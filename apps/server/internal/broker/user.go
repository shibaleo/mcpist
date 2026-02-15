package broker

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// UserBroker manages user context queries via PostgREST RPC
type UserBroker struct {
	postgrestURL string
	apiKey       string
	client       *http.Client
	cache        *userCache
}

// UserContext represents the user's context from get_user_context RPC
type UserContext struct {
	AccountStatus      string              `json:"account_status"`
	PlanID             string              `json:"plan_id"`
	DailyUsed          int                 `json:"daily_used"`
	DailyLimit         int                 `json:"daily_limit"`
	EnabledModules     []string            `json:"enabled_modules"`
	EnabledTools       map[string][]string `json:"enabled_tools"`       // module -> []tool_id (whitelist)
	Language           string              `json:"language"`            // BCP47 language code (e.g., "en-US", "ja-JP")
	ModuleDescriptions ModuleDescriptions  `json:"module_descriptions"` // module -> custom description
}

// WithinDailyLimit checks if the user can execute the given number of tools
func (uc *UserContext) WithinDailyLimit(count int) bool {
	return uc.DailyUsed+count <= uc.DailyLimit
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

// NewUserBroker creates a new user store
func NewUserBroker() *UserBroker {
	return &UserBroker{
		postgrestURL: os.Getenv("POSTGREST_URL"),
		apiKey:       os.Getenv("POSTGREST_API_KEY"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: &userCache{
			items: make(map[string]*userCacheItem),
			ttl:   30 * time.Second, // Cache for 30 seconds
		},
	}
}

// HealthCheck verifies connectivity to the PostgREST endpoint.
func (s *UserBroker) HealthCheck() error {
	req, err := http.NewRequest("HEAD", s.postgrestURL+"/", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}
	req.Header.Set("apikey", s.apiKey)
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := doWithRetry(s.client, req, defaultRetry)
	if err != nil {
		return fmt.Errorf("postgrest unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("postgrest returned status %d", resp.StatusCode)
	}
	return nil
}

// GetUserContext retrieves the user's context (account status, credits, modules, tools).
// On fetch failure, returns stale cached data if available (graceful degradation).
func (s *UserBroker) GetUserContext(userID string) (*UserContext, error) {
	// Check cache first (non-expired)
	if cached := s.cache.get(userID); cached != nil {
		return cached, nil
	}

	// Query PostgREST RPC
	ctx, err := s.fetchUserContext(userID)
	if err != nil {
		// Fall back to stale cache on transient failure
		if stale := s.cache.getStale(userID); stale != nil {
			log.Printf("GetUserContext: using stale cache for %s due to: %v", userID, err)
			s.cache.set(userID, stale) // extend TTL
			return stale, nil
		}
		return nil, err
	}

	// Cache the result
	s.cache.set(userID, ctx)

	return ctx, nil
}

// fetchUserContext calls the PostgREST RPC function
func (s *UserBroker) fetchUserContext(userID string) (*UserContext, error) {
	if s.apiKey == "" {
		// Return default context for development without service key
		// All tools enabled for all modules (dev mode)
		return &UserContext{
			AccountStatus:      "active",
			PlanID:             "free",
			DailyUsed:          0,
			DailyLimit:         100,
			EnabledModules:     []string{"notion", "github", "jira", "confluence", "supabase", "airtable", "google_calendar", "microsoft_todo", "google_tasks"},
			EnabledTools:       map[string][]string{}, // Empty means check disabled (dev mode fallback)
			Language:           "en-US",
			ModuleDescriptions: ModuleDescriptions{},
		}, nil
	}

	reqBody, _ := json.Marshal(map[string]string{"p_user_id": userID})
	req, err := http.NewRequest(
		"POST",
		s.postgrestURL+"/rpc/get_user_context",
		strings.NewReader(string(reqBody)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", s.apiKey)
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := doWithRetry(s.client, req, defaultRetry)
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
		PlanID             string          `json:"plan_id"`
		DailyUsed          int             `json:"daily_used"`
		DailyLimit         int             `json:"daily_limit"`
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
		PlanID:             r.PlanID,
		DailyUsed:          r.DailyUsed,
		DailyLimit:         r.DailyLimit,
		EnabledModules:     r.EnabledModules,
		EnabledTools:       enabledTools,
		Language:           language,
		ModuleDescriptions: moduleDescriptions,
	}, nil
}

// ToolDetail represents a single tool execution in the details array
type ToolDetail struct {
	TaskID string `json:"task_id,omitempty"`
	Module string `json:"module"`
	Tool   string `json:"tool"`
}

// RecordUsage records tool usage asynchronously (fire-and-forget).
// metaTool: "run" or "batch"
// details: array of ToolDetail for tracking individual tool executions
// This is non-blocking: failures are logged but do not affect the caller.
func (s *UserBroker) RecordUsage(userID, metaTool, requestID string, details []ToolDetail) {
	if s.apiKey == "" {
		return // Skip in development
	}

	go func() {
		reqBody, err := json.Marshal(map[string]interface{}{
			"p_user_id":    userID,
			"p_meta_tool":  metaTool,
			"p_request_id": requestID,
			"p_details":    details,
		})
		if err != nil {
			log.Printf("RecordUsage: failed to marshal request: %v", err)
			return
		}

		req, err := http.NewRequest(
			"POST",
			s.postgrestURL+"/rpc/record_usage",
			strings.NewReader(string(reqBody)),
		)
		if err != nil {
			log.Printf("RecordUsage: failed to create request: %v", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("apikey", s.apiKey)
		req.Header.Set("Authorization", "Bearer "+s.apiKey)

		resp, err := s.client.Do(req)
		if err != nil {
			log.Printf("RecordUsage: failed to call record_usage: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			log.Printf("RecordUsage: record_usage returned status %d", resp.StatusCode)
		}
	}()
}

// InvalidateCache removes a user's cached context
func (s *UserBroker) InvalidateCache(userID string) {
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

// getStale returns cached context even if expired (for graceful degradation).
func (c *userCache) getStale(userID string) *UserContext {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[userID]
	if !ok {
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

// =============================================================================
// User Prompts (Templates)
// =============================================================================

// UserPrompt represents a user's saved prompt template
type UserPrompt struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"` // Short description for prompts/list (MCP spec)
	Content     string  `json:"content"`     // Full content for prompts/get
	Enabled     bool    `json:"enabled"`
}

// GetUserPrompts retrieves all enabled prompts for a user
func (s *UserBroker) GetUserPrompts(userID string) ([]UserPrompt, error) {
	return s.fetchUserPrompts(userID, "")
}

// GetUserPromptByName retrieves a specific prompt by name for a user
func (s *UserBroker) GetUserPromptByName(userID, promptName string) (*UserPrompt, error) {
	prompts, err := s.fetchUserPrompts(userID, promptName)
	if err != nil {
		return nil, err
	}
	if len(prompts) == 0 {
		return nil, nil
	}
	return &prompts[0], nil
}

// fetchUserPrompts calls the unified get_user_prompts RPC.
// When promptName is empty, returns all enabled prompts.
// When promptName is specified, returns the matching prompt.
func (s *UserBroker) fetchUserPrompts(userID, promptName string) ([]UserPrompt, error) {
	if s.apiKey == "" {
		return []UserPrompt{}, nil
	}

	params := map[string]interface{}{
		"p_user_id":      userID,
		"p_enabled_only": true,
	}
	if promptName != "" {
		params["p_prompt_name"] = promptName
	}

	reqBody, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest(
		"POST",
		s.postgrestURL+"/rpc/get_user_prompts",
		strings.NewReader(string(reqBody)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", s.apiKey)
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := doWithRetry(s.client, req, defaultRetry)
	if err != nil {
		return nil, fmt.Errorf("failed to call get_user_prompts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get_user_prompts failed: status %d", resp.StatusCode)
	}

	var prompts []UserPrompt
	if err := json.NewDecoder(resp.Body).Decode(&prompts); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return prompts, nil
}
