# @zinetic/sdk

TypeScript SDK for Zinetic NHI secretless credential exchange.

```ts
import { NHIProvider, fetchWithZinetic } from "@zinetic/sdk";

const provider = new NHIProvider({
  backendURL: "https://api.zinetic.net",
  target: "postgres-staging"
});

await provider.start();

const password = provider.getCredential("password");
const client = fetchWithZinetic(provider);
```

Credentials are kept in memory only. The provider uses environment-native
attestation tokens, DPoP proofs, encrypted exchange responses, and background
renewal.
