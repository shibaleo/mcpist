// Package asanaapi provides a typed Asana API client powered by ogen.
package asanaapi

import (
	"context"

	gen "mcpist/server/pkg/asanaapi/gen"
)

const serverURL = "https://app.asana.com/api/1.0"

// tokenSecuritySource implements gen.SecuritySource using a static token.
type tokenSecuritySource struct {
	token string
}

func (s *tokenSecuritySource) BearerAuth(_ context.Context, _ gen.OperationName) (gen.BearerAuth, error) {
	return gen.BearerAuth{Token: s.token}, nil
}

// NewClient creates a new Asana API client with the given access token.
func NewClient(token string) (*gen.Client, error) {
	return gen.NewClient(serverURL, &tokenSecuritySource{token: token})
}
