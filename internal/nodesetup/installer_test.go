package nodesetup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstaller_Install(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, hostRoot, binarySource string)
		gcpConfig   string
		registries  string
		wantChanged bool
		wantErr     bool
	}{
		{
			name: "fresh install",
			setup: func(t *testing.T, _, binarySource string) {
				err := os.WriteFile(binarySource, []byte("binary-content"), 0755)
				require.NoError(t, err)
			},
			gcpConfig:   `{"type":"external_account"}`,
			registries:  "*.pkg.dev",
			wantChanged: true,
		},
		{
			name: "idempotent - no changes needed",
			setup: func(t *testing.T, hostRoot, binarySource string) {
				err := os.WriteFile(binarySource, []byte("binary-content"), 0755)
				require.NoError(t, err)

				configDir := filepath.Join(hostRoot, "/etc/eks/image-credential-provider")
				err = os.MkdirAll(configDir, 0750)
				require.NoError(t, err)

				err = os.WriteFile(filepath.Join(configDir, "gar-credential-provider"), []byte("binary-content"), 0755)
				require.NoError(t, err)

				err = os.WriteFile(filepath.Join(configDir, "gcp-credential-config.json"), []byte(`{"type":"external_account"}`), 0600)
				require.NoError(t, err)

				kubeletConfig := `{
  "apiVersion": "kubelet.config.k8s.io/v1",
  "kind": "CredentialProviderConfig",
  "providers": [
    {
      "name": "gar-credential-provider",
      "matchImages": [
        "*.pkg.dev"
      ],
      "defaultCacheDuration": "50m",
      "apiVersion": "credentialprovider.kubelet.k8s.io/v1",
      "args": [
        "--config=/etc/eks/image-credential-provider/gcp-credential-config.json"
      ]
    }
  ]
}
`
				err = os.WriteFile(filepath.Join(configDir, "config.json"), []byte(kubeletConfig), 0600)
				require.NoError(t, err)
			},
			gcpConfig:   `{"type":"external_account"}`,
			registries:  "*.pkg.dev",
			wantChanged: false,
		},
		{
			name: "binary source not found",
			setup: func(_ *testing.T, _, _ string) {
				// Don't create the binary source
			},
			gcpConfig:  `{"type":"external_account"}`,
			registries: "*.pkg.dev",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hostRoot := t.TempDir()
			binarySource := filepath.Join(t.TempDir(), "source-binary")

			if tt.setup != nil {
				tt.setup(t, hostRoot, binarySource)
			}

			installer := NewInstaller(InstallerConfig{
				HostRoot:     hostRoot,
				GCPConfig:    tt.gcpConfig,
				Registries:   tt.registries,
				BinarySource: binarySource,
			})

			result, err := installer.Install()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantChanged, result.Changed)

			if tt.wantChanged {
				binaryPath := filepath.Join(hostRoot, "/etc/eks/image-credential-provider/gar-credential-provider")
				_, err = os.Stat(binaryPath)
				assert.NoError(t, err, "binary should exist")

				gcpConfigPath := filepath.Join(hostRoot, "/etc/eks/image-credential-provider/gcp-credential-config.json")
				content, err := os.ReadFile(gcpConfigPath)
				assert.NoError(t, err)
				assert.Equal(t, tt.gcpConfig, string(content))

				kubeletConfigPath := filepath.Join(hostRoot, "/etc/eks/image-credential-provider/config.json")
				_, err = os.Stat(kubeletConfigPath)
				assert.NoError(t, err, "kubelet config should exist")
			}
		})
	}
}
