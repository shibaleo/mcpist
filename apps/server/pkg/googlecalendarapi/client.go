// Package googlecalendarapi provides a typed Google Calendar API client powered by ogen.
package googlecalendarapi

import (
	"context"

	gen "mcpist/server/pkg/googlecalendarapi/gen"
)

const serverURL = "https://www.googleapis.com/calendar/v3"

// tokenSecuritySource implements gen.SecuritySource using a static Bearer token.
type tokenSecuritySource struct {
	token string
}

func (s *tokenSecuritySource) BearerAuth(_ context.Context, _ gen.OperationName) (gen.BearerAuth, error) {
	return gen.BearerAuth{Token: s.token}, nil
}

// NewClient creates a new Google Calendar API client with the given access token.
func NewClient(token string) (*gen.Client, error) {
	return gen.NewClient(serverURL, &tokenSecuritySource{token: token})
}
