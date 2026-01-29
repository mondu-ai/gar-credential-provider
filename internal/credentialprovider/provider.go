package credentialprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/mondu-ai/gar-credential-provider/internal/gcpauth"
)

type Provider interface {
	HandleRequest(stdin io.Reader, stdout io.Writer) error
}

var _ Provider = (*provider)(nil)

type provider struct {
	tokenFetcher gcpauth.TokenFetcher
}

func New(tokenFetcher gcpauth.TokenFetcher) Provider {
	return &provider{
		tokenFetcher: tokenFetcher,
	}
}

func (p *provider) HandleRequest(stdin io.Reader, stdout io.Writer) error {
	var req CredentialProviderRequest
	if err := json.NewDecoder(stdin).Decode(&req); err != nil {
		return fmt.Errorf("failed to decode request: %w", err)
	}

	if req.APIVersion != APIVersion {
		return fmt.Errorf("unsupported API version: %s (expected %s)", req.APIVersion, APIVersion)
	}

	if req.Kind != KindRequest {
		return fmt.Errorf("unexpected kind: %s (expected %s)", req.Kind, KindRequest)
	}

	registry, err := extractRegistry(req.Image)
	if err != nil {
		return fmt.Errorf("failed to extract registry from image %q: %w", req.Image, err)
	}

	token, err := p.tokenFetcher.GetAccessToken(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	resp := CredentialProviderResponse{
		TypeMeta: TypeMeta{
			APIVersion: APIVersion,
			Kind:       KindResponse,
		},
		CacheKeyType: CacheKeyTypeRegistry,
		Auth: map[string]AuthConfig{
			registry: {
				Username: DockerUsername,
				Password: token,
			},
		},
	}

	if err := json.NewEncoder(stdout).Encode(&resp); err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}

	return nil
}

// Examples:
//   - "europe-west3-docker.pkg.dev/project/repo/image:tag" → "europe-west3-docker.pkg.dev"
//   - "gcr.io/project/image" → "gcr.io"
func extractRegistry(image string) (string, error) {
	// Remove tag or digest suffix
	image = strings.Split(image, "@")[0]
	image = strings.Split(image, ":")[0]

	// First part before / is the registry
	parts := strings.SplitN(image, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "", fmt.Errorf("invalid image reference: %q", image)
	}

	registry := parts[0]

	// Validate it looks like a hostname (contains a dot or is localhost)
	if !strings.Contains(registry, ".") && registry != "localhost" {
		// Might be a Docker Hub image like "nginx" or "library/nginx"
		return "", fmt.Errorf("image %q does not appear to be from a custom registry", image)
	}

	// Ensure it's a valid hostname
	if _, err := url.Parse("https://" + registry); err != nil {
		return "", fmt.Errorf("invalid registry hostname %q: %w", registry, err)
	}

	return registry, nil
}
