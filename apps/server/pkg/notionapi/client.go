// Package notionapi provides a typed Notion REST API client powered by ogen.
package notionapi

import (
	"context"
	"net/http"

	gen "mcpist/server/pkg/notionapi/gen"
)

const serverURL = "https://api.notion.com/v1"

// notionVersionTransport injects the Notion-Version header into every request.
type notionVersionTransport struct {
	base    http.RoundTripper
	version string
}

func (t *notionVersionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Notion-Version", t.version)
	return t.base.RoundTrip(req)
}

// bearerSecuritySource implements gen.SecuritySource using a Bearer token.
type bearerSecuritySource struct {
	token string
}

func (s *bearerSecuritySource) BearerAuth(_ context.Context, _ gen.OperationName) (gen.BearerAuth, error) {
	return gen.BearerAuth{Token: s.token}, nil
}

// NewClient creates a new Notion API client with Bearer token authentication.
// The Notion-Version header is automatically injected by a custom RoundTripper.
func NewClient(token, version string) (*gen.Client, error) {
	httpClient := &http.Client{
		Transport: &notionVersionTransport{
			base:    http.DefaultTransport,
			version: version,
		},
	}
	return gen.NewClient(serverURL, &bearerSecuritySource{token: token}, gen.WithClient(httpClient))
}
