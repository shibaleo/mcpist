// Package googletasksapi provides a typed Google Tasks API client powered by ogen.
package googletasksapi

import (
	"context"

	gen "mcpist/server/pkg/googletasksapi/gen"
)

const serverURL = "https://tasks.googleapis.com/tasks/v1"

// tokenSecuritySource implements gen.SecuritySource using a static Bearer token.
type tokenSecuritySource struct {
	token string
}

func (s *tokenSecuritySource) BearerAuth(_ context.Context, _ gen.OperationName) (gen.BearerAuth, error) {
	return gen.BearerAuth{Token: s.token}, nil
}

// NewClient creates a new Google Tasks API client with the given access token.
func NewClient(token string) (*gen.Client, error) {
	return gen.NewClient(serverURL, &tokenSecuritySource{token: token})
}
