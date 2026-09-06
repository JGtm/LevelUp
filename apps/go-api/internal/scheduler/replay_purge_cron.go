// Package scheduler — replay_purge_cron.go : purge récurrente des artefacts de rejeu 2D
// hors fenêtre de rétention (lot 6 v7.5).
//
// CE CRON SUPPRIME DES ARTEFACTS, JAMAIS DES FILMS. Les artefacts
// (data/cache/replays/{title}/{short8}.json) se re-cuisent en une commande depuis le
// cache film (levelup backfill-replay) ; les films, eux, sont IRREMPLAÇABLES une fois
// expirés côté serveur Halo — le cron ne touche ni film_chunks/ ni film_manifests/.
//
// La fenêtre `replay_retention_months` est relue à CHAQUE tick (patron scheduler,
// settings live) : 0 = illimité → tick no-op, rapporté en succès (0 purgé). L'âge d'un
// artefact est celui de SON MATCH (start_time canonique du registre), jamais le mtime du
// fichier — re-cuire un vieux match ne le rajeunit pas. Un artefact sans ligne de
// registre n'est PAS supprimé (on ne détruit pas ce qu'on ne sait pas dater).
//
// Visible sur /admin/monitoring/crons via observability.ReportCronRun (patron
// world_leaderboard_cron).
package scheduler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/replaybuild"
)

// DefaultReplayPurgeInterval : 1×/jour suffit — la fenêtre est en MOIS.
const DefaultReplayPurgeInterval = 24 * time.Hour

// replayPurgeCronName : nom du cron sur /admin/monitoring/crons.
const replayPurgeCronName = "replay_purge"

// ReplayPurgeCron purge périodiquement les artefacts de rejeu hors fenêtre.
type ReplayPurgeCron struct {
	repoRoot string
	// retention relit replay_retention_months à chaque tick (0 = illimité, no-op).
	retention func() int
	registry  *titlePkg.Registry
	interval  time.Duration
}

// NewReplayPurgeCron construit le cron. retention requis (nil → noop).
func NewReplayPurgeCron(repoRoot string, retention func() int, interval time.Duration) *ReplayPurgeCron {
	if interval <= 0 {
		interval = DefaultReplayPurgeInterval
	}
	return &ReplayPurgeCron{
		repoRoot:  repoRoot,
		retention: retention,
		registry:  titlePkg.DefaultRegistry(),
		interval:  interval,
	}
}

// Run lance le cron : premier tick immédiat, puis toutes les `interval`. Bloque
// jusqu'à ctx.Done().
func (c *ReplayPurgeCron) Run(ctx context.Context) {
	if c == nil || c.retention == nil {
		slog.WarnContext(ctx, "replay_purge_cron: noop (retention nil)")
		return
	}
	slog.InfoContext(ctx, "replay_purge_cron: started", "interval", c.interval)
	c.RunOnce(ctx)
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "replay_purge_cron: stopped (ctx done)")
			return
		case <-ticker.C:
			c.RunOnce(ctx)
		}
	}
}

// RunOnce exécute un cycle complet (exporté pour les tests). Best-effort par titre ; le
// cycle rapporte l'erreur agrégée réelle à ReportCronRun.
func (c *ReplayPurgeCron) RunOnce(ctx context.Context) {
	if c == nil || c.retention == nil {
		return
	}
	start := time.Now()
	months := c.retention()
	if months <= 0 {
		// Fenêtre illimitée : no-op NOMINAL (pas un skip d'erreur), rapporté en succès.
		observability.ReportCronRun(replayPurgeCronName, start, nil, time.Since(start).Milliseconds())
		return
	}
	cutoff := time.Now().UTC().AddDate(0, -months, 0)

	var errs []error
	totalPurged := 0
	reg := c.registry
	if reg == nil {
		reg = titlePkg.DefaultRegistry()
	}
	pr := titlePkg.NewPathResolver(c.repoRoot)
	for _, desc := range reg.Active() {
		if desc == nil {
			continue
		}
		purged, kept, unknown, err := purgeReplayArtifactsForTitle(ctx,
			pr.SharedDBPath(desc.Slug), pr.ReplayArtifactsDir(desc.Slug), cutoff)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", desc.Slug, err))
			continue
		}
		totalPurged += purged
		if purged > 0 || unknown > 0 {
			slog.InfoContext(ctx, "replay_purge_cron: titre purgé",
				"titleSlug", desc.Slug, "purged", purged, "kept", kept,
				"unknown", unknown, "cutoff", cutoff.Format(time.RFC3339))
		}
	}
	if totalPurged > 0 {
		observability.AddInt("replay_purge_artifacts_deleted_total", int64(totalPurged))
	}
	observability.ReportCronRun(replayPurgeCronName, start, errors.Join(errs...), time.Since(start).Milliseconds())
}

