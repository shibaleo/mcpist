// Package grafanaapi provides a typed Grafana HTTP API client powered by ogen.
package grafanaapi

import (
	"context"

	"github.com/ogen-go/ogen/ogenerrors"

	gen "mcpist/server/pkg/grafanaapi/gen"
)

// bearerSecuritySource implements gen.SecuritySource using a Bearer token.
type bearerSecuritySource struct {
	token string
}

func (s *bearerSecuritySource) BearerAuth(_ context.Context, _ gen.OperationName) (gen.BearerAuth, error) {
	return gen.BearerAuth{Token: s.token}, nil
}

func (s *bearerSecuritySource) BasicAuth(_ context.Context, _ gen.OperationName) (gen.BasicAuth, error) {
	return gen.BasicAuth{}, ogenerrors.ErrSkipClientSecurity
}

// basicSecuritySource implements gen.SecuritySource using Basic auth.
type basicSecuritySource struct {
	username string
	password string
}

func (s *basicSecuritySource) BearerAuth(_ context.Context, _ gen.OperationName) (gen.BearerAuth, error) {
	return gen.BearerAuth{}, ogenerrors.ErrSkipClientSecurity
}

func (s *basicSecuritySource) BasicAuth(_ context.Context, _ gen.OperationName) (gen.BasicAuth, error) {
	return gen.BasicAuth{Username: s.username, Password: s.password}, nil
}

// NewBearerClient creates a new Grafana API client with Bearer token authentication.
func NewBearerClient(serverURL, token string) (*gen.Client, error) {
	return gen.NewClient(serverURL, &bearerSecuritySource{token: token})
}

// NewBasicClient creates a new Grafana API client with Basic authentication.
func NewBasicClient(serverURL, username, password string) (*gen.Client, error) {
	return gen.NewClient(serverURL, &basicSecuritySource{username: username, password: password})
}
