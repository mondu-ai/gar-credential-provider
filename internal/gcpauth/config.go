package gcpauth

import (
	"encoding/json"
	"fmt"
	"os"
)

// GCP external account credential configuration for Workload Identity Federation.
// https://cloud.google.com/iam/docs/workload-identity-federation-with-other-clouds
type Config struct {
	Type                           string           `json:"type"`
	Audience                       string           `json:"audience"`
	SubjectTokenType               string           `json:"subject_token_type"`
	TokenURL                       string           `json:"token_url"`
	ServiceAccountImpersonationURL string           `json:"service_account_impersonation_url,omitempty"`
	CredentialSource               CredentialSource `json:"credential_source"`
	QuotaProjectID                 string           `json:"quota_project_id,omitempty"`
}

type CredentialSource struct {
	EnvironmentID               string            `json:"environment_id,omitempty"`
	RegionURL                   string            `json:"region_url,omitempty"`
	URL                         string            `json:"url,omitempty"`
	RegionalCredVerificationURL string            `json:"regional_cred_verification_url,omitempty"`
	IMDSv2SessionTokenURL       string            `json:"imdsv2_session_token_url,omitempty"`
	File                        string            `json:"file,omitempty"`
	Headers                     map[string]string `json:"headers,omitempty"`
	Format                      *CredentialFormat `json:"format,omitempty"`
}

type CredentialFormat struct {
	Type                  string `json:"type,omitempty"`
	SubjectTokenFieldName string `json:"subject_token_field_name,omitempty"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path) //#nosec G304 -- path from CLI flag
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

func (c *Config) Validate() error {
	if c.Type != "external_account" {
		return fmt.Errorf("unsupported credential type: %q (expected \"external_account\")", c.Type)
	}

	if c.Audience == "" {
		return fmt.Errorf("audience is required")
	}

	if c.SubjectTokenType == "" {
		return fmt.Errorf("subject_token_type is required")
	}

	if c.TokenURL == "" {
		return fmt.Errorf("token_url is required")
	}

	// Check credential source
	cs := c.CredentialSource
	hasAWS := cs.EnvironmentID != ""
	hasFile := cs.File != ""
	hasURL := cs.URL != ""

	if !hasAWS && !hasFile && !hasURL {
		return fmt.Errorf("credential_source must specify environment_id (AWS), file, or url")
	}

	return nil
}

func (c *Config) IsAWS() bool {
	return c.CredentialSource.EnvironmentID == "aws1"
}
