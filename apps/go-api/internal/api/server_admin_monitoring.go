// Package api — server_admin_monitoring.go : montage des routes du dashboard
// monitoring admin. Extrait de server.go (déjà >1000 L) — server.go n'ajoute
// qu'un appel à mountAdminMonitoringRoutes dans le groupe /admin existant.
package api

import (
	"context"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/observability/logging"
	jobs_platform "levelup/go-api/internal/platform/jobs"
	"levelup/go-api/internal/scheduler"
)

// mountAdminMonitoringRoutes monte les endpoints monitoring + actions sous le
// groupe /admin (déjà gardé RequireAuth + RequireAdmin). NoStore sur les GET :
// état courant, jamais de cache.
//
//   - sched peut être nil (watcher/scheduler désactivés) : sections dégradées.
//   - serverCtx : parent des goroutines de jobs (annulé au shutdown, avant
//     duckdb.CloseAll) — pattern syncH.WithServerContext.
//
// Le HealthScheduler n'est PAS passé ici : il est créé par main.go APRÈS
// NewRouter et attaché via reg.WithHealthScheduler (lecture lazy par les
// runners à chaque requête — ordre de boot sûr).
func mountAdminMonitoringRoutes(
	r chi.Router,
	reg *ServiceRegistry,
	sched *scheduler.AutoSyncScheduler,
	jobStore *jobs_platform.Store,
	serverCtx context.Context,
) {
	reg.WithAutoSyncScheduler(sched)

	monitoringH := handlers.NewAdminMonitoringHandler(
		reg.MonitoringOverview, reg.ConvergenceReport, reg.PerfStats, reg.ErrorStats, sched, jobStore)
	r.With(middleware.NoStore).Get("/monitoring/overview", monitoringH.GetOverview)
	r.With(middleware.NoStore).Get("/monitoring/scheduler", monitoringH.GetScheduler)
	r.With(middleware.NoStore).Get("/monitoring/convergence", monitoringH.GetConvergence)
	r.With(middleware.NoStore).Get("/monitoring/jobs", monitoringH.GetJobs)
	r.With(middleware.NoStore).Get("/monitoring/perf", monitoringH.GetPerf)
	r.With(middleware.NoStore).Get("/monitoring/errors", monitoringH.GetErrors)

	actionsH := handlers.NewAdminActionsHandler(
		reg.RunDataHealthNow, sched, jobStore, serverCtx)
	r.Post("/actions/data-health/run", actionsH.RunDataHealth)
	r.Post("/actions/auto-sync/run", actionsH.RunSyncCycle)

	// Qualité données : compteurs/listes d'inconnus + actions de résolution
	// (backfill registry names, traductions metadata) — Phase 2.
	dqH := handlers.NewAdminDataQualityHandler(
		reg.DataQualityCounts, reg.DataQualityIssues,
		reg.RunRegistryNamesBackfill, reg.ResolveModeTranslation, reg.ResolveAssetTranslation,
		reg.RunCatalogRefresh, reg.RunLyingBitsReset,
		ErrActionBusy)
	r.With(middleware.NoStore).Get("/monitoring/data-quality", dqH.GetCounts)
	r.With(middleware.NoStore).Get("/monitoring/data-quality/issues", dqH.GetIssues)
	r.Post("/actions/registry-names/backfill", dqH.RunRegistryNamesBackfill)
	r.Post("/actions/translations/mode", dqH.ResolveModeTranslation)
	r.Post("/actions/translations/asset", dqH.ResolveAssetTranslation)
	r.Post("/actions/catalog/refresh", dqH.RunCatalogRefresh)
	r.Post("/actions/lying-bits/reset", dqH.RunLyingBitsReset)

	// Convergence ciblée d'un joueur (job asynchrone, claim SyncGate).
	convH := handlers.NewAdminConvergenceActionHandler(
		reg.RunPlayerConvergence, jobStore, serverCtx, ErrSyncInFlight)
	r.Post("/actions/convergence/run", convH.Run)

	// Drain DiscoveryUGC (job asynchrone — réseau, rate-limité). Complète le
	// catalog/refresh zéro-réseau en hydratant les assets absents de match_registry.
	drainH := handlers.NewAdminCatalogDrainHandler(reg.RunCatalogUGCDrain, jobStore, serverCtx)
	r.Post("/actions/catalog/ugc-drain", drainH.Run)

	// Viewer de logs : modules + tail filtré (lecture par la fin chunkée).
	// LogsDir résolu par logging.LoadConfig (respecte LEVELUP_LOGS_DIR) — pas
	// PathResolver : les logs ne vivent pas sous data/.
	logsH := handlers.NewAdminLogsHandler(logging.LoadConfig(reg.cfg.RepoRoot).LogsDir)
	r.With(middleware.NoStore).Get("/monitoring/logs/modules", logsH.GetModules)
	r.With(middleware.NoStore).Get("/monitoring/logs/tail", logsH.GetTail)
}
