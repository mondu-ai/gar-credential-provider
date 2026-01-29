package credentialprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTokenFetcher struct {
	token string
	err   error
}

func (m *mockTokenFetcher) GetAccessToken(_ context.Context) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.token, nil
}

func TestHandleRequest_Success(t *testing.T) {
	tests := []struct {
		name             string
		image            string
		expectedRegistry string
	}{
		{
			name:             "artifact registry with tag",
			image:            "europe-west3-docker.pkg.dev/some-name/repo/image:v1.0.0",
			expectedRegistry: "europe-west3-docker.pkg.dev",
		},
		{
			name:             "artifact registry with digest",
			image:            "us-docker.pkg.dev/project/repo/image@sha256:abc123",
			expectedRegistry: "us-docker.pkg.dev",
		},
		{
			name:             "gcr.io image",
			image:            "gcr.io/project/image:latest",
			expectedRegistry: "gcr.io",
		},
		{
			name:             "artifact registry without tag",
			image:            "asia-docker.pkg.dev/project/repo/image",
			expectedRegistry: "asia-docker.pkg.dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := &mockTokenFetcher{token: "test-access-token"}
			provider := New(fetcher)

			req := CredentialProviderRequest{
				TypeMeta: TypeMeta{
					APIVersion: APIVersion,
					Kind:       KindRequest,
				},
				Image: tt.image,
			}

			reqBytes, err := json.Marshal(req)
			require.NoError(t, err)

			stdin := bytes.NewReader(reqBytes)
			stdout := &bytes.Buffer{}

			err = provider.HandleRequest(stdin, stdout)
			require.NoError(t, err)

			var resp CredentialProviderResponse
			err = json.Unmarshal(stdout.Bytes(), &resp)
			require.NoError(t, err)

			assert.Equal(t, APIVersion, resp.APIVersion)
			assert.Equal(t, KindResponse, resp.Kind)
			assert.Equal(t, CacheKeyTypeRegistry, resp.CacheKeyType)

			auth, ok := resp.Auth[tt.expectedRegistry]
			require.True(t, ok, "Auth missing for registry %q", tt.expectedRegistry)
			assert.Equal(t, DockerUsername, auth.Username)
			assert.Equal(t, "test-access-token", auth.Password)
		})
	}
}

func TestHandleRequest_InvalidRequest(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		errContains string
	}{
		{
			name:        "invalid JSON",
			input:       "not json",
			errContains: "failed to decode request",
		},
		{
			name: "wrong API version",
			input: `{
				"apiVersion": "credentialprovider.kubelet.k8s.io/v1beta1",
				"kind": "CredentialProviderRequest",
				"image": "gcr.io/project/image"
			}`,
			errContains: "unsupported API version",
		},
		{
			name: "wrong kind",
			input: `{
				"apiVersion": "credentialprovider.kubelet.k8s.io/v1",
				"kind": "WrongKind",
				"image": "gcr.io/project/image"
			}`,
			errContains: "unexpected kind",
		},
		{
			name: "docker hub image",
			input: `{
				"apiVersion": "credentialprovider.kubelet.k8s.io/v1",
				"kind": "CredentialProviderRequest",
				"image": "nginx:latest"
			}`,
			errContains: "does not appear to be from a custom registry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := &mockTokenFetcher{token: "test-token"}
			provider := New(fetcher)

			stdin := strings.NewReader(tt.input)
			stdout := &bytes.Buffer{}

			err := provider.HandleRequest(stdin, stdout)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

func TestExtractRegistry(t *testing.T) {
	tests := []struct {
		image   string
		want    string
		wantErr bool
	}{
		{"europe-west3-docker.pkg.dev/project/repo/image:tag", "europe-west3-docker.pkg.dev", false},
		{"gcr.io/project/image", "gcr.io", false},
		{"us-docker.pkg.dev/project/repo/image@sha256:abc", "us-docker.pkg.dev", false},
		{"localhost/image", "localhost", false},
		{"localhost:5000/image", "localhost", false},
		{"nginx", "", true},
		{"library/nginx", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.image, func(t *testing.T) {
			got, err := extractRegistry(tt.image)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
