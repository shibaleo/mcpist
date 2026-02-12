// Package ticktickapi provides a typed TickTick API client powered by ogen.
package ticktickapi

import (
	"context"

	gen "mcpist/server/pkg/ticktickapi/gen"
)

const serverURL = "https://api.ticktick.com/open/v1"

// tokenSecuritySource implements gen.SecuritySource using a static Bearer token.
type tokenSecuritySource struct {
	token string
}

func (s *tokenSecuritySource) BearerAuth(_ context.Context, _ gen.OperationName) (gen.BearerAuth, error) {
	return gen.BearerAuth{Token: s.token}, nil
}

// NewClient creates a new TickTick API client with the given access token.
func NewClient(token string) (*gen.Client, error) {
	return gen.NewClient(serverURL, &tokenSecuritySource{token: token})
}
