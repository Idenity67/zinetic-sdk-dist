package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"

	"sdk.zinetic.net/zinetic"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	provider, err := zinetic.NewNHIProvider(zinetic.NHIProviderConfig{
		BackendURL: os.Getenv("ZINETIC_API_URL"),
		Target:     os.Getenv("ZINETIC_NHI_TARGET"),
		Audience:   os.Getenv("ZINETIC_AUDIENCE"),
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := provider.Start(ctx); err != nil {
		log.Fatal(err)
	}
	defer provider.Stop()

	connector, err := zinetic.NewNHIConnector(zinetic.NHIConnectorConfig{
		Provider:    provider,
		DSNTemplate: "postgres://{{username}}:{{password}}@" + os.Getenv("DB_HOST") + "/app?sslmode=require",
	})
	if err != nil {
		log.Fatal(err)
	}

	db := sql.OpenDB(connector)
	defer db.Close()

	var result int
	if err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Database connected via NHI secretless credentials: %d\n", result)
}
