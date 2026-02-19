package rest

import (
	"net/http"

	"mcpist/server/internal/db"
)

// PUT /v1/admin/oauth/apps/{provider}
func (h *Handler) upsertOAuthApp(w http.ResponseWriter, r *http.Request) {
	provider := r.PathValue("provider")
	if provider == "" {
		writeError(w, http.StatusBadRequest, "provider is required")
		return
	}

	var body struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		RedirectURI  string `json:"redirect_uri"`
		Enabled      *bool  `json:"enabled,omitempty"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	app := &db.OAuthApp{
		Provider:     provider,
		ClientID:     body.ClientID,
		ClientSecret: body.ClientSecret,
		RedirectURI:  body.RedirectURI,
		Enabled:      true,
	}
	if body.Enabled != nil {
		app.Enabled = *body.Enabled
	}

	if err := db.UpsertOAuthApp(h.db, app); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to upsert OAuth app")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// GET /v1/admin/oauth/consents
func (h *Handler) listAllOAuthConsents(w http.ResponseWriter, r *http.Request) {
	consents, err := db.ListAllOAuthConsents(h.db)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list consents")
		return
	}
	writeJSON(w, http.StatusOK, consents)
}
