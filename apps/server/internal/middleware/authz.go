package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"mcpist/server/internal/auth"
	"mcpist/server/internal/broker"
	"mcpist/server/internal/db"
	"mcpist/server/internal/observability"

	"gorm.io/gorm"
)

// ContextKey is the type for context keys
type ContextKey string

const (
	// AuthContextKey is the context key for auth context
	AuthContextKey ContextKey = "authContext"
	// RequestIDKey is the context key for request tracing ID
	RequestIDKey ContextKey = "requestID"
)

// AuthContext contains user authentication and authorization info
type AuthContext struct {
	UserID             string
	AuthType           string // "jwt" or "api_key"
	AccountStatus      string
	PlanID             string
	DailyUsed          int
	DailyLimit         int
	EnabledModules     []string            // Modules with at least one enabled tool (derived by RPC)
	EnabledTools       map[string][]string // module -> []tool_id (whitelist)
	ModuleDescriptions broker.ModuleDescriptions
}

// WithinDailyLimit checks if the user can execute the given number of additional tools
func (ctx *AuthContext) WithinDailyLimit(count int) bool {
	return ctx.DailyUsed+count <= ctx.DailyLimit
}

// Authorizer handles authorization checks
type Authorizer struct {
	gatewayVerifier *auth.GatewayVerifier
	store           *broker.UserBroker
	db              *gorm.DB
}

// NewAuthorizer creates a new authorizer.
func NewAuthorizer(userStore *broker.UserBroker, database *gorm.DB, gatewayVerifier *auth.GatewayVerifier) *Authorizer {
	return &Authorizer{
		gatewayVerifier: gatewayVerifier,
		store:           userStore,
		db:              database,
	}
}

