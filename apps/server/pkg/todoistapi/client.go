// Package todoistapi provides a typed Todoist API client powered by ogen.
package todoistapi

import (
	"context"

	gen "mcpist/server/pkg/todoistapi/gen"
)

const serverURL = "https://api.todoist.com/api/v1"

// tokenSecuritySource implements gen.SecuritySource using a static Bearer token.
type tokenSecuritySource struct {
	token string
}

func (s *tokenSecuritySource) BearerAuth(_ context.Context, _ gen.OperationName) (gen.BearerAuth, error) {
	return gen.BearerAuth{Token: s.token}, nil
}

// NewClient creates a new Todoist API client with the given access token.
func NewClient(token string) (*gen.Client, error) {
	return gen.NewClient(serverURL, &tokenSecuritySource{token: token})
}
