package store

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	UserID      string            `json:"user_id"`
	Service     string            `json:"service"`
	AuthType    string            `json:"auth_type"`
	Credentials *Credentials      `json:"credentials,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"` // e.g., domain for Atlassian
	Error       string            `json:"error,omitempty"`
}

// NewTokenStore creates a new token store
func NewTokenStore() *TokenStore {
	// Use SUPABASE_PUBLISHABLE_KEY (anon key) for RPC calls
	// The get_module_token RPC is SECURITY DEFINER, so anon key works
	serviceKey := os.Getenv("SUPABASE_PUBLISHABLE_KEY")
	if serviceKey == "" {
		// Fallback to SUPABASE_SECRET_KEY for backwards compatibility
		serviceKey = os.Getenv("SUPABASE_SECRET_KEY")
	}
	supabaseURL := os.Getenv("SUPABASE_URL")
	log.Printf("[TokenStore] Initialized - URL: %s, Key: %s...", supabaseURL, serviceKey[:min(20, len(serviceKey))])
	return &TokenStore{
		supabaseURL: supabaseURL,
		serviceKey:  serviceKey,
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

	// Copy metadata from result to credentials (for Basic auth like Jira/Confluence)
	if result.Metadata != nil && result.Credentials.Metadata == nil {
		result.Credentials.Metadata = result.Metadata
	}

	return result.Credentials, nil
}

// OAuthAppCredentials represents the OAuth app configuration from Vault
type OAuthAppCredentials struct {
	Provider     string `json:"provider"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURI  string `json:"redirect_uri"`
	Error        string `json:"error,omitempty"`
	Message      string `json:"message,omitempty"`
}

// GetOAuthAppCredentials retrieves OAuth app credentials (client_id, client_secret) for a provider
// Used for token refresh operations
func (s *TokenStore) GetOAuthAppCredentials(ctx context.Context, provider string) (*OAuthAppCredentials, error) {
	if s.serviceKey == "" {
		return nil, fmt.Errorf("OAuth app credentials not available in development mode")
	}

	// Need service_role key for get_oauth_app_credentials
	secretKey := os.Getenv("SUPABASE_SECRET_KEY")
	if secretKey == "" {
		return nil, fmt.Errorf("SUPABASE_SECRET_KEY required for OAuth app credentials")
	}

	reqBody := fmt.Sprintf(`{"p_provider": "%s"}`, provider)
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		s.supabaseURL+"/rest/v1/rpc/get_oauth_app_credentials",
		strings.NewReader(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", secretKey)
	req.Header.Set("Authorization", "Bearer "+secretKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call get_oauth_app_credentials: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get_oauth_app_credentials failed: status %d", resp.StatusCode)
	}

	var result OAuthAppCredentials
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("OAuth app error: %s - %s", result.Error, result.Message)
	}

	return &result, nil
}

// UpdateModuleToken saves refreshed credentials to Vault via RPC
// Called after OAuth2 token refresh to persist new access_token/expires_at
func (s *TokenStore) UpdateModuleToken(ctx context.Context, userID, module string, credentials *Credentials) error {
	if s.serviceKey == "" {
		// Skip in development
		return nil
	}

	// Need service_role key for update_module_token (writes to Vault)
	secretKey := os.Getenv("SUPABASE_SECRET_KEY")
	if secretKey == "" {
		return fmt.Errorf("SUPABASE_SECRET_KEY required for token update")
	}

	credJSON, err := json.Marshal(credentials)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	reqBody := fmt.Sprintf(
		`{"p_user_id": "%s", "p_module": "%s", "p_credentials": %s}`,
		userID, module, string(credJSON),
	)

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		s.supabaseURL+"/rest/v1/rpc/update_module_token",
		strings.NewReader(reqBody),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", secretKey)
	req.Header.Set("Authorization", "Bearer "+secretKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call update_module_token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update_module_token failed: status %d", resp.StatusCode)
	}

	var result struct {
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("update_module_token failed: %s", result.Error)
	}

	return nil
}
