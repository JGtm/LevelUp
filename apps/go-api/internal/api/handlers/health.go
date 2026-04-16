// Package handlers — handler GET /health.
package handlers

import (
	"net/http"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// HealthHandler gère GET /health.
type HealthHandler struct {
	repo       port.BootstrapRepository
	appVersion string
}

// NewHealthHandler crée un HealthHandler.
func NewHealthHandler(repo port.BootstrapRepository) *HealthHandler {
	return &HealthHandler{repo: repo}
}

// NewHealthHandlerWithVersion crée un HealthHandler avec la version de l'application.
func NewHealthHandlerWithVersion(repo port.BootstrapRepository, version string) *HealthHandler {
	return &HealthHandler{repo: repo, appVersion: version}
}

// ServeHTTP retourne {"status": "ok", "match_count": N, "app_version": "X.Y.Z"}.
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
		AppVersion: h.appVersion,
	})
}
