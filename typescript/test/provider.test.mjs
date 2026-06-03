import assert from "node:assert/strict";
import { createServer } from "node:http";
import { after, test } from "node:test";
import { NHIProvider, createFetch, detectEnvironment, fetchWithZinetic } from "../dist/index.js";

const server = createServer((req, res) => {
  if (req.url === "/resource") {
    assert.equal(req.headers.authorization, "Bearer bearer-token");
    assert.ok(req.headers.dpop);
    res.writeHead(200).end("ok");
    return;
  }
  if (req.url === "/status") {
    res.writeHead(200, { "content-type": "application/json" });
    res.end(JSON.stringify({ available: true, key_id: "hw-key-1", provider: "mock" }));
    return;
  }
  if (req.url === "/attest" && req.method === "POST") {
    res.writeHead(200, { "content-type": "application/json" });
    res.end(JSON.stringify({ key_id: "hw-key-1", format: "mock", data: "attestation" }));
    return;
  }
  if (req.url !== "/api/v1/decision/exchange" || req.method !== "POST") {
    res.writeHead(404).end();
    return;
  }
  assert.ok(req.headers.dpop);
  let body = "";
  req.setEncoding("utf8");
  req.on("data", (chunk) => {
    body += chunk;
  });
  req.on("end", () => {
    const payload = JSON.parse(body);
    assert.equal(payload.exchange_version, 2);
    assert.equal(payload.target, "postgres-staging");
    assert.equal(payload.tenant_id, "tenant-123");
    assert.equal(payload.hardware_mode, "auto");
    assert.equal(payload.hardware_key_id, "hw-key-1");
    assert.equal(payload.credential_encryption_jwk.kty, "EC");
    res.writeHead(200, { "content-type": "application/json" });
    res.end(JSON.stringify({
      credentials: { token: "bearer-token", username: "app", password: "secret" },
      ttl_seconds: 300,
      policy_signature: "dev",
      policy_version: "policy-v1",
      policy_signing_key_id: "key-1",
      audit_id: "audit-123",
      transaction_hash: "tx-abc",
      ledger_anchor_hash: "ledger-def"
    }));
  });
});

await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
after(() => new Promise((resolve) => server.close(resolve)));

test("detectEnvironment recognizes GitLab CI", () => {
  process.env.GITLAB_CI = "true";
  assert.equal(detectEnvironment(), "gitlab-ci");
  delete process.env.GITLAB_CI;
});

test("NHIProvider exchanges credentials in memory and injects fetch auth", async () => {
  process.env.ZINETIC_ACCESS_TOKEN = "local-session";
  const { port } = server.address();
  const events = [];
  const provider = new NHIProvider({
    backendURL: `http://127.0.0.1:${port}`,
    target: "postgres-staging",
    tenantID: "tenant-123",
    environment: "local",
    allowPlaintextResponse: true,
    hardwareMode: "auto",
    hardwareAgentURL: `http://127.0.0.1:${port}`,
    refreshThreshold: 0.25,
    eventCallback: (event) => events.push(event)
  });
  await provider.start();
  try {
    assert.equal(provider.getCredential("password"), "secret");
    assert.deepEqual(await provider.pgConfig({ host: "db.local" }), {
      host: "db.local",
      user: "app",
      password: "secret"
    });

    const wrappedFetch = createFetch(provider);
    const resource = await wrappedFetch(`http://127.0.0.1:${port}/resource`);
    assert.equal(resource.status, 200);
    const aliasFetch = fetchWithZinetic(provider);
    const aliasResource = await aliasFetch(`http://127.0.0.1:${port}/resource`);
    assert.equal(aliasResource.status, 200);
    assert.equal(provider.getCredentials().token, "bearer-token");
    assert.deepEqual(provider.getMetadata(), {
      auditID: "audit-123",
      policyVersion: "policy-v1",
      policySigningKeyID: "key-1",
      transactionHash: "tx-abc",
      ledgerAnchorHash: "ledger-def",
      target: "postgres-staging",
      expiresAt: provider.getMetadata().expiresAt,
      ttlSeconds: 300,
      policySignature: "dev"
    });
    assert.equal(events[0].type, "exchange_succeeded");
    assert.equal(events[0].auditID, "audit-123");
    assert.equal(events[0].hardwareAvailable, true);
  } finally {
    provider.stop();
    delete process.env.ZINETIC_ACCESS_TOKEN;
  }
});

