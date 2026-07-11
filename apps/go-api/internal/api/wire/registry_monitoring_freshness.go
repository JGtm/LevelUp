// Package api — registry_monitoring_freshness.go : runner de la fraîcheur des
// données (plan monitoring A4, seuils DC-3).
//
// Par titre ACTIF non-interne du registre (capability matchmaking requise —
// jamais de comparaison de slug), pour chaque joueur suivi : dernier match
// persisté (timestamp canonique COALESCE sur match_registry via
// match_participants) + dernier cycle sync réussi (snapshot scheduler, quand le
// titre y passe — les titres live-only comme Halo 5 n'y sont pas : l'âge du
// match fait alors foi, c'est le trou de visibilité que ce panneau couvre).
// Best-effort par joueur (pattern ConvergenceReport) : une DB inaccessible pose
// CheckError, jamais d'échec global.
//
// A4.2 — âge du dernier backup : source = manifest du scheduler duckdbbackup
// (Status().LastBackupAt), PAS cron_runs (câblé seulement en A6) ni mtime de
// log (fragile). Décision consignée au plan.
package wire

import (
	"context"
	"database/sql"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/ops"
	"levelup/go-api/pkg/duckdbbackup"
)

// WithBackupScheduler attache le scheduler de backup au registry (âge du
// dernier backup dans la réponse fraîcheur). Nil possible : section absente.
func (r *ServiceRegistry) WithBackupScheduler(s *duckdbbackup.Scheduler) *ServiceRegistry {
	r.backupScheduler = s
	return r
}

// FreshnessReport calcule la fraîcheur des données par joueur suivi.
// titleFilter vide = tous les titres actifs non-internes du registre.
func (r *ServiceRegistry) FreshnessReport(ctx context.Context, titleFilter string) (domain.AdminFreshnessResponse, error) {
	now := time.Now().UTC()
	resp := domain.AdminFreshnessResponse{
		GeneratedAt: now.Format(time.RFC3339),
		Titles:      []domain.TitleFreshnessReport{},
	}
	th := r.freshnessThresholds()
	lastSyncOK := r.lastSyncOKByXUID()

	// DefaultRegistry : registre PILOTÉ PAR CONFIG posé au boot (built-in
	// halo_infinite + titres découverts sous config/titles/* comme halo_5) —
	// PAS NewRegistry() qui ne connaît que le titre par défaut.
	for _, desc := range titlePkg.DefaultRegistry().NonArchived() {
		if desc.IsInternal || !desc.IsActive() {
			continue
		}
		if titleFilter != "" && desc.Slug != titleFilter {
			continue
		}
		resp.Titles = append(resp.Titles, r.titleFreshness(ctx, desc, now, th, lastSyncOK))
	}
	for _, t := range resp.Titles {
		resp.CriticalTotal += t.CriticalCount
	}
	resp.Backup = r.freshnessBackupInfo(now)

	// Gauge pour le badge « État » (l'overview la lit sans I/O).
	observability.SetInt("monitoring_freshness_critical", int64(resp.CriticalTotal))
	return resp, nil
}

// freshnessThresholds lit les seuils DC-3 depuis app_settings.json (défauts sinon).
func (r *ServiceRegistry) freshnessThresholds() ops.FreshnessThresholds {
	settings, err := r.cfg.LoadAppSettings()
	if err != nil {
		monitoringLog.Warn("admin_freshness: app_settings illisible — seuils par défaut", "err", err)
		return ops.DefaultFreshnessThresholds()
	}
	return ops.FreshnessThresholdsFromSettings(settings)
}

// titleFreshness évalue tous les joueurs suivis d'un titre (best-effort).
func (r *ServiceRegistry) titleFreshness(
	ctx context.Context,
	desc *titlePkg.TitleDescriptor,
	now time.Time,
	th ops.FreshnessThresholds,
	lastSyncOK map[string]time.Time,
) domain.TitleFreshnessReport {
	report := domain.TitleFreshnessReport{TitleSlug: desc.Slug, Players: []domain.PlayerFreshness{}}
	if !desc.HasCapability(titlePkg.CapMatchmaking) {
		report.Note = "titre sans capability matchmaking — fraîcheur non applicable"
		return report
	}
	players, err := r.cfg.LoadPlayers(desc.Slug)
	if err != nil {
		report.Note = "joueurs suivis illisibles : " + err.Error()
		return report
	}
	if len(players) == 0 {
		report.Note = "aucun joueur suivi pour ce titre"
		return report
	}
	for _, p := range players {
		in := ops.PlayerFreshnessInput{Gamertag: p.Gamertag, XUID: p.XUID}
		if syncAt, ok := lastSyncOK[p.XUID]; ok {
			t := syncAt
			in.LastSyncOKAt = &t
		}
		lastMatch, errMsg := r.lastMatchAt(ctx, desc.Slug, p.Gamertag, p.XUID)
		in.LastMatchAt = lastMatch
		in.CheckError = errMsg
		pf := ops.EvaluatePlayerFreshness(in, now, th)
		switch pf.Status {
		case domain.FreshnessStatusWarn:
			report.WarnCount++
		case domain.FreshnessStatusCritical:
			report.CriticalCount++
		}
		report.Players = append(report.Players, pf)
	}
	return report
}

// lastMatchAt lit le dernier match persisté d'un joueur (timestamp canonique,
// règle CLAUDE.md n°8 — jamais start_time brut). Best-effort : erreur → message.
func (r *ServiceRegistry) lastMatchAt(ctx context.Context, titleSlug, gamertag, xuid string) (*time.Time, string) {
	_, sharedSQL, release, errMsg := r.resolveMonitoringDBs(ctx, titleSlug, gamertag, xuid)
	if errMsg != "" {
		return nil, errMsg
	}
	defer release()
	var last sql.NullTime
	q := `SELECT MAX(` + analysis.SQLStartTimeCanonical("mr") + `)
		FROM match_registry mr
		JOIN match_participants mp ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?`
	if err := sharedSQL.QueryRowContext(ctx, q, xuid).Scan(&last); err != nil {
		monitoringLog.WarnContext(ctx, "admin_freshness: last match query failed",
			"title", titleSlug, "gamertag", gamertag, "err", err)
		return nil, "lecture du dernier match impossible"
	}
	if !last.Valid {
		return nil, ""
	}
	t := last.Time
	return &t, ""
}

// lastSyncOKByXUID indexe le dernier cycle sync réussi par joueur depuis le
// snapshot scheduler (halo_infinite ; les titres live-only n'y figurent pas).
func (r *ServiceRegistry) lastSyncOKByXUID() map[string]time.Time {
	out := map[string]time.Time{}
	if r.autoSyncScheduler == nil {
		return out
	}
	snap := r.autoSyncScheduler.Snapshot()
	for _, p := range snap.Players {
		if p.Outcome == "ok" && !p.AttemptedAt.IsZero() {
			out[p.XUID] = p.AttemptedAt
		}
	}
	return out
}

// freshnessBackupInfo expose l'âge du dernier backup réussi (nil si non câblé).
func (r *ServiceRegistry) freshnessBackupInfo(now time.Time) *domain.FreshnessBackupInfo {
	if r.backupScheduler == nil {
		return nil
	}
	st := r.backupScheduler.Status()
	info := &domain.FreshnessBackupInfo{Enabled: st.Enabled, LastBackupAt: st.LastBackupAt}
	if st.LastBackupAt != "" {
		if t, err := time.Parse(time.RFC3339, st.LastBackupAt); err == nil {
			info.AgeSeconds = int64(now.Sub(t).Seconds())
		}
	}
	return info
}
