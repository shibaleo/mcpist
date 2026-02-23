package trelloapi

import (
	"context"

	gen "mcpist/server/pkg/trelloapi/gen"
)

const serverURL = "https://api.trello.com/1"

type trelloSecuritySource struct {
	apiKey string
	token  string
}

func (s *trelloSecuritySource) ApiKey(_ context.Context, _ gen.OperationName) (gen.ApiKey, error) {
	return gen.ApiKey{APIKey: s.apiKey}, nil
}

func (s *trelloSecuritySource) ApiToken(_ context.Context, _ gen.OperationName) (gen.ApiToken, error) {
	return gen.ApiToken{APIKey: s.token}, nil
}

// NewClient creates a new ogen-generated Trello API client.
func NewClient(apiKey, token string) (*gen.Client, error) {
	return gen.NewClient(serverURL, &trelloSecuritySource{apiKey: apiKey, token: token})
}
