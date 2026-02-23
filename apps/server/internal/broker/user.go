package broker

import (
	"log"
	"sync"
	"time"

	"gorm.io/gorm"

	"mcpist/server/internal/db"
)

// UserBroker manages user context queries via GORM
type UserBroker struct {
	db    *gorm.DB
	cache *userCache
}

// UserContext represents the user's context for MCP tool execution
type UserContext struct {
	AccountStatus      string              `json:"account_status"`
	PlanID             string              `json:"plan_id"`
	DailyUsed          int                 `json:"daily_used"`
	DailyLimit         int                 `json:"daily_limit"`
	EnabledModules     []string            `json:"enabled_modules"`
	EnabledTools       map[string][]string `json:"enabled_tools"`
	ModuleDescriptions ModuleDescriptions  `json:"module_descriptions"`
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
func (uc *UserContext) IsToolEnabled(module, tool string) bool {
	enabledTools, ok := uc.EnabledTools[module]
	if !ok {
		return false
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

// NewUserBroker creates a new user broker with GORM DB
func NewUserBroker(database *gorm.DB) *UserBroker {
	return &UserBroker{
		db: database,
		cache: &userCache{
			items: make(map[string]*userCacheItem),
			ttl:   30 * time.Second,
		},
	}
}

// HealthCheck verifies database connectivity.
func (s *UserBroker) HealthCheck() error {
	return db.HealthCheck(s.db)
}

// GetUserContext retrieves the user's context (account status, credits, modules, tools).
// On fetch failure, returns stale cached data if available (graceful degradation).
func (s *UserBroker) GetUserContext(userID string) (*UserContext, error) {
	// Check cache first (non-expired)
	if cached := s.cache.get(userID); cached != nil {
		return cached, nil
	}

	// Query DB via GORM
	ctx, err := s.fetchUserContext(userID)
	if err != nil {
		// Fall back to stale cache on transient failure
		if stale := s.cache.getStale(userID); stale != nil {
			log.Printf("GetUserContext: using stale cache for %s due to: %v", userID, err)
			s.cache.set(userID, stale)
			return stale, nil
		}
		return nil, err
	}

	s.cache.set(userID, ctx)
	return ctx, nil
}

// fetchUserContext queries the database for user context
func (s *UserBroker) fetchUserContext(userID string) (*UserContext, error) {
	mcpCtx, err := db.GetMCPContext(s.db, userID)
	if err != nil {
		return nil, err
	}

	return &UserContext{
		AccountStatus:      mcpCtx.AccountStatus,
		PlanID:             mcpCtx.PlanID,
		DailyUsed:          mcpCtx.DailyUsed,
		DailyLimit:         mcpCtx.DailyLimit,
		EnabledModules:     mcpCtx.EnabledModules,
		EnabledTools:       mcpCtx.EnabledTools,
		ModuleDescriptions: ModuleDescriptions(mcpCtx.ModuleDescriptions),
	}, nil
}

// ToolDetail represents a single tool execution in the details array
type ToolDetail struct {
	TaskID string `json:"task_id,omitempty"`
	Module string `json:"module"`
	Tool   string `json:"tool"`
}

// RecordUsage records tool usage asynchronously (fire-and-forget).
func (s *UserBroker) RecordUsage(userID, metaTool, requestID string, details []ToolDetail) {
	go func() {
		if err := db.RecordUsage(s.db, userID, metaTool, requestID, details); err != nil {
			log.Printf("RecordUsage: failed: %v", err)
		}
	}()
}

// SyncModuleEntry represents a module to sync to the database
type SyncModuleEntry struct {
	Name         string            `json:"name"`
	Status       string            `json:"status"`
	Descriptions map[string]string `json:"descriptions,omitempty"`
	Tools        interface{}       `json:"tools"`
}

// SyncModules upserts module+tool data to the database.
func (s *UserBroker) SyncModules(entries []SyncModuleEntry) error {
	dbEntries := make([]db.SyncModuleEntry, len(entries))
	for i, e := range entries {
		dbEntries[i] = db.SyncModuleEntry{
			Name:   e.Name,
			Status: e.Status,
			Tools:  e.Tools,
		}
	}

	upserted, err := db.SyncModules(s.db, dbEntries)
	if err != nil {
		return err
	}

	log.Printf("SyncModules: upserted %d/%d modules", upserted, len(entries))
	return nil
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
	Description *string `json:"description"`
	Content     string  `json:"content"`
	Enabled     bool    `json:"enabled"`
}

// GetUserPrompts retrieves all enabled prompts for a user
func (s *UserBroker) GetUserPrompts(userID string) ([]UserPrompt, error) {
	prompts, err := db.GetEnabledPrompts(s.db, userID)
	if err != nil {
		return nil, err
	}

	result := make([]UserPrompt, len(prompts))
	for i, p := range prompts {
		result[i] = UserPrompt{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Content:     p.Content,
			Enabled:     p.Enabled,
		}
	}
	return result, nil
}

// GetUserPromptByName retrieves a specific prompt by name for a user
func (s *UserBroker) GetUserPromptByName(userID, promptName string) (*UserPrompt, error) {
	p, err := db.GetEnabledPromptByName(s.db, userID, promptName)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, nil
	}
	return &UserPrompt{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Content:     p.Content,
		Enabled:     p.Enabled,
	}, nil
}
