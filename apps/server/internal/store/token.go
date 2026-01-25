package store

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	defaultTokenStore *TokenStore
	tokenOnce         sync.Once
)

// GetTokenStore returns the singleton token store instance
func GetTokenStore() *TokenStore {
	tokenOnce.Do(func() {
		defaultTokenStore = NewTokenStore()
	})
	return defaultTokenStore
}

// TokenStore manages token retrieval from Supabase Vault via RPC
type TokenStore struct {
	supabaseURL string
	serviceKey  string
	client      *http.Client
}

// AuthType constants for API request authentication methods
const (
	AuthTypeOAuth2       = "oauth2"        // OAuth 2.0 (with refresh token support)
	AuthTypeOAuth1       = "oauth1"        // OAuth 1.0a signature
	AuthTypeAPIKey       = "api_key"       // API Key (Bearer token, no refresh)
	AuthTypeBasic        = "basic"         // Basic authentication (username:password)
	AuthTypeCustomHeader = "custom_header" // Custom header
)

// Credentials represents the credentials from Vault
// Supports multiple authentication types as defined in dtl-itr-MOD-TVL.md
type Credentials struct {
	// Common fields
	AuthType string `json:"auth_type"` // oauth2, oauth1, api_key, basic, custom_header

	// OAuth 2.0 (with refresh support)
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresAt    int64  `json:"expires_at,omitempty"` // Unix timestamp

	// OAuth 1.0a
	ConsumerKey       string `json:"consumer_key,omitempty"`
	ConsumerSecret    string `json:"consumer_secret,omitempty"`
	AccessTokenSecret string `json:"access_token_secret,omitempty"`

	// API Key (also uses AccessToken field for the key)
	// Uses AccessToken field above

	// Basic authentication
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`

	// Custom header
	Token      string `json:"token,omitempty"`
	HeaderName string `json:"header_name,omitempty"`

	// Additional metadata (e.g., domain for Atlassian)
	Metadata map[string]string `json:"metadata,omitempty"`
}

// TokenResult represents the result of get_module_token RPC
// Matches the TVL response format from dtl-itr-MOD-TVL.md
type TokenResult struct {
	UserID      string       `json:"user_id"`
	Service     string       `json:"service"`
	AuthType    string       `json:"auth_type"`
	Credentials *Credentials `json:"credentials,omitempty"`
	Error       string       `json:"error,omitempty"`
}

// NewTokenStore creates a new token store
func NewTokenStore() *TokenStore {
	return &TokenStore{
		supabaseURL: os.Getenv("SUPABASE_URL"),
		serviceKey:  os.Getenv("SUPABASE_SECRET_KEY"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetModuleToken retrieves the module's credentials from Vault via RPC
// Returns credentials with auth_type indicating how to use them for API requests
func (s *TokenStore) GetModuleToken(ctx context.Context, userID, module string) (*Credentials, error) {
	if s.serviceKey == "" {
		// Return mock token for development
		return &Credentials{
			AuthType:    AuthTypeAPIKey,
			AccessToken: "dev_mock_token",
		}, nil
	}

	reqBody := fmt.Sprintf(`{"p_user_id": "%s", "p_module": "%s"}`, userID, module)
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		s.supabaseURL+"/rest/v1/rpc/get_module_token",
		strings.NewReader(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", s.serviceKey)
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call get_module_token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get_module_token failed: status %d", resp.StatusCode)
	}

	var result TokenResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("token error: %s", result.Error)
	}

	if result.Credentials == nil {
		return nil, fmt.Errorf("no token configured for user: %s, service: %s", userID, module)
	}

	// Copy auth_type from result to credentials if not set
	if result.Credentials.AuthType == "" {
		result.Credentials.AuthType = result.AuthType
	}

	return result.Credentials, nil
}
