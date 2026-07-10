// Package api — registry_monitoring.go : runners du dashboard monitoring
// admin (vue d'ensemble, convergence, action data-health).
//
// Vue d'ensemble : ZÉRO I/O DuckDB — agrège des états mémoire (snapshot
// scheduler, dernier data health check, JobStore, token store fichiers,
// gauges expvar) pour supporter un polling 30 s sans coût.
//
// Convergence : lectures seules best-effort par joueur (pattern
// registry_invariants : une DB inaccessible pose CheckError sur le joueur,
// jamais d'échec global).
//
// Logs : module explicite « monitoring » → logs/monitoring.log (le package
// api routerait sinon vers http.log — pattern invariantsLog).
package wire

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/scheduler"
	sync_pkg "levelup/go-api/internal/sync"
)

// monitoringLog : logger taggé module=monitoring (fichier logs/monitoring.log).
var monitoringLog = slog.With("module", "monitoring")

// WithAutoSyncScheduler attache le scheduler auto-sync au registry pour les
// agrégats monitoring (overview). Nil possible : la section scheduler de
// l'overview est alors marquée indisponible.
func (r *ServiceRegistry) WithAutoSyncScheduler(s *scheduler.AutoSyncScheduler) *ServiceRegistry {
	r.autoSyncScheduler = s
	return r
}

// WithHealthScheduler attache le HealthScheduler (audit data health 24h) au
// registry — lu lazily à chaque requête, donc câblable par main.go APRÈS
// NewRouter. Nil possible : section data_health absente + action 503.
func (r *ServiceRegistry) WithHealthScheduler(h *scheduler.HealthScheduler) *ServiceRegistry {
	r.healthScheduler = h
	return r
}

// MonitoringOverview agrège les KPIs du dashboard monitoring.
// Best-effort : chaque section dégrade indépendamment (champ *_error ou nil).
func (r *ServiceRegistry) MonitoringOverview(ctx context.Context, titleSlug string) (domain.AdminMonitoringOverview, error) {
	resp := domain.AdminMonitoringOverview{
		TitleSlug:   titleSlug,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Server: domain.MonitoringServerInfo{
			UptimeS:   int64(time.Since(r.startedAt).Seconds()),
			StartedAt: r.startedAt.UTC().Format(time.RFC3339),
			Version:   r.cfg.AppVersion,
		},
		Scheduler: r.monitoringSchedulerSummary(),
		Jobs:      r.monitoringJobsSummary(),
		Invariants: domain.MonitoringInvariantsSummary{
			RunsTotal: observability.LoadCounter("invariants_runs_total"),
			FailLast:  observability.LoadCounter("invariants_fail_last"),
			WarnLast:  observability.LoadCounter("invariants_warn_last"),
		},
		Snapshot: monitoringSnapshotSummary(titleSlug),
	}
	if res, at := r.lastDataHealth(); res != nil {
		resp.DataHealth = toMonitoringDataHealth(res, at)
	}
	r.fillMonitoringTokens(ctx, titleSlug, &resp)
	return resp, nil
}

// monitoringSnapshotSummary lit l'état du producteur de snapshot immuable depuis les
// gauges + cumuls expvar titrés (zéro I/O DuckDB — posés en fin de cycle par le
// SnapshotCutter). 0 partout = aucun cut depuis le boot / titre sans snapshot.
func monitoringSnapshotSummary(titleSlug string) domain.MonitoringSnapshotSummary {
	return domain.MonitoringSnapshotSummary{
		Version:                 observability.LoadCounterT(titleSlug, "snapshot_version"),
		ReadyMatchCount:         observability.LoadCounterT(titleSlug, "snapshot_ready_match_count"),
		PendingTotal:            observability.LoadCounterT(titleSlug, "snapshot_pending_total"),
		PartialTotal:            observability.LoadCounterT(titleSlug, "snapshot_partial_total"),
		PendingOldestAgeSeconds: observability.LoadCounterT(titleSlug, "snapshot_pending_oldest_age_seconds"),
		CutsProduced:            observability.LoadCounterT(titleSlug, "snapshot_cut_produced_total"),
		CutFailures:             observability.LoadCounterT(titleSlug, "snapshot_cut_failures_total"),
		CutNoop:                 observability.LoadCounterT(titleSlug, "snapshot_cut_noop_total"),
		ReadsServed:             observability.LoadCounterT(titleSlug, "snapshot_read_served_total"),
		ReadsFallback:           observability.LoadCounterT(titleSlug, "snapshot_read_live_fallback_total"),
	}
}

