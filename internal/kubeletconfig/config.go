// Package kubeletconfig manages kubelet credential provider configuration files.
package kubeletconfig

import (
	"encoding/json"
	"fmt"
	"os"
)

// Manager manages kubelet credential provider config files.
type Manager interface {
	Load() (*Config, error)
	Save(cfg *Config) error
	EnsureProvider(cfg *Config, provider Provider) (changed bool)
}

var _ Manager = (*mgr)(nil)

type mgr struct {
	path string
}

// New creates a Manager for the given config file path.
func New(path string) Manager {
	return &mgr{path: path}
}

// NewConfig creates an empty credential provider config.
func NewConfig() *Config {
	return &Config{
		APIVersion: "kubelet.config.k8s.io/v1",
		Kind:       "CredentialProviderConfig",
		Providers:  []Provider{},
	}
}

// NewGARProvider creates a Provider entry for gar-credential-provider.
func NewGARProvider(gcpConfigPath string, cacheDuration string, matchImages []string) Provider {
	return Provider{
		Name:                 "gar-credential-provider",
		MatchImages:          matchImages,
		DefaultCacheDuration: cacheDuration,
		APIVersion:           "credentialprovider.kubelet.k8s.io/v1",
		Args:                 []string{"--config=" + gcpConfigPath},
	}
}

func (m *mgr) Load() (*Config, error) {
	data, err := os.ReadFile(m.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read kubelet config %s: %w", m.path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse kubelet config %s: %w", m.path, err)
	}

	return &cfg, nil
}

func (m *mgr) Save(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal kubelet config: %w", err)
	}

	if err := os.WriteFile(m.path, append(data, '\n'), 0600); err != nil {
		return fmt.Errorf("write kubelet config %s: %w", m.path, err)
	}

	return nil
}

func (m *mgr) EnsureProvider(cfg *Config, provider Provider) bool {
	for i, p := range cfg.Providers {
		if p.Name == provider.Name {
			if providersEqual(p, provider) {
				return false
			}
			cfg.Providers[i] = provider
			return true
		}
	}

	cfg.Providers = append(cfg.Providers, provider)
	return true
}

func providersEqual(a, b Provider) bool {
	if a.Name != b.Name || a.APIVersion != b.APIVersion || a.DefaultCacheDuration != b.DefaultCacheDuration {
		return false
	}

	if len(a.MatchImages) != len(b.MatchImages) {
		return false
	}
	for i := range a.MatchImages {
		if a.MatchImages[i] != b.MatchImages[i] {
			return false
		}
	}

	if len(a.Args) != len(b.Args) {
		return false
	}
	for i := range a.Args {
		if a.Args[i] != b.Args[i] {
			return false
		}
	}

	return true
}