// Authorize is HTTP middleware that checks authorization
func (a *Authorizer) Authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx, err := a.ValidateRequest(r)
		if err != nil {
			a.writeErrorResponse(w, err)
			return
		}

		// Generate or propagate request ID for tracing
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		// Add auth context and request ID to request context
		ctx := context.WithValue(r.Context(), AuthContextKey, authCtx)
		ctx = context.WithValue(ctx, RequestIDKey, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ValidateRequest validates the request and returns auth context
func (a *Authorizer) ValidateRequest(r *http.Request) (*AuthContext, error) {
	// 1. Verify gateway JWT (ensures request came from Worker)
	token := r.Header.Get("X-Gateway-Token")
	if token == "" {
		observability.LogSecurityEvent("", "", "missing_gateway_token", map[string]any{
			"remote_addr": r.RemoteAddr,
		})
		return nil, &AuthError{
			Code:    "MISSING_GATEWAY_TOKEN",
			Message: "Missing gateway token",
			Status:  http.StatusUnauthorized,
		}
	}

	claims, err := a.gatewayVerifier.VerifyToken(token)
	if err != nil {
		observability.LogSecurityEvent("", "", "invalid_gateway_token", map[string]any{
			"remote_addr": r.RemoteAddr,
			"error":       err.Error(),
		})
		return nil, &AuthError{
			Code:    "INVALID_GATEWAY_TOKEN",
			Message: "Invalid gateway token",
			Status:  http.StatusUnauthorized,
		}
	}

	// 2. Determine user ID and auth type from JWT claims
	var userID string
	var authType string

	if claims.UserID != "" {
		// API key auth — claims.UserID is already the internal mcpist UUID
		userID = claims.UserID
		authType = "api_key"
	} else if claims.ClerkID != "" {
		// Clerk JWT auth — resolve Clerk ID to internal UUID
		authType = "jwt"
		internalID, err := db.FindOrCreateByClerkID(a.db, claims.ClerkID, claims.Email)
		if err != nil {
			log.Printf("Failed to resolve clerk_id %s: %v", claims.ClerkID, err)
			return nil, &AuthError{
				Code:    "USER_RESOLUTION_ERROR",
				Message: "Failed to resolve user identity",
				Status:  http.StatusInternalServerError,
			}
		}
		userID = internalID
	} else {
		return nil, &AuthError{
			Code:    "MISSING_USER_ID",
			Message: "Missing user identity in gateway token",
			Status:  http.StatusUnauthorized,
		}
	}

	// 3. Get user's context from Store
	userContext, err := a.store.GetUserContext(userID)
	if err != nil {
		log.Printf("Failed to get user context for user %s: %v", userID, err)
		return nil, &AuthError{
			Code:    "CONTEXT_ERROR",
			Message: "Failed to verify user context",
			Status:  http.StatusInternalServerError,
		}
	}

	// 4. Check account status
	if userContext.AccountStatus != "active" {
		return nil, &AuthError{
			Code:    "ACCOUNT_NOT_ACTIVE",
			Message: fmt.Sprintf("Account is %s", userContext.AccountStatus),
			Status:  http.StatusForbidden,
		}
	}

	// Build auth context (EnabledModules derived from EnabledTools keys by RPC)
	authCtx := &AuthContext{
		UserID:             userID,
		AuthType:           authType,
		AccountStatus:      userContext.AccountStatus,
		PlanID:             userContext.PlanID,
		DailyUsed:          userContext.DailyUsed,
		DailyLimit:         userContext.DailyLimit,
		EnabledModules:     userContext.EnabledModules,
		EnabledTools:       userContext.EnabledTools,
		ModuleDescriptions: userContext.ModuleDescriptions,
	}

	return authCtx, nil
}

// CanAccessModule checks if the user can access a specific module.
func (ctx *AuthContext) CanAccessModule(moduleName string) error {
	for _, m := range ctx.EnabledModules {
		if m == moduleName {
			return nil
		}
	}
	return &AuthError{
		Code:    "MODULE_NOT_ENABLED",
		Message: fmt.Sprintf("Module '%s' is not enabled for your account", moduleName),
		Status:  http.StatusForbidden,
	}
}

// CanAccessTool checks if the user can access a specific tool.
// Optimized: single map lookup + slice search (no separate module check needed).
func (ctx *AuthContext) CanAccessTool(moduleName, toolName string, usageCount int) error {
	toolID := moduleName + ":" + toolName

	// 1. Check if tool is enabled (whitelist approach)
	//    This implicitly checks module access (module must have enabled tools)
	enabledTools, ok := ctx.EnabledTools[moduleName]
	if !ok {
		// Module not in EnabledTools = no enabled tools for this module
		return &AuthError{
			Code:    "MODULE_NOT_ENABLED",
			Message: fmt.Sprintf("Module '%s' is not enabled for your account", moduleName),
			Status:  http.StatusForbidden,
		}
	}

	// 2. Check if specific tool is enabled
	toolEnabled := false
	for _, t := range enabledTools {
		if t == toolID {
			toolEnabled = true
			break
		}
	}
	if !toolEnabled {
		return &AuthError{
			Code:    "TOOL_DISABLED",
			Message: fmt.Sprintf("Tool '%s' is not enabled for your account", toolID),
			Status:  http.StatusForbidden,
		}
	}

	// 3. Check daily usage limit
	if usageCount > 0 && !ctx.WithinDailyLimit(usageCount) {
		consoleURL := os.Getenv("CONSOLE_URL")
		upgradeURL := ""
		if consoleURL != "" {
			upgradeURL = fmt.Sprintf(" Upgrade your plan at: %s/plan", consoleURL)
		}
		return &AuthError{
			Code:    "USAGE_LIMIT_EXCEEDED",
			Message: fmt.Sprintf("Daily usage limit exceeded. Used: %d, Limit: %d.%s", ctx.DailyUsed, ctx.DailyLimit, upgradeURL),
			Status:  http.StatusTooManyRequests,
		}
	}

	return nil
}

// AuthError represents an authorization error
type AuthError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

func (e *AuthError) Error() string {
	return e.Message
}

// writeErrorResponse writes an authorization error response
func (a *Authorizer) writeErrorResponse(w http.ResponseWriter, err error) {
	authErr, ok := err.(*AuthError)
	if !ok {
		authErr = &AuthError{
			Code:    "AUTHORIZATION_ERROR",
			Message: err.Error(),
			Status:  http.StatusInternalServerError,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(authErr.Status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   authErr.Code,
		"message": authErr.Message,
	})
}

// GetAuthContext extracts auth context from request context
func GetAuthContext(ctx context.Context) *AuthContext {
	authCtx, _ := ctx.Value(AuthContextKey).(*AuthContext)
	return authCtx
}

// GetRequestID extracts request ID from context
func GetRequestID(ctx context.Context) string {
	id, _ := ctx.Value(RequestIDKey).(string)
	return id
}

// generateRequestID creates a random 16-byte hex request ID
func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("fallback-%d", os.Getpid())
	}
	return hex.EncodeToString(b)
}
