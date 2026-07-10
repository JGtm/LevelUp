// Package handlers — admin_monitoring.go : endpoints lecture du dashboard
// monitoring admin (vue d'ensemble, scheduler + historique, convergence,
// jobs récents).
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// /admin (middleware RequireAuth/RequireAdmin/NoStore hérités) et enregistre les
// 6 GET via huma.Get. Logique métier inchangée (runners injectés), seul le
// wrapping HTTP change.
//
// Routes (montées sous /api/v1/admin/monitoring/, RequireAuth+RequireAdmin+
// NoStore) :
//   - GET /monitoring/overview    : KPIs agrégés (zéro I/O DuckDB, polling 30 s ok)
//   - GET /monitoring/scheduler   : snapshot complet + historique des cycles (ring mémoire)
//   - GET /monitoring/convergence : backlog d'enrichissement par joueur (lectures seules)
//   - GET /monitoring/jobs        : jobs asynchrones récents (JobStore)
//   - GET /monitoring/perf        : agrégats de performance depuis le boot
//   - GET /monitoring/errors      : logs WARN/ERROR agrégés depuis le boot
package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
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

// PerfStatsRunner retourne les agrégats de performance depuis le boot, filtrés
// par titre (MT-05 ; implémenté par ServiceRegistry.PerfStats — expvar pur).
type PerfStatsRunner func(ctx context.Context, titleSlug string) (domain.AdminPerfStats, error)

// ErrorStatsRunner retourne les logs WARN/ERROR agrégés depuis le boot, filtrés
// par titre (MT-05 ; implémenté par ServiceRegistry.ErrorStats — collecteur mémoire).
type ErrorStatsRunner func(ctx context.Context, titleSlug string) (domain.AdminErrorStats, error)

// DetectionsRunner liste les détections PERSISTÉES avec cycle de vie (vue
// detections_latest, survit au restart), filtrées. Implémenté par
// ServiceRegistry.DetectionsReport (flush de l'ErrorCollector puis lecture).
type DetectionsRunner func(ctx context.Context, status, level, module, title string, limit int) (domain.AdminDetectionsResponse, error)

// DetectionStatusRunner statue une détection (append d'un événement de cycle de
// vie). Implémenté par ServiceRegistry.SetDetectionStatus.
type DetectionStatusRunner func(ctx context.Context, fingerprint, status, note string) error

// FreshnessRunner calcule la fraîcheur des données par joueur suivi
// (implémenté par ServiceRegistry.FreshnessReport — lectures shared par joueur).
type FreshnessRunner func(ctx context.Context, titleFilter string) (domain.AdminFreshnessResponse, error)

// ResourcesRunner agrège l'état ressources machine & process (implémenté par
// ServiceRegistry.ResourcesReport — os.Stat + expvar + runtime, pas de DuckDB).
type ResourcesRunner func(ctx context.Context) (domain.AdminResourcesResponse, error)

// AdminMonitoringHandler sert les endpoints lecture du dashboard monitoring.
type AdminMonitoringHandler struct {
	overview     MonitoringOverviewRunner
	convergence  ConvergenceReportRunner
	perf         PerfStatsRunner
	errors       ErrorStatsRunner
	detections   DetectionsRunner             // nil → section détections vide
	setDetection DetectionStatusRunner        // nil → PATCH 503
	freshness    FreshnessRunner              // nil → réponse vide
	resources    ResourcesRunner              // nil → réponse vide
	sched        *scheduler.AutoSyncScheduler // nil → scheduler indisponible
	jobs         *jobs.Store                  // nil → liste jobs vide
}

// NewAdminMonitoringHandler construit le handler. sched et jobs peuvent être
// nil (sections dégradées, jamais de panic). detections/setDetection/freshness
// peuvent être nil (dégradation propre).
func NewAdminMonitoringHandler(
	overview MonitoringOverviewRunner,
	convergence ConvergenceReportRunner,
	perf PerfStatsRunner,
	errors ErrorStatsRunner,
	detections DetectionsRunner,
	setDetection DetectionStatusRunner,
	freshness FreshnessRunner,
	resources ResourcesRunner,
	sched *scheduler.AutoSyncScheduler,
	jobStore *jobs.Store,
) *AdminMonitoringHandler {
	return &AdminMonitoringHandler{
		overview: overview, convergence: convergence, perf: perf, errors: errors,
		detections: detections, setDetection: setDetection, freshness: freshness,
		resources: resources, sched: sched, jobs: jobStore,
	}
}

