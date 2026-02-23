package ogenserver

import (
	"context"
	"fmt"

	"mcpist/server/internal/auth"
	"mcpist/server/internal/db"
	gen "mcpist/server/internal/ogenserver/gen"

	"gorm.io/gorm"
)

// securityHandler implements gen.SecurityHandler.
type securityHandler struct {
	verifier *auth.GatewayVerifier
	db       *gorm.DB
}

var _ gen.SecurityHandler = (*securityHandler)(nil)

// NewSecurityHandler creates a SecurityHandler that validates gateway JWTs
// and resolves users from the token claims.
func NewSecurityHandler(verifier *auth.GatewayVerifier, database *gorm.DB) gen.SecurityHandler {
	return &securityHandler{
		verifier: verifier,
		db:       database,
	}
}

// HandleGatewayToken validates the X-Gateway-Token JWT and stores the
// resolved user ID in context. The behaviour varies by operation:
//
//   - ListModules: no security (ogen won't call this — no security on that op)
//   - RegisterUser: JWT only (user may not exist yet), stores email in ctx
//   - Admin ops: JWT + user lookup + admin check
//   - All others: JWT + user lookup
func (s *securityHandler) HandleGatewayToken(ctx context.Context, operationName gen.OperationName, t gen.GatewayToken) (context.Context, error) {
	claims, err := s.verifier.VerifyToken(t.APIKey)
	if err != nil {
		return ctx, fmt.Errorf("invalid gateway token")
	}

	// Internal operations: JWT validation only (no user context needed)
	if isInternalOperation(operationName) {
		return ctx, nil
	}

	// Registration: user may not exist yet — just validate JWT and store claims
	if operationName == gen.RegisterUserOperation {
		if claims.ClerkID == "" {
			return ctx, fmt.Errorf("missing clerk_id in gateway token")
		}
		if claims.Email == "" {
			return ctx, fmt.Errorf("missing email in gateway token")
		}
		// Store clerk_id as "userID" temporarily (register handler reads it)
		ctx = withUserID(ctx, claims.ClerkID)
		ctx = withEmail(ctx, claims.Email)
		return ctx, nil
	}

	// All other operations: resolve to internal user ID
	if claims.UserID == "" && claims.ClerkID == "" {
		return ctx, fmt.Errorf("missing user_id or clerk_id in token")
	}

	var userID string
	if claims.UserID != "" {
		user, err := db.FindByID(s.db, claims.UserID)
		if err != nil {
			return ctx, fmt.Errorf("user not found")
		}
		userID = user.ID
	} else {
		user, err := db.FindByClerkID(s.db, claims.ClerkID)
		if err != nil {
			return ctx, fmt.Errorf("user not found")
		}
		userID = user.ID
	}

	ctx = withUserID(ctx, userID)
	if claims.Email != "" {
		ctx = withEmail(ctx, claims.Email)
	}

	// Admin check for admin operations
	if isAdminOperation(operationName) {
		user, err := db.FindByID(s.db, userID)
		if err != nil || user.Role != "admin" {
			return ctx, fmt.Errorf("admin access required")
		}
	}

	return ctx, nil
}

func isInternalOperation(op gen.OperationName) bool {
	switch op {
	case gen.GetApiKeyStatusOperation:
		return true
	}
	return false
}

func isAdminOperation(op gen.OperationName) bool {
	switch op {
	case gen.ListOAuthAppsOperation,
		gen.UpsertOAuthAppOperation,
		gen.DeleteOAuthAppOperation,
		gen.ListAllOAuthConsentsOperation:
		return true
	}
	return false
}