// purgeReplayArtifactsForTitle purge les artefacts d'UN titre. Rend
// (purgés, conservés, sans-registre). Un dossier d'artefacts absent est nominal (aucun
// artefact construit pour ce titre) : (0, 0, 0, nil).
func purgeReplayArtifactsForTitle(
	ctx context.Context, sharedPath, artifactsDir string, cutoff time.Time,
) (purged, kept, unknown int, err error) {
	entries, rerr := os.ReadDir(artifactsDir)
	if os.IsNotExist(rerr) {
		return 0, 0, 0, nil
	}
	if rerr != nil {
		return 0, 0, 0, fmt.Errorf("lecture du dossier d'artefacts %s: %w", artifactsDir, rerr)
	}
	if len(entries) == 0 {
		return 0, 0, 0, nil
	}
	dates, derr := matchDatesByShortID(ctx, sharedPath)
	if derr != nil {
		return 0, 0, 0, derr
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".json") {
			continue // on ne touche QUE des artefacts {short8}.json
		}
		// LA MARQUE DE DÉRIVATION N'EST PAS UN ARTEFACT (constat C6 de la revue A-R1).
		// `<short8>.derived.json` vit dans le MÊME dossier et finit par `.json` : elle donnait
		// `short = "<short8>.derived"`, absent du registre, donc `unknown++` — une ligne INFO à
		// CHAQUE passage dès qu'une dérivation existait, et une marque qui SURVIVAIT à la purge
		// de son propre artefact. Elle se traite avec lui, plus bas, jamais pour elle-même.
		if replaybuild.EstMarqueDerivations(name) {
			continue
		}
		short := strings.TrimSuffix(name, ".json")
		at, ok := dates[short]
		if !ok || at.IsZero() {
			unknown++ // indatable : on ne détruit pas ce qu'on ne sait pas dater
			continue
		}
		if !at.Before(cutoff) {
			kept++
			continue
		}
		chemin := filepath.Join(artifactsDir, name)
		if rmErr := os.Remove(chemin); rmErr != nil {
			slog.WarnContext(ctx, "replay_purge_cron: suppression échouée",
				"artifact", name, "err", rmErr)
			continue
		}
		// LA MARQUE SUIT SON ARTEFACT : sans cette ligne elle resterait indéfiniment, à
		// décrire des dérivations d'un fichier qui n'existe plus.
		if rmErr := replaybuild.RemoveDerivationsMark(chemin); rmErr != nil {
			slog.WarnContext(ctx, "replay_purge_cron: marque de dérivation non supprimée",
				"artifact", name, "err", rmErr)
		}
		purged++
	}
	return purged, kept, unknown, nil
}

// matchDatesByShortID lit (forme courte -> start_time canonique) de tout le registre.
// OpenReadForQuery : la shared peut être tenue par le serveur (RO ou fenêtre RW B-swap),
// le handle process-wide est réutilisé — jamais un second open concurrent.
func matchDatesByShortID(ctx context.Context, sharedPath string) (map[string]time.Time, error) {
	if _, err := os.Stat(sharedPath); err != nil {
		return nil, fmt.Errorf("shared introuvable: %w", err)
	}
	db, release, err := duckdb.OpenReadForQuery(sharedPath)
	if err != nil {
		return nil, fmt.Errorf("open shared (lecture): %w", err)
	}
	defer release()
	q := `SELECT match_id, ` + analysis.SQLStartTimeCanonical("match_registry") + ` FROM match_registry`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("lecture match_registry: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]time.Time{}
	for rows.Next() {
		var id string
		var at sql.NullTime
		if err := rows.Scan(&id, &at); err != nil {
			return nil, fmt.Errorf("lecture match_registry (scan): %w", err)
		}
		if at.Valid {
			out[titlePkg.FilmShortMatchID(id)] = at.Time
		}
	}
	return out, rows.Err()
}
