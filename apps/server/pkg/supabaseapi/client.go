// Package supabaseapi provides a typed Supabase Management API client powered by ogen.
package supabaseapi

import (
	"context"

	gen "mcpist/server/pkg/supabaseapi/gen"
)

const serverURL = "https://api.supabase.com/v1"

// tokenSecuritySource implements gen.SecuritySource using a static token.
type tokenSecuritySource struct {
	token string
}

func (s *tokenSecuritySource) BearerAuth(_ context.Context, _ gen.OperationName) (gen.BearerAuth, error) {
	return gen.BearerAuth{Token: s.token}, nil
}

// NewClient creates a new Supabase Management API client with the given access token.
func NewClient(token string) (*gen.Client, error) {
	return gen.NewClient(serverURL, &tokenSecuritySource{token: token})
}
