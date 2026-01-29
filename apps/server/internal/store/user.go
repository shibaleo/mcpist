package store

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// UserStore manages user context queries to Supabase
type UserStore struct {
	supabaseURL string
	serviceKey  string
	client      *http.Client
	cache       *userCache
}

// UserContext represents the user's context from get_user_context RPC
type UserContext struct {
	AccountStatus      string              `json:"account_status"`
	FreeCredits        int                 `json:"free_credits"`
	PaidCredits        int                 `json:"paid_credits"`
	EnabledModules     []string            `json:"enabled_modules"`
	EnabledTools       map[string][]string `json:"enabled_tools"`       // module -> []tool_id (whitelist)
	Language           string              `json:"language"`            // BCP47 language code (e.g., "en-US", "ja-JP")
	ModuleDescriptions ModuleDescriptions  `json:"module_descriptions"` // module -> custom description
}

// TotalCredits returns the sum of free and paid credits
func (uc *UserContext) TotalCredits() int {
	return uc.FreeCredits + uc.PaidCredits
}

// IsModuleEnabled checks if a module is enabled for the user.
func (uc *UserContext) IsModuleEnabled(module string) bool {
	for _, m := range uc.EnabledModules {
		if m == module {
			return true
		}
	}
	return false
}

// IsToolEnabled checks if a tool is enabled for the user
// Uses whitelist approach: tool must be in EnabledTools to be enabled
func (uc *UserContext) IsToolEnabled(module, tool string) bool {
	enabledTools, ok := uc.EnabledTools[module]
	if !ok {
		return false // Module has no enabled tools
	}
	for _, t := range enabledTools {
		if t == tool {
			return true
		}
	}
	return false
}

// ConsumeResult represents the result of consume_credit RPC
type ConsumeResult struct {
	Success          bool   `json:"success"`
	FreeCredits      int    `json:"free_credits"`
	PaidCredits      int    `json:"paid_credits"`
	AlreadyProcessed bool   `json:"already_processed,omitempty"`
	Error            string `json:"error,omitempty"`
}

// userCache stores user context with TTL
type userCache struct {
	mu    sync.RWMutex
	items map[string]*userCacheItem
	ttl   time.Duration
}

type userCacheItem struct {
	context   *UserContext
	expiresAt time.Time
}

