package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"mcpist/server/internal/db"
)

var (
	defaultBroker *TokenBroker
	brokerOnce    sync.Once
)

// InitTokenBroker initializes the singleton token broker with the given DB.
// Must be called once at startup before GetTokenBroker().
func InitTokenBroker(database *gorm.DB) {
	brokerOnce.Do(func() {
		defaultBroker = NewTokenBroker(database)
	})
}

// GetTokenBroker returns the singleton token broker instance
func GetTokenBroker() *TokenBroker {
	if defaultBroker == nil {
		log.Fatal("TokenBroker not initialized. Call InitTokenBroker() first.")
	}
	return defaultBroker
}

// TokenBroker manages token retrieval from DB via GORM
// and transparently refreshes OAuth2 tokens when needed.
type TokenBroker struct {
	db     *gorm.DB
	client *http.Client
}

// AuthType constants for API request authentication methods
const (
	AuthTypeOAuth2       = "oauth2"
	AuthTypeOAuth1       = "oauth1"
	AuthTypeAPIKey       = "api_key"
	AuthTypeBasic        = "basic"
	AuthTypeCustomHeader = "custom_header"
)

// tokenRefreshBuffer is the number of seconds before expiry to trigger refresh
const tokenRefreshBuffer = 5 * 60

// FlexibleTime handles both Unix timestamp (int64) and ISO string formats
type FlexibleTime int64

func (ft *FlexibleTime) UnmarshalJSON(data []byte) error {
	var num int64
	if err := json.Unmarshal(data, &num); err == nil {
		*ft = FlexibleTime(num)
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		if str == "" {
			*ft = 0
			return nil
		}
		t, err := time.Parse(time.RFC3339, str)
		if err != nil {
			t, err = time.Parse("2006-01-02T15:04:05.999Z", str)
			if err != nil {
				return fmt.Errorf("failed to parse time string: %w", err)
			}
		}
		*ft = FlexibleTime(t.Unix())
		return nil
	}

	return fmt.Errorf("expires_at must be number or string")
}

