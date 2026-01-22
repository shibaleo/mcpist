package entitlement

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Store manages user entitlement queries to Supabase
type Store struct {
	supabaseURL string
	serviceKey  string
	client      *http.Client
	cache       *cache
}

// UserEntitlement represents a user's current entitlements
type UserEntitlement struct {
	UserStatus        string   `json:"user_status"`
	PlanName          string   `json:"plan_name"`
	RateLimitRPM      int      `json:"rate_limit_rpm"`
	RateLimitBurst    int      `json:"rate_limit_burst"`
	QuotaMonthly      *int     `json:"quota_monthly"` // nil = unlimited
	CreditEnabled     bool     `json:"credit_enabled"`
	CreditBalance     int      `json:"credit_balance"`
	UsageCurrentMonth int      `json:"usage_current_month"`
	EnabledModules    []string `json:"enabled_modules"`
}

// cache stores entitlements with TTL
type cache struct {
	mu    sync.RWMutex
	items map[string]*cacheItem
	ttl   time.Duration
}

type cacheItem struct {
	entitlement *UserEntitlement
	expiresAt   time.Time
}

// NewStore creates a new entitlement store
func NewStore() *Store {
	return &Store{
		supabaseURL: os.Getenv("SUPABASE_URL"),
		serviceKey:  os.Getenv("SUPABASE_SECRET_KEY"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: &cache{
			items: make(map[string]*cacheItem),
			ttl:   30 * time.Second, // Cache for 30 seconds
		},
	}
}

// GetUserEntitlement retrieves the user's entitlements
func (s *Store) GetUserEntitlement(userID string) (*UserEntitlement, error) {
	// Check cache first
	if cached := s.cache.get(userID); cached != nil {
		return cached, nil
	}

	// Query Supabase RPC
	entitlement, err := s.fetchEntitlement(userID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	s.cache.set(userID, entitlement)

	return entitlement, nil
}

// fetchEntitlement calls the Supabase RPC function
func (s *Store) fetchEntitlement(userID string) (*UserEntitlement, error) {
	if s.serviceKey == "" {
		// Return default entitlement for development without service key
		return &UserEntitlement{
			UserStatus:        "active",
			PlanName:          "free",
			RateLimitRPM:      10,
			RateLimitBurst:    5,
			QuotaMonthly:      intPtr(1000),
			CreditEnabled:     false,
			CreditBalance:     0,
			UsageCurrentMonth: 0,
			EnabledModules:    []string{},
		}, nil
	}

	reqBody := fmt.Sprintf(`{"p_user_id": "%s"}`, userID)
	req, err := http.NewRequest(
		"POST",
		s.supabaseURL+"/rest/v1/rpc/get_user_entitlement",
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
		return nil, fmt.Errorf("failed to call get_user_entitlement: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get_user_entitlement failed: status %d", resp.StatusCode)
	}

	var results []UserEntitlement
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(results) == 0 {
		// User not found - return default free plan entitlement
		return &UserEntitlement{
			UserStatus:        "active",
			PlanName:          "free",
			RateLimitRPM:      10,
			RateLimitBurst:    5,
			QuotaMonthly:      intPtr(1000),
			CreditEnabled:     false,
			CreditBalance:     0,
			UsageCurrentMonth: 0,
			EnabledModules:    []string{},
		}, nil
	}

	return &results[0], nil
}

// IncrementUsage increments the user's usage count for the current month
func (s *Store) IncrementUsage(userID string) (int, error) {
	if s.serviceKey == "" {
		return 0, nil // Skip in development
	}

	reqBody := fmt.Sprintf(`{"p_user_id": "%s"}`, userID)
	req, err := http.NewRequest(
		"POST",
		s.supabaseURL+"/rest/v1/rpc/increment_usage",
		strings.NewReader(reqBody),
	)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", s.serviceKey)
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("increment_usage failed: status %d", resp.StatusCode)
	}

	var count int
	if err := json.NewDecoder(resp.Body).Decode(&count); err != nil {
		return 0, err
	}

	// Invalidate cache to reflect new usage
	s.cache.delete(userID)

	return count, nil
}

// DeductCredits deducts credits from user's balance
// Returns new balance, or -1 if insufficient credits
func (s *Store) DeductCredits(userID string, amount int, description, referenceID string) (int, error) {
	if s.serviceKey == "" {
		return 0, nil // Skip in development
	}

	reqBody := fmt.Sprintf(
		`{"p_user_id": "%s", "p_amount": %d, "p_description": %s, "p_reference_id": %s}`,
		userID,
		amount,
		jsonString(description),
		jsonString(referenceID),
	)
	req, err := http.NewRequest(
		"POST",
		s.supabaseURL+"/rest/v1/rpc/deduct_credits",
		strings.NewReader(reqBody),
	)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", s.serviceKey)
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("deduct_credits failed: status %d", resp.StatusCode)
	}

	var balance int
	if err := json.NewDecoder(resp.Body).Decode(&balance); err != nil {
		return 0, err
	}

	// Invalidate cache to reflect new balance
	s.cache.delete(userID)

	return balance, nil
}

// GetToolCost retrieves the credit cost for a tool
func (s *Store) GetToolCost(moduleName, toolName string) (int, error) {
	if s.serviceKey == "" {
		return 1, nil // Default cost in development
	}

	reqBody := fmt.Sprintf(`{"p_module_name": "%s", "p_tool_name": "%s"}`, moduleName, toolName)
	req, err := http.NewRequest(
		"POST",
		s.supabaseURL+"/rest/v1/rpc/get_tool_cost",
		strings.NewReader(reqBody),
	)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", s.serviceKey)
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 1, nil // Default to 1 if RPC fails
	}

	var cost int
	if err := json.NewDecoder(resp.Body).Decode(&cost); err != nil {
		return 1, nil
	}

	return cost, nil
}

// InvalidateCache removes a user's cached entitlement
func (s *Store) InvalidateCache(userID string) {
	s.cache.delete(userID)
}

// Cache methods

func (c *cache) get(userID string) *UserEntitlement {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[userID]
	if !ok {
		return nil
	}

	if time.Now().After(item.expiresAt) {
		return nil
	}

	return item.entitlement
}

func (c *cache) set(userID string, entitlement *UserEntitlement) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[userID] = &cacheItem{
		entitlement: entitlement,
		expiresAt:   time.Now().Add(c.ttl),
	}
}

func (c *cache) delete(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, userID)
}

// Helper functions

func intPtr(i int) *int {
	return &i
}

func jsonString(s string) string {
	if s == "" {
		return "null"
	}
	b, _ := json.Marshal(s)
	return string(b)
}
