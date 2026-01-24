package token

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
	defaultStore *Store
	once         sync.Once
)

// GetStore returns the singleton token store instance
func GetStore() *Store {
	once.Do(func() {
		defaultStore = NewStore()
	})
	return defaultStore
}

// Store manages token retrieval from Supabase Vault via RPC
type Store struct {
	supabaseURL string
	serviceKey  string
	client      *http.Client
}

// Credentials represents the credentials from Vault
type Credentials struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	Scope        string `json:"scope,omitempty"`
	ExpiresAt    string `json:"_expires_at,omitempty"`
	AuthType     string `json:"_auth_type,omitempty"`
}

// ModuleTokenResult represents the result of get_module_token RPC
type ModuleTokenResult struct {
	Found       bool        `json:"found"`
	Credentials *Credentials `json:"credentials,omitempty"`
	Error       string      `json:"error,omitempty"`
}

// NewStore creates a new token store
func NewStore() *Store {
	return &Store{
		supabaseURL: os.Getenv("SUPABASE_URL"),
		serviceKey:  os.Getenv("SUPABASE_SECRET_KEY"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetModuleToken retrieves the module's credentials from Vault via RPC
func (s *Store) GetModuleToken(ctx context.Context, userID, module string) (*Credentials, error) {
	if s.serviceKey == "" {
		// Return mock token for development
		return &Credentials{
			AccessToken: "dev_mock_token",
			AuthType:    "api_key",
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

	var result ModuleTokenResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Found {
		if result.Error != "" {
			return nil, fmt.Errorf("token not found: %s", result.Error)
		}
		return nil, fmt.Errorf("token not found for module: %s", module)
	}

	return result.Credentials, nil
}
