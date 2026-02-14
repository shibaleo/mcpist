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

	"mcpist/server/internal/observability"
	"mcpist/server/internal/broker"
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
	FreeCredits        int
	PaidCredits        int
	EnabledModules     []string            // Modules with at least one enabled tool (derived by RPC)
	EnabledTools       map[string][]string // module -> []tool_id (whitelist)
	Language           string              // BCP47 language code (e.g., "en-US", "ja-JP")
	ModuleDescriptions broker.ModuleDescriptions
}

// TotalCredits returns the sum of free and paid credits
func (ctx *AuthContext) TotalCredits() int {
	return ctx.FreeCredits + ctx.PaidCredits
}

// Authorizer handles authorization checks
type Authorizer struct {
	gatewaySecret string
	store         *broker.UserStore
}

// NewAuthorizer creates a new authorizer.
// Panics if GATEWAY_SECRET is not set — required in all environments.
func NewAuthorizer(userStore *broker.UserStore) *Authorizer {
	secret := os.Getenv("GATEWAY_SECRET")
	if secret == "" {
		log.Fatal("GATEWAY_SECRET is not set. Set it via environment variable or .env.dev")
	}
	return &Authorizer{
		gatewaySecret: secret,
		store:         userStore,
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
	// 1. Verify gateway secret (ensures request came from Worker)
	requestSecret := r.Header.Get("X-Gateway-Secret")
	if requestSecret != a.gatewaySecret {
		observability.LogSecurityEvent("", "", "invalid_gateway_secret", map[string]any{
			"remote_addr": r.RemoteAddr,
		})
		return nil, &AuthError{
			Code:    "INVALID_GATEWAY_SECRET",
			Message: "Invalid gateway secret",
			Status:  http.StatusUnauthorized,
		}
	}

	// 2. Get user ID from X-User-ID header (set by Worker after authentication)
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		return nil, &AuthError{
			Code:    "MISSING_USER_ID",
			Message: "Missing user ID",
			Status:  http.StatusUnauthorized,
		}
	}

	// 3. Get auth type from header
	authType := r.Header.Get("X-Auth-Type")
	if authType == "" {
		authType = "unknown"
	}

	// 4. Get user's context from Store
	userContext, err := a.store.GetUserContext(userID)
	if err != nil {
		log.Printf("Failed to get user context for user %s: %v", userID, err)
		return nil, &AuthError{
			Code:    "CONTEXT_ERROR",
			Message: "Failed to verify user context",
			Status:  http.StatusInternalServerError,
		}
	}

	// 5. Check account status
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
		FreeCredits:        userContext.FreeCredits,
		PaidCredits:        userContext.PaidCredits,
		EnabledModules:     userContext.EnabledModules,
		EnabledTools:       userContext.EnabledTools,
		Language:           userContext.Language,
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
func (ctx *AuthContext) CanAccessTool(moduleName, toolName string, creditCost int) error {
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

	// 3. Check credit balance
	if ctx.TotalCredits() < creditCost {
		consoleURL := os.Getenv("CONSOLE_URL")
		billingURL := ""
		if consoleURL != "" {
			billingURL = fmt.Sprintf(" Add credits at: %s/billing", consoleURL)
		}
		return &AuthError{
			Code:    "INSUFFICIENT_CREDITS",
			Message: fmt.Sprintf("Insufficient credits. Required: %d, Available: %d.%s", creditCost, ctx.TotalCredits(), billingURL),
			Status:  http.StatusPaymentRequired,
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
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
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
