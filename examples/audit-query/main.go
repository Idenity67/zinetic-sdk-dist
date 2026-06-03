package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"time"

	"sdk.zinetic.net/audit"
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

	since := time.Now().Add(-24 * time.Hour)
	resp, err := client.Audit.Search(ctx, &audit.SearchRequest{
		TimeRangeStart: &since,
		Action:         "credential.issued",
		Limit:          100,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d audit events (has_more: %v)\n", len(resp.Data), resp.HasMore)
	for _, evt := range resp.Data {
		fmt.Printf("  [%s] %s → %s (%s)\n",
			evt.Timestamp.Format(time.RFC3339),
			evt.Actor.ActorID,
			evt.Action,
			evt.Outcome,
		)
	}
}
