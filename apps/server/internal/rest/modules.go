package rest

import (
	"net/http"

	"mcpist/server/internal/db"
)

// GET /v1/modules — public, no auth required
func (h *Handler) listModules(w http.ResponseWriter, r *http.Request) {
	modules, err := db.ListModules(h.db)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list modules")
		return
	}
	writeJSON(w, http.StatusOK, modules)
}
