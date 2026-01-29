package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mondu-ai/gar-credential-provider/internal/kubeletconfig"
)

const (
	defaultKubeletConfig = "/etc/eks/image-credential-provider/config.json"
	defaultBinaryPath    = "/etc/eks/image-credential-provider/gar-credential-provider"
	defaultCacheDuration = "50m"
	defaultRegistry      = "*.pkg.dev"

	exitSuccess    = 0
	exitError      = 1
	exitDryRunDiff = 2
)

type installOptions struct {
	gcpConfig     string
	kubeletConfig string
	binaryPath    string
	cacheDuration string
	registries    string
	dryRun        bool
	force         bool
}

func runInstall(args []string) {
	opts := parseInstallFlags(args)

	if opts.gcpConfig == "" {
		fmt.Fprintln(os.Stderr, "error: --gcp-config is required")
		os.Exit(exitError)
	}

	installer := &installer{opts: opts}
	exitCode := installer.run()
	os.Exit(exitCode)
}

func parseInstallFlags(args []string) installOptions {
	fs := flag.NewFlagSet("install", flag.ExitOnError)

	var opts installOptions
	fs.StringVar(&opts.gcpConfig, "gcp-config", "", "GCP credential config as JSON string")
	fs.StringVar(&opts.kubeletConfig, "kubelet-config", defaultKubeletConfig, "Path to kubelet credential provider config")
	fs.StringVar(&opts.binaryPath, "binary-path", defaultBinaryPath, "Path to install the binary")
	fs.StringVar(&opts.cacheDuration, "cache-duration", defaultCacheDuration, "Default cache duration for credentials")
	fs.StringVar(&opts.registries, "registries", defaultRegistry, "Comma-separated list of registry patterns to match")
	fs.BoolVar(&opts.dryRun, "dry-run", false, "Show what would be done without making changes")
	fs.BoolVar(&opts.force, "force", false, "Overwrite existing configuration")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `Usage: gar-credential-provider install [options]

Install and configure the GAR credential provider on this node.

Options:`)
		fs.PrintDefaults()
		fmt.Fprintln(os.Stderr, `
Exit codes:
  0  Success (installed or already configured)
  1  Error
  2  Dry-run would make changes

Examples:
  gar-credential-provider install --gcp-config='{"type":"external_account",...}'
  gar-credential-provider install --gcp-config='...' --dry-run`)
	}

	_ = fs.Parse(args)
	return opts
}

type installer struct {
	opts    installOptions
	changes []string
}

func (i *installer) run() int {
	gcpConfigContent := i.resolveGCPConfig()

	if err := i.installBinary(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return exitError
	}

	gcpConfigPath := filepath.Join(filepath.Dir(i.opts.kubeletConfig), "gcp-credential-config.json")
	if err := i.writeGCPConfig(gcpConfigPath, gcpConfigContent); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return exitError
	}

	if err := i.updateKubeletConfig(gcpConfigPath); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return exitError
	}

	// Print summary
	if len(i.changes) == 0 {
		fmt.Println("Already configured, no changes needed")
		return exitSuccess
	}

	if i.opts.dryRun {
		fmt.Println("Dry-run: the following changes would be made:")
		for _, change := range i.changes {
			fmt.Printf("  - %s\n", change)
		}
		return exitDryRunDiff
	}

	fmt.Println("Installation complete:")
	for _, change := range i.changes {
		fmt.Printf("  - %s\n", change)
	}
	return exitSuccess
}

func (i *installer) resolveGCPConfig() []byte {
	return []byte(i.opts.gcpConfig)
}

func (i *installer) installBinary() error {
	currentBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get current executable path: %w", err)
	}

	// Resolve symlinks
	currentBinary, err = filepath.EvalSymlinks(currentBinary)
	if err != nil {
		return fmt.Errorf("resolve executable symlinks: %w", err)
	}

	targetPath := i.opts.binaryPath

	// Check if already installed at target path
	if currentBinary == targetPath {
		// Running from target location, no copy needed
		return nil
	}

	// Check if target already exists
	if _, err := os.Stat(targetPath); err == nil {
		if !i.opts.force {
			// Check if it's the same file (by size comparison as quick check)
			currentInfo, err1 := os.Stat(currentBinary)
			targetInfo, err2 := os.Stat(targetPath)
			if err1 == nil && err2 == nil && currentInfo.Size() == targetInfo.Size() {
				return nil
			}
			return fmt.Errorf("binary already exists at %s, use --force to overwrite", targetPath)
		}
	}

	i.changes = append(i.changes, fmt.Sprintf("Copy binary to %s", targetPath))

	if i.opts.dryRun {
		return nil
	}

	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0750); err != nil {
		return fmt.Errorf("create directory %s: %w", targetDir, err)
	}

	// Copy binary
	if err := copyFile(currentBinary, targetPath, 0755); err != nil {
		return fmt.Errorf("copy binary: %w", err)
	}

	return nil
}

func (i *installer) writeGCPConfig(path string, content []byte) error {
	existing, err := os.ReadFile(path) //#nosec G304 -- path from CLI flag
	if err == nil && string(existing) == string(content) {
		return nil
	}

	if err == nil && !i.opts.force {
		return fmt.Errorf("GCP config already exists at %s, use --force to overwrite", path)
	}

	i.changes = append(i.changes, fmt.Sprintf("Write GCP credential config to %s", path))

	if i.opts.dryRun {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	if err := os.WriteFile(path, content, 0600); err != nil {
		return fmt.Errorf("write GCP config: %w", err)
	}

	return nil
}

func (i *installer) updateKubeletConfig(gcpConfigPath string) error {
	mgr := kubeletconfig.New(i.opts.kubeletConfig)

	cfg, err := mgr.Load()
	if err != nil {
		return err
	}

	isNew := cfg == nil
	if isNew {
		cfg = kubeletconfig.NewConfig()
	}

	registries := strings.Split(i.opts.registries, ",")
	for j := range registries {
		registries[j] = strings.TrimSpace(registries[j])
	}

	provider := kubeletconfig.NewGARProvider(gcpConfigPath, i.opts.cacheDuration, registries)
	changed := mgr.EnsureProvider(cfg, provider)

	if !changed && !isNew {
		return nil
	}

	if isNew {
		i.changes = append(i.changes, fmt.Sprintf("Create kubelet config at %s", i.opts.kubeletConfig))
	} else {
		i.changes = append(i.changes, fmt.Sprintf("Update kubelet config at %s", i.opts.kubeletConfig))
	}

	if i.opts.dryRun {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(i.opts.kubeletConfig), 0750); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	return mgr.Save(cfg)
}

func copyFile(src, dst string, perm os.FileMode) error {
	srcFile, err := os.Open(src) //#nosec G304 -- src from os.Executable()
	if err != nil {
		return fmt.Errorf("open source %s: %w", src, err)
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm) //#nosec G304 -- dst from CLI flag
	if err != nil {
		return fmt.Errorf("create destination %s: %w", dst, err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copy data: %w", err)
	}

	return nil
}
