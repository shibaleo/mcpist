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
	UserStatus     string
	PlanName       string
	RateLimitRPM   int
	RateLimitBurst int
	QuotaMonthly   *int // nil = unlimited
	CreditEnabled  bool
	CreditBalance  int
	UsageCount     int
	EnabledModules []string
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

	// 4. Get user's entitlements from Entitlement Store
	entitlement, err := a.store.GetUserEntitlement(userID)
	if err != nil {
		log.Printf("Failed to get entitlement for user %s: %v", userID, err)
		return nil, &AuthError{
			Code:    "ENTITLEMENT_ERROR",
			Message: "Failed to verify user entitlements",
			Status:  http.StatusInternalServerError,
		}
	}

	// 5. Check account status
	if entitlement.UserStatus != "active" {
		return nil, &AuthError{
			Code:    "ACCOUNT_NOT_ACTIVE",
			Message: fmt.Sprintf("Account is %s", entitlement.UserStatus),
			Status:  http.StatusForbidden,
		}
	}

	// 6. Check monthly quota (if not unlimited)
	if entitlement.QuotaMonthly != nil && entitlement.UsageCurrentMonth >= *entitlement.QuotaMonthly {
		return nil, &AuthError{
			Code:    "QUOTA_EXCEEDED",
			Message: "Monthly quota exceeded",
			Status:  http.StatusTooManyRequests,
		}
	}

	// Build auth context
	authCtx := &AuthContext{
		UserID:         userID,
		AuthType:       authType,
		UserStatus:     entitlement.UserStatus,
		PlanName:       entitlement.PlanName,
		RateLimitRPM:   entitlement.RateLimitRPM,
		RateLimitBurst: entitlement.RateLimitBurst,
		QuotaMonthly:   entitlement.QuotaMonthly,
		CreditEnabled:  entitlement.CreditEnabled,
		CreditBalance:  entitlement.CreditBalance,
		UsageCount:     entitlement.UsageCurrentMonth,
		EnabledModules: entitlement.EnabledModules,
	}

	log.Printf("Authorization: user=%s plan=%s usage=%d/%v",
		userID, entitlement.PlanName, entitlement.UsageCurrentMonth,
		quotaString(entitlement.QuotaMonthly))

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
// This includes module access check and credit check if applicable
func (ctx *AuthContext) CanAccessTool(moduleName, toolName string, store *Store) error {
	// 1. Check module access
	if err := ctx.CanAccessModule(moduleName); err != nil {
		return err
	}

	// 2. If credit-based billing, check credits
	if ctx.CreditEnabled {
		cost, err := store.GetToolCost(moduleName, toolName)
		if err != nil {
			log.Printf("Failed to get tool cost: %v", err)
			cost = 1 // Default cost
		}

		if ctx.CreditBalance < cost {
			return &AuthError{
				Code:    "INSUFFICIENT_CREDITS",
				Message: fmt.Sprintf("Insufficient credits. Required: %d, Available: %d", cost, ctx.CreditBalance),
				Status:  http.StatusPaymentRequired,
			}
		}
	}

	return nil
}

// ConsumeUsage increments usage after a successful tool call
func (ctx *AuthContext) ConsumeUsage(store *Store) error {
	_, err := store.IncrementUsage(ctx.UserID)
	return err
}

// ConsumeCredits deducts credits after a successful tool call (if credit-based)
func (ctx *AuthContext) ConsumeCredits(store *Store, moduleName, toolName, referenceID string) error {
	if !ctx.CreditEnabled {
		return nil
	}

	cost, err := store.GetToolCost(moduleName, toolName)
	if err != nil {
		cost = 1 // Default cost
	}

	description := fmt.Sprintf("Tool call: %s/%s", moduleName, toolName)
	newBalance, err := store.DeductCredits(ctx.UserID, cost, description, referenceID)
	if err != nil {
		return err
	}

	if newBalance < 0 {
		return &AuthError{
			Code:    "CREDIT_DEDUCTION_FAILED",
			Message: "Failed to deduct credits",
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
		"error": authErr.Code,
		"message": authErr.Message,
	})
}

// GetAuthContext extracts auth context from request context
func GetAuthContext(ctx context.Context) *AuthContext {
	authCtx, _ := ctx.Value(AuthContextKey).(*AuthContext)
	return authCtx
}

// Helper functions

func quotaString(quota *int) string {
	if quota == nil {
		return "unlimited"
	}
	return fmt.Sprintf("%d", *quota)
}
