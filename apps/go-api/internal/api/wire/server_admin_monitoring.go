// Package api — server_admin_monitoring.go : montage des routes du dashboard
// monitoring admin. Extrait de server.go (déjà >1000 L) — server.go n'ajoute
// qu'un appel à MountAdminMonitoringRoutes dans le groupe /admin existant.
package wire

import (
	"context"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/observability/logging"
	jobs_platform "levelup/go-api/internal/platform/jobs"
	"levelup/go-api/internal/scheduler"
)

// MountAdminMonitoringRoutes monte les endpoints monitoring + actions sous le
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
func MountAdminMonitoringRoutes(
	r chi.Router,
	reg *ServiceRegistry,
	sched *scheduler.AutoSyncScheduler,
	jobStore *jobs_platform.Store,
	serverCtx context.Context,
) {
	reg.WithAutoSyncScheduler(sched)

	monitoringH := handlers.NewAdminMonitoringHandler(
		reg.MonitoringOverview, reg.ConvergenceReport, reg.PerfStats, reg.ErrorStats, sched, jobStore)
	// 6 GET /monitoring/* migrés vers Huma (Phase 3b), NoStore.
	monitoringH.Mount(r.With(middleware.NoStore))

	// Couverture de résolution d'arme (registre vs weapon_labels vs non résolu),
	// par titre. GET /monitoring/weapon-coverage?title=.
	coverageH := handlers.NewAdminWeaponCoverageHandler(reg.WeaponCoverage)
	coverageH.Mount(r.With(middleware.NoStore))

	actionsH := handlers.NewAdminActionsHandler(
		reg.RunDataHealthNow, sched, jobStore, serverCtx)
	actionsH.Mount(r) // POST /actions/data-health/run, /actions/auto-sync/run

	// Qualité données : compteurs/listes d'inconnus + actions de résolution
	// (backfill registry names, traductions metadata) — Phase 2.
	// Mount gère le Cache-Control:no-store des 2 GET en interne (champ header).
	dqH := handlers.NewAdminDataQualityHandler(
		reg.DataQualityCounts, reg.DataQualityIssues,
		reg.RunRegistryNamesBackfill, reg.ResolveModeTranslation, reg.ResolveAssetTranslation,
		reg.RunCatalogRefresh, reg.RunLyingBitsReset,
		ErrActionBusy)
	dqH.Mount(r)

	// Convergence ciblée d'un joueur (job asynchrone, claim SyncGate).
	convH := handlers.NewAdminConvergenceActionHandler(
		reg.RunPlayerConvergence, jobStore, serverCtx, ErrSyncInFlight)
	convH.Mount(r) // POST /actions/convergence/run

	// Drain DiscoveryUGC (job asynchrone — réseau, rate-limité). Complète le
	// catalog/refresh zéro-réseau en hydratant les assets absents de match_registry.
	drainH := handlers.NewAdminCatalogDrainHandler(reg.RunCatalogUGCDrain, jobStore, serverCtx)
	drainH.Mount(r) // POST /actions/catalog/ugc-drain

	// Viewer de logs : modules + tail filtré (lecture par la fin chunkée).
	logsH := handlers.NewAdminLogsHandler(logging.LoadConfig(reg.cfg.RepoRoot).LogsDir)
	logsH.Mount(r.With(middleware.NoStore)) // GET /monitoring/logs/{modules,tail}
}
