package credentialprovider

const (
	APIVersion           = "credentialprovider.kubelet.k8s.io/v1"
	KindRequest          = "CredentialProviderRequest"
	KindResponse         = "CredentialProviderResponse"
	CacheKeyTypeRegistry = "Registry"
	DockerUsername       = "oauth2accesstoken"
)

type TypeMeta struct {
	Kind       string `json:"kind,omitempty"`
	APIVersion string `json:"apiVersion,omitempty"`
}

type Duration struct {
	Duration string `json:"duration,omitempty"`
}

// CredentialProviderRequest is sent by kubelet to request credentials for an image.
// https://kubernetes.io/docs/reference/config-api/kubelet-credentialprovider.v1/
type CredentialProviderRequest struct {
	TypeMeta `json:",inline"`
	Image    string `json:"image"`
}

type CredentialProviderResponse struct {
	TypeMeta      `json:",inline"`
	CacheKeyType  string                `json:"cacheKeyType"`
	CacheDuration *Duration             `json:"cacheDuration,omitempty"`
	Auth          map[string]AuthConfig `json:"auth,omitempty"`
}

type AuthConfig struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}
