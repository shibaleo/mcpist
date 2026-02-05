package notion

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"mcpist/server/internal/httpclient"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/store"
)

const (
	notionAPIBase       = "https://api.notion.com/v1"
	notionTokenURL      = "https://api.notion.com/v1/oauth/token"
	notionVersion       = "2022-06-28"
	tokenRefreshBuffer  = 300 // Refresh 5 minutes before expiry
)

var client = httpclient.New()

// getCredentials retrieves credentials from Vault via RPC for the given user
// and refreshes the token if needed (for OAuth2)
func getCredentials(ctx context.Context) *store.Credentials {
	// Get user_id from AuthContext (set by authorization middleware)
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		log.Printf("[notion] No auth context for token retrieval")
		return nil
	}

	userID := authCtx.UserID
	if userID == "" {
		log.Printf("[notion] No user_id in auth context for token retrieval")
		return nil
	}

	credentials, err := store.GetTokenStore().GetModuleToken(ctx, userID, "notion")
	if err != nil {
		log.Printf("[notion] Failed to get token from vault: %v", err)
		return nil
	}

	// Check if token needs refresh (OAuth2 only)
	if credentials.AuthType == store.AuthTypeOAuth2 && credentials.RefreshToken != "" {
		if needsRefresh(credentials) {
			log.Printf("[notion] Token expired or expiring soon, refreshing...")
			refreshed, err := refreshToken(ctx, userID, credentials)
			if err != nil {
				log.Printf("[notion] Token refresh failed: %v", err)
				// Return original credentials and let the API call fail
				return credentials
			}
			log.Printf("[notion] Token refreshed successfully")
			return refreshed
		}
	}

	return credentials
}

// needsRefresh checks if the token needs to be refreshed
func needsRefresh(creds *store.Credentials) bool {
	// If no expiry set, assume token doesn't expire (legacy behavior)
	if creds.ExpiresAt == 0 {
		return false
	}
	now := time.Now().Unix()
	// Refresh if expired or expiring within buffer period
	return now >= (int64(creds.ExpiresAt) - tokenRefreshBuffer)
}

// refreshToken exchanges the refresh token for a new access token
func refreshToken(ctx context.Context, userID string, creds *store.Credentials) (*store.Credentials, error) {
	// Get OAuth app credentials (client_id, client_secret)
	oauthApp, err := store.GetTokenStore().GetOAuthAppCredentials(ctx, "notion")
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth app credentials: %w", err)
	}

	// Notion uses HTTP Basic Auth for token refresh
	basicAuth := base64.StdEncoding.EncodeToString([]byte(oauthApp.ClientID + ":" + oauthApp.ClientSecret))

	// Create refresh request body
	reqBody := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": creds.RefreshToken,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", notionTokenURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+basicAuth)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed: status %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		TokenType    string `json:"token_type"`
		BotID        string `json:"bot_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode refresh response: %w", err)
	}

	// Calculate expiry time
	var expiresAt store.FlexibleTime
	if tokenResp.ExpiresIn > 0 {
		expiresAt = store.FlexibleTime(time.Now().Unix() + tokenResp.ExpiresIn)
	}

	// Update credentials with new tokens
	newCreds := &store.Credentials{
		AuthType:     store.AuthTypeOAuth2,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken, // Notion may return new refresh token
		ExpiresAt:    expiresAt,
		Metadata:     creds.Metadata, // Preserve metadata
	}

	// If Notion didn't return a new refresh token, keep the old one
	if newCreds.RefreshToken == "" {
		newCreds.RefreshToken = creds.RefreshToken
	}

	// Save updated credentials to Vault
	err = store.GetTokenStore().UpdateModuleToken(ctx, userID, "notion", newCreds)
	if err != nil {
		log.Printf("[notion] Failed to save refreshed token: %v", err)
		// Continue anyway, the token is still valid for this request
	}

	return newCreds, nil
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
	default:
		// Fallback: if auth type is unknown but token exists, use it
		if creds.AccessToken != "" {
			h["Authorization"] = "Bearer " + creds.AccessToken
		}
	}

	return h
}
