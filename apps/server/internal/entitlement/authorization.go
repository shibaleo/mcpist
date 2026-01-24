package entitlement

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// ContextKey is the type for context keys
type ContextKey string

const (
	// AuthContextKey is the context key for auth context
	AuthContextKey ContextKey = "authContext"
)

// AuthContext contains user authentication and authorization info
type AuthContext struct {
	UserID         string
	AuthType       string // "jwt" or "api_key"
	AccountStatus  string
	FreeCredits    int
	PaidCredits    int
	EnabledModules []string
	DisabledTools  map[string][]string // module -> []tool
}

// TotalCredits returns the sum of free and paid credits
func (ctx *AuthContext) TotalCredits() int {
	return ctx.FreeCredits + ctx.PaidCredits
}

// Authorizer handles authorization checks
type Authorizer struct {
	gatewaySecret string
	store         *Store
}

// NewAuthorizer creates a new authorizer
func NewAuthorizer(store *Store) *Authorizer {
	return &Authorizer{
		gatewaySecret: os.Getenv("GATEWAY_SECRET"),
		store:         store,
	}
}

// Authorize is HTTP middleware that checks authorization
func (a *Authorizer) Authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx, err := a.ValidateRequest(r)
		if err != nil {
			log.Printf("Authorization failed: %v", err)
			a.writeErrorResponse(w, err)
			return
		}

		// Add auth context to request context
		ctx := context.WithValue(r.Context(), AuthContextKey, authCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ValidateRequest validates the request and returns auth context
func (a *Authorizer) ValidateRequest(r *http.Request) (*AuthContext, error) {
	// 1. Verify gateway secret (ensures request came from Worker)
	if a.gatewaySecret != "" {
		requestSecret := r.Header.Get("X-Gateway-Secret")
		if requestSecret != a.gatewaySecret {
			return nil, &AuthError{
				Code:    "INVALID_GATEWAY_SECRET",
				Message: "Invalid gateway secret",
				Status:  http.StatusUnauthorized,
			}
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

	// Build auth context
	authCtx := &AuthContext{
		UserID:         userID,
		AuthType:       authType,
		AccountStatus:  userContext.AccountStatus,
		FreeCredits:    userContext.FreeCredits,
		PaidCredits:    userContext.PaidCredits,
		EnabledModules: userContext.EnabledModules,
		DisabledTools:  userContext.DisabledTools,
	}

	log.Printf("Authorization: user=%s credits=free:%d+paid:%d modules=%d",
		userID, userContext.FreeCredits, userContext.PaidCredits, len(userContext.EnabledModules))

	return authCtx, nil
}

// CanAccessModule checks if the user can access a specific module
func (ctx *AuthContext) CanAccessModule(moduleName string) error {
	// Check if module is enabled for this user
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

// CanAccessTool checks if the user can access a specific tool
func (ctx *AuthContext) CanAccessTool(moduleName, toolName string, creditCost int) error {
	// 1. Check module access
	if err := ctx.CanAccessModule(moduleName); err != nil {
		return err
	}

	// 2. Check if tool is disabled
	if disabledTools, ok := ctx.DisabledTools[moduleName]; ok {
		for _, t := range disabledTools {
			if t == toolName {
				return &AuthError{
					Code:    "TOOL_DISABLED",
					Message: fmt.Sprintf("Tool '%s/%s' is disabled for your account", moduleName, toolName),
					Status:  http.StatusForbidden,
				}
			}
		}
	}

	// 3. Check credit balance
	if ctx.TotalCredits() < creditCost {
		return &AuthError{
			Code:    "INSUFFICIENT_CREDITS",
			Message: fmt.Sprintf("Insufficient credits. Required: %d, Available: %d", creditCost, ctx.TotalCredits()),
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