test("NHIProvider requires policy key when signature verification is required", () => {
  assert.throws(() => new NHIProvider({
    backendURL: "https://api.zinetic.net",
    target: "postgres",
    requirePolicySignature: true
  }), /policy signature verification requires policyPublicKey/);
});

test("NHIProvider verifies Ed25519 policy signatures", async () => {
  process.env.ZINETIC_ACCESS_TOKEN = "local-session";
  const keyPair = await crypto.subtle.generateKey("Ed25519", true, ["sign", "verify"]);
  const policyPublicKey = Buffer.from(await crypto.subtle.exportKey("raw", keyPair.publicKey)).toString("base64");
  const expiresAt = new Date(Date.now() + 300_000).toISOString();
  const response = {
    credential_type: "env",
    credentials: { token: "signed-token" },
    expires_at: expiresAt,
    ttl_seconds: 300,
    audit_id: "audit-signed",
    policy_version: "policy-signed",
    policy_signing_key_id: "key-signed",
    transaction_hash: "tx-signed",
    ledger_anchor_hash: "ledger-signed"
  };
  const envelope = {
    credential_type: response.credential_type,
    credentials: response.credentials,
    encrypted_credentials: undefined,
    expires_at: response.expires_at,
    ttl_seconds: response.ttl_seconds,
    audit_id: response.audit_id,
    policy_version: response.policy_version,
    policy_signing_key_id: response.policy_signing_key_id,
    transaction_hash: response.transaction_hash,
    ledger_anchor_hash: response.ledger_anchor_hash
  };
  const signature = await crypto.subtle.sign("Ed25519", keyPair.privateKey, new TextEncoder().encode(JSON.stringify(envelope)));
  const signedServer = createServer((req, res) => {
    if (req.url !== "/api/v1/decision/exchange" || req.method !== "POST") {
      res.writeHead(404).end();
      return;
    }
    res.writeHead(200, { "content-type": "application/json" });
    res.end(JSON.stringify({
      ...response,
      policy_signature: Buffer.from(signature).toString("base64url")
    }));
  });
  await new Promise((resolve) => signedServer.listen(0, "127.0.0.1", resolve));
  try {
    const { port } = signedServer.address();
    const provider = new NHIProvider({
      backendURL: `http://127.0.0.1:${port}`,
      target: "signed",
      environment: "local",
      allowPlaintextResponse: true,
      policyPublicKey,
      requirePolicySignature: true
    });
    await provider.start();
    assert.equal(provider.getCredential("token"), "signed-token");
    assert.equal(provider.getMetadata().policySigningKeyID, "key-signed");
    provider.stop();
  } finally {
    await new Promise((resolve) => signedServer.close(resolve));
    delete process.env.ZINETIC_ACCESS_TOKEN;
  }
});

test("NHIProvider reports unsupported required hardware mode clearly", async () => {
  process.env.ZINETIC_ACCESS_TOKEN = "local-session";
  const { port } = server.address();
  const provider = new NHIProvider({
    backendURL: `http://127.0.0.1:${port}`,
    target: "postgres-staging",
    environment: "local",
    allowPlaintextResponse: true,
    hardwareMode: "required"
  });
  await assert.rejects(() => provider.start(), /hardware-bound NHI is required/);
  delete process.env.ZINETIC_ACCESS_TOKEN;
});

test("NHIProvider rejects browser runtime", () => {
  const original = globalThis.process;
  Object.defineProperty(globalThis, "process", { value: undefined, configurable: true });
  try {
    assert.throws(() => new NHIProvider({
      backendURL: "https://api.zinetic.net",
      target: "postgres"
    }), /requires a Node\.js server runtime/);
  } finally {
    Object.defineProperty(globalThis, "process", { value: original, configurable: true });
  }
});
