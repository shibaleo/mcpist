package notion

import (
	"context"
	"log"

	"mcpist/server/internal/httpclient"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/store"
)

const (
	notionAPIBase = "https://api.notion.com/v1"
	notionVersion = "2022-06-28"
)

var client = httpclient.New()

// getCredentials retrieves credentials from Vault via RPC for the given user
func getCredentials(ctx context.Context) *store.Credentials {
	// Get user_id from AuthContext (set by authorization middleware)
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		log.Printf("No auth context for notion token retrieval")
		return nil
	}

	userID := authCtx.UserID
	if userID == "" {
		log.Printf("No user_id in auth context for notion token retrieval")
		return nil
	}

	credentials, err := store.GetTokenStore().GetModuleToken(ctx, userID, "notion")
	if err != nil {
		log.Printf("Failed to get notion token from vault: %v", err)
		return nil
	}

	return credentials
}

func headers(ctx context.Context) map[string]string {
	creds := getCredentials(ctx)
	if creds == nil {
		return map[string]string{}
	}

	h := map[string]string{
		"Notion-Version": notionVersion,
	}

	// Notion supports OAuth2 and API Key (both use Bearer token)
	switch creds.AuthType {
	case store.AuthTypeOAuth2, store.AuthTypeAPIKey:
		h["Authorization"] = "Bearer " + creds.AccessToken
	}

	return h
}
