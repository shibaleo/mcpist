// Package githubapi provides a typed GitHub API client powered by ogen.
package githubapi

import (
	"context"

	gen "mcpist/server/pkg/githubapi/gen"
)

const serverURL = "https://api.github.com"

// tokenSecuritySource implements gen.SecuritySource using a static token.
type tokenSecuritySource struct {
	token string
}

func (s *tokenSecuritySource) BearerAuth(_ context.Context, _ gen.OperationName) (gen.BearerAuth, error) {
	return gen.BearerAuth{Token: s.token}, nil
}

// NewClient creates a new GitHub API client with the given access token.
func NewClient(token string) (*gen.Client, error) {
	return gen.NewClient(serverURL, &tokenSecuritySource{token: token})
}
