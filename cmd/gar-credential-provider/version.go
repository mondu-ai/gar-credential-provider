package main

import (
	"fmt"

	"github.com/mondu-ai/gar-credential-provider/internal/version"
)

func runVersion() {
	fmt.Printf("gar-credential-provider %s (commit: %s, built: %s)\n",
		version.Version, version.GitCommit, version.BuildTime)
}
