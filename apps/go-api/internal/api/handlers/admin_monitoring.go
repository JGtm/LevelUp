// Package handlers — admin_monitoring.go : endpoints lecture du dashboard
// monitoring admin (vue d'ensemble, scheduler + historique, convergence,
// jobs récents).
//
// Routes (montées sous /api/v1/admin/monitoring/, RequireAuth+RequireAdmin+
// NoStore) :
//   - GET /overview    : KPIs agrégés (zéro I/O DuckDB, polling 30 s ok)
//   - GET /scheduler   : snapshot complet + historique des cycles (ring mémoire)
//   - GET /convergence : backlog d'enrichissement par joueur (lectures seules)
//   - GET /jobs        : jobs asynchrones récents (JobStore)
package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/jobs"
	"levelup/go-api/internal/scheduler"
)

// MonitoringOverviewRunner retourne les KPIs agrégés (implémenté par
// ServiceRegistry.MonitoringOverview — injecté pour éviter le cycle d'import).
type MonitoringOverviewRunner func(ctx context.Context, titleSlug string) (domain.AdminMonitoringOverview, error)

// ConvergenceReportRunner retourne le backlog de convergence par joueur
// (implémenté par ServiceRegistry.ConvergenceReport).
type ConvergenceReportRunner func(ctx context.Context, titleSlug string) (domain.AdminConvergenceReport, error)

// PerfStatsRunner retourne les agrégats de performance depuis le boot
// (implémenté par ServiceRegistry.PerfStats — expvar pur).
type PerfStatsRunner func(ctx context.Context) (domain.AdminPerfStats, error)

// AdminMonitoringHandler sert les endpoints lecture du dashboard monitoring.
type AdminMonitoringHandler struct {
	overview    MonitoringOverviewRunner
	convergence ConvergenceReportRunner
	perf        PerfStatsRunner
	sched       *scheduler.AutoSyncScheduler // nil → scheduler indisponible
	jobs        *jobs.Store                  // nil → liste jobs vide
}

// NewAdminMonitoringHandler construit le handler. sched et jobs peuvent être
// nil (sections dégradées, jamais de panic).
func NewAdminMonitoringHandler(
	overview MonitoringOverviewRunner,
	convergence ConvergenceReportRunner,
	perf PerfStatsRunner,
	sched *scheduler.AutoSyncScheduler,
	jobStore *jobs.Store,
) *AdminMonitoringHandler {
	return &AdminMonitoringHandler{overview: overview, convergence: convergence, perf: perf, sched: sched, jobs: jobStore}
}

// GetPerf retourne les agrégats de performance depuis le boot.
// GET /admin/monitoring/perf.
func (h *AdminMonitoringHandler) GetPerf(w http.ResponseWriter, r *http.Request) {
	resp, err := h.perf(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "admin_monitoring: perf failed", "err", err)
		writeError(r.Context(), w, http.StatusInternalServerError, "monitoring_perf_error",
			"Impossible d'agréger les statistiques de performance.")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// AdminSchedulerStatusResponse est la réponse de GET /monitoring/scheduler.
// Réutilise les types JSON-contractés du package scheduler (déjà exposés par
// /_diag/auto-sync/snapshot) plutôt que de les dupliquer en domain.
type AdminSchedulerStatusResponse struct {
	Available bool                         `json:"available"`
	Snapshot  *scheduler.SchedulerSnapshot `json:"snapshot,omitempty"`
	// History : cycles depuis le boot, plus récent en premier (ring mémoire
	// 48 entrées — l'historique long terme vit dans logs/scheduler.log).
	History          []scheduler.CycleRecord `json:"history"`
	HistorySinceBoot bool                    `json:"history_since_boot"`
	// ZeroInsertWarnThreshold : seuil d'alerte consecutive_zero_inserts,
	// exposé pour que le front n'ait pas à dupliquer la constante.
	ZeroInsertWarnThreshold int `json:"zero_insert_warn_threshold"`
}

// AdminJobsResponse est la réponse de GET /monitoring/jobs.
type AdminJobsResponse struct {
	GeneratedAt string                   `json:"generated_at"`
	Jobs        []*domain.AsyncJobStatus `json:"jobs"`
}

// titleOrDefault lit ?title= avec fallback sur le titre par défaut.
func titleOrDefault(r *http.Request) string {
	if t := r.URL.Query().Get("title"); t != "" {
		return t
	}
	return titlePkg.DefaultSlug
}

// GetOverview retourne les KPIs agrégés.
// GET /admin/monitoring/overview?title={slug}.
func (h *AdminMonitoringHandler) GetOverview(w http.ResponseWriter, r *http.Request) {
	titleSlug := titleOrDefault(r)
	resp, err := h.overview(r.Context(), titleSlug)
	if err != nil {
		slog.ErrorContext(r.Context(), "admin_monitoring: overview failed", "title", titleSlug, "err", err)
		writeError(r.Context(), w, http.StatusInternalServerError, "monitoring_overview_error",
			"Impossible d'agréger la vue d'ensemble monitoring.")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetScheduler retourne le snapshot scheduler + l'historique des cycles.
// GET /admin/monitoring/scheduler.
func (h *AdminMonitoringHandler) GetScheduler(w http.ResponseWriter, r *http.Request) {
	resp := AdminSchedulerStatusResponse{
		History:                 []scheduler.CycleRecord{},
		HistorySinceBoot:        true,
		ZeroInsertWarnThreshold: scheduler.ConsecutiveZeroInsertWarnThreshold,
	}
	if h.sched != nil {
		snap := h.sched.Snapshot()
		resp.Available = true
		resp.Snapshot = &snap
		resp.History = h.sched.History()
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetConvergence retourne le backlog de convergence par joueur.
// GET /admin/monitoring/convergence?title={slug}.
func (h *AdminMonitoringHandler) GetConvergence(w http.ResponseWriter, r *http.Request) {
	titleSlug := titleOrDefault(r)
	resp, err := h.convergence(r.Context(), titleSlug)
	if err != nil {
		slog.ErrorContext(r.Context(), "admin_monitoring: convergence failed", "title", titleSlug, "err", err)
		writeError(r.Context(), w, http.StatusInternalServerError, "monitoring_convergence_error",
			"Impossible de calculer le backlog de convergence.")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetJobs retourne les jobs asynchrones récents.
// GET /admin/monitoring/jobs?limit=20 (max 50).
func (h *AdminMonitoringHandler) GetJobs(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 50 {
		limit = 50
	}
	resp := AdminJobsResponse{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Jobs:        []*domain.AsyncJobStatus{},
	}
	if h.jobs != nil {
		resp.Jobs = h.jobs.List(limit)
	}
	writeJSON(w, http.StatusOK, resp)
}
