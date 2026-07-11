// Package api — registry_monitoring_crons.go : runner du statut unifié des
// crons + feature liveness (plan monitoring A6, DC-5).
//
// Fusionne le registre mémoire (observability.CronStatusSnapshot — état du
// boot courant, dont consecutive_failures) avec le dernier run persisté
// (cron_runs_latest — réhydratation après restart, marqué SinceBoot=false).
// Heartbeats : liste FERMÉE de features (DC-5), timestamp expvar posé au
// passage réel dans le code — « jamais vu » = accent destructive (le hook
// câblé mais jamais invoqué devient visible).
package wire

import (
	"context"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/ops"
)

// monitoredFeatures : liste fermée des heartbeats (DC-5).
var monitoredFeatures = []string{"prestige_hook", "notifications_push", "watcher_rta", "media_pipeline"}

// CronsReport agrège le statut des crons + les heartbeats de features (A6.3).
func (r *ServiceRegistry) CronsReport(ctx context.Context) (domain.AdminCronsResponse, error) {
	now := time.Now().UTC()
	resp := domain.AdminCronsResponse{
		GeneratedAt: now.Format(time.RFC3339),
		Crons:       []domain.CronStatusEntry{},
		Features:    []domain.FeatureHeartbeat{},
	}

	seen := map[string]bool{}
	for _, rec := range observability.CronStatusSnapshot() {
		if rec.Name == "server_boot" {
			continue // marqueur de démarrage (A5), pas un cron
		}
		seen[rec.Name] = true
		resp.Crons = append(resp.Crons, cronEntryFromRecord(rec))
	}
	// Réhydratation : crons persistés jamais vus depuis ce boot (restart récent).
	if r.monitoringStore != nil {
		runs, err := r.monitoringStore.LatestCronRuns(ctx)
		if err != nil {
			monitoringLog.WarnContext(ctx, "admin_crons: cron_runs_latest illisible", "err", err)
		}
		for _, run := range runs {
			if run.Name == "server_boot" || seen[run.Name] {
				continue
			}
			resp.Crons = append(resp.Crons, cronEntryFromPersisted(run))
		}
	}

	for _, f := range monitoredFeatures {
		resp.Features = append(resp.Features, featureHeartbeat(f, now))
	}
	return resp, nil
}

// cronEntryFromRecord mappe une entrée du registre mémoire (boot courant).
func cronEntryFromRecord(rec observability.CronStatusRecord) domain.CronStatusEntry {
	out := domain.CronStatusEntry{
		Name:                rec.Name,
		LastError:           rec.LastError,
		ConsecutiveFailures: rec.ConsecutiveFailures,
		Runs:                rec.Runs,
		LastDurationMs:      rec.LastDurationMs,
		SinceBoot:           true,
	}
	if !rec.LastRunAt.IsZero() {
		out.LastRunAt = rec.LastRunAt.UTC().Format(time.RFC3339)
	}
	if !rec.LastSuccessAt.IsZero() {
		out.LastSuccessAt = rec.LastSuccessAt.UTC().Format(time.RFC3339)
	}
	switch {
	case rec.ConsecutiveFailures >= domain.CronFailuresCriticalThreshold:
		out.Status = domain.FreshnessStatusCritical
	case rec.ConsecutiveFailures > 0:
		out.Status = domain.FreshnessStatusWarn
	default:
		out.Status = domain.FreshnessStatusOK
	}
	return out
}

// cronEntryFromPersisted mappe un run réhydraté depuis cron_runs_latest
// (pas encore couru depuis ce boot — consecutive_failures inconnu, donc au
// mieux warn si le dernier run persisté était en échec).
func cronEntryFromPersisted(run ops.PersistedCronRun) domain.CronStatusEntry {
	out := domain.CronStatusEntry{
		Name:           run.Name,
		LastError:      run.Err,
		LastDurationMs: run.DurationMs,
		SinceBoot:      false,
	}
	if !run.StartedAt.IsZero() {
		out.LastRunAt = run.StartedAt.UTC().Format(time.RFC3339)
		if run.OK {
			out.LastSuccessAt = out.LastRunAt
		}
	}
	if run.OK {
		out.Status = domain.FreshnessStatusOK
	} else {
		out.Status = domain.FreshnessStatusWarn
	}
	return out
}

// featureHeartbeat lit le heartbeat expvar d'une feature (0 = jamais vu).
func featureHeartbeat(feature string, now time.Time) domain.FeatureHeartbeat {
	out := domain.FeatureHeartbeat{Feature: feature}
	unix := observability.HeartbeatUnix(feature)
	if unix <= 0 {
		out.Status = "never"
		return out
	}
	seen := time.Unix(unix, 0).UTC()
	out.LastSeenAt = seen.Format(time.RFC3339)
	out.AgeSeconds = int64(now.Sub(seen).Seconds())
	out.Status = domain.FreshnessStatusOK
	return out
}
