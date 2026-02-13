// Package googledocsapi provides a typed Google Docs API client powered by ogen.
package googledocsapi

import (
	"context"

	gen "mcpist/server/pkg/googledocsapi/gen"
)

const serverURL = "https://docs.googleapis.com/v1"

// tokenSecuritySource implements gen.SecuritySource using a static Bearer token.
type tokenSecuritySource struct {
	token string
}

func (s *tokenSecuritySource) BearerAuth(_ context.Context, _ gen.OperationName) (gen.BearerAuth, error) {
	return gen.BearerAuth{Token: s.token}, nil
}

// NewClient creates a new Google Docs API client with the given access token.
func NewClient(token string) (*gen.Client, error) {
	return gen.NewClient(serverURL, &tokenSecuritySource{token: token})
}
