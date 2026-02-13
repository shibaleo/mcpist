package broker

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// ModuleStore manages module registration in Supabase
type ModuleStore struct {
	supabaseURL string
	serviceKey  string
	client      *http.Client
}

// NewModuleStore creates a new module store
func NewModuleStore() *ModuleStore {
	return &ModuleStore{
		supabaseURL: os.Getenv("SUPABASE_URL"),
		serviceKey:  os.Getenv("SUPABASE_SECRET_KEY"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SyncModulesResult represents the response from sync_modules RPC
type SyncModulesResult struct {
	Success  bool `json:"success"`
	Inserted int  `json:"inserted"`
	Total    int  `json:"total"`
}

// SyncModules ensures all provided modules exist in the database
// Uses RPC to access mcpist schema
func (s *ModuleStore) SyncModules(moduleNames []string) error {
	if s.serviceKey == "" {
		log.Println("[ModuleStore] No service key configured, skipping module sync")
		return nil
	}

	if len(moduleNames) == 0 {
		return nil
	}

	// Build RPC payload: {"p_modules": ["notion", "github", ...]}
	payload := fmt.Sprintf(`{"p_modules": ["%s"]}`, strings.Join(moduleNames, `","`))

	req, err := http.NewRequest(
		"POST",
		s.supabaseURL+"/rest/v1/rpc/sync_modules",
		strings.NewReader(payload),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", s.serviceKey)
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to sync modules: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("module sync failed: status %d", resp.StatusCode)
	}

	var result SyncModulesResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Inserted > 0 {
		log.Printf("[ModuleStore] Synced modules: %d new, %d total", result.Inserted, result.Total)
	} else {
		log.Printf("[ModuleStore] All %d modules already registered", result.Total)
	}

	return nil
}
