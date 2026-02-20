package rest

import (
	"context"
	"net/http"
	"strings"

	authpkg "mcpist/server/internal/auth"
	"mcpist/server/internal/db"
)

type contextKey string

const (
	userIDKey        contextKey = "userID"
	gatewayClaimsKey contextKey = "gatewayClaims"
)

// verifyGatewayToken extracts and verifies the X-Gateway-Token JWT.
func (h *Handler) verifyGatewayToken(r *http.Request) (*authpkg.GatewayClaims, error) {
	token := r.Header.Get("X-Gateway-Token")
	if token == "" {
		return nil, &httpError{status: http.StatusUnauthorized, msg: "missing gateway token"}
	}
	claims, err := h.gatewayVerifier.VerifyToken(token)
	if err != nil {
		return nil, &httpError{status: http.StatusUnauthorized, msg: "invalid gateway token"}
	}
	return claims, nil
}

type httpError struct {
	status int
	msg    string
}

func (e *httpError) Error() string { return e.msg }

// withAuth validates gateway JWT and resolves the user to internal UUID.
// JWT claims carry user identity:
//   - claims.UserID: internal mcpist UUID (from API key auth)
//   - claims.ClerkID: Clerk user ID (from Clerk JWT auth)
//
// Returns 404 if user not found.
func (h *Handler) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, err := h.verifyGatewayToken(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid gateway token")
			return
		}

		if claims.UserID == "" && claims.ClerkID == "" {
			writeError(w, http.StatusUnauthorized, "missing user_id or clerk_id in token")
			return
		}

		var userID string
		if claims.UserID != "" {
			user, err := db.FindByID(h.db, claims.UserID)
			if err != nil {
				writeError(w, http.StatusNotFound, "user not found")
				return
			}
			userID = user.ID
		} else {
			user, err := db.FindByClerkID(h.db, claims.ClerkID)
			if err != nil {
				writeError(w, http.StatusNotFound, "user not found")
				return
			}
			userID = user.ID
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next(w, r.WithContext(ctx))
	}
}

// getUserID extracts the user ID from the request context.
func getUserID(r *http.Request) string {
	id, _ := r.Context().Value(userIDKey).(string)
	return id
}

// withGateway validates the gateway JWT only (no user lookup).
// Used for endpoints where the user may not exist yet (e.g. registration).
// Stores GatewayClaims in context for handler access.
func (h *Handler) withGateway(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, err := h.verifyGatewayToken(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid gateway token")
			return
		}

		ctx := context.WithValue(r.Context(), gatewayClaimsKey, claims)
		next(w, r.WithContext(ctx))
	}
}

// getGatewayClaims extracts gateway claims from the request context.
func getGatewayClaims(r *http.Request) *authpkg.GatewayClaims {
	claims, _ := r.Context().Value(gatewayClaimsKey).(*authpkg.GatewayClaims)
	return claims
}

// isAdmin checks if an email is in the admin list (from ADMIN_EMAILS env var).
func (h *Handler) isAdmin(email string) bool {
	return h.adminEmails[strings.ToLower(email)]
}

// withAdmin checks that the user is an admin.
func (h *Handler) withAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)

		user, err := db.FindByID(h.db, userID)
		if err != nil || user.Email == nil {
			writeError(w, http.StatusForbidden, "admin access required")
			return
		}
		if !h.isAdmin(*user.Email) {
			writeError(w, http.StatusForbidden, "admin access required")
			return
		}

		next(w, r)
	}
}
