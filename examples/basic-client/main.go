package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"log"
	"os"

	"sdk.zinetic.net/zinetic"
)

func main() {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}

	client, err := zinetic.NewClient(
		zinetic.WithBaseURL(os.Getenv("ZINETIC_API_URL")),
		zinetic.WithTenantID(os.Getenv("ZINETIC_TENANT_ID")),
		zinetic.WithAccessToken(os.Getenv("ZINETIC_ACCESS_TOKEN")),
		zinetic.WithDPoPKey(key),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	health, err := client.Health.Health(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Backend status: %s\n", health.Status)

	version, err := client.Health.Version(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Backend version: %s (commit: %s)\n", version.Version, version.CommitSHA)
}
