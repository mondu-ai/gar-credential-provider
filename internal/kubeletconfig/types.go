package kubeletconfig

// Config represents kubelet credential provider configuration.
// See: https://kubernetes.io/docs/reference/config-api/kubelet-credentialprovider.v1/
type Config struct {
	APIVersion string     `json:"apiVersion"`
	Kind       string     `json:"kind"`
	Providers  []Provider `json:"providers"`
}

// Provider represents a single credential provider entry.
type Provider struct {
	Name                 string   `json:"name"`
	MatchImages          []string `json:"matchImages"`
	DefaultCacheDuration string   `json:"defaultCacheDuration"`
	APIVersion           string   `json:"apiVersion"`
	Args                 []string `json:"args,omitempty"`
	Env                  []EnvVar `json:"env,omitempty"`
}

// EnvVar represents an environment variable for a provider.
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
