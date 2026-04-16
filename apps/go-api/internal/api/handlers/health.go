// Package handlers — handler GET /health.
package handlers

import (
	"net/http"
	"runtime"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// HealthHandler gère GET /health.
type HealthHandler struct {
	repo       port.BootstrapRepository
	appVersion string
	startedAt  time.Time
}

// NewHealthHandler crée un HealthHandler.
func NewHealthHandler(repo port.BootstrapRepository) *HealthHandler {
	return &HealthHandler{repo: repo, startedAt: time.Now()}
}

// NewHealthHandlerWithVersion crée un HealthHandler avec la version de l'application.
func NewHealthHandlerWithVersion(repo port.BootstrapRepository, version string) *HealthHandler {
	return &HealthHandler{repo: repo, appVersion: version, startedAt: time.Now()}
}

// ServeHTTP retourne le healthcheck enrichi (Sprint 41 T3).
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	count, err := h.repo.GetMatchCount(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "db_unavailable",
			"Impossible de lire shared_matches_v2: "+err.Error())
		return
	}

	dbVersion, _ := h.repo.GetDBVersion(r.Context())
	playerCount, _ := h.repo.GetPlayerCount(r.Context())
	lastSync, _ := h.repo.GetLastSyncAt(r.Context())

	writeJSON(w, http.StatusOK, domain.HealthResponse{
		Status:      "ok",
		MatchCount:  count,
		DBVersion:   dbVersion,
		AppVersion:  h.appVersion,
		PlayerCount: playerCount,
		LastSyncAt:  lastSync,
		Uptime:      time.Since(h.startedAt).Round(time.Second).String(),
		GoVersion:   runtime.Version(),
	})
}
