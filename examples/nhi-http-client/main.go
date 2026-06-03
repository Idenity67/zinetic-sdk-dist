package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

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

	httpClient := zinetic.NewNHIHTTPClient(provider)

	downstreamURL, err := downstreamDataURL()
	if err != nil {
		log.Fatal(err)
	}
	resp, err := httpClient.Get(downstreamURL) // #nosec G107 G704 -- downstream URL is operator-configured and validated before use.
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	fmt.Printf("Downstream response status: %d\n", resp.StatusCode)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"healthy"}`)
	})

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("server shutdown: %v", err)
		}
	}()
	log.Fatal(server.ListenAndServe())
}

func downstreamDataURL() (string, error) {
	raw := strings.TrimSpace(os.Getenv("DOWNSTREAM_SERVICE_URL"))
	if raw == "" {
		return "", fmt.Errorf("DOWNSTREAM_SERVICE_URL is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse DOWNSTREAM_SERVICE_URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("DOWNSTREAM_SERVICE_URL must use http or https")
	}
	if parsed.Host == "" || parsed.User != nil {
		return "", fmt.Errorf("DOWNSTREAM_SERVICE_URL must include a host and no user info")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/api/data"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}
