// Package airtableapi provides a typed Airtable API client powered by ogen.
package airtableapi

import (
	"context"

	gen "mcpist/server/pkg/airtableapi/gen"
)

const serverURL = "https://api.airtable.com/v0"

// tokenSecuritySource implements gen.SecuritySource using a static Bearer token.
type tokenSecuritySource struct {
	token string
}

func (s *tokenSecuritySource) BearerAuth(_ context.Context, _ gen.OperationName) (gen.BearerAuth, error) {
	return gen.BearerAuth{Token: s.token}, nil
}

// NewClient creates a new Airtable API client with the given access token.
func NewClient(token string) (*gen.Client, error) {
	return gen.NewClient(serverURL, &tokenSecuritySource{token: token})
}
