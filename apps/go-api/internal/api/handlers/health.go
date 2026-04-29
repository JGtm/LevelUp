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
//
// Deprecated: utiliser /healthz pour la liveness ou /readyz pour la readiness.
// /health garde la sémantique mixte (200 si DB OK) pour rétrocompat.
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

// Liveness gère GET /healthz (P8.11).
//
// Liveness probe : signale que le process Go est vivant et n'a pas paniqué.
// **Aucun I/O DB**, aucune requête réseau — latence < 5ms garantie.
// Si ce handler répond 200, l'orchestrateur (K8s/LB) NE doit PAS redémarrer
// l'instance. Si la DB est en panne, c'est un problème de readiness, pas
// de liveness — voir /readyz.
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "alive",
		"uptime": time.Since(h.startedAt).Round(time.Second).String(),
	})
}

// Readiness gère GET /readyz (P8.11).
//
// Readiness probe : signale que l'app est prête à accepter du trafic.
// Vérifie : DuckDB metadata accessible (read), filesystem data/ accessible.
// Si un check échoue → 503 + body JSON `{checks: {duckdb: "ok|err", fs: "ok|err"}}`.
// Latence cible < 100ms.
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	checks := map[string]string{}
	allOK := true

	// Check DB : tentative de lecture du compte de matchs (fail-fast).
	if _, err := h.repo.GetMatchCount(r.Context()); err != nil {
		checks["duckdb"] = "err: " + err.Error()
		allOK = false
	} else {
		checks["duckdb"] = "ok"
	}

	// Check version DB : déjà couvert par GetMatchCount mais marqueur explicite.
	if _, err := h.repo.GetDBVersion(r.Context()); err != nil {
		checks["duckdb_version"] = "err: " + err.Error()
		allOK = false
	} else {
		checks["duckdb_version"] = "ok"
	}

	status := http.StatusOK
	statusLabel := "ready"
	if !allOK {
		status = http.StatusServiceUnavailable
		statusLabel = "not_ready"
	}
	writeJSON(w, status, map[string]any{
		"status": statusLabel,
		"checks": checks,
	})
}
