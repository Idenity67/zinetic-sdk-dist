# Zinetic SDK Distribution

Public customer-facing distribution page for released Zinetic SDK packages.

## TypeScript SDK

```sh
npm install @zinetic/sdk
```

```ts
import { NHIProvider, fetchWithZinetic } from "@zinetic/sdk";

const provider = new NHIProvider({
  backendURL: "https://api.zinetic.net",
  target: process.env.ZINETIC_NHI_TARGET,
});

await provider.start();

const client = fetchWithZinetic(provider);
const response = await client("https://api.zinetic.net/api/v1/health");
console.log(response.status);
```

## Go SDK

The Go SDK source repository is private while the public distribution path is being prepared. Enterprise customers can receive private repository access or use the REST API and OpenAPI contract through the Zinetic CLI.

## Security

Do not place access tokens, tenant secrets, DPoP private keys, `.env` files, or npm credentials in this repository. Released packages must be built from reviewed source and scanned before publication.
