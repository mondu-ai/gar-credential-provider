package gcpauth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_AWSCredentials(t *testing.T) {
	configContent := `{
		"type": "external_account",
		"audience": "//iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/my-pool/providers/aws-provider",
		"subject_token_type": "urn:ietf:params:aws:token-type:aws4_request",
		"token_url": "https://sts.googleapis.com/v1/token",
		"service_account_impersonation_url": "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/sa@project.iam.gserviceaccount.com:generateAccessToken",
		"credential_source": {
			"environment_id": "aws1",
			"regional_cred_verification_url": "https://sts.{region}.amazonaws.com?Action=GetCallerIdentity&Version=2011-06-15"
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	config, err := LoadConfig(configPath)
	require.NoError(t, err)

	assert.Equal(t, "external_account", config.Type)
	assert.True(t, config.IsAWS())
	assert.Equal(t, "aws1", config.CredentialSource.EnvironmentID)
}

func TestLoadConfig_OIDCFileCredentials(t *testing.T) {
	configContent := `{
		"type": "external_account",
		"audience": "//iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/my-pool/providers/oidc-provider",
		"subject_token_type": "urn:ietf:params:oauth:token-type:jwt",
		"token_url": "https://sts.googleapis.com/v1/token",
		"credential_source": {
			"file": "/var/run/secrets/tokens/gcp-token"
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	config, err := LoadConfig(configPath)
	require.NoError(t, err)

	assert.False(t, config.IsAWS())
	assert.Equal(t, "/var/run/secrets/tokens/gcp-token", config.CredentialSource.File)
}

func TestLoadConfig_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		errContains string
	}{
		{
			name:        "wrong type",
			config:      `{"type": "service_account"}`,
			errContains: "unsupported credential type",
		},
		{
			name:        "missing audience",
			config:      `{"type": "external_account", "subject_token_type": "jwt", "token_url": "https://sts.googleapis.com/v1/token", "credential_source": {"file": "/token"}}`,
			errContains: "audience is required",
		},
		{
			name:        "missing token_url",
			config:      `{"type": "external_account", "audience": "aud", "subject_token_type": "jwt", "credential_source": {"file": "/token"}}`,
			errContains: "token_url is required",
		},
		{
			name:        "missing credential_source",
			config:      `{"type": "external_account", "audience": "aud", "subject_token_type": "jwt", "token_url": "https://sts.googleapis.com/v1/token"}`,
			errContains: "credential_source must specify",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.json")
			err := os.WriteFile(configPath, []byte(tt.config), 0644)
			require.NoError(t, err)

			_, err = LoadConfig(configPath)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.json")
	require.Error(t, err)
}
