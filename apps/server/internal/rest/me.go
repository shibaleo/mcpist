package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"mcpist/server/internal/auth"
	"mcpist/server/internal/db"
)

// POST /v1/me/register
func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	clerkID := r.Header.Get("X-Clerk-ID")
	if clerkID == "" {
		writeError(w, http.StatusBadRequest, "missing X-Clerk-ID")
		return
	}
	email := r.Header.Get("X-User-Email")
	if email == "" {
		writeError(w, http.StatusBadRequest, "missing X-User-Email")
		return
	}

	userID, err := db.FindOrCreateByClerkID(h.db, clerkID, email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to register user")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": userID})
}

// GET /v1/me/profile
func (h *Handler) getProfile(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	user, err := db.FindByID(h.db, userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	admin := user.Email != nil && h.isAdmin(*user.Email)
	profile, err := db.GetMyProfile(h.db, userID, admin)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

// PUT /v1/me/settings
func (h *Handler) updateSettings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Settings json.RawMessage `json:"settings"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := db.UpdateSettings(h.db, getUserID(r), body.Settings); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update settings")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// POST /v1/me/onboarding
func (h *Handler) completeOnboarding(w http.ResponseWriter, r *http.Request) {
	var body struct {
		EventID string `json:"event_id"`
	}
	if err := decodeJSON(r, &body); err != nil || body.EventID == "" {
		writeError(w, http.StatusBadRequest, "event_id is required")
		return
	}

	if err := db.CompleteOnboarding(h.db, getUserID(r), body.EventID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to complete onboarding")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// GET /v1/me/usage
func (h *Handler) getUsage(w http.ResponseWriter, r *http.Request) {
	usage, err := db.GetUsage(h.db, getUserID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get usage")
		return
	}
	writeJSON(w, http.StatusOK, usage)
}

// GET /v1/me/stripe
func (h *Handler) getStripe(w http.ResponseWriter, r *http.Request) {
	customerID, err := db.GetStripeCustomerID(h.db, getUserID(r))
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]*string{"stripe_customer_id": customerID})
}

// PUT /v1/me/stripe
func (h *Handler) linkStripe(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CustomerID string `json:"customer_id"`
	}
	if err := decodeJSON(r, &body); err != nil || body.CustomerID == "" {
		writeError(w, http.StatusBadRequest, "customer_id is required")
		return
	}

	if err := db.LinkStripeCustomer(h.db, getUserID(r), body.CustomerID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to link stripe customer")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// GET /v1/me/credentials
func (h *Handler) listCredentials(w http.ResponseWriter, r *http.Request) {
	creds, err := db.ListCredentials(h.db, getUserID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list credentials")
		return
	}
	writeJSON(w, http.StatusOK, creds)
}

// PUT /v1/me/credentials/{module}
func (h *Handler) upsertCredential(w http.ResponseWriter, r *http.Request) {
	module := r.PathValue("module")
	if module == "" {
		writeError(w, http.StatusBadRequest, "module is required")
		return
	}

	var body struct {
		Credentials json.RawMessage `json:"credentials"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := db.UpsertCredential(h.db, getUserID(r), module, string(body.Credentials)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to upsert credential")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// DELETE /v1/me/credentials/{module}
func (h *Handler) deleteCredential(w http.ResponseWriter, r *http.Request) {
	module := r.PathValue("module")
	if module == "" {
		writeError(w, http.StatusBadRequest, "module is required")
		return
	}

	if err := db.DeleteCredential(h.db, getUserID(r), module); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// GET /v1/me/apikeys
func (h *Handler) listAPIKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := db.ListAPIKeys(h.db, getUserID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list API keys")
		return
	}
	writeJSON(w, http.StatusOK, keys)
}

// POST /v1/me/apikeys
func (h *Handler) createAPIKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string  `json:"name"`
		DisplayName string  `json:"display_name"`
		ExpiresIn   *int    `json:"expires_in,omitempty"`  // seconds
		ExpiresAt   *string `json:"expires_at,omitempty"`  // ISO 8601
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	// Accept both "name" and "display_name"
	name := body.Name
	if name == "" {
		name = body.DisplayName
	}
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	userID := getUserID(r)

	// Compute expiration: accept expires_in (seconds) or expires_at (ISO 8601)
	var expiresAt *time.Time
	if body.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *body.ExpiresAt)
		if err == nil {
			expiresAt = &t
		}
	} else if body.ExpiresIn != nil {
		t := time.Now().Add(time.Duration(*body.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	// Create DB record first to get the key ID
	key, err := db.CreateAPIKey(h.db, userID, "", "mpt_", name, expiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create API key record")
		return
	}

	// Generate JWT signed with Ed25519
	token, err := auth.GenerateAPIKeyJWT(userID, key.ID, expiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate API key")
		return
	}

	// Update the key record with the JWT KID and prefix
	prefix := token[:12] + "..."
	h.db.Model(&db.APIKey{}).Where("id = ?", key.ID).Updates(map[string]interface{}{
		"jwt_kid":    auth.GetKeyPair().KID,
		"key_prefix": prefix,
	})

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":           key.ID,
		"api_key":      token,
		"key":          token,
		"key_prefix":   prefix,
		"name":         name,
		"display_name": name,
		"expires_at":   expiresAt,
		"created_at":   key.CreatedAt,
	})
}

// DELETE /v1/me/apikeys/{id}
func (h *Handler) revokeAPIKey(w http.ResponseWriter, r *http.Request) {
	keyID := r.PathValue("id")
	if keyID == "" {
		writeError(w, http.StatusBadRequest, "key id is required")
		return
	}

	if err := db.RevokeAPIKey(h.db, getUserID(r), keyID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// GET /v1/me/prompts
func (h *Handler) listPrompts(w http.ResponseWriter, r *http.Request) {
	moduleName := r.URL.Query().Get("module")
	var modulePtr *string
	if moduleName != "" {
		modulePtr = &moduleName
	}

	prompts, err := db.ListPrompts(h.db, getUserID(r), modulePtr)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list prompts")
		return
	}
	writeJSON(w, http.StatusOK, prompts)
}

// GET /v1/me/prompts/{id}
func (h *Handler) getPrompt(w http.ResponseWriter, r *http.Request) {
	promptID := r.PathValue("id")
	prompt, err := db.GetPrompt(h.db, getUserID(r), promptID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, prompt)
}

// POST /v1/me/prompts
func (h *Handler) createPrompt(w http.ResponseWriter, r *http.Request) {
	var prompt db.Prompt
	if err := decodeJSON(r, &prompt); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	prompt.UserID = getUserID(r)

	if err := db.CreatePrompt(h.db, &prompt); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create prompt")
		return
	}
	writeJSON(w, http.StatusCreated, prompt)
}

// PUT /v1/me/prompts/{id}
func (h *Handler) updatePrompt(w http.ResponseWriter, r *http.Request) {
	promptID := r.PathValue("id")

	var updates map[string]interface{}
	if err := decodeJSON(r, &updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := db.UpdatePrompt(h.db, getUserID(r), promptID, updates); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// DELETE /v1/me/prompts/{id}
func (h *Handler) deletePrompt(w http.ResponseWriter, r *http.Request) {
	promptID := r.PathValue("id")
	if err := db.DeletePrompt(h.db, getUserID(r), promptID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// GET /v1/me/modules/config
func (h *Handler) getModulesConfig(w http.ResponseWriter, r *http.Request) {
	configs, err := db.GetModuleConfig(h.db, getUserID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get module config")
		return
	}
	writeJSON(w, http.StatusOK, configs)
}

// PUT /v1/me/modules/{name}/tools
func (h *Handler) updateToolSettings(w http.ResponseWriter, r *http.Request) {
	moduleName := r.PathValue("name")
	var body struct {
		Enabled  []string `json:"enabled"`
		Disabled []string `json:"disabled"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := db.UpsertToolSettings(h.db, getUserID(r), moduleName, body.Enabled, body.Disabled); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// PUT /v1/me/modules/{name}/description
func (h *Handler) updateModuleDescription(w http.ResponseWriter, r *http.Request) {
	moduleName := r.PathValue("name")
	var body struct {
		Description string `json:"description"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := db.UpsertModuleDescription(h.db, getUserID(r), moduleName, body.Description); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// GET /v1/me/oauth/consents
func (h *Handler) listOAuthConsents(w http.ResponseWriter, r *http.Request) {
	consents, err := db.ListOAuthConsents(h.db, getUserID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list consents")
		return
	}
	writeJSON(w, http.StatusOK, consents)
}

// DELETE /v1/me/oauth/consents/{id}
func (h *Handler) revokeOAuthConsent(w http.ResponseWriter, r *http.Request) {
	consentID := r.PathValue("id")
	if err := db.RevokeOAuthConsent(h.db, getUserID(r), consentID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}
