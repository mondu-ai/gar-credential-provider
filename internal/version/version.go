// Package version provides build-time version information.
package version

// Build-time variables set via ldflags.
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)
