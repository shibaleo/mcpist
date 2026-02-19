package rest

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"gorm.io/gorm"
)

// Handler holds dependencies for REST endpoints.
type Handler struct {
	db            *gorm.DB
	gatewaySecret string
	adminEmails   map[string]bool
}

// NewHandler creates a new REST handler.
// adminEmailsCSV is a comma-separated list of admin email addresses.
func NewHandler(db *gorm.DB, gatewaySecret, adminEmailsCSV string) *Handler {
	emails := map[string]bool{}
	for _, e := range strings.Split(adminEmailsCSV, ",") {
		e = strings.TrimSpace(e)
		if e != "" {
			emails[strings.ToLower(e)] = true
		}
	}
	return &Handler{
		db:            db,
		gatewaySecret: gatewaySecret,
		adminEmails:   emails,
	}
}

// Register registers all REST routes on the given mux.
func (h *Handler) Register(mux *http.ServeMux) {
	// Public
	mux.HandleFunc("GET /v1/modules", h.listModules)

	// User registration (gateway auth only — user may not exist yet)
	mux.HandleFunc("POST /v1/me/register", h.withGateway(h.register))

	// /v1/me/* (requires gateway auth + existing user)
	mux.HandleFunc("GET /v1/me/profile", h.withAuth(h.getProfile))
	mux.HandleFunc("PUT /v1/me/settings", h.withAuth(h.updateSettings))
	mux.HandleFunc("POST /v1/me/onboarding", h.withAuth(h.completeOnboarding))
	mux.HandleFunc("GET /v1/me/usage", h.withAuth(h.getUsage))
	mux.HandleFunc("GET /v1/me/stripe", h.withAuth(h.getStripe))
	mux.HandleFunc("PUT /v1/me/stripe", h.withAuth(h.linkStripe))
	mux.HandleFunc("GET /v1/me/credentials", h.withAuth(h.listCredentials))
	mux.HandleFunc("PUT /v1/me/credentials/{module}", h.withAuth(h.upsertCredential))
	mux.HandleFunc("DELETE /v1/me/credentials/{module}", h.withAuth(h.deleteCredential))
	mux.HandleFunc("GET /v1/me/apikeys", h.withAuth(h.listAPIKeys))
	mux.HandleFunc("POST /v1/me/apikeys", h.withAuth(h.createAPIKey))
	mux.HandleFunc("DELETE /v1/me/apikeys/{id}", h.withAuth(h.revokeAPIKey))
	mux.HandleFunc("GET /v1/me/prompts", h.withAuth(h.listPrompts))
	mux.HandleFunc("GET /v1/me/prompts/{id}", h.withAuth(h.getPrompt))
	mux.HandleFunc("POST /v1/me/prompts", h.withAuth(h.createPrompt))
	mux.HandleFunc("PUT /v1/me/prompts/{id}", h.withAuth(h.updatePrompt))
	mux.HandleFunc("DELETE /v1/me/prompts/{id}", h.withAuth(h.deletePrompt))
	mux.HandleFunc("GET /v1/me/modules/config", h.withAuth(h.getModulesConfig))
	mux.HandleFunc("PUT /v1/me/modules/{name}/tools", h.withAuth(h.updateToolSettings))
	mux.HandleFunc("PUT /v1/me/modules/{name}/description", h.withAuth(h.updateModuleDescription))
	mux.HandleFunc("GET /v1/me/oauth/consents", h.withAuth(h.listOAuthConsents))
	mux.HandleFunc("DELETE /v1/me/oauth/consents/{id}", h.withAuth(h.revokeOAuthConsent))

	// Admin (requires gateway auth + admin check)
	mux.HandleFunc("PUT /v1/admin/oauth/apps/{provider}", h.withAuth(h.withAdmin(h.upsertOAuthApp)))
	mux.HandleFunc("GET /v1/admin/oauth/consents", h.withAuth(h.withAdmin(h.listAllOAuthConsents)))

	// Stripe webhook (no gateway auth — signature validation only)
	mux.HandleFunc("POST /v1/stripe/webhook", h.handleStripeWebhook)
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON: %v", err)
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// decodeJSON decodes a JSON request body into v.
func decodeJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
