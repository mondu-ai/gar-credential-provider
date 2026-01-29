package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mondu-ai/gar-credential-provider/internal/kubeletconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstaller_ResolveGCPConfig(t *testing.T) {
	gcpJSON := `{"type":"external_account","audience":"//iam.googleapis.com/projects/123"}`

	i := &installer{opts: installOptions{gcpConfig: gcpJSON}}
	content := i.resolveGCPConfig()

	assert.Equal(t, gcpJSON, string(content))
}

func TestInstaller_WriteGCPConfig_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir", "gcp-config.json")
	content := []byte(`{"type":"external_account"}`)

	i := &installer{opts: installOptions{}}
	err := i.writeGCPConfig(configPath, content)

	require.NoError(t, err)
	assert.Len(t, i.changes, 1)
	assert.Contains(t, i.changes[0], "Write GCP credential config")

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestInstaller_WriteGCPConfig_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "gcp-config.json")
	content := []byte(`{"type":"external_account"}`)

	err := os.WriteFile(configPath, content, 0600)
	require.NoError(t, err)

	i := &installer{opts: installOptions{}}
	err = i.writeGCPConfig(configPath, content)

	require.NoError(t, err)
	assert.Empty(t, i.changes, "no changes should be recorded for identical content")
}

func TestInstaller_WriteGCPConfig_ExistingDifferent_NoForce(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "gcp-config.json")

	err := os.WriteFile(configPath, []byte(`{"old":"content"}`), 0600)
	require.NoError(t, err)

	i := &installer{opts: installOptions{force: false}}
	err = i.writeGCPConfig(configPath, []byte(`{"new":"content"}`))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "use --force to overwrite")
}

func TestInstaller_WriteGCPConfig_ExistingDifferent_WithForce(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "gcp-config.json")
	newContent := []byte(`{"new":"content"}`)

	err := os.WriteFile(configPath, []byte(`{"old":"content"}`), 0600)
	require.NoError(t, err)

	i := &installer{opts: installOptions{force: true}}
	err = i.writeGCPConfig(configPath, newContent)

	require.NoError(t, err)
	assert.Len(t, i.changes, 1)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, newContent, data)
}

func TestInstaller_WriteGCPConfig_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "gcp-config.json")
	content := []byte(`{"type":"external_account"}`)

	i := &installer{opts: installOptions{dryRun: true}}
	err := i.writeGCPConfig(configPath, content)

	require.NoError(t, err)
	assert.Len(t, i.changes, 1)

	_, err = os.Stat(configPath)
	assert.True(t, os.IsNotExist(err))
}

func TestInstaller_UpdateKubeletConfig_NewConfig(t *testing.T) {
	tmpDir := t.TempDir()
	kubeletConfigPath := filepath.Join(tmpDir, "config.json")
	gcpConfigPath := filepath.Join(tmpDir, "gcp-credential-config.json")

	i := &installer{opts: installOptions{
		kubeletConfig: kubeletConfigPath,
		cacheDuration: "50m",
		registries:    "*.pkg.dev",
	}}

	err := i.updateKubeletConfig(gcpConfigPath)
	require.NoError(t, err)
	assert.Len(t, i.changes, 1)
	assert.Contains(t, i.changes[0], "Create kubelet config")

	data, err := os.ReadFile(kubeletConfigPath)
	require.NoError(t, err)

	var cfg kubeletconfig.Config
	err = json.Unmarshal(data, &cfg)
	require.NoError(t, err)

	assert.Len(t, cfg.Providers, 1)
	assert.Equal(t, "gar-credential-provider", cfg.Providers[0].Name)
	assert.Equal(t, []string{"*.pkg.dev"}, cfg.Providers[0].MatchImages)
	assert.Equal(t, "50m", cfg.Providers[0].DefaultCacheDuration)
}

func TestInstaller_UpdateKubeletConfig_AddToExisting(t *testing.T) {
	tmpDir := t.TempDir()
	kubeletConfigPath := filepath.Join(tmpDir, "config.json")
	gcpConfigPath := filepath.Join(tmpDir, "gcp-credential-config.json")

	existingConfig := kubeletconfig.Config{
		APIVersion: "kubelet.config.k8s.io/v1",
		Kind:       "CredentialProviderConfig",
		Providers: []kubeletconfig.Provider{
			{
				Name:                 "ecr-credential-provider",
				MatchImages:          []string{"*.dkr.ecr.*.amazonaws.com"},
				DefaultCacheDuration: "12h",
				APIVersion:           "credentialprovider.kubelet.k8s.io/v1",
			},
		},
	}
	data, _ := json.MarshalIndent(existingConfig, "", "  ")
	err := os.WriteFile(kubeletConfigPath, data, 0644)
	require.NoError(t, err)

	i := &installer{opts: installOptions{
		kubeletConfig: kubeletConfigPath,
		cacheDuration: "50m",
		registries:    "*.pkg.dev",
	}}

	err = i.updateKubeletConfig(gcpConfigPath)
	require.NoError(t, err)
	assert.Len(t, i.changes, 1)
	assert.Contains(t, i.changes[0], "Update kubelet config")

	data, err = os.ReadFile(kubeletConfigPath)
	require.NoError(t, err)

	var cfg kubeletconfig.Config
	err = json.Unmarshal(data, &cfg)
	require.NoError(t, err)

	assert.Len(t, cfg.Providers, 2)

	var hasECR, hasGAR bool
	for _, p := range cfg.Providers {
		if p.Name == "ecr-credential-provider" {
			hasECR = true
		}
		if p.Name == "gar-credential-provider" {
			hasGAR = true
		}
	}
	assert.True(t, hasECR, "ECR provider should be preserved")
	assert.True(t, hasGAR, "GAR provider should be added")
}