// monitoringSchedulerSummary résume le snapshot scheduler (sans le détail
// par joueur — exposé par /admin/monitoring/scheduler).
func (r *ServiceRegistry) monitoringSchedulerSummary() domain.MonitoringSchedulerSummary {
	if r.autoSyncScheduler == nil {
		return domain.MonitoringSchedulerSummary{Available: false}
	}
	snap := r.autoSyncScheduler.Snapshot()
	out := domain.MonitoringSchedulerSummary{
		Available:       true,
		IntervalMinutes: snap.IntervalMinutes,
		PoolSize:        snap.PoolSize,
		InFlightClaims:  len(snap.Gate.Claims),
	}
	if !snap.LastCycleAt.IsZero() {
		out.LastCycleAt = snap.LastCycleAt.UTC().Format(time.RFC3339)
	}
	if res := snap.LastCycleResult; res != nil {
		out.LastTotal = res.Total
		out.LastSynced = res.Synced
		out.LastSkipped = res.Skipped
		out.LastFailed = res.Failed
		out.LastDurationMs = res.Duration.Milliseconds()
	}
	for _, p := range snap.Players {
		if p.ConsecutiveZeroInserts >= scheduler.ConsecutiveZeroInsertWarnThreshold {
			out.ZeroInsertAlerts++
		}
	}
	return out
}

// monitoringJobsSummary résume l'activité du JobStore (8 jobs récents).
func (r *ServiceRegistry) monitoringJobsSummary() domain.MonitoringJobsSummary {
	out := domain.MonitoringJobsSummary{Recent: []domain.AsyncJobStatus{}}
	if r.jobStore == nil {
		return out
	}
	jobs := r.jobStore.List(20)
	for i, j := range jobs {
		if !j.IsTerminal() {
			out.ActiveCount++
		}
		if i < 8 {
			out.Recent = append(out.Recent, *j)
		}
	}
	return out
}

// lastDataHealth retourne le dernier audit complet (nil si scheduler non câblé
// ou jamais couru).
func (r *ServiceRegistry) lastDataHealth() (*scheduler.DataHealthCheckResult, time.Time) {
	if r.healthScheduler == nil {
		return nil, time.Time{}
	}
	return r.healthScheduler.LastResult()
}

// toMonitoringDataHealth mappe le résultat scheduler vers le DTO domain.
func toMonitoringDataHealth(res *scheduler.DataHealthCheckResult, at time.Time) *domain.MonitoringDataHealth {
	return &domain.MonitoringDataHealth{
		RanAt:                at.UTC().Format(time.RFC3339),
		UUIDsRawCount:        res.UUIDsRawCount,
		LyingBitsEvents:      res.LyingBitsEvents,
		LyingBitsWeaponKills: res.LyingBitsWeaponKills,
		OrphanXUIDs:          res.OrphanXUIDs,
		GarbageBannerURLs:    res.GarbageBannerURLs,
		WarningsTotal:        res.WarningsTotal,
		DurationMs:           res.Duration.Milliseconds(),
	}
}

// fillMonitoringTokens agrège la santé tokens (réutilise reg.TokenHealth,
// lectures fichiers sans refresh réseau). Best-effort : erreur → TokensError.
func (r *ServiceRegistry) fillMonitoringTokens(ctx context.Context, titleSlug string, resp *domain.AdminMonitoringOverview) {
	health, err := r.TokenHealth(ctx, titleSlug)
	if err != nil {
		resp.TokensError = err.Error()
		return
	}
	if health.StoreUnavailable {
		resp.TokensError = "token store non câblé (mode legacy)"
		return
	}
	sum := domain.MonitoringTokensSummary{Players: len(health.Players)}
	for _, p := range health.Players {
		switch p.Refresh {
		case "ok":
			sum.OK++
		case "expiring":
			sum.Expiring++
		case "expired":
			sum.Expired++
		case "absent":
			sum.Absent++
		case "reauth":
			sum.Reauth++
		}
		if p.LastAuthError != "" || p.LoadError != "" {
			sum.WithAuthError++
		}
	}
	resp.Tokens = &sum
}

