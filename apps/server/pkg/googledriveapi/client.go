// Package googledriveapi provides a typed Google Drive API client powered by ogen.
// This package is shared by google_drive, google_docs, google_sheets, and google_apps_script modules.
package googledriveapi

import (
	"context"

	gen "mcpist/server/pkg/googledriveapi/gen"
)

const serverURL = "https://www.googleapis.com/drive/v3"

// tokenSecuritySource implements gen.SecuritySource using a static Bearer token.
type tokenSecuritySource struct {
	token string
}

func (s *tokenSecuritySource) BearerAuth(_ context.Context, _ gen.OperationName) (gen.BearerAuth, error) {
	return gen.BearerAuth{Token: s.token}, nil
}

// NewClient creates a new Google Drive API client with the given access token.
func NewClient(token string) (*gen.Client, error) {
	return gen.NewClient(serverURL, &tokenSecuritySource{token: token})
}
