package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mondu-ai/gar-credential-provider/internal/nodesetup"
)

const (
	hostRoot       = "/host"
	setupTimeout   = 2 * time.Minute
	defaultVersion = "unknown"
)

func runSetup(_ []string) {
	gcpConfig := os.Getenv("GCP_CONFIG")
	nodeName := os.Getenv("NODE_NAME")
	version := os.Getenv("VERSION")
	registries := os.Getenv("REGISTRIES")

	if gcpConfig == "" {
		log.Fatal("GCP_CONFIG env var is required")
	}
	if nodeName == "" {
		log.Fatal("NODE_NAME env var is required")
	}
	if version == "" {
		version = defaultVersion
	}
	if registries == "" {
		registries = "*.pkg.dev"
	}

	if _, err := os.Stat(hostRoot); os.IsNotExist(err) {
		log.Fatalf("/host is not mounted - this command must run in a DaemonSet pod with hostPath volume")
	}

	installer := nodesetup.NewInstaller(nodesetup.InstallerConfig{
		HostRoot:   hostRoot,
		GCPConfig:  gcpConfig,
		Registries: registries,
	})

	result, err := installer.Install()
	if err != nil {
		log.Fatalf("Installation failed: %v", err)
	}

	if result.Changed {
		log.Printf("Changes made: %v", result.Actions)
		log.Println("Restarting kubelet...")
		if err := installer.RestartKubelet(); err != nil {
			log.Fatalf("Failed to restart kubelet: %v", err)
		}
		log.Println("Kubelet restarted successfully")
	} else {
		log.Println("No changes needed, skipping kubelet restart")
	}

	if err := labelNode(nodeName, version); err != nil {
		log.Fatalf("Failed to label node: %v", err)
	}

	fmt.Printf("Setup complete on node %s (version: %s)\n", nodeName, version)
}

func labelNode(nodeName, version string) error {
	labeler, err := nodesetup.NewLabeler()
	if err != nil {
		return fmt.Errorf("create node labeler: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), setupTimeout)
	defer cancel()

	return labeler.LabelNode(ctx, nodeName, version)
}
