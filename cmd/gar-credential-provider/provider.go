package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mondu-ai/gar-credential-provider/internal/credentialprovider"
	"github.com/mondu-ai/gar-credential-provider/internal/gcpauth"
)

func runProvider() {
	fs := flag.NewFlagSet("provider", flag.ExitOnError)
	configPath := fs.String("config", "", "Path to GCP credential config file")
	_ = fs.Parse(os.Args[1:])

	if *configPath == "" {
		defaultPath := "/etc/eks/image-credential-provider/gcp-credential-config.json"
		if _, err := os.Stat(defaultPath); err == nil {
			*configPath = defaultPath
		} else {
			fmt.Fprintln(os.Stderr, "error: --config flag is required or config must exist at", defaultPath)
			os.Exit(1)
		}
	}

	if err := runProviderWithConfig(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runProviderWithConfig(configPath string) error {
	tokenFetcher, err := gcpauth.NewTokenFetcher(configPath)
	if err != nil {
		return fmt.Errorf("failed to create token fetcher: %w", err)
	}

	provider := credentialprovider.New(tokenFetcher)
	return provider.HandleRequest(os.Stdin, os.Stdout)
}
