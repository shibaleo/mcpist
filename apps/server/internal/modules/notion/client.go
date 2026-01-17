package notion

import (
	"context"
	"log"

	"mcpist/server/internal/httpclient"
	"mcpist/server/internal/vault"
)

const (
	notionAPIBase = "https://api.notion.com/v1"
	notionVersion = "2022-06-28"
)

var client = httpclient.New()

// getToken retrieves token from vault for the given user
// Priority: OAuth token > Long-term token
func getToken(ctx context.Context) string {
	// Get user_id from context (will be set by auth middleware)
	// For now, use "dev" as mock user_id
	userID := "dev"
	if uid := ctx.Value("user_id"); uid != nil {
		if s, ok := uid.(string); ok {
			userID = s
		}
	}

	tokens, err := vault.GetTokens(userID, "notion")
	if err != nil {
		log.Printf("Failed to get notion tokens from vault: %v", err)
		return ""
	}

	// Prefer OAuth token if available, fallback to long-term token
	if tokens.OAuthToken != "" {
		return tokens.OAuthToken
	}
	return tokens.LongTermToken
}

func headers(ctx context.Context) map[string]string {
	return map[string]string{
		"Authorization":  "Bearer " + getToken(ctx),
		"Notion-Version": notionVersion,
	}
}
