# Security Policy

## Supported Versions

Only the latest released minor version receives security fixes.

## Reporting Vulnerabilities

Report suspected vulnerabilities privately through GitHub private vulnerability reporting or by emailing the Zinetic security contact published on the organization profile.

Do not open public issues for vulnerabilities, leaked credentials, bypasses, or exploit details.

## Public SDK Rules

This repository must not contain real access tokens, refresh tokens, API keys, DPoP private keys, signing keys, customer tenant identifiers, `.env` files, `.npmrc` files with tokens, generated credential responses, or production e2e secrets.

Examples and tests must use placeholders or locally generated material only.

The SDK is public client code. All authorization, tenant isolation, DPoP validation, policy enforcement, rate limiting, audit, and replay protection must be enforced by the Zinetic backend.

## Required Release Gates

Every release must pass:

- Go tests and race tests
- TypeScript build and tests
- `go vet`
- `govulncheck`
- `gosec`
- `gitleaks`
- npm production audit
- backend OpenAPI contract tests

## Credential Handling

Use GitHub Actions secrets, npm trusted publishing, OIDC, short-lived tokens, and environment-scoped credentials. Never commit credentials or embed them in source, examples, workflows, package artifacts, or generated files.
