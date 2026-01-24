package notion

import (
	"context"
	"log"

	"mcpist/server/internal/httpclient"
	"mcpist/server/internal/token"
)

const (
	notionAPIBase = "https://api.notion.com/v1"
	notionVersion = "2022-06-28"
)

var client = httpclient.New()

// getToken retrieves token from Vault via RPC for the given user
func getToken(ctx context.Context) string {
	// Get user_id from context (set by auth middleware via X-User-ID header)
	userID := ""
	if uid := ctx.Value("user_id"); uid != nil {
		if s, ok := uid.(string); ok {
			userID = s
		}
	}

	if userID == "" {
		log.Printf("No user_id in context for notion token retrieval")
		return ""
	}

	credentials, err := token.GetStore().GetModuleToken(ctx, userID, "notion")
	if err != nil {
		log.Printf("Failed to get notion token from vault: %v", err)
		return ""
	}

	return credentials.AccessToken
}

func headers(ctx context.Context) map[string]string {
	return map[string]string{
		"Authorization":  "Bearer " + getToken(ctx),
		"Notion-Version": notionVersion,
	}
}
