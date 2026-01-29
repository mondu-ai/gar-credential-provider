// Package credentialprovider implements the kubelet credential provider API v1.
package credentialprovider

// Kubelet credential provider API constants.
const (
	APIVersion           = "credentialprovider.kubelet.k8s.io/v1"
	KindRequest          = "CredentialProviderRequest"
	KindResponse         = "CredentialProviderResponse"
	CacheKeyTypeRegistry = "Registry"
	DockerUsername       = "oauth2accesstoken"
)

// TypeMeta describes the API version and kind of a resource.
type TypeMeta struct {
	Kind       string `json:"kind,omitempty"`
	APIVersion string `json:"apiVersion,omitempty"`
}

// Duration represents a cache duration.
type Duration struct {
	Duration string `json:"duration,omitempty"`
}

// Request is sent by kubelet to request credentials for an image.
// https://kubernetes.io/docs/reference/config-api/kubelet-credentialprovider.v1/
type Request struct {
	TypeMeta `json:",inline"`
	Image    string `json:"image"`
}

// Response is returned to kubelet with credentials for pulling images.
type Response struct {
	TypeMeta      `json:",inline"`
	CacheKeyType  string                `json:"cacheKeyType"`
	CacheDuration *Duration             `json:"cacheDuration,omitempty"`
	Auth          map[string]AuthConfig `json:"auth,omitempty"`
}

// AuthConfig contains Docker registry authentication credentials.
type AuthConfig struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}
