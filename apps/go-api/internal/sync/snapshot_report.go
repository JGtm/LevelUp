package sync

// snapshot_report.go — agrégat GLOBAL-par-titre du backlog de snapshot (gauges expvar
// pour le monitoring opérateur, ADR 0009). Calculé après chaque cut.
//
// CRITIQUE — agrégat GLOBAL par titre : on SOMME le backlog de TOUS les joueurs en
// mémoire puis on publie UN SEUL SetIntT par titre. Un SetIntT par joueur s'écraserait
// entre joueurs (une gauge = overwrite, pas un cumul) → on ne verrait que la valeur du
// dernier joueur (bug). Idem pour l'âge : on garde le MAX cross-joueurs.
//
// « Pending » = matchs enrichis mais pas encore snapshot-ready (snapshot_ready_at IS
// NULL) — PAS le backlog de convergence amont. Un pending qui ne se résorbe pas =
// dérivation bloquée (alerte : la grâce bornée finira par le forcer, cf. plan §2).

import (
	"context"
	"database/sql"
	"time"

	"levelup/go-api/internal/observability"
	duckdb "levelup/go-api/internal/platform/duckdb"
)

// SnapshotPendingReport agrège le backlog snapshot d'un titre (tous joueurs confondus).
type SnapshotPendingReport struct {
	PendingTotal        int   // matchs snapshot_ready_at IS NULL (en attente)
	PartialTotal        int   // matchs ready AVEC partial_reasons (terminalement partiels)
	OldestPendingAgeSec int64 // âge du plus vieux pending (proxy created_at), 0 si aucun
}

// reportPending calcule l'agrégat global et publie les gauges expvar titrées.
// Best-effort (read-only) : appelé après chaque cut pour garder les gauges fraîches.
func (c *SnapshotCutter) reportPending(ctx context.Context, titleSlug string, gamertags []string) {
	if c == nil || c.paths == nil {
		return
	}
	rep := c.computeSnapshotPending(ctx, titleSlug, gamertags)
	observability.SetIntT(titleSlug, "snapshot_pending_total", int64(rep.PendingTotal))
	observability.SetIntT(titleSlug, "snapshot_partial_total", int64(rep.PartialTotal))
	observability.SetIntT(titleSlug, "snapshot_pending_oldest_age_seconds", rep.OldestPendingAgeSec)
}

// computeSnapshotPending somme le backlog sur tous les joueurs (player DB RO via
// OpenReadForQuery — handle cached, zéro ATTACH).
func (c *SnapshotCutter) computeSnapshotPending(ctx context.Context, titleSlug string, gamertags []string) SnapshotPendingReport {
	var rep SnapshotPendingReport
	now := time.Now()
	for _, gt := range gamertags {
		db, release, err := duckdb.OpenReadForQuery(c.paths.PlayerDBPath(titleSlug, gt))
		if err != nil {
			continue // player DB absente → ignorée (best-effort)
		}
		addPlayerSnapshotBacklog(ctx, db, now, &rep)
		release()
	}
	return rep
}

// addPlayerSnapshotBacklog ajoute le backlog d'un joueur à l'agrégat. Une requête
// agrégée sur la vue _latest ; toute erreur (table absente, titre fraîchement activé)
// est silencieusement tolérée (le joueur ne contribue rien).
func addPlayerSnapshotBacklog(ctx context.Context, db *sql.DB, now time.Time, rep *SnapshotPendingReport) {
	var pending, partial int
	var oldest sql.NullTime
	err := db.QueryRowContext(ctx, `
		SELECT
		  COUNT(*) FILTER (WHERE snapshot_ready_at IS NULL),
		  COUNT(*) FILTER (WHERE snapshot_ready_at IS NOT NULL AND partial_reasons IS NOT NULL AND partial_reasons <> '[]'),
		  MIN(created_at) FILTER (WHERE snapshot_ready_at IS NULL)
		FROM player_match_enrichment_latest`).Scan(&pending, &partial, &oldest)
	if err != nil {
		return
	}
	rep.PendingTotal += pending
	rep.PartialTotal += partial
	if oldest.Valid {
		if age := int64(now.Sub(oldest.Time).Seconds()); age > rep.OldestPendingAgeSec {
			rep.OldestPendingAgeSec = age
		}
	}
}
