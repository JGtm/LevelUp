// Package sync — sessions_postsync_persist.go : Phase 4.4 du refactor
// Collect→Persist — chemin batch pour WriteSessionAssignments.

package sync

import (
	"context"
	"database/sql"
	"log/slog"
	"strconv"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/persist"
)

// writeSessionAssignmentsBatch est la variante INSERT-only-friendly de
// WriteSessionAssignments. 1 single UPDATE multi-row au lieu de N UPDATE
// row-by-row sur player_match_enrichment.
func writeSessionAssignmentsBatch(ctx context.Context, db *sql.DB, assignments []domain.SessionAssignment) (int, error) {
	if len(assignments) == 0 {
		return 0, nil
	}
	rows := make([]persist.EnrichmentMultiColumnUpdate, 0, len(assignments))
	for _, a := range assignments {
		rows = append(rows, persist.EnrichmentMultiColumnUpdate{
			MatchID: a.MatchID,
			Fields: map[string]any{
				"session_id":    strconv.Itoa(a.SessionID),
				"session_label": a.SessionLabel,
			},
		})
	}
	p := persist.NewPostSyncEnrichmentPersister(db)
	if err := p.BatchUpdateMulti(ctx, rows); err != nil {
		slog.ErrorContext(ctx, "writeSessionAssignmentsBatch: BatchUpdateMulti échoué",
			"batch_size", len(rows), "err", err)
		return 0, err
	}
	slog.InfoContext(ctx, "writeSessionAssignmentsBatch: sessions persistées (INSERT-only path)",
		"updated", len(rows))
	return len(rows), nil
}
