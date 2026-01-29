package gcpauth

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google/externalaccount"
)

// TokenFetcher obtains GCP access tokens via Workload Identity Federation.
type TokenFetcher interface {
	GetAccessToken(ctx context.Context) (string, error)
}

var _ TokenFetcher = (*tokenFetcher)(nil)

type tokenFetcher struct {
	tokenSource oauth2.TokenSource
}

// NewTokenFetcher creates a TokenFetcher from a config file path.
func NewTokenFetcher(configPath string) (TokenFetcher, error) {
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return NewTokenFetcherFromConfig(config)
}

// NewTokenFetcherFromConfig creates a TokenFetcher from a parsed config.
func NewTokenFetcherFromConfig(config *Config) (TokenFetcher, error) {
	credSource := &externalaccount.CredentialSource{}

	if config.IsAWS() {
		credSource.EnvironmentID = config.CredentialSource.EnvironmentID
		credSource.RegionURL = config.CredentialSource.RegionURL
		credSource.URL = config.CredentialSource.URL
		credSource.RegionalCredVerificationURL = config.CredentialSource.RegionalCredVerificationURL
		credSource.IMDSv2SessionTokenURL = config.CredentialSource.IMDSv2SessionTokenURL
	} else if config.CredentialSource.File != "" {
		credSource.File = config.CredentialSource.File
		if config.CredentialSource.Format != nil {
			credSource.Format = externalaccount.Format{
				Type:                  config.CredentialSource.Format.Type,
				SubjectTokenFieldName: config.CredentialSource.Format.SubjectTokenFieldName,
			}
		}
	} else if config.CredentialSource.URL != "" {
		credSource.URL = config.CredentialSource.URL
		credSource.Headers = config.CredentialSource.Headers
		if config.CredentialSource.Format != nil {
			credSource.Format = externalaccount.Format{
				Type:                  config.CredentialSource.Format.Type,
				SubjectTokenFieldName: config.CredentialSource.Format.SubjectTokenFieldName,
			}
		}
	}

	externalConfig := externalaccount.Config{
		Audience:                       config.Audience,
		SubjectTokenType:               config.SubjectTokenType,
		TokenURL:                       config.TokenURL,
		CredentialSource:               credSource,
		ServiceAccountImpersonationURL: config.ServiceAccountImpersonationURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/cloud-platform",
		},
	}

	ctx := context.Background()
	tokenSource, err := externalaccount.NewTokenSource(ctx, externalConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create token source: %w", err)
	}

	return &tokenFetcher{
		tokenSource: tokenSource,
	}, nil
}

func (t *tokenFetcher) GetAccessToken(_ context.Context) (string, error) {
	token, err := t.tokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}

	return token.AccessToken, nil
}