// Credentials represents the credentials from the database
type Credentials struct {
	AuthType string `json:"auth_type,omitempty"`

	// OAuth 2.0
	AccessToken  string       `json:"access_token,omitempty"`
	RefreshToken string       `json:"refresh_token,omitempty"`
	ExpiresAt    FlexibleTime `json:"expires_at,omitempty"`

	// OAuth 1.0a
	ConsumerKey       string `json:"consumer_key,omitempty"`
	ConsumerSecret    string `json:"consumer_secret,omitempty"`
	AccessTokenSecret string `json:"access_token_secret,omitempty"`

	// API Key
	APIKey string `json:"api_key,omitempty"`

	// Basic authentication
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`

	// Custom header
	Token      string `json:"token,omitempty"`
	HeaderName string `json:"header_name,omitempty"`

	// Additional metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// CredentialResult represents the result of credential lookup
type CredentialResult struct {
	Found       bool                   `json:"found"`
	UserID      string                 `json:"user_id"`
	Service     string                 `json:"service"`
	AuthType    string                 `json:"auth_type"`
	Credentials *Credentials           `json:"credentials,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// NewTokenBroker creates a new token broker
func NewTokenBroker(database *gorm.DB) *TokenBroker {
	log.Printf("[broker] TokenBroker initialized with GORM")
	return &TokenBroker{
		db: database,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// =============================================================================
// OAuth Refresh Configuration
// =============================================================================

// OAuthRefreshConfig defines how to refresh tokens for each OAuth provider.
type OAuthRefreshConfig struct {
	Provider            string
	TokenURL            string
	AuthMethod          string // "form" or "basic"
	ContentType         string // "urlencoded" or "json"
	ExtraParams         map[string]string
	RotatesRefreshToken bool
}

var oauthRefreshConfigs = map[string]OAuthRefreshConfig{
	"google_calendar":    {Provider: "google", TokenURL: "https://oauth2.googleapis.com/token", AuthMethod: "form", ContentType: "urlencoded"},
	"google_tasks":       {Provider: "google", TokenURL: "https://oauth2.googleapis.com/token", AuthMethod: "form", ContentType: "urlencoded"},
	"google_drive":       {Provider: "google", TokenURL: "https://oauth2.googleapis.com/token", AuthMethod: "form", ContentType: "urlencoded"},
	"google_docs":        {Provider: "google", TokenURL: "https://oauth2.googleapis.com/token", AuthMethod: "form", ContentType: "urlencoded"},
	"google_sheets":      {Provider: "google", TokenURL: "https://oauth2.googleapis.com/token", AuthMethod: "form", ContentType: "urlencoded"},
	"google_apps_script": {Provider: "google", TokenURL: "https://oauth2.googleapis.com/token", AuthMethod: "form", ContentType: "urlencoded"},
	"asana":              {Provider: "asana", TokenURL: "https://app.asana.com/-/oauth_token", AuthMethod: "form", ContentType: "urlencoded", RotatesRefreshToken: true},
	"dropbox":            {Provider: "dropbox", TokenURL: "https://api.dropboxapi.com/oauth2/token", AuthMethod: "form", ContentType: "urlencoded"},
	"microsoft_todo":     {Provider: "microsoft", TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token", AuthMethod: "form", ContentType: "urlencoded", ExtraParams: map[string]string{"scope": "offline_access Tasks.ReadWrite"}, RotatesRefreshToken: true},
	"notion":             {Provider: "notion", TokenURL: "https://api.notion.com/v1/oauth/token", AuthMethod: "basic", ContentType: "json", RotatesRefreshToken: true},
	"airtable":           {Provider: "airtable", TokenURL: "https://airtable.com/oauth2/v1/token", AuthMethod: "basic", ContentType: "urlencoded", RotatesRefreshToken: true},
	"jira":               {Provider: "atlassian", TokenURL: "https://auth.atlassian.com/oauth/token", AuthMethod: "form", ContentType: "urlencoded", RotatesRefreshToken: true},
	"confluence":         {Provider: "atlassian", TokenURL: "https://auth.atlassian.com/oauth/token", AuthMethod: "form", ContentType: "urlencoded", RotatesRefreshToken: true},
}

// =============================================================================
// GetModuleToken — transparently refreshes OAuth2 tokens
// =============================================================================

// GetModuleToken retrieves the module's credentials from DB.
// For OAuth2 modules with refresh tokens, it transparently refreshes expired tokens.
func (b *TokenBroker) GetModuleToken(ctx context.Context, userID, module string) (*Credentials, error) {
	creds, err := b.fetchCredentials(ctx, userID, module)
	if err != nil {
		return nil, err
	}

	// Skip refresh for non-OAuth2 or tokens without refresh_token
	if creds.AuthType != AuthTypeOAuth2 || creds.RefreshToken == "" {
		return creds, nil
	}
	if !needsRefresh(creds) {
		return creds, nil
	}

	// Look up refresh config for this module
	config, ok := oauthRefreshConfigs[module]
	if !ok {
		return creds, nil
	}

	log.Printf("[broker] Token expired or expiring soon for %s, refreshing...", module)
	refreshed, err := b.refreshOAuthToken(ctx, userID, module, creds, config)
	if err != nil {
		log.Printf("[broker] Token refresh failed for %s: %v", module, err)
		return creds, nil // Fall back to existing token
	}
	log.Printf("[broker] Token refreshed successfully for %s", module)
	return refreshed, nil
}

// fetchCredentials retrieves raw credentials from DB (no refresh)
func (b *TokenBroker) fetchCredentials(ctx context.Context, userID, module string) (*Credentials, error) {
	cred, err := db.GetCredential(b.db, userID, module)
	if err != nil {
		return nil, fmt.Errorf("no credential configured for user: %s, module: %s: %w", userID, module, err)
	}

	var credentials Credentials
	if err := json.Unmarshal([]byte(cred.Credentials), &credentials); err != nil {
		return nil, fmt.Errorf("failed to parse credentials for module %s: %w", module, err)
	}

	return &credentials, nil
}

// needsRefresh checks if the token should be refreshed
func needsRefresh(creds *Credentials) bool {
	if creds.ExpiresAt == 0 {
		return false
	}
	return time.Now().Unix() >= (int64(creds.ExpiresAt) - tokenRefreshBuffer)
}

// refreshOAuthToken performs the OAuth2 token refresh using the provider-specific config
func (b *TokenBroker) refreshOAuthToken(ctx context.Context, userID, module string, creds *Credentials, cfg OAuthRefreshConfig) (*Credentials, error) {
	oauthApp, err := b.GetOAuthAppCredentials(ctx, cfg.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth app credentials: %w", err)
	}

	var req *http.Request
	switch cfg.ContentType {
	case "json":
		body, _ := json.Marshal(map[string]string{
			"grant_type":    "refresh_token",
			"refresh_token": creds.RefreshToken,
		})
		req, err = http.NewRequestWithContext(ctx, "POST", cfg.TokenURL, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create refresh request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
	default:
		data := url.Values{}
		data.Set("grant_type", "refresh_token")
		data.Set("refresh_token", creds.RefreshToken)
		if cfg.AuthMethod == "form" {
			data.Set("client_id", oauthApp.ClientID)
			data.Set("client_secret", oauthApp.ClientSecret)
		}
		for k, v := range cfg.ExtraParams {
			data.Set(k, v)
		}
		req, err = http.NewRequestWithContext(ctx, "POST", cfg.TokenURL, strings.NewReader(data.Encode()))
		if err != nil {
			return nil, fmt.Errorf("failed to create refresh request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	if cfg.AuthMethod == "basic" {
		req.SetBasicAuth(oauthApp.ClientID, oauthApp.ClientSecret)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed: status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	newCreds := &Credentials{
		AuthType:     AuthTypeOAuth2,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: creds.RefreshToken,
		Metadata:     creds.Metadata,
	}
	if tokenResp.ExpiresIn > 0 {
		newCreds.ExpiresAt = FlexibleTime(time.Now().Unix() + tokenResp.ExpiresIn)
	}
	if cfg.RotatesRefreshToken && tokenResp.RefreshToken != "" {
		newCreds.RefreshToken = tokenResp.RefreshToken
	}

	if err := b.UpdateModuleToken(ctx, userID, module, newCreds); err != nil {
		log.Printf("[broker] Failed to save refreshed token for %s: %v", module, err)
	}
	return newCreds, nil
}

// =============================================================================
// OAuth App Credentials & Token Update
// =============================================================================

// OAuthAppCredentials represents the OAuth app configuration
type OAuthAppCredentials struct {
	Provider     string `json:"provider"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURI  string `json:"redirect_uri"`
}

// GetOAuthAppCredentials retrieves OAuth app credentials (client_id, client_secret) for a provider
func (b *TokenBroker) GetOAuthAppCredentials(ctx context.Context, provider string) (*OAuthAppCredentials, error) {
	app, err := db.GetOAuthAppCredentials(b.db, provider)
	if err != nil {
		return nil, err
	}

	return &OAuthAppCredentials{
		Provider:     app.Provider,
		ClientID:     app.ClientID,
		ClientSecret: app.ClientSecret,
		RedirectURI:  app.RedirectURI,
	}, nil
}

// UpdateModuleToken saves refreshed credentials to DB
func (b *TokenBroker) UpdateModuleToken(ctx context.Context, userID, module string, credentials *Credentials) error {
	credJSON, err := json.Marshal(credentials)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	return db.UpsertCredential(b.db, userID, module, string(credJSON))
}
