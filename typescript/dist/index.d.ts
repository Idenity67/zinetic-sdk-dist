export type Environment = "github-actions" | "gitlab-ci" | "kubernetes" | "aws" | "gcp" | "local";
export type HardwareMode = "auto" | "required" | "off";
export interface CredentialMetadata {
    auditID?: string;
    policyVersion?: string;
    policySigningKeyID?: string;
    transactionHash?: string;
    ledgerAnchorHash?: string;
    target: string;
    expiresAt: Date;
    ttlSeconds?: number;
    policySignature?: string;
}
export interface Credential extends CredentialMetadata {
    values: Record<string, string>;
}
export interface ProviderEvent {
    type: "exchange_succeeded" | "exchange_failed" | "renewal_succeeded" | "renewal_failed";
    target: string;
    auditID?: string;
    policyVersion?: string;
    transactionHash?: string;
    ledgerAnchorHash?: string;
    expiresAt?: Date;
    attempt?: number;
    error?: unknown;
    hardwareMode: HardwareMode;
    hardwareAvailable: boolean;
}
export interface ProviderConfig {
    backendURL: string;
    target: string;
    tenantID?: string;
    audience?: string;
    environment?: Environment;
    allowPlaintextResponse?: boolean;
    policyPublicKey?: string;
    requirePolicySignature?: boolean;
    hardwareMode?: HardwareMode;
    hardwareAgentURL?: string;
    refreshThreshold?: number;
    eventCallback?: (event: ProviderEvent) => void;
    exchangeTimeoutMs?: number;
    fetch?: typeof fetch;
}
export interface CredentialKeyOptions {
    usernameKey?: string;
    passwordKey?: string;
}
export interface PgConfigOptions extends CredentialKeyOptions {
}
export interface MysqlConfigOptions extends CredentialKeyOptions {
}
export declare class NHIProvider {
    private readonly cfg;
    private readonly fetchImpl;
    private dpopKey?;
    private credential?;
    private renewTimer?;
    private stopped;
    private serverNonce?;
    private hardwareStatus?;
    constructor(cfg: ProviderConfig);
    start(): Promise<void>;
    stop(): void;
    getCredential(key: string): string;
    getCredentials(): Record<string, string>;
    getMetadata(): CredentialMetadata;
    fetch(input: RequestInfo | URL, init?: RequestInit, tokenKey?: string): Promise<Response>;
    pgConfig<T extends Record<string, unknown>>(base: T, opts?: PgConfigOptions): Promise<T & {
        user?: string;
        password?: string;
    }>;
    mysqlConfig<T extends Record<string, unknown>>(base: T, opts?: MysqlConfigOptions): Promise<T & {
        user?: string;
        password?: string;
    }>;
    private currentCredential;
    private exchange;
    private exchangeFetch;
    private retryExchangeWithNonce;
    private hardwarePayload;
    private hardwareAvailable;
    private scheduleRenewal;
    private renewWithBackoff;
    private emit;
}
export declare function createFetch(provider: NHIProvider, tokenKey?: string): typeof fetch;
export declare function fetchWithZinetic(provider: NHIProvider, tokenKey?: string): typeof fetch;
export declare function createPgPool(provider: NHIProvider, options?: Record<string, unknown>, credentialKeys?: PgConfigOptions): Promise<unknown>;
export declare function createMysqlPool(provider: NHIProvider, options?: Record<string, unknown>, credentialKeys?: MysqlConfigOptions): Promise<unknown>;
export declare function detectEnvironment(): Environment;
