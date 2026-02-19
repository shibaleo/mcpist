package rest

import (
	"context"
	"net/http"
	"strings"

	"mcpist/server/internal/db"
)

type contextKey string

const userIDKey contextKey = "userID"

// withAuth validates gateway secret and resolves the user to internal UUID.
// Two mutually exclusive headers are supported:
//   - X-User-ID: internal mcpist UUID (from API key auth)
//   - X-Clerk-ID: Clerk user ID (from Clerk JWT auth)
//
// Exactly one must be present. Returns 404 if user not found.
func (h *Handler) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Gateway-Secret") != h.gatewaySecret {
			writeError(w, http.StatusUnauthorized, "invalid gateway secret")
			return
		}

		mcpistUserID := r.Header.Get("X-User-ID")
		clerkID := r.Header.Get("X-Clerk-ID")

		if mcpistUserID == "" && clerkID == "" {
			writeError(w, http.StatusUnauthorized, "missing X-User-ID or X-Clerk-ID")
			return
		}
		if mcpistUserID != "" && clerkID != "" {
			writeError(w, http.StatusBadRequest, "provide X-User-ID or X-Clerk-ID, not both")
			return
		}

		var userID string
		if mcpistUserID != "" {
			user, err := db.FindByID(h.db, mcpistUserID)
			if err != nil {
				writeError(w, http.StatusNotFound, "user not found")
				return
			}
			userID = user.ID
		} else {
			user, err := db.FindByClerkID(h.db, clerkID)
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

// withGateway validates the gateway secret only (no user lookup).
// Used for endpoints where the user may not exist yet (e.g. registration).
func (h *Handler) withGateway(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Gateway-Secret") != h.gatewaySecret {
			writeError(w, http.StatusUnauthorized, "invalid gateway secret")
			return
		}
		next(w, r)
	}
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
