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
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/pkg/duckdbbackup"
)

// freshnessLastMatchReadErr : message best-effort quand la lecture groupée du
// dernier match échoue (shared absent/illisible). Posé sur chaque joueur du titre.
const freshnessLastMatchReadErr = "lecture du dernier match impossible"

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
	// E3 (revue 2026-07) : UNE requête groupée par titre sur le shared reader.
	// On ne résout AUCUNE player-DB ici (avant : une résolution jetée par joueur,
	// qui CRÉAIT même la DB des profils auth_only, + un scan MAX+JOIN par joueur).
	xuids := make([]string, 0, len(players))
	for _, p := range players {
		if p.XUID != "" {
			xuids = append(xuids, p.XUID)
		}
	}
	lastByXUID, sharedErr := r.lastMatchByXUID(ctx, desc.Slug, xuids)
	for _, p := range players {
		in := ops.PlayerFreshnessInput{Gamertag: p.Gamertag, XUID: p.XUID}
		if syncAt, ok := lastSyncOK[p.XUID]; ok {
			t := syncAt
			in.LastSyncOKAt = &t
		}
		if sharedErr != "" {
			in.CheckError = sharedErr
		} else if t, ok := lastByXUID[p.XUID]; ok {
			tt := t
			in.LastMatchAt = &tt
		}
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

// lastMatchByXUID lit, en UNE requête groupée par titre, le dernier match
// persisté de chaque joueur (timestamp canonique, règle CLAUDE.md n°8 — jamais
// start_time brut). Lit le shared reader du titre via OpenReadForQuery (réutilise
// le handle en cache RO/RW s'il est tenu — B-swap-safe, jamais OpenReadOnly
// forcé). Ne résout AUCUNE player-DB : un profil auth_only (sans match) est
// simplement absent du résultat, et AUCUNE DB n'est créée pour lui (E3). Best-
// effort : shared absent/illisible → message d'erreur global, jamais de panic.
func (r *ServiceRegistry) lastMatchByXUID(ctx context.Context, titleSlug string, xuids []string) (map[string]time.Time, string) {
	out := map[string]time.Time{}
	if len(xuids) == 0 {
		return out, ""
	}
	sharedPath := titlePkg.NewPathResolver(r.cfg.RepoRoot).SharedDBPath(titleSlug)
	sqlDB, release, err := duckdb.OpenReadForQuery(sharedPath)
	if err != nil {
		monitoringLog.WarnContext(ctx, "admin_freshness: shared reader indisponible",
			"title", titleSlug, "err", err)
		return out, freshnessLastMatchReadErr
	}
	defer release()

	placeholders := make([]string, len(xuids))
	args := make([]any, len(xuids))
	for i, x := range xuids {
		placeholders[i] = "?"
		args[i] = x
	}
	q := `SELECT mp.xuid, MAX(` + analysis.SQLStartTimeCanonical("mr") + `)
		FROM match_registry mr
		JOIN match_participants mp ON mr.match_id = mp.match_id
		WHERE mp.xuid IN (` + strings.Join(placeholders, ", ") + `)
		GROUP BY mp.xuid`
	rows, err := sqlDB.QueryContext(ctx, q, args...)
	if err != nil {
		monitoringLog.WarnContext(ctx, "admin_freshness: requête groupée dernier match échouée",
			"title", titleSlug, "err", err)
		return out, freshnessLastMatchReadErr
	}
	defer rows.Close()
	for rows.Next() {
		var xuid string
		var last sql.NullTime
		if err := rows.Scan(&xuid, &last); err != nil {
			monitoringLog.WarnContext(ctx, "admin_freshness: scan dernier match",
				"title", titleSlug, "err", err)
			return out, freshnessLastMatchReadErr
		}
		if last.Valid {
			out[xuid] = last.Time
		}
	}
	if err := rows.Err(); err != nil {
		monitoringLog.WarnContext(ctx, "admin_freshness: itération dernier match",
			"title", titleSlug, "err", err)
		return out, freshnessLastMatchReadErr
	}
	return out, ""
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
