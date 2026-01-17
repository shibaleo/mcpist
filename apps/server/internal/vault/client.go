package vault

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
)

var (
	baseURL string
	anonKey string
	once    sync.Once
)

func initConfig() {
	once.Do(func() {
		baseURL = os.Getenv("VAULT_URL")
		if baseURL == "" {
			baseURL = "http://localhost:8089"
		}
		anonKey = os.Getenv("SUPABASE_ANON_KEY")
	})
}

func getBaseURL() string {
	initConfig()
	return baseURL
}

func getAnonKey() string {
	initConfig()
	return anonKey
}

// TokenRequest is the request body for token retrieval
type TokenRequest struct {
	UserID  string `json:"user_id"`
	Service string `json:"service"`
}

// TokenResult contains both token types from vault
type TokenResult struct {
	LongTermToken string `json:"long_term_token,omitempty"`
	OAuthToken    string `json:"oauth_token,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// GetTokens retrieves tokens for the specified user and service from Token Vault
// Returns both long-term and OAuth tokens if available
func GetTokens(userID, service string) (*TokenResult, error) {
	url := fmt.Sprintf("%s/token-vault", getBaseURL())

	reqBody := TokenRequest{
		UserID:  userID,
		Service: service,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if key := getAnonKey(); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vault request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("vault error: %s", errResp.Error)
		}
		return nil, fmt.Errorf("vault error (status %d): %s", resp.StatusCode, string(body))
	}

	var result TokenResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}