// Mount enregistre les routes via Huma sur le sous-routeur chi /admin
// (middleware RequireAuth/RequireAdmin/NoStore hérités).
func (h *AdminMonitoringHandler) Mount(r chi.Router) {
	api := humacore.NewAPI(r)
	huma.Get(api, "/monitoring/overview", h.handleGetOverview)
	huma.Get(api, "/monitoring/scheduler", h.handleGetScheduler)
	huma.Get(api, "/monitoring/convergence", h.handleGetConvergence)
	huma.Get(api, "/monitoring/jobs", h.handleGetJobs)
	huma.Get(api, "/monitoring/perf", h.handleGetPerf)
	huma.Get(api, "/monitoring/errors", h.handleGetErrors)
	huma.Get(api, "/monitoring/detections", h.handleGetDetections)
	huma.Patch(api, "/monitoring/detections/{fingerprint}", h.handlePatchDetection)
	huma.Get(api, "/monitoring/freshness", h.handleGetFreshness)
	huma.Get(api, "/monitoring/resources", h.handleGetResources)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// titleInput : ?title= optionnel (fallback titre par défaut via titleOrDefaultSlug).
type titleInput struct {
	Title string `query:"title"`
}

// jobsInput : ?limit= optionnel (défaut 20, max 50) — pris en STRING pour
// reproduire le contrat d'origine (limit non numérique ou <=0 ignoré, défaut 20),
// PAS le 422 de validation Huma qu'un `int` produirait.
type jobsInput struct {
	Limit string `query:"limit"`
}

type adminOverviewOutput struct {
	Body domain.AdminMonitoringOverview
}
type adminConvergenceOutput struct {
	Body domain.AdminConvergenceReport
}
type adminPerfOutput struct{ Body domain.AdminPerfStats }
type adminErrorsOutput struct{ Body domain.AdminErrorStats }
type adminSchedulerOutput struct {
	Body AdminSchedulerStatusResponse
}
type adminJobsOutput struct{ Body AdminJobsResponse }

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

// detectionsInput : filtres optionnels de GET /monitoring/detections. limit en
// STRING (contrat souple : non numérique/<=0 → défaut store), pas de 422 Huma.
type detectionsInput struct {
	Status string `query:"status"`
	Level  string `query:"level"`
	Module string `query:"module"`
	Title  string `query:"title"`
	Limit  string `query:"limit"`
}
type adminDetectionsOutput struct {
	Body domain.AdminDetectionsResponse
}

// detectionPatchInput : {fingerprint} + body {status, note?}.
type detectionPatchInput struct {
	Fingerprint string `path:"fingerprint"`
	Body        struct {
		Status string `json:"status"`
		Note   string `json:"note,omitempty"`
	}
}

// AdminDetectionPatchResponse : accusé d'un changement de statut de détection.
type AdminDetectionPatchResponse struct {
	Fingerprint string `json:"fingerprint"`
	Status      string `json:"status"`
	OK          bool   `json:"ok"`
}
type detectionPatchOutput struct{ Body AdminDetectionPatchResponse }

type adminFreshnessOutput struct{ Body domain.AdminFreshnessResponse }

type adminResourcesOutput struct{ Body domain.AdminResourcesResponse }

// titleOrDefaultSlug lit ?title= avec fallback sur le titre par défaut.
func titleOrDefaultSlug(title string) string {
	if title != "" {
		return title
	}
	return titlePkg.DefaultSlug
}

// ─── Endpoints ───────────────────────────────────────────────────────────────

// handleGetOverview retourne les KPIs agrégés.
// GET /admin/monitoring/overview?title={slug}.
func (h *AdminMonitoringHandler) handleGetOverview(ctx context.Context, in *titleInput) (*adminOverviewOutput, error) {
	titleSlug := titleOrDefaultSlug(in.Title)
	resp, err := h.overview(ctx, titleSlug)
	if err != nil {
		slog.ErrorContext(ctx, "admin_monitoring: overview failed", "title", titleSlug, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "monitoring_overview_error",
			"Impossible d'agréger la vue d'ensemble monitoring.")
	}
	return &adminOverviewOutput{Body: resp}, nil
}

// handleGetScheduler retourne le snapshot scheduler + l'historique des cycles.
// GET /admin/monitoring/scheduler.
func (h *AdminMonitoringHandler) handleGetScheduler(_ context.Context, _ *struct{}) (*adminSchedulerOutput, error) {
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
	return &adminSchedulerOutput{Body: resp}, nil
}

// handleGetConvergence retourne le backlog de convergence par joueur.
// GET /admin/monitoring/convergence?title={slug}.
func (h *AdminMonitoringHandler) handleGetConvergence(ctx context.Context, in *titleInput) (*adminConvergenceOutput, error) {
	titleSlug := titleOrDefaultSlug(in.Title)
	resp, err := h.convergence(ctx, titleSlug)
	if err != nil {
		slog.ErrorContext(ctx, "admin_monitoring: convergence failed", "title", titleSlug, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "monitoring_convergence_error",
			"Impossible de calculer le backlog de convergence.")
	}
	return &adminConvergenceOutput{Body: resp}, nil
}

// handleGetJobs retourne les jobs asynchrones récents.
// GET /admin/monitoring/jobs?limit=20 (max 50).
func (h *AdminMonitoringHandler) handleGetJobs(_ context.Context, in *jobsInput) (*adminJobsOutput, error) {
	limit := 20
	if in.Limit != "" {
		if n, err := strconv.Atoi(in.Limit); err == nil && n > 0 {
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
	return &adminJobsOutput{Body: resp}, nil
}

// handleGetPerf retourne les agrégats de performance depuis le boot.
// GET /admin/monitoring/perf.
func (h *AdminMonitoringHandler) handleGetPerf(ctx context.Context, in *titleInput) (*adminPerfOutput, error) {
	resp, err := h.perf(ctx, titleOrDefaultSlug(in.Title))
	if err != nil {
		slog.ErrorContext(ctx, "admin_monitoring: perf failed", "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "monitoring_perf_error",
			"Impossible d'agréger les statistiques de performance.")
	}
	return &adminPerfOutput{Body: resp}, nil
}

// handleGetErrors retourne les logs WARN/ERROR agrégés depuis le boot.
// GET /admin/monitoring/errors.
func (h *AdminMonitoringHandler) handleGetErrors(ctx context.Context, in *titleInput) (*adminErrorsOutput, error) {
	resp, err := h.errors(ctx, titleOrDefaultSlug(in.Title))
	if err != nil {
		slog.ErrorContext(ctx, "admin_monitoring: errors failed", "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "monitoring_errors_error",
			"Impossible d'agréger les erreurs récentes.")
	}
	return &adminErrorsOutput{Body: resp}, nil
}

// handleGetDetections retourne les détections persistées avec leur cycle de vie.
// GET /admin/monitoring/detections?status=&level=&module=&title=&limit=.
func (h *AdminMonitoringHandler) handleGetDetections(ctx context.Context, in *detectionsInput) (*adminDetectionsOutput, error) {
	if h.detections == nil {
		return &adminDetectionsOutput{Body: domain.AdminDetectionsResponse{
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			Detections:  []domain.MonitoringDetection{},
		}}, nil
	}
	limit := 0
	if in.Limit != "" {
		if n, err := strconv.Atoi(in.Limit); err == nil && n > 0 {
			limit = n
		}
	}
	resp, err := h.detections(ctx, in.Status, in.Level, in.Module, in.Title, limit)
	if err != nil {
		slog.ErrorContext(ctx, "admin_monitoring: detections failed", "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "monitoring_detections_error",
			"Impossible de lister les détections.")
	}
	return &adminDetectionsOutput{Body: resp}, nil
}

// handlePatchDetection statue une détection (Reconnaître / Sourdine / Résoudre).
// PATCH /admin/monitoring/detections/{fingerprint} — body {status, note?}.
func (h *AdminMonitoringHandler) handlePatchDetection(ctx context.Context, in *detectionPatchInput) (*detectionPatchOutput, error) {
	if h.setDetection == nil {
		return nil, humacore.NewError(http.StatusServiceUnavailable, "monitoring_store_unavailable",
			"Le store monitoring n'est pas disponible.")
	}
	if !domain.IsValidDetectionStatus(in.Body.Status) {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_status",
			"Statut invalide (open, acked, muted ou resolved).")
	}
	if err := h.setDetection(ctx, in.Fingerprint, in.Body.Status, in.Body.Note); err != nil {
		slog.ErrorContext(ctx, "admin_monitoring: patch detection failed",
			"fingerprint", in.Fingerprint, "status", in.Body.Status, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "monitoring_detection_patch_error",
			"Impossible de mettre à jour la détection.")
	}
	slog.InfoContext(ctx, "admin_monitoring: détection statuée",
		"fingerprint", in.Fingerprint, "status", in.Body.Status)
	return &detectionPatchOutput{Body: AdminDetectionPatchResponse{
		Fingerprint: in.Fingerprint, Status: in.Body.Status, OK: true,
	}}, nil
}

// handleGetFreshness retourne la fraîcheur des données par joueur suivi.
// GET /admin/monitoring/freshness?title= (vide = tous les titres actifs).
func (h *AdminMonitoringHandler) handleGetFreshness(ctx context.Context, in *titleInput) (*adminFreshnessOutput, error) {
	if h.freshness == nil {
		return &adminFreshnessOutput{Body: domain.AdminFreshnessResponse{
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			Titles:      []domain.TitleFreshnessReport{},
		}}, nil
	}
	// PAS de fallback titre par défaut : vide = tous les titres actifs (le
	// panneau État affiche halo_infinite ET halo_5 côte à côte).
	resp, err := h.freshness(ctx, in.Title)
	if err != nil {
		slog.ErrorContext(ctx, "admin_monitoring: freshness failed", "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "monitoring_freshness_error",
			"Impossible de calculer la fraîcheur des données.")
	}
	return &adminFreshnessOutput{Body: resp}, nil
}

// handleGetResources retourne l'état ressources machine & process (A5).
// GET /admin/monitoring/resources.
func (h *AdminMonitoringHandler) handleGetResources(ctx context.Context, _ *struct{}) (*adminResourcesOutput, error) {
	if h.resources == nil {
		return &adminResourcesOutput{Body: domain.AdminResourcesResponse{
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			Databases:   []domain.ResourceDBFile{},
		}}, nil
	}
	resp, err := h.resources(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "admin_monitoring: resources failed", "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "monitoring_resources_error",
			"Impossible d'agréger l'état des ressources.")
	}
	return &adminResourcesOutput{Body: resp}, nil
}
