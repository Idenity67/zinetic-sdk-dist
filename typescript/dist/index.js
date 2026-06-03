import { readFile } from "node:fs/promises";
import { createHash, createHmac } from "node:crypto";
const encoder = new TextEncoder();
const dynamicImport = new Function("specifier", "return import(specifier)");
export class NHIProvider {
    cfg;
    fetchImpl;
    dpopKey;
    credential;
    renewTimer;
    stopped = true;
    serverNonce;
    hardwareStatus;
    constructor(cfg) {
        assertNodeRuntime();
        if (!cfg.backendURL) {
            throw new Error("backendURL is required");
        }
        if (!cfg.target) {
            throw new Error("target is required");
        }
        if (cfg.requirePolicySignature && !cfg.policyPublicKey?.trim()) {
            throw new Error("policy signature verification requires policyPublicKey");
        }
        const refreshThreshold = cfg.refreshThreshold ?? 0.33;
        if (refreshThreshold <= 0 || refreshThreshold >= 1) {
            throw new Error("refreshThreshold must be greater than 0 and less than 1");
        }
        const hardwareMode = normalizeHardwareMode(cfg.hardwareMode);
        const backendURL = normalizeBackendURL(cfg.backendURL);
        this.cfg = {
            ...cfg,
            backendURL,
            target: cfg.target,
            audience: cfg.audience || backendURL,
            hardwareMode,
            refreshThreshold,
            exchangeTimeoutMs: cfg.exchangeTimeoutMs ?? 30_000
        };
        this.fetchImpl = cfg.fetch || fetch;
    }
    async start() {
        this.stopped = false;
        this.dpopKey = await generateDPoPKey();
        try {
            this.credential = await this.exchange();
            this.emit(credentialEvent("exchange_succeeded", this.cfg, this.credential, this.hardwareAvailable()));
            this.scheduleRenewal();
        }
        catch (error) {
            this.emit({
                type: "exchange_failed",
                target: this.cfg.target,
                error,
                hardwareMode: this.cfg.hardwareMode,
                hardwareAvailable: this.hardwareAvailable()
            });
            throw error;
        }
    }
    stop() {
        this.stopped = true;
        if (this.renewTimer) {
            clearTimeout(this.renewTimer);
            this.renewTimer = undefined;
        }
        zeroizeCredential(this.credential);
        this.credential = undefined;
    }
    getCredential(key) {
        const credential = this.currentCredential();
        if (!(key in credential.values)) {
            throw new Error(`credential key ${key} not found`);
        }
        return credential.values[key];
    }
    getCredentials() {
        return { ...this.currentCredential().values };
    }
    getMetadata() {
        const credential = this.currentCredential();
        return {
            auditID: credential.auditID,
            policyVersion: credential.policyVersion,
            policySigningKeyID: credential.policySigningKeyID,
            transactionHash: credential.transactionHash,
            ledgerAnchorHash: credential.ledgerAnchorHash,
            target: credential.target,
            expiresAt: new Date(credential.expiresAt),
            ttlSeconds: credential.ttlSeconds,
            policySignature: credential.policySignature
        };
    }
    async fetch(input, init = {}, tokenKey = "token") {
        const token = this.getCredential(tokenKey);
        const request = new Request(input, init);
        request.headers.set("Authorization", `Bearer ${token}`);
        if (this.dpopKey) {
            request.headers.set("DPoP", await createDPoPProof(this.dpopKey, request.method, request.url, token, this.serverNonce));
        }
        return this.fetchImpl(request);
    }
    async pgConfig(base, opts = {}) {
        const creds = this.getCredentials();
        return {
            ...base,
            user: creds[opts.usernameKey || "username"] || base.user,
            password: creds[opts.passwordKey || "password"] || base.password
        };
    }
    async mysqlConfig(base, opts = {}) {
        return this.pgConfig(base, opts);
    }
    currentCredential() {
        if (!this.credential || this.credential.expiresAt.getTime() <= Date.now()) {
            throw new Error("no valid credential available");
        }
        return this.credential;
    }
    async exchange() {
        const environment = this.cfg.environment || detectEnvironment();
        const attestationToken = await fetchAttestationToken(environment, this.cfg.audience, this.fetchImpl);
        const dpopKey = this.dpopKey || await generateDPoPKey();
        this.dpopKey = dpopKey;
        const encryptionKey = await generateEncryptionKey();
        const endpoint = `${this.cfg.backendURL}/api/v1/decision/exchange`;
        const hardware = await this.hardwarePayload();
        const payload = {
            environment,
            attestation_token: attestationToken,
            attestation_token_type: tokenTypeForEnvironment(environment),
            target: this.cfg.target,
            tenant_id: this.cfg.tenantID,
            dpop_jwk: await exportPublicJWK(dpopKey),
            credential_encryption_jwk: await exportPublicJWK(encryptionKey),
            exchange_version: 2,
            client_metadata: collectMetadata(environment),
            hardware_mode: this.cfg.hardwareMode,
            ...hardware
        };
        const body = JSON.stringify(payload);
        const resp = await this.exchangeFetch(endpoint, dpopKey, body);
        const retryNonce = resp.status === 401 ? resp.headers.get("dpop-nonce") : "";
        const finalResp = retryNonce ? await this.retryExchangeWithNonce(endpoint, dpopKey, body, retryNonce) : resp;
        if (!finalResp.ok) {
            throw new Error(`exchange failed (HTTP ${finalResp.status}): ${await finalResp.text()}`);
        }
        const pkg = await finalResp.json();
        let values = pkg.credentials || {};
        if (pkg.encrypted_credentials) {
            values = await decryptCredentials(encryptionKey, pkg.encrypted_credentials);
            pkg.credentials = values;
        }
        else if (!this.cfg.allowPlaintextResponse && environment !== "local") {
            throw new Error("exchange returned plaintext credentials; enable allowPlaintextResponse only for local development");
        }
        if (Object.keys(values).length === 0) {
            throw new Error("exchange returned empty credentials");
        }
        await verifyPolicySignature(pkg, this.cfg.policyPublicKey, Boolean(this.cfg.requirePolicySignature));
        const expiresAt = pkg.expires_at ? new Date(pkg.expires_at) : new Date(Date.now() + (pkg.ttl_seconds || 300) * 1000);
        const next = {
            values,
            expiresAt,
            ttlSeconds: pkg.ttl_seconds,
            policySignature: pkg.policy_signature,
            policyVersion: pkg.policy_version,
            policySigningKeyID: pkg.policy_signing_key_id,
            auditID: pkg.audit_id,
            transactionHash: pkg.transaction_hash,
            ledgerAnchorHash: pkg.ledger_anchor_hash,
            target: this.cfg.target
        };
        zeroizeCredential(this.credential);
        return next;
    }
    async exchangeFetch(endpoint, dpopKey, body) {
        const controller = new AbortController();
        const timeout = setTimeout(() => controller.abort(), this.cfg.exchangeTimeoutMs);
        try {
            return await this.fetchImpl(endpoint, {
                method: "POST",
                signal: controller.signal,
                headers: {
                    "Content-Type": "application/json",
                    "DPoP": await createDPoPProof(dpopKey, "POST", endpoint, "", this.serverNonce)
                },
                body
            });
        }
        finally {
            clearTimeout(timeout);
        }
    }
    async retryExchangeWithNonce(endpoint, dpopKey, body, nonce) {
        this.serverNonce = nonce;
        return this.exchangeFetch(endpoint, dpopKey, body);
    }
    async hardwarePayload() {
        if (this.cfg.hardwareMode === "off") {
            return {};
        }
        if (!this.cfg.hardwareAgentURL) {
            if (this.cfg.hardwareMode === "required") {
                throw new Error("hardware-bound NHI is required but hardwareAgentURL is not configured");
            }
            return {};
        }
        const base = this.cfg.hardwareAgentURL.replace(/\/$/, "");
        try {
            const statusResp = await this.fetchImpl(`${base}/status`);
            if (!statusResp.ok) {
                throw new Error(`hardware agent status failed (HTTP ${statusResp.status})`);
            }
            const status = await statusResp.json();
            if (!status.available) {
                if (this.cfg.hardwareMode === "required") {
                    throw new Error("hardware-bound NHI is required but the hardware agent is unavailable");
                }
                return {};
            }
            this.hardwareStatus = status;
            const attestResp = await this.fetchImpl(`${base}/attest`, { method: "POST" });
            if (!attestResp.ok) {
                throw new Error(`hardware agent attestation failed (HTTP ${attestResp.status})`);
            }
            const attestation = await attestResp.json();
            return {
                hardware_key_id: String(status.key_id || attestation.key_id || ""),
                hardware_attestation: {
                    status,
                    attestation
                }
            };
        }
        catch (error) {
            if (this.cfg.hardwareMode === "required") {
                throw error;
            }
            return {};
        }
    }
    hardwareAvailable() {
        return Boolean(this.hardwareStatus?.available);
    }
    scheduleRenewal() {
        if (this.stopped || !this.credential) {
            return;
        }
        const remaining = Math.max(0, this.credential.expiresAt.getTime() - Date.now());
        const delay = Math.max(1000, Math.floor(remaining * (1 - this.cfg.refreshThreshold)));
        this.renewTimer = setTimeout(() => {
            void this.renewWithBackoff();
        }, delay);
    }
    async renewWithBackoff() {
        for (let attempt = 1; !this.stopped && attempt <= 10; attempt++) {
            try {
                this.credential = await this.exchange();
                this.emit(credentialEvent("renewal_succeeded", this.cfg, this.credential, this.hardwareAvailable(), attempt));
                this.scheduleRenewal();
                return;
            }
            catch (error) {
                this.emit({
                    type: "renewal_failed",
                    target: this.cfg.target,
                    attempt,
                    error,
                    hardwareMode: this.cfg.hardwareMode,
                    hardwareAvailable: this.hardwareAvailable()
                });
                await sleep(backoffWithJitter(attempt));
            }
        }
    }
    emit(event) {
        this.cfg.eventCallback?.(event);
    }
}
export function createFetch(provider, tokenKey = "token") {
    return (input, init) => provider.fetch(input, init, tokenKey);
}
export function fetchWithZinetic(provider, tokenKey = "token") {
    return createFetch(provider, tokenKey);
}
export async function createPgPool(provider, options = {}, credentialKeys = {}) {
    const mod = await dynamicImport("pg");
    const Pool = mod.Pool || mod.default?.Pool;
    if (!Pool) {
        throw new Error("pg peer dependency does not export Pool");
    }
    return new Pool(await provider.pgConfig(options, credentialKeys));
}
export async function createMysqlPool(provider, options = {}, credentialKeys = {}) {
    let mod;
    try {
        mod = await dynamicImport("mysql2/promise");
    }
    catch {
        mod = await dynamicImport("mysql2");
    }
    const createPool = mod.createPool || mod.default?.createPool;
    if (!createPool) {
        throw new Error("mysql2 peer dependency does not export createPool");
    }
    return createPool(await provider.mysqlConfig(options, credentialKeys));
}
export function detectEnvironment() {
    assertNodeRuntime();
    if (process.env.ACTIONS_ID_TOKEN_REQUEST_URL && process.env.ACTIONS_ID_TOKEN_REQUEST_TOKEN) {
        return "github-actions";
    }
    if (process.env.GITLAB_CI === "true" || process.env.CI_PROJECT_PATH || process.env.CI_SERVER_URL) {
        return "gitlab-ci";
    }
    if (process.env.AWS_WEB_IDENTITY_TOKEN_FILE || process.env.AWS_CONTAINER_CREDENTIALS_FULL_URI || process.env.AWS_CONTAINER_CREDENTIALS_RELATIVE_URI || process.env.AWS_EXECUTION_ENV) {
        return "aws";
    }
    if (process.env.KUBERNETES_SERVICE_HOST) {
        return "kubernetes";
    }
    if (process.env.GCE_METADATA_HOST || process.env.GOOGLE_CLOUD_PROJECT) {
        return "gcp";
    }
    return "local";
}
async function fetchAttestationToken(environment, audience, fetchImpl) {
    switch (environment) {
        case "github-actions":
            return fetchGitHubToken(audience, fetchImpl);
        case "gitlab-ci":
            return firstEnv("ZINETIC_OIDC_TOKEN", "ZINETIC_ID_TOKEN", "ZINETIC_GITLAB_OIDC_TOKEN", "CI_JOB_JWT_V2", "CI_JOB_JWT");
        case "kubernetes":
            return (await readFile("/var/run/secrets/kubernetes.io/serviceaccount/token", "utf8")).trim();
        case "aws":
            return fetchAWSIdentityDocument(fetchImpl);
        case "gcp":
            return fetchGCPIdentityToken(audience, fetchImpl);
        case "local":
            return firstEnv("ZINETIC_ACCESS_TOKEN");
    }
}
async function fetchGitHubToken(audience, fetchImpl) {
    const requestURL = firstEnv("ACTIONS_ID_TOKEN_REQUEST_URL");
    const requestToken = firstEnv("ACTIONS_ID_TOKEN_REQUEST_TOKEN");
    const url = new URL(requestURL);
    if (url.protocol !== "https:" || !url.hostname.endsWith("actions.githubusercontent.com")) {
        throw new Error("GitHub OIDC request URL is not trusted");
    }
    url.searchParams.set("audience", audience);
    const resp = await fetchImpl(url, { headers: { Authorization: `bearer ${requestToken}` } });
    if (!resp.ok) {
        throw new Error(`GitHub OIDC token request failed (HTTP ${resp.status})`);
    }
    const body = await resp.json();
    if (!body.value) {
        throw new Error("empty token in GitHub OIDC response");
    }
    return body.value;
}
async function fetchAWSIdentityDocument(fetchImpl) {
    if (process.env.AWS_WEB_IDENTITY_TOKEN_FILE) {
        const token = (await readFile(process.env.AWS_WEB_IDENTITY_TOKEN_FILE, "utf8")).trim();
        if (!token) {
            throw new Error("AWS web identity token file is empty");
        }
        return token;
    }
    if (process.env.AWS_CONTAINER_CREDENTIALS_FULL_URI || process.env.AWS_CONTAINER_CREDENTIALS_RELATIVE_URI) {
        return fetchECSIdentityEnvelope(fetchImpl);
    }
    const tokenResp = await fetchImpl("http://169.254.169.254/latest/api/token", {
        method: "PUT",
        headers: { "X-aws-ec2-metadata-token-ttl-seconds": "21600" }
    });
    if (!tokenResp.ok) {
        throw new Error(`IMDS token request failed (HTTP ${tokenResp.status})`);
    }
    const token = await tokenResp.text();
    const headers = { "X-aws-ec2-metadata-token": token };
    const [documentResp, signatureResp] = await Promise.all([
        fetchImpl("http://169.254.169.254/latest/dynamic/instance-identity/document", { headers }),
        fetchImpl("http://169.254.169.254/latest/dynamic/instance-identity/pkcs7", { headers })
    ]);
    if (!documentResp.ok || !signatureResp.ok) {
        throw new Error("AWS identity document request failed");
    }
    return JSON.stringify({ document: await documentResp.text(), signature: await signatureResp.text() });
}
async function fetchECSIdentityEnvelope(fetchImpl) {
    const credentialsURL = process.env.AWS_CONTAINER_CREDENTIALS_FULL_URI ||
        `http://169.254.170.2${process.env.AWS_CONTAINER_CREDENTIALS_RELATIVE_URI || ""}`;
    const headers = {};
    if (process.env.AWS_CONTAINER_AUTHORIZATION_TOKEN) {
        headers.Authorization = process.env.AWS_CONTAINER_AUTHORIZATION_TOKEN;
    }
    const credsResp = await fetchImpl(credentialsURL, { headers });
    if (!credsResp.ok) {
        throw new Error(`ECS task credentials request failed (HTTP ${credsResp.status})`);
    }
    return buildAWSSTSAttestation(await credsResp.json(), firstEnvOptional("AWS_REGION", "AWS_DEFAULT_REGION") || "us-east-1", "ecs", new Date());
}
async function fetchGCPIdentityToken(audience, fetchImpl) {
    const url = new URL("http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity");
    url.searchParams.set("audience", audience);
    url.searchParams.set("format", "full");
    const resp = await fetchImpl(url, { headers: { "Metadata-Flavor": "Google" } });
    if (!resp.ok) {
        throw new Error(`GCP identity token request failed (HTTP ${resp.status})`);
    }
    const token = await resp.text();
    if (!token) {
        throw new Error("empty GCP identity token");
    }
    return token;
}
function tokenTypeForEnvironment(environment) {
    if (environment === "github-actions" || environment === "gitlab-ci" || environment === "gcp") {
        return "oidc";
    }
    if (environment === "aws") {
        if (process.env.AWS_WEB_IDENTITY_TOKEN_FILE) {
            return "oidc";
        }
        if (process.env.AWS_CONTAINER_CREDENTIALS_FULL_URI || process.env.AWS_CONTAINER_CREDENTIALS_RELATIVE_URI) {
            return "aws_sts";
        }
        return "instance_identity";
    }
    if (environment === "local") {
        return "local_session";
    }
    return "jwt";
}
function collectMetadata(environment) {
    if (environment === "github-actions") {
        return compact({
            repo: process.env.GITHUB_REPOSITORY,
            workflow: process.env.GITHUB_WORKFLOW,
            run_id: process.env.GITHUB_RUN_ID,
            sha: process.env.GITHUB_SHA
        });
    }
    if (environment === "gitlab-ci") {
        return compact({
            repo: process.env.CI_PROJECT_PATH,
            workflow: process.env.CI_PIPELINE_SOURCE,
            run_id: process.env.CI_PIPELINE_ID,
            sha: process.env.CI_COMMIT_SHA
        });
    }
    if (environment === "aws") {
        return compact({ cloud_provider: "aws", region: process.env.AWS_REGION || process.env.AWS_DEFAULT_REGION });
    }
    if (environment === "gcp") {
        return compact({ cloud_provider: "gcp", project: process.env.GOOGLE_CLOUD_PROJECT });
    }
    return {};
}
function compact(values) {
    return Object.fromEntries(Object.entries(values).filter((entry) => Boolean(entry[1])));
}
async function verifyPolicySignature(pkg, publicKey, required) {
    const key = publicKey?.trim();
    if (!key) {
        if (required) {
            throw new Error("policy signature verification requires policyPublicKey");
        }
        return;
    }
    if (!pkg.policy_signature) {
        throw new Error("exchange response is missing policy signature");
    }
    const rawKey = decodePolicyPublicKey(key);
    const cryptoKey = await crypto.subtle.importKey("raw", rawKey, "Ed25519", false, ["verify"]);
    const envelope = {
        credential_type: pkg.credential_type,
        credentials: pkg.credentials,
        encrypted_credentials: pkg.encrypted_credentials,
        expires_at: pkg.expires_at,
        ttl_seconds: pkg.ttl_seconds,
        audit_id: pkg.audit_id,
        policy_version: pkg.policy_version,
        policy_signing_key_id: pkg.policy_signing_key_id,
        transaction_hash: pkg.transaction_hash,
        ledger_anchor_hash: pkg.ledger_anchor_hash
    };
    const ok = await crypto.subtle.verify("Ed25519", cryptoKey, decodeBase64Flexible(pkg.policy_signature), encoder.encode(JSON.stringify(envelope)));
    if (!ok) {
        throw new Error("policy signature verification failed");
    }
}
function decodePolicyPublicKey(raw) {
    const pem = raw.match(/-----BEGIN [^-]+-----([\s\S]+?)-----END [^-]+-----/);
    const body = (pem ? pem[1] : raw).replace(/\s+/g, "");
    const decoded = decodeBase64Flexible(body);
    if (decoded.byteLength !== 32) {
        throw new Error("Ed25519 policy public key must be 32 bytes");
    }
    return decoded;
}
function credentialEvent(type, cfg, credential, hardwareAvailable, attempt) {
    return {
        type,
        target: cfg.target,
        auditID: credential.auditID,
        policyVersion: credential.policyVersion,
        transactionHash: credential.transactionHash,
        ledgerAnchorHash: credential.ledgerAnchorHash,
        expiresAt: credential.expiresAt,
        attempt,
        hardwareMode: cfg.hardwareMode,
        hardwareAvailable
    };
}
async function generateDPoPKey() {
    return crypto.subtle.generateKey({ name: "ECDSA", namedCurve: "P-256" }, false, ["sign", "verify"]);
}
async function generateEncryptionKey() {
    return crypto.subtle.generateKey({ name: "ECDH", namedCurve: "P-256" }, false, ["deriveBits"]);
}
async function exportPublicJWK(keyPair) {
    const jwk = await crypto.subtle.exportKey("jwk", keyPair.publicKey);
    return { kty: jwk.kty, crv: jwk.crv, x: jwk.x, y: jwk.y };
}
async function createDPoPProof(keyPair, method, uri, accessToken = "", nonce = "") {
    const header = { typ: "dpop+jwt", alg: "ES256", jwk: await exportPublicJWK(keyPair) };
    const claims = {
        jti: base64url(crypto.getRandomValues(new Uint8Array(16))),
        htm: method.toUpperCase(),
        htu: uri,
        iat: Math.floor(Date.now() / 1000),
        exp: Math.floor(Date.now() / 1000) + 60
    };
    if (nonce) {
        claims.nonce = nonce;
    }
    if (accessToken) {
        claims.ath = base64url(await crypto.subtle.digest("SHA-256", encoder.encode(accessToken)));
    }
    const signingInput = `${base64url(JSON.stringify(header))}.${base64url(JSON.stringify(claims))}`;
    const signature = await crypto.subtle.sign({ name: "ECDSA", hash: "SHA-256" }, keyPair.privateKey, encoder.encode(signingInput));
    return `${signingInput}.${base64url(signature)}`;
}
async function decryptCredentials(keyPair, pkg) {
    if (pkg.alg !== "ECDH-ES+HKDF-SHA256" || pkg.enc !== "A256GCM") {
        throw new Error(`unsupported encrypted credential package: ${pkg.alg}/${pkg.enc}`);
    }
    const peer = await crypto.subtle.importKey("jwk", pkg.epk, { name: "ECDH", namedCurve: "P-256" }, false, []);
    const sharedSecret = await crypto.subtle.deriveBits({ name: "ECDH", public: peer }, keyPair.privateKey, 256);
    const nonce = base64urlDecode(pkg.nonce);
    const hkdfKey = await crypto.subtle.importKey("raw", sharedSecret, "HKDF", false, ["deriveKey"]);
    const aesKey = await crypto.subtle.deriveKey({ name: "HKDF", hash: "SHA-256", salt: nonce, info: encoder.encode("zinetic-nhi-credential-package-v2") }, hkdfKey, { name: "AES-GCM", length: 256 }, false, ["decrypt"]);
    const plaintext = await crypto.subtle.decrypt({ name: "AES-GCM", iv: nonce }, aesKey, base64urlDecode(pkg.ciphertext));
    return JSON.parse(new TextDecoder().decode(plaintext));
}
function normalizeBackendURL(raw) {
    const url = new URL(raw);
    const host = url.hostname.replace(/^\[|\]$/g, "");
    if (url.protocol === "http:" && !["localhost", "127.0.0.1", "::1"].includes(host)) {
        throw new Error("backendURL must use HTTPS except for localhost development URLs");
    }
    url.pathname = url.pathname.replace(/\/?(api\/v1|v1)\/?$/, "");
    url.search = "";
    url.hash = "";
    return url.toString().replace(/\/$/, "");
}
function normalizeHardwareMode(raw) {
    const mode = raw || "auto";
    if (mode !== "auto" && mode !== "required" && mode !== "off") {
        throw new Error("hardwareMode must be one of auto, required, or off");
    }
    return mode;
}
function assertNodeRuntime() {
    if (typeof process === "undefined" || !process.versions?.node) {
        throw new Error("@zinetic/sdk requires a Node.js server runtime and does not run in browsers");
    }
}
function firstEnv(...names) {
    for (const name of names) {
        const value = process.env[name]?.trim();
        if (value) {
            return value;
        }
    }
    throw new Error(`missing required environment variable: ${names.join(" or ")}`);
}
function firstEnvOptional(...names) {
    for (const name of names) {
        const value = process.env[name]?.trim();
        if (value) {
            return value;
        }
    }
    return "";
}
function buildAWSSTSAttestation(creds, region, runtime, now) {
    const accessKey = String(creds.AccessKeyId || "");
    const secretKey = String(creds.SecretAccessKey || "");
    const sessionToken = String(creds.Token || "");
    if (!accessKey || !secretKey) {
        throw new Error("AWS credentials are missing access key or secret key");
    }
    const host = `sts.${region}.amazonaws.com`;
    const scopeDate = awsDate(now).slice(0, 8);
    const amzDate = awsDate(now);
    const scope = `${scopeDate}/${region}/sts/aws4_request`;
    const params = new URLSearchParams({
        Action: "GetCallerIdentity",
        Version: "2011-06-15",
        "X-Amz-Algorithm": "AWS4-HMAC-SHA256",
        "X-Amz-Credential": `${accessKey}/${scope}`,
        "X-Amz-Date": amzDate,
        "X-Amz-Expires": "60",
        "X-Amz-SignedHeaders": "host"
    });
    if (sessionToken) {
        params.set("X-Amz-Security-Token", sessionToken);
    }
    const canonicalQuery = params.toString();
    const canonicalRequest = ["GET", "/", canonicalQuery, `host:${host}\n`, "host", "UNSIGNED-PAYLOAD"].join("\n");
    const requestHash = createHash("sha256").update(canonicalRequest).digest("hex");
    const stringToSign = ["AWS4-HMAC-SHA256", amzDate, scope, requestHash].join("\n");
    const signature = hmac(signingKey(secretKey, scopeDate, region), stringToSign).toString("hex");
    params.set("X-Amz-Signature", signature);
    return JSON.stringify({
        source: "aws_sts",
        runtime,
        region,
        url: `https://${host}/?${params.toString()}`
    });
}
function awsDate(now) {
    return now.toISOString().replace(/[:-]|\.\d{3}/g, "");
}
function signingKey(secret, date, region) {
    const kDate = hmac(Buffer.from(`AWS4${secret}`, "utf8"), date);
    const kRegion = hmac(kDate, region);
    const kService = hmac(kRegion, "sts");
    return hmac(kService, "aws4_request");
}
function hmac(key, data) {
    return createHmac("sha256", key).update(data).digest();
}
function zeroizeCredential(credential) {
    if (!credential) {
        return;
    }
    for (const key of Object.keys(credential.values)) {
        credential.values[key] = "\0".repeat(credential.values[key].length);
        delete credential.values[key];
    }
}
function base64url(input) {
    const bytes = typeof input === "string" ? Buffer.from(input) : Buffer.from(input instanceof Uint8Array ? input : new Uint8Array(input));
    return bytes.toString("base64url");
}
function base64urlDecode(input) {
    const buf = Buffer.from(input, "base64url");
    return buf.buffer.slice(buf.byteOffset, buf.byteOffset + buf.byteLength);
}
function decodeBase64Flexible(input) {
    const normalized = input.replace(/\s+/g, "");
    const buf = Buffer.from(normalized, normalized.includes("-") || normalized.includes("_") ? "base64url" : "base64");
    return buf.buffer.slice(buf.byteOffset, buf.byteOffset + buf.byteLength);
}
function backoffWithJitter(attempt) {
    const base = Math.min(60_000, 500 * 2 ** Math.max(0, attempt - 1));
    return base + Math.floor(Math.random() * base);
}
function sleep(ms) {
    return new Promise((resolve) => setTimeout(resolve, ms));
}
