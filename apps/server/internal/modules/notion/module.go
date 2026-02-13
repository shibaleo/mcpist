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

	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/store"
	"mcpist/server/pkg/notionapi"
	gen "mcpist/server/pkg/notionapi/gen"
)

const (
	notionTokenURL     = "https://api.notion.com/v1/oauth/token"
	notionVersion      = "2022-06-28"
	tokenRefreshBuffer = 300 // Refresh 5 minutes before expiry
)

// NotionModule implements the Module interface for Notion API
type NotionModule struct{}

// New creates a new NotionModule instance
func New() *NotionModule {
	return &NotionModule{}
}

// Module descriptions in multiple languages
var moduleDescriptions = modules.LocalizedText{
	"en-US": "Notion API - Page, Database, Block, Comment, and User operations",
	"ja-JP": "Notion API - ページ、データベース、ブロック、コメント、ユーザー操作",
}

// Name returns the module name
func (m *NotionModule) Name() string {
	return "notion"
}

// Descriptions returns the module descriptions in all languages
func (m *NotionModule) Descriptions() modules.LocalizedText {
	return moduleDescriptions
}

// Description returns the module description for a specific language
func (m *NotionModule) Description(lang string) string {
	return modules.GetLocalizedText(moduleDescriptions, lang)
}

// APIVersion returns the Notion API version
func (m *NotionModule) APIVersion() string {
	return notionVersion
}

// Tools returns all available tools
func (m *NotionModule) Tools() []modules.Tool {
	return toolDefinitions()
}

// ExecuteTool executes a tool by name and returns JSON response
func (m *NotionModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// ToCompact converts JSON result to compact format (MD or CSV)
// Implements modules.CompactConverter interface
func (m *NotionModule) ToCompact(toolName string, jsonResult string) string {
	return formatCompact(toolName, jsonResult)
}

// Resources returns all available resources
func (m *NotionModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI
func (m *NotionModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

// =============================================================================
// Client / Auth
// =============================================================================

// getCredentials retrieves credentials from Vault via RPC for the given user
// and refreshes the token if needed (for OAuth2)
func getCredentials(ctx context.Context) *store.Credentials {
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
				return credentials
			}
			log.Printf("[notion] Token refreshed successfully")
			return refreshed
		}
	}

	return credentials
}

func needsRefresh(creds *store.Credentials) bool {
	if creds.ExpiresAt == 0 {
		return false
	}
	now := time.Now().Unix()
	return now >= (int64(creds.ExpiresAt) - tokenRefreshBuffer)
}

func refreshToken(ctx context.Context, userID string, creds *store.Credentials) (*store.Credentials, error) {
	oauthApp, err := store.GetTokenStore().GetOAuthAppCredentials(ctx, "notion")
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth app credentials: %w", err)
	}

	basicAuth := base64.StdEncoding.EncodeToString([]byte(oauthApp.ClientID + ":" + oauthApp.ClientSecret))

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

	var expiresAt store.FlexibleTime
	if tokenResp.ExpiresIn > 0 {
		expiresAt = store.FlexibleTime(time.Now().Unix() + tokenResp.ExpiresIn)
	}

	newCreds := &store.Credentials{
		AuthType:     store.AuthTypeOAuth2,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    expiresAt,
		Metadata:     creds.Metadata,
	}

	if newCreds.RefreshToken == "" {
		newCreds.RefreshToken = creds.RefreshToken
	}

	err = store.GetTokenStore().UpdateModuleToken(ctx, userID, "notion", newCreds)
	if err != nil {
		log.Printf("[notion] Failed to save refreshed token: %v", err)
	}

	return newCreds, nil
}

// newOgenClient creates a new ogen-generated Notion API client
func newOgenClient(ctx context.Context) (*gen.Client, error) {
	creds := getCredentials(ctx)
	if creds == nil {
		return nil, fmt.Errorf("no credentials available")
	}
	return notionapi.NewClient(creds.AccessToken, notionVersion)
}
