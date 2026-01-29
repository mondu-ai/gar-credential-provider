// Package nodesetup installs the credential provider on Kubernetes nodes.
package nodesetup

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mondu-ai/gar-credential-provider/internal/kubeletconfig"
)

const (
	defaultKubeletConfigPath = "/etc/eks/image-credential-provider/config.json"
	defaultBinaryName        = "gar-credential-provider"
	defaultCacheDuration     = "50m"
)

// InstallerConfig configures the node setup installer.
type InstallerConfig struct {
	HostRoot      string
	GCPConfig     string
	Registries    string
	CacheDuration string
	BinarySource  string
}

// InstallResult contains the outcome of an installation.
type InstallResult struct {
	Changed bool
	Actions []string
}

// Installer installs the credential provider on a node.
type Installer interface {
	Install() (InstallResult, error)
	RestartKubelet() error
}

var _ Installer = (*installer)(nil)

type installer struct {
	cfg InstallerConfig
}

// NewInstaller creates an Installer with the given config.
func NewInstaller(cfg InstallerConfig) Installer {
	if cfg.CacheDuration == "" {
		cfg.CacheDuration = defaultCacheDuration
	}
	if cfg.BinarySource == "" {
		cfg.BinarySource = "/gar-credential-provider"
	}
	return &installer{cfg: cfg}
}

func (i *installer) Install() (InstallResult, error) {
	result := InstallResult{}

	binaryChanged, err := i.installBinary()
	if err != nil {
		return result, fmt.Errorf("install binary: %w", err)
	}
	if binaryChanged {
		result.Actions = append(result.Actions, "Copied binary")
		result.Changed = true
	}

	gcpConfigChanged, err := i.writeGCPConfig()
	if err != nil {
		return result, fmt.Errorf("write GCP config: %w", err)
	}
	if gcpConfigChanged {
		result.Actions = append(result.Actions, "Wrote GCP config")
		result.Changed = true
	}

	kubeletConfigChanged, err := i.updateKubeletConfig()
	if err != nil {
		return result, fmt.Errorf("update kubelet config: %w", err)
	}
	if kubeletConfigChanged {
		result.Actions = append(result.Actions, "Updated kubelet config")
		result.Changed = true
	}

	return result, nil
}

func (i *installer) installBinary() (bool, error) {
	src := i.cfg.BinarySource
	dst := i.hostPath(defaultKubeletConfigPath)
	dst = filepath.Join(filepath.Dir(dst), defaultBinaryName)

	srcInfo, err := os.Stat(src)
	if err != nil {
		return false, fmt.Errorf("stat source binary %s: %w", src, err)
	}

	dstInfo, err := os.Stat(dst)
	if err == nil && srcInfo.Size() == dstInfo.Size() {
		return false, nil
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0750); err != nil {
		return false, fmt.Errorf("create directory %s: %w", filepath.Dir(dst), err)
	}

	if err := copyFile(src, dst, 0755); err != nil {
		return false, fmt.Errorf("copy binary: %w", err)
	}

	return true, nil
}

func (i *installer) writeGCPConfig() (bool, error) {
	path := i.gcpConfigHostPath()
	content := []byte(i.cfg.GCPConfig)

	existing, err := os.ReadFile(path) //#nosec G304 -- controlled path
	if err == nil && bytes.Equal(existing, content) {
		return false, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return false, fmt.Errorf("create directory: %w", err)
	}

	if err := os.WriteFile(path, content, 0600); err != nil {
		return false, fmt.Errorf("write GCP config: %w", err)
	}

	return true, nil
}

func (i *installer) updateKubeletConfig() (bool, error) {
	kubeletConfigHostPath := i.hostPath(defaultKubeletConfigPath)
	mgr := kubeletconfig.New(kubeletConfigHostPath)

	cfg, err := mgr.Load()
	if err != nil {
		return false, err
	}

	isNew := cfg == nil
	if isNew {
		cfg = kubeletconfig.NewConfig()
	}

	registries := strings.Split(i.cfg.Registries, ",")
	for j := range registries {
		registries[j] = strings.TrimSpace(registries[j])
	}

	gcpConfigPath := i.gcpConfigKubeletPath()
	provider := kubeletconfig.NewGARProvider(gcpConfigPath, i.cfg.CacheDuration, registries)
	changed := mgr.EnsureProvider(cfg, provider)

	if !changed && !isNew {
		return false, nil
	}

	if err := os.MkdirAll(filepath.Dir(kubeletConfigHostPath), 0750); err != nil {
		return false, fmt.Errorf("create directory: %w", err)
	}

	if err := mgr.Save(cfg); err != nil {
		return false, err
	}

	return true, nil
}

func (i *installer) RestartKubelet() error {
	cmd := exec.Command("chroot", i.cfg.HostRoot, "systemctl", "restart", "kubelet") //#nosec G204 -- controlled input
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restart kubelet: %w (output: %s)", err, string(output))
	}
	return nil
}

func (i *installer) hostPath(path string) string {
	return filepath.Join(i.cfg.HostRoot, path)
}

func (i *installer) gcpConfigHostPath() string {
	return filepath.Join(filepath.Dir(i.hostPath(defaultKubeletConfigPath)), "gcp-credential-config.json")
}

func (i *installer) gcpConfigKubeletPath() string {
	return filepath.Join(filepath.Dir(defaultKubeletConfigPath), "gcp-credential-config.json")
}

func copyFile(src, dst string, perm os.FileMode) error {
	srcFile, err := os.Open(src) //#nosec G304 -- src from controlled input
	if err != nil {
		return fmt.Errorf("open source %s: %w", src, err)
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm) //#nosec G304 -- dst from controlled input
	if err != nil {
		return fmt.Errorf("create destination %s: %w", dst, err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copy data: %w", err)
	}

	return nil
}
