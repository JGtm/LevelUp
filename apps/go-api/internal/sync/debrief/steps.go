// Package debrief capte, côté synchronisation, les instants du délai du débrief
// (domain.MatchStep) que le post-sync est seul à connaître : l'écriture des stats d'un match et
// la fin de ce match selon l'API.
//
// Sous-package parce que la racine de internal/sync/ est gelée (archlint
// TestSyncRootPackageFrozen) : le post-sync n'y fait qu'un appel.
package debrief

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
)

// ReportInserted capte, pour les matchs que le cycle vient d'insérer, l'écriture de leurs stats
// (maintenant) et leur fin (`match_registry.end_time_utc`, lue dans le segment de lecture que
// le post-sync prête). Une lecture en échec perd seulement la fin de match, journalisé.
func ReportInserted(ctx context.Context, sharedDB *sql.DB, titleSlug string, insertedIDs []string) {
	if len(insertedIDs) == 0 {
		return
	}
	for _, id := range insertedIDs {
		observability.ReportMatchStep(titleSlug, id, domain.MatchStepStatsWritten, "")
	}
	if sharedDB == nil {
		return
	}
	args := make([]any, len(insertedIDs))
	for i, id := range insertedIDs {
		args[i] = id
	}
	rows, err := sharedDB.QueryContext(ctx, `SELECT match_id, end_time_utc FROM match_registry
		WHERE end_time_utc IS NOT NULL AND match_id IN (`+
		strings.TrimSuffix(strings.Repeat("?,", len(insertedIDs)), ",")+`)`, args...)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: fin des matchs insérés illisible — délai du débrief sans référence",
			"err", err, "matchs", len(insertedIDs))
		return
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id string
		var fin time.Time
		if err := rows.Scan(&id, &fin); err != nil {
			slog.WarnContext(ctx, "post-sync: fin d'un match inséré illisible", "err", err)
			return
		}
		observability.ReportMatchStepAt(titleSlug, id, domain.MatchStepEnd, fin, "")
	}
	if err := rows.Err(); err != nil {
		slog.WarnContext(ctx, "post-sync: fin des matchs insérés illisible", "err", err)
	}
}

// ReportCitations capte l'écriture des citations d'un match pour UN joueur (xuid en détail).
func ReportCitations(titleSlug, xuid, matchID string) {
	observability.ReportMatchStep(titleSlug, matchID, domain.MatchStepCitationsWritten, xuid)
}

// ReportKillSource capte l'écriture de la passe film d'un match (lecture des morts).
func ReportKillSource(titleSlug, matchID, detail string) {
	observability.ReportMatchStep(titleSlug, matchID, domain.MatchStepKillSourceWritten, detail)
}
