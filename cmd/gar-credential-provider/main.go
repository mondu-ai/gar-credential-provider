package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		// Default mode: kubelet credential provider
		runProvider()
		return
	}

	switch os.Args[1] {
	case "install":
		runInstall(os.Args[2:])
	case "setup":
		runSetup(os.Args[2:])
	case "version", "--version", "-v":
		runVersion()
	case "--config":
		// Legacy flag usage: --config=/path/to/config.json
		runProvider()
	case "--help", "-h", "help":
		printUsage()
	default:
		// Check if first arg looks like a flag
		if len(os.Args[1]) > 0 && os.Args[1][0] == '-' {
			runProvider()
			return
		}
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: gar-credential-provider [command]

Commands:
  install     Install and configure the credential provider on this node
  setup       DaemonSet mode: install, restart kubelet, and label node
  version     Show version information

Kubelet credential provider mode (default):
  gar-credential-provider [--config=<path>]
    Reads kubelet credential request from stdin, returns credentials to stdout.
    If --config is not specified, uses /etc/eks/image-credential-provider/gcp-credential-config.json

Examples:
  # Install with GCP config from file
  gar-credential-provider install --gcp-config=/path/to/config.json

  # Install with GCP config as JSON string
  gar-credential-provider install --gcp-config='{"type":"external_account",...}'

  # Run as kubelet credential provider (default)
  gar-credential-provider --config=/etc/eks/image-credential-provider/gcp-credential-config.json

  # DaemonSet setup (reads from environment variables)
  GCP_CONFIG='...' NODE_NAME=node-1 VERSION=v1.0.0 gar-credential-provider setup`)
}