func TestInstaller_UpdateKubeletConfig_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	kubeletConfigPath := filepath.Join(tmpDir, "config.json")
	gcpConfigPath := "/etc/gcp-credential-config.json"

	existingConfig := kubeletconfig.Config{
		APIVersion: "kubelet.config.k8s.io/v1",
		Kind:       "CredentialProviderConfig",
		Providers: []kubeletconfig.Provider{
			{
				Name:                 "gar-credential-provider",
				MatchImages:          []string{"*.pkg.dev"},
				DefaultCacheDuration: "50m",
				APIVersion:           "credentialprovider.kubelet.k8s.io/v1",
				Args:                 []string{"--config=" + gcpConfigPath},
			},
		},
	}
	data, _ := json.MarshalIndent(existingConfig, "", "  ")
	err := os.WriteFile(kubeletConfigPath, data, 0644)
	require.NoError(t, err)

	i := &installer{opts: installOptions{
		kubeletConfig: kubeletConfigPath,
		cacheDuration: "50m",
		registries:    "*.pkg.dev",
	}}

	err = i.updateKubeletConfig(gcpConfigPath)
	require.NoError(t, err)
	assert.Empty(t, i.changes, "no changes should be recorded for identical config")
}

func TestInstaller_UpdateKubeletConfig_MultipleRegistries(t *testing.T) {
	tmpDir := t.TempDir()
	kubeletConfigPath := filepath.Join(tmpDir, "config.json")
	gcpConfigPath := filepath.Join(tmpDir, "gcp-credential-config.json")

	i := &installer{opts: installOptions{
		kubeletConfig: kubeletConfigPath,
		cacheDuration: "50m",
		registries:    "*.pkg.dev, gcr.io, eu.gcr.io",
	}}

	err := i.updateKubeletConfig(gcpConfigPath)
	require.NoError(t, err)

	data, err := os.ReadFile(kubeletConfigPath)
	require.NoError(t, err)

	var cfg kubeletconfig.Config
	err = json.Unmarshal(data, &cfg)
	require.NoError(t, err)

	assert.Equal(t, []string{"*.pkg.dev", "gcr.io", "eu.gcr.io"}, cfg.Providers[0].MatchImages)
}

func TestInstaller_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	kubeletConfigPath := filepath.Join(tmpDir, "config.json")
	gcpConfig := `{"type":"external_account","audience":"test"}`

	i := &installer{opts: installOptions{
		gcpConfig:     gcpConfig,
		kubeletConfig: kubeletConfigPath,
		binaryPath:    filepath.Join(tmpDir, "target", "binary"),
		cacheDuration: "50m",
		registries:    "*.pkg.dev",
		dryRun:        true,
	}}

	gcpConfigPath := filepath.Join(tmpDir, "gcp-credential-config.json")
	err := i.writeGCPConfig(gcpConfigPath, []byte(gcpConfig))
	require.NoError(t, err)

	_, err = os.Stat(gcpConfigPath)
	assert.True(t, os.IsNotExist(err), "file should not be created in dry-run mode")

	err = i.updateKubeletConfig(gcpConfigPath)
	require.NoError(t, err)

	_, err = os.Stat(kubeletConfigPath)
	assert.True(t, os.IsNotExist(err), "kubelet config should not be created in dry-run mode")
}

func TestParseInstallFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want installOptions
	}{
		{
			name: "defaults",
			args: []string{"--gcp-config={\"type\":\"external_account\"}"},
			want: installOptions{
				gcpConfig:     `{"type":"external_account"}`,
				kubeletConfig: defaultKubeletConfig,
				binaryPath:    defaultBinaryPath,
				cacheDuration: defaultCacheDuration,
				registries:    defaultRegistry,
				dryRun:        false,
				force:         false,
			},
		},
		{
			name: "all flags",
			args: []string{
				"--gcp-config={\"type\":\"external_account\"}",
				"--kubelet-config=/my/kubelet.json",
				"--binary-path=/my/binary",
				"--cache-duration=1h",
				"--registries=*.pkg.dev,gcr.io",
				"--dry-run",
				"--force",
			},
			want: installOptions{
				gcpConfig:     `{"type":"external_account"}`,
				kubeletConfig: "/my/kubelet.json",
				binaryPath:    "/my/binary",
				cacheDuration: "1h",
				registries:    "*.pkg.dev,gcr.io",
				dryRun:        true,
				force:         true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseInstallFlags(tt.args)
			assert.Equal(t, tt.want, got)
		})
	}
}