// ConvergenceReport compte le backlog de convergence par joueur suivi
// (lectures seules — le travail reste porté par les cycles de sync).
func (r *ServiceRegistry) ConvergenceReport(ctx context.Context, titleSlug string) (domain.AdminConvergenceReport, error) {
	resp := domain.AdminConvergenceReport{
		TitleSlug:   titleSlug,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Horizon:     sync_pkg.ConvergenceHorizon,
		Players:     []domain.PlayerConvergenceReport{},
		// Cumuls « rattrapé depuis le boot » (expvar AddInt posés par le
		// pipeline post-sync — étapes 1.54/1.55/1.56 + alias PSA).
		TotalsSinceBoot: domain.ConvergenceTotalsSinceBoot{
			EventsProcessed:  observability.LoadCounter("convergence_events_processed_total"),
			WeaponsProcessed: observability.LoadCounter("convergence_weapons_processed_total"),
			PSAProcessed:     observability.LoadCounter("convergence_psa_processed_total"),
			AliasesUpserted:  observability.LoadCounter("convergence_aliases_upserted_total"),
		},
	}
	players, err := r.cfg.LoadPlayers(titleSlug)
	if err != nil {
		return resp, err
	}
	for _, p := range players {
		report := domain.PlayerConvergenceReport{
			PlayerSlug: p.PlayerSlug,
			Gamertag:   p.Gamertag,
			XUID:       p.XUID,
		}
		playerSQL, sharedSQL, release, errMsg := r.resolveMonitoringDBs(ctx, titleSlug, p.Gamertag, p.XUID)
		if errMsg != "" {
			report.CheckError = errMsg
		} else {
			counts := sync_pkg.ConvergenceBacklog(ctx, playerSQL, sharedSQL, p.XUID)
			report.MissingEnrichment = counts.MissingEnrichment
			report.MissingPSA = counts.MissingPSA
			report.MissingEvents = counts.MissingEvents
			report.MissingWeapons = counts.MissingWeapons
			release()
		}
		resp.Players = append(resp.Players, report)
	}
	return resp, nil
}

// resolveMonitoringDBs résout les handles DB d'un joueur (best-effort, miroir
// de resolveInvariantDBs avec retour d'erreur libre). release() après usage.
func (r *ServiceRegistry) resolveMonitoringDBs(
	ctx context.Context, titleSlug, gamertag, xuid string,
) (playerSQL, sharedSQL *sql.DB, release func(), errMsg string) {
	if r.resolveByGT == nil || gamertag == "" || xuid == "" {
		return nil, nil, nil, "resolver ou identité joueur indisponible"
	}
	tpdb, err := r.resolveByGT(ctx, titleSlug, gamertag)
	if err != nil || tpdb == nil || tpdb.Player == nil {
		monitoringLog.WarnContext(ctx, "admin_monitoring: resolve player failed",
			"gamertag", gamertag, "err", err)
		return nil, nil, nil, "player DB inaccessible"
	}
	shared, rel, err := tpdb.SharedReadDB().Get(ctx)
	if err != nil {
		monitoringLog.WarnContext(ctx, "admin_monitoring: shared reader failed",
			"gamertag", gamertag, "err", err)
		return nil, nil, nil, "shared DB inaccessible"
	}
	return tpdb.Player.SQLDb(), shared, rel, ""
}

// persistPhaseNames : phases d'écriture instrumentées (combined persister).
var persistPhaseNames = []string{"shared_acquire", "shared_write", "player_lease", "player_write"}

