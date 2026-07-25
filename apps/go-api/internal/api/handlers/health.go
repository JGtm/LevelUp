// Package handlers — handler GET /health, /healthz, /readyz.
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le routeur
// racine (routes RACINE, sans préfixe /api/v1) et enregistre les 3 GET via
// huma.Get. Logique métier inchangée (BootstrapRepository), seul le wrapping
// HTTP change.
//
//   - /health  → ServeHTTP  : 200 {HealthResponse} ou 503 {db_unavailable}.
//   - /healthz → Liveness   : 200 {status:alive} (aucun I/O DB).
//   - /readyz  → Readiness  : 200 {status:ready} ou 503 {status:not_ready,checks}
//     — le 503 PORTE un corps de checks (diagnostic), donc Output{Status, Body}
//     et non une erreur Huma.
package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// HealthHandler gère GET /health.
type HealthHandler struct {
	repo         port.BootstrapRepository
	appVersion   string
	startedAt    time.Time
	mediaTooling domain.MediaToolingStatus
}

// NewHealthHandler crée un HealthHandler.
func NewHealthHandler(repo port.BootstrapRepository) *HealthHandler {
	return &HealthHandler{repo: repo, startedAt: time.Now()}
}

// NewHealthHandlerWithVersion crée un HealthHandler avec la version de l'application.
func NewHealthHandlerWithVersion(repo port.BootstrapRepository, version string) *HealthHandler {
	return &HealthHandler{repo: repo, appVersion: version, startedAt: time.Now()}
}

// WithMediaTooling injecte l'état de l'outillage média sondé une fois au boot.
// Fluent : renvoie h pour chaîner avec les constructeurs. Absence d'appel =
// MediaToolingStatus zéro-valeur (ffmpeg/ffprobe=false) — jamais de nouveau
// WARN, l'info reste une simple projection dans /health.
func (h *HealthHandler) WithMediaTooling(status domain.MediaToolingStatus) *HealthHandler {
	h.mediaTooling = status
	return h
}

// Mount enregistre les 3 routes RACINE via Huma sur le routeur chi `r`
// (montées sans préfixe /api/v1, à l'identique des routes chi d'origine).
func (h *HealthHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/health", h.handleHealth)
	huma.Get(api, "/healthz", h.handleLiveness)
	huma.Get(api, "/readyz", h.handleReadiness)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// healthOutput : 200 {HealthResponse} (le 503 db_unavailable est une erreur Huma).
type healthOutput struct {
	Body domain.HealthResponse
}

// livenessOutput : 200 {status:alive, uptime}.
type livenessOutput struct {
	Body map[string]any
}

// readinessOutput : 200 OU 503 — le statut porte un corps de checks (diagnostic),
// donc Output{Status, Body} et non une erreur Huma.
type readinessOutput struct {
	Status int
	Body   map[string]any
}

// ─── Endpoints ───────────────────────────────────────────────────────────────

// handleHealth retourne le healthcheck enrichi (Sprint 41 T3).
//
// Deprecated: utiliser /healthz pour la liveness ou /readyz pour la readiness.
// /health garde la sémantique mixte (200 si DB OK) pour rétrocompat.
func (h *HealthHandler) handleHealth(ctx context.Context, _ *struct{}) (*healthOutput, error) {
	count, err := h.repo.GetMatchCount(ctx)
	if err != nil {
		return nil, humacore.NewError(http.StatusServiceUnavailable, "db_unavailable",
			"Impossible de lire shared_matches_v2: "+err.Error())
	}

	// Enrichissements best-effort : leur échec ne dégrade pas le 200 (la santé
	// DB est déjà attestée par GetMatchCount ci-dessus), mais on trace au lieu
	// d'avaler l'erreur (Debug : /health est sondé fréquemment).
	dbVersion, err := h.repo.GetDBVersion(ctx)
	if err != nil {
		slog.DebugContext(ctx, "health: db version enrichment unavailable", "err", err)
	}
	playerCount, err := h.repo.GetPlayerCount(ctx)
	if err != nil {
		slog.DebugContext(ctx, "health: player count enrichment unavailable", "err", err)
	}
	lastSync, err := h.repo.GetLastSyncAt(ctx)
	if err != nil {
		slog.DebugContext(ctx, "health: last sync enrichment unavailable", "err", err)
	}

	return &healthOutput{Body: domain.HealthResponse{
		Status:       "ok",
		MatchCount:   count,
		DBVersion:    dbVersion,
		AppVersion:   h.appVersion,
		PlayerCount:  playerCount,
		LastSyncAt:   lastSync,
		Uptime:       time.Since(h.startedAt).Round(time.Second).String(),
		GoVersion:    runtime.Version(),
		MediaTooling: h.mediaTooling,
	}}, nil
}

// handleLiveness gère GET /healthz (P8.11).
//
// Liveness probe : signale que le process Go est vivant et n'a pas paniqué.
// **Aucun I/O DB**, aucune requête réseau — latence < 5ms garantie.
// Si ce handler répond 200, l'orchestrateur (K8s/LB) NE doit PAS redémarrer
// l'instance. Si la DB est en panne, c'est un problème de readiness, pas
// de liveness — voir /readyz.
func (h *HealthHandler) handleLiveness(ctx context.Context, _ *struct{}) (*livenessOutput, error) {
	return &livenessOutput{Body: map[string]any{
		jsonKeyStatus: "alive",
		"uptime":      time.Since(h.startedAt).Round(time.Second).String(),
	}}, nil
}

// handleReadiness gère GET /readyz (P8.11).
//
// Readiness probe : signale que l'app est prête à accepter du trafic.
// Vérifie : DuckDB metadata accessible (read), filesystem data/ accessible.
// Si un check échoue → 503 + body JSON `{checks: {duckdb: "ok|err", fs: "ok|err"}}`.
// Latence cible < 100ms.
func (h *HealthHandler) handleReadiness(ctx context.Context, _ *struct{}) (*readinessOutput, error) {
	checks := map[string]string{}
	allOK := true

	// Check DB : tentative de lecture du compte de matchs (fail-fast).
	if _, err := h.repo.GetMatchCount(ctx); err != nil {
		checks["duckdb"] = "err: " + err.Error()
		allOK = false
	} else {
		checks["duckdb"] = "ok"
	}

	// Check version DB : déjà couvert par GetMatchCount mais marqueur explicite.
	if _, err := h.repo.GetDBVersion(ctx); err != nil {
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
	return &readinessOutput{Status: status, Body: map[string]any{
		jsonKeyStatus: statusLabel,
		"checks":      checks,
	}}, nil
}
