package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mondu-ai/gar-credential-provider/internal/nodesetup"
)

const hostRoot = "/host"

func runSetup(_ []string) {
	gcpConfig := os.Getenv("GCP_CONFIG")
	registries := os.Getenv("REGISTRIES")

	if gcpConfig == "" {
		log.Fatal("GCP_CONFIG env var is required")
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

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()

	runSetupWithInstaller(ctx, installer)
}

func runSetupWithInstaller(ctx context.Context, installer nodesetup.Installer) {
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

	log.Println("Setup complete, waiting for termination signal...")
	<-ctx.Done()
	log.Println("Received termination signal, exiting")
}