// NewUserStore creates a new user store
func NewUserStore() *UserStore {
	return &UserStore{
		supabaseURL: os.Getenv("SUPABASE_URL"),
		serviceKey:  os.Getenv("SUPABASE_SECRET_KEY"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: &userCache{
			items: make(map[string]*userCacheItem),
			ttl:   30 * time.Second, // Cache for 30 seconds
		},
	}
}

// GetUserContext retrieves the user's context (account status, credits, modules, tools)
func (s *UserStore) GetUserContext(userID string) (*UserContext, error) {
	// Check cache first
	if cached := s.cache.get(userID); cached != nil {
		return cached, nil
	}

	// Query Supabase RPC
	ctx, err := s.fetchUserContext(userID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	s.cache.set(userID, ctx)

	return ctx, nil
}

// fetchUserContext calls the Supabase RPC function
func (s *UserStore) fetchUserContext(userID string) (*UserContext, error) {
	if s.serviceKey == "" {
		// Return default context for development without service key
		// All tools enabled for all modules (dev mode)
		return &UserContext{
			AccountStatus:      "active",
			FreeCredits:        100,
			PaidCredits:        0,
			EnabledModules:     []string{"notion", "github", "jira", "confluence", "supabase", "airtable", "google_calendar", "microsoft_todo", "rag"},
			EnabledTools:       map[string][]string{}, // Empty means check disabled (dev mode fallback)
			Language:           "en-US",
			ModuleDescriptions: ModuleDescriptions{},
		}, nil
	}

	reqBody := fmt.Sprintf(`{"p_user_id": "%s"}`, userID)
	req, err := http.NewRequest(
		"POST",
		s.supabaseURL+"/rest/v1/rpc/get_user_context",
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
		return nil, fmt.Errorf("failed to call get_user_context: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get_user_context failed: status %d", resp.StatusCode)
	}

	// RPC returns a table, so we get an array
	var results []struct {
		AccountStatus      string          `json:"account_status"`
		FreeCredits        int             `json:"free_credits"`
		PaidCredits        int             `json:"paid_credits"`
		EnabledModules     []string        `json:"enabled_modules"`
		EnabledTools       json.RawMessage `json:"enabled_tools"`
		Language           string          `json:"language"`
		ModuleDescriptions json.RawMessage `json:"module_descriptions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(results) == 0 {
		// User not found - return error
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	r := results[0]

	// Parse enabled_tools JSONB (whitelist)
	enabledTools := make(map[string][]string)
	if len(r.EnabledTools) > 0 && string(r.EnabledTools) != "{}" {
		if err := json.Unmarshal(r.EnabledTools, &enabledTools); err != nil {
			// Log but don't fail - just use empty map
			enabledTools = map[string][]string{}
		}
	}

	// Parse module_descriptions JSONB
	moduleDescriptions := make(ModuleDescriptions)
	if len(r.ModuleDescriptions) > 0 && string(r.ModuleDescriptions) != "{}" {
		if err := json.Unmarshal(r.ModuleDescriptions, &moduleDescriptions); err != nil {
			// Log but don't fail - just use empty map
			moduleDescriptions = ModuleDescriptions{}
		}
	}

	// Default to en-US if language is empty
	language := r.Language
	if language == "" {
		language = "en-US"
	}

	return &UserContext{
		AccountStatus:      r.AccountStatus,
		FreeCredits:        r.FreeCredits,
		PaidCredits:        r.PaidCredits,
		EnabledModules:     r.EnabledModules,
		EnabledTools:       enabledTools,
		Language:           language,
		ModuleDescriptions: moduleDescriptions,
	}, nil
}

// ConsumeCredit consumes credits for a tool execution (idempotent)
func (s *UserStore) ConsumeCredit(userID, module, tool string, amount int, requestID string, taskID *string) (*ConsumeResult, error) {
	if s.serviceKey == "" {
		// Skip in development
		return &ConsumeResult{
			Success:     true,
			FreeCredits: 100,
			PaidCredits: 0,
		}, nil
	}

	// Build request body
	var reqBody string
	if taskID != nil {
		reqBody = fmt.Sprintf(
			`{"p_user_id": "%s", "p_module": "%s", "p_tool": "%s", "p_amount": %d, "p_request_id": "%s", "p_task_id": "%s"}`,
			userID, module, tool, amount, requestID, *taskID,
		)
	} else {
		reqBody = fmt.Sprintf(
			`{"p_user_id": "%s", "p_module": "%s", "p_tool": "%s", "p_amount": %d, "p_request_id": "%s"}`,
			userID, module, tool, amount, requestID,
		)
	}

	req, err := http.NewRequest(
		"POST",
		s.supabaseURL+"/rest/v1/rpc/consume_credit",
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
		return nil, fmt.Errorf("failed to call consume_credit: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("consume_credit failed: status %d", resp.StatusCode)
	}

	var result ConsumeResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Invalidate cache to reflect new balance
	s.cache.delete(userID)

	return &result, nil
}

// InvalidateCache removes a user's cached context
func (s *UserStore) InvalidateCache(userID string) {
	s.cache.delete(userID)
}

// ModuleDescriptions maps module_name -> custom_description
type ModuleDescriptions map[string]string

// Cache methods

func (c *userCache) get(userID string) *UserContext {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[userID]
	if !ok {
		return nil
	}

	if time.Now().After(item.expiresAt) {
		return nil
	}

	return item.context
}

func (c *userCache) set(userID string, ctx *UserContext) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[userID] = &userCacheItem{
		context:   ctx,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *userCache) delete(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, userID)
}