// PerfStats agrège les latences depuis le boot (expvar pur, zéro I/O DuckDB) :
// appels API Halo, phases d'écriture persist, étapes post-sync, fenêtre
// d'indisponibilité des lectures shared.
// MT-05 (PMT-10) : titleSlug filtre les agrégats par titre. Pour Halo
// (titre par défaut → effectif "") les clés expvar restent nues → sortie
// byte-identique. Un 2e titre lit ses propres clés/buckets.
func (r *ServiceRegistry) PerfStats(_ context.Context, titleSlug string) (domain.AdminPerfStats, error) {
	resp := domain.AdminPerfStats{
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		APICalls:      []domain.PerfCallStats{},
		PersistPhases: []domain.PerfCallStats{},
		PostSyncSteps: []domain.PerfCallStats{},
		APIBuckets: domain.PerfAPIBuckets{
			RateLimited429: observability.LoadCounterT(titleSlug, "halo_api_429_total"),
			Auth:           observability.LoadCounterT(titleSlug, "halo_api_auth_total"),
			Server5xx:      observability.LoadCounterT(titleSlug, "halo_api_5xx_total"),
			Network:        observability.LoadCounterT(titleSlug, "halo_api_network_total"),
			Other:          observability.LoadCounterT(titleSlug, "halo_api_other_total"),
		},
	}
	for _, call := range sync_pkg.HaloAPICallNames() {
		stats := loadPerfCallStats(titleSlug, call, "halo_api_ms_"+call)
		stats.Errors = observability.LoadCounterT(titleSlug, "halo_api_err_"+call+"_total")
		if stats.Count > 0 || stats.Errors > 0 {
			resp.APICalls = append(resp.APICalls, stats)
		}
	}
	for _, phase := range persistPhaseNames {
		stats := loadPerfCallStats(titleSlug, phase, "persist_"+phase+"_ms")
		stats.Errors = observability.LoadCounterT(titleSlug, "persist_"+phase+"_err_total")
		if stats.Count > 0 || stats.Errors > 0 {
			resp.PersistPhases = append(resp.PersistPhases, stats)
		}
	}
	for _, step := range sync_pkg.PostSyncStepNames() {
		if stats := loadPerfCallStats(titleSlug, step, "postsync_step_ms_"+step); stats.Count > 0 {
			resp.PostSyncSteps = append(resp.PostSyncSteps, stats)
		}
	}
	resp.PostSyncTotal = loadPerfCallStats(titleSlug, "postsync_total", "postsync_total_ms")
	resp.BlockedWindow = loadPerfCallStats(titleSlug, "blocked_window", "shared_provider_blocked_window_ms")

	// Breakdown par joueur des appels attribuables (collecteur dédié, filtré titre).
	resp.APIByPlayer = []domain.PerfPlayerCallStats{}
	for _, s := range observability.PlayerAPIStatsForTitle(titleSlug) {
		resp.APIByPlayer = append(resp.APIByPlayer, domain.PerfPlayerCallStats{
			Title: s.Title, Player: s.Player, Call: s.Call, Count: s.Count,
			AvgMs: s.AvgMs, MaxMs: s.MaxMs, Errors: s.Errors,
		})
	}
	return resp, nil
}

// loadPerfCallStats mappe un agrégat expvar RecordDurationMS(T) vers le DTO,
// filtré par titre (MT-05).
func loadPerfCallStats(titleSlug, name, metric string) domain.PerfCallStats {
	count, sum, avg, max := observability.LoadDurationStatsT(titleSlug, metric)
	return domain.PerfCallStats{
		Title: observability.EffectiveTitle(titleSlug),
		Name:  name, Count: count, SumMs: sum, AvgMs: avg, MaxMs: max,
	}
}

// ErrorStats retourne les logs WARN/ERROR agrégés depuis le boot (collecteur
// mémoire). Zéro I/O.
func (r *ServiceRegistry) ErrorStats(_ context.Context, titleSlug string) (domain.AdminErrorStats, error) {
	buckets := observability.ErrorBucketsForTitle(titleSlug) // MT-05 : filtré par titre
	resp := domain.AdminErrorStats{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Buckets:     make([]domain.AdminErrorBucket, 0, len(buckets)),
	}
	for _, b := range buckets {
		resp.Buckets = append(resp.Buckets, domain.AdminErrorBucket{
			Title:      b.Title,
			Level:      b.Level,
			Module:     b.Module,
			Message:    b.Message,
			Count:      b.Count,
			FirstSeen:  b.FirstSeen.UTC().Format(time.RFC3339),
			LastSeen:   b.LastSeen.UTC().Format(time.RFC3339),
			LastDetail: b.LastDetail,
		})
	}
	return resp, nil
}

// RunDataHealthNow exécute un audit data health immédiat (action admin —
// lectures RO uniquement, safe pendant un sync). Le résultat est aussi
// mémorisé par le HealthScheduler (LastResult) pour l'overview.
func (r *ServiceRegistry) RunDataHealthNow(ctx context.Context) (*domain.MonitoringDataHealth, error) {
	if r.healthScheduler == nil {
		return nil, fmt.Errorf("health scheduler non câblé")
	}
	res := r.healthScheduler.RunOnce(ctx)
	monitoringLog.InfoContext(ctx, "admin_actions: data health check exécuté",
		"warnings_total", res.WarningsTotal,
		"orphan_xuids", res.OrphanXUIDs,
		"duration_ms", res.Duration.Milliseconds(),
	)
	observability.IncCounter("admin_action_data_health_total")
	return toMonitoringDataHealth(res, time.Now()), nil
}
