// Package googlesheetsapi provides a typed Google Sheets API client powered by ogen.
package googlesheetsapi

import (
	"context"

	gen "mcpist/server/pkg/googlesheetsapi/gen"
)

const serverURL = "https://sheets.googleapis.com/v4"

// tokenSecuritySource implements gen.SecuritySource using a static Bearer token.
type tokenSecuritySource struct {
	token string
}

func (s *tokenSecuritySource) BearerAuth(_ context.Context, _ gen.OperationName) (gen.BearerAuth, error) {
	return gen.BearerAuth{Token: s.token}, nil
}

// NewClient creates a new Google Sheets API client with the given access token.
func NewClient(token string) (*gen.Client, error) {
	return gen.NewClient(serverURL, &tokenSecuritySource{token: token})
}
