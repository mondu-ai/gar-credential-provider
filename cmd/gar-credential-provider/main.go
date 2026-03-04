// Package main is the entrypoint for gar-credential-provider.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		runProvider()
		return
	}

	switch os.Args[1] {
	case "install", "setup":
		runSetup(os.Args[2:])
	case "version", "--version", "-v":
		runVersion()
	case "--config":
		runProvider()
	case "--help", "-h", "help":
		printUsage()
	default:
		if len(os.Args[1]) > 0 && os.Args[1][0] == '-' {
			runProvider()
			return
		}
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1]) //nolint:gosec // stderr output, not web response
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: gar-credential-provider [command]

Commands:
  install     Install on node: copy binary, configure kubelet, restart (alias: setup)
  version     Show version information

Kubelet credential provider mode (default):
  gar-credential-provider [--config=<path>]
    Reads kubelet credential request from stdin, returns credentials to stdout.
    If --config is not specified, uses /etc/eks/image-credential-provider/gcp-credential-config.json

Examples:
  # Run as kubelet credential provider (default)
  gar-credential-provider --config=/etc/eks/image-credential-provider/gcp-credential-config.json

  # DaemonSet install (reads from environment variables, stays running)
  GCP_CONFIG='...' gar-credential-provider install`)
}
