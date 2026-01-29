package kubeletconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_Load(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		want        *Config
		wantErr     bool
	}{
		{
			name: "valid config with providers",
			fileContent: `{
				"apiVersion": "kubelet.config.k8s.io/v1",
				"kind": "CredentialProviderConfig",
				"providers": [
					{
						"name": "ecr-credential-provider",
						"matchImages": ["*.dkr.ecr.*.amazonaws.com"],
						"defaultCacheDuration": "12h",
						"apiVersion": "credentialprovider.kubelet.k8s.io/v1"
					}
				]
			}`,
			want: &Config{
				APIVersion: "kubelet.config.k8s.io/v1",
				Kind:       "CredentialProviderConfig",
				Providers: []Provider{
					{
						Name:                 "ecr-credential-provider",
						MatchImages:          []string{"*.dkr.ecr.*.amazonaws.com"},
						DefaultCacheDuration: "12h",
						APIVersion:           "credentialprovider.kubelet.k8s.io/v1",
					},
				},
			},
		},
		{
			name:        "invalid json",
			fileContent: `{broken`,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, "config.json")

			err := os.WriteFile(path, []byte(tt.fileContent), 0644)
			require.NoError(t, err)

			mgr := New(path)
			got, err := mgr.Load()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestManager_Load_FileNotExists(t *testing.T) {
	mgr := New("/nonexistent/path/config.json")
	cfg, err := mgr.Load()

	require.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestManager_Save(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.json")

	cfg := &Config{
		APIVersion: "kubelet.config.k8s.io/v1",
		Kind:       "CredentialProviderConfig",
		Providers: []Provider{
			{
				Name:                 "test-provider",
				MatchImages:          []string{"*.example.com"},
				DefaultCacheDuration: "1h",
				APIVersion:           "credentialprovider.kubelet.k8s.io/v1",
				Args:                 []string{"--config=/etc/config.json"},
			},
		},
	}

	mgr := New(path)
	err := mgr.Save(cfg)
	require.NoError(t, err)

	// Verify file was written
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "test-provider")
	assert.Contains(t, string(data), "*.example.com")

	// Verify we can load it back
	loaded, err := mgr.Load()
	require.NoError(t, err)
	assert.Equal(t, cfg, loaded)
}

func TestManager_EnsureProvider(t *testing.T) {
	tests := []struct {
		name         string
		existingCfg  *Config
		newProvider  Provider
		wantChanged  bool
		wantNumProvs int
	}{
		{
			name: "add new provider to empty config",
			existingCfg: &Config{
				APIVersion: "kubelet.config.k8s.io/v1",
				Kind:       "CredentialProviderConfig",
				Providers:  []Provider{},
			},
			newProvider: Provider{
				Name:                 "gar-credential-provider",
				MatchImages:          []string{"*.pkg.dev"},
				DefaultCacheDuration: "50m",
				APIVersion:           "credentialprovider.kubelet.k8s.io/v1",
			},
			wantChanged:  true,
			wantNumProvs: 1,
		},
		{
			name: "add provider alongside existing",
			existingCfg: &Config{
				APIVersion: "kubelet.config.k8s.io/v1",
				Kind:       "CredentialProviderConfig",
				Providers: []Provider{
					{
						Name:                 "ecr-credential-provider",
						MatchImages:          []string{"*.dkr.ecr.*.amazonaws.com"},
						DefaultCacheDuration: "12h",
						APIVersion:           "credentialprovider.kubelet.k8s.io/v1",
					},
				},
			},
			newProvider: Provider{
				Name:                 "gar-credential-provider",
				MatchImages:          []string{"*.pkg.dev"},
				DefaultCacheDuration: "50m",
				APIVersion:           "credentialprovider.kubelet.k8s.io/v1",
			},
			wantChanged:  true,
			wantNumProvs: 2,
		},
		{
			name: "no change when identical provider exists",
			existingCfg: &Config{
				APIVersion: "kubelet.config.k8s.io/v1",
				Kind:       "CredentialProviderConfig",
				Providers: []Provider{
					{
						Name:                 "gar-credential-provider",
						MatchImages:          []string{"*.pkg.dev"},
						DefaultCacheDuration: "50m",
						APIVersion:           "credentialprovider.kubelet.k8s.io/v1",
						Args:                 []string{"--config=/etc/gcp.json"},
					},
				},
			},
			newProvider: Provider{
				Name:                 "gar-credential-provider",
				MatchImages:          []string{"*.pkg.dev"},
				DefaultCacheDuration: "50m",
				APIVersion:           "credentialprovider.kubelet.k8s.io/v1",
				Args:                 []string{"--config=/etc/gcp.json"},
			},
			wantChanged:  false,
			wantNumProvs: 1,
		},
		{
			name: "update existing provider with different config",
			existingCfg: &Config{
				APIVersion: "kubelet.config.k8s.io/v1",
				Kind:       "CredentialProviderConfig",
				Providers: []Provider{
					{
						Name:                 "gar-credential-provider",
						MatchImages:          []string{"*.pkg.dev"},
						DefaultCacheDuration: "30m",
						APIVersion:           "credentialprovider.kubelet.k8s.io/v1",
					},
				},
			},
			newProvider: Provider{
				Name:                 "gar-credential-provider",
				MatchImages:          []string{"*.pkg.dev"},
				DefaultCacheDuration: "50m",
				APIVersion:           "credentialprovider.kubelet.k8s.io/v1",
			},
			wantChanged:  true,
			wantNumProvs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := New("/tmp/unused").(*mgr)
			changed := mgr.EnsureProvider(tt.existingCfg, tt.newProvider)

			assert.Equal(t, tt.wantChanged, changed)
			assert.Len(t, tt.existingCfg.Providers, tt.wantNumProvs)

			// Verify the provider is in the config
			var found bool
			for _, p := range tt.existingCfg.Providers {
				if p.Name == tt.newProvider.Name {
					found = true
					assert.Equal(t, tt.newProvider.DefaultCacheDuration, p.DefaultCacheDuration)
				}
			}
			assert.True(t, found, "provider should be in config")
		})
	}
}

func TestNewGARProvider(t *testing.T) {
	provider := NewGARProvider("/etc/gcp-config.json", "50m", []string{"*.pkg.dev", "gcr.io"})

	assert.Equal(t, "gar-credential-provider", provider.Name)
	assert.Equal(t, []string{"*.pkg.dev", "gcr.io"}, provider.MatchImages)
	assert.Equal(t, "50m", provider.DefaultCacheDuration)
	assert.Equal(t, "credentialprovider.kubelet.k8s.io/v1", provider.APIVersion)
	assert.Equal(t, []string{"--config=/etc/gcp-config.json"}, provider.Args)
}

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	assert.Equal(t, "kubelet.config.k8s.io/v1", cfg.APIVersion)
	assert.Equal(t, "CredentialProviderConfig", cfg.Kind)
	assert.Empty(t, cfg.Providers)
}
