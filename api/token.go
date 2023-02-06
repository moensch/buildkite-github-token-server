package api

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/moensch/buildkite-github-token-server/internal/repoparser"
)

// TokenRequest represents the HTTP request body to the /token endpoint
type TokenRequest struct {
	Repositories []repoparser.RepositoryName `json:"repositories"`
	AccessLevel  AccessLevel                 `json:"access_level" default:"read"`
}

type AccessLevel string

const (
	AccessLevelRead  AccessLevel = "read"
	AccessLevelWrite AccessLevel = "write"
)

type rawTokenRequest struct {
	Repositories []string    `json:"repositories"`
	AccessLevel  AccessLevel `json:"access_level" default:"read"`
}

// UnmarshalJSON is a custom unmarshaller that ensures all "repositories" point to a valid repository name
func (tr *TokenRequest) UnmarshalJSON(data []byte) error {
	raw := rawTokenRequest{}
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	if raw.AccessLevel == "" {
		raw.AccessLevel = "read"
	}
	tr.AccessLevel = raw.AccessLevel

	tr.Repositories = make([]repoparser.RepositoryName, len(raw.Repositories))
	for idx, repoString := range raw.Repositories {
		repo, err := repoparser.ExtractOrgRepoFromURL(repoString)
		if err != nil {
			return fmt.Errorf("cannot parse repo %s: %w", repoString, err)
		}
		tr.Repositories[idx] = repo
	}
	return nil
}

// TokenResponse represents the HTTP response to the /token request
type TokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	RequestID string    `json:"req_id"`
}
