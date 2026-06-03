# Zinetic SDK

Public customer-facing distribution repository for released Zinetic SDK packages.

## Go SDK

```sh
go get sdk.zinetic.net/zinetic@latest
```

```go
package main

import (
	"context"
	"fmt"
	"os"

	"sdk.zinetic.net/zinetic"
)

func main() {
	client, err := zinetic.NewClient(
		zinetic.WithBaseURL("https://api.zinetic.net"),
		zinetic.WithTenantID(os.Getenv("ZINETIC_TENANT_ID")),
		zinetic.WithAccessToken(os.Getenv("ZINETIC_ACCESS_TOKEN")),
	)
	if err != nil {
		panic(err)
	}

	health, err := client.Health.Health(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Println(health.Status)
}
```

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

## Security

Do not place access tokens, tenant secrets, DPoP private keys, `.env` files, or npm credentials in this repository. Released packages must be built from reviewed source and scanned before publication.
