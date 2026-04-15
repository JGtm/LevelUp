// Package handlers — handler GET /health.
package handlers

import (
	"net/http"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// HealthHandler gère GET /health.
type HealthHandler struct {
	repo port.BootstrapRepository
}

// NewHealthHandler crée un HealthHandler.
func NewHealthHandler(repo port.BootstrapRepository) *HealthHandler {
	return &HealthHandler{repo: repo}
}

// ServeHTTP retourne {"status": "ok", "match_count": N} en lisant shared_matches_v2.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	count, err := h.repo.GetMatchCount(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "db_unavailable",
			"Impossible de lire shared_matches_v2: "+err.Error())
		return
	}

	dbVersion, _ := h.repo.GetDBVersion(r.Context())

	writeJSON(w, http.StatusOK, domain.HealthResponse{
		Status:     "ok",
		MatchCount: count,
		DBVersion:  dbVersion,
	})
}
