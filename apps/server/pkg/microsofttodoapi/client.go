// Package microsofttodoapi provides a typed Microsoft Graph To Do API client powered by ogen.
package microsofttodoapi

import (
	"context"

	gen "mcpist/server/pkg/microsofttodoapi/gen"
)

const serverURL = "https://graph.microsoft.com/v1.0"

// tokenSecuritySource implements gen.SecuritySource using a static Bearer token.
type tokenSecuritySource struct {
	token string
}

func (s *tokenSecuritySource) BearerAuth(_ context.Context, _ gen.OperationName) (gen.BearerAuth, error) {
	return gen.BearerAuth{Token: s.token}, nil
}

// NewClient creates a new Microsoft Graph To Do API client with the given access token.
func NewClient(token string) (*gen.Client, error) {
	return gen.NewClient(serverURL, &tokenSecuritySource{token: token})
}
