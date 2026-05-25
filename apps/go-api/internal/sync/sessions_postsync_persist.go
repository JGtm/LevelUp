// Package sync — sessions_postsync_persist.go : persistance des sessions
// avec delta filter pour éviter le bug ART DuckDB (cf. ADR 0019).

package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/persist"
)

// writeSessionAssignmentsBatch ne persiste que les assignments dont session_id
// ou session_label ont changé vs la DB courante (delta filter).
// Sans ce filtre, le recalcul complet des sessions génère 700+ UPDATEs sur des
// rows inchangées → bug ART DuckDB "Failed to delete all rows from index".
func writeSessionAssignmentsBatch(ctx context.Context, db *sql.DB, assignments []domain.SessionAssignment) (int, error) {
	if len(assignments) == 0 {
		return 0, nil
	}

	changed, err := deltaSessionAssignments(ctx, db, assignments)
	if err != nil {
		slog.WarnContext(ctx, "writeSessionAssignmentsBatch: delta load échoué, fallback full update",
			"total", len(assignments), "err", err)
		changed = assignments // fallback safe : on écrit tout plutôt que de rater des sessions
	}

	if len(changed) == 0 {
		slog.InfoContext(ctx, "writeSessionAssignmentsBatch: aucun changement de session détecté",
			"total_matchs", len(assignments))
		return 0, nil
	}

	rows := make([]persist.EnrichmentMultiColumnUpdate, 0, len(changed))
	for _, a := range changed {
		rows = append(rows, persist.EnrichmentMultiColumnUpdate{
			MatchID: a.MatchID,
			Fields: map[string]any{
				"session_id":    strconv.Itoa(a.SessionID),
				"session_label": a.SessionLabel,
			},
		})
	}

	p := persist.NewPostSyncEnrichmentPersister(db)
	affected, err := p.BatchUpdateMulti(ctx, rows)
	if err != nil {
		slog.ErrorContext(ctx, "writeSessionAssignmentsBatch: BatchUpdateMulti échoué",
			"batch_size", len(rows), "err", err)
		return 0, err
	}
	slog.InfoContext(ctx, "writeSessionAssignmentsBatch: sessions persistées",
		"planned", len(rows), "affected", affected, "total_matchs", len(assignments))
	return int(affected), nil
}

// deltaSessionAssignments compare les nouveaux assignments calculés avec les
// valeurs actuelles dans player_match_enrichment. Retourne uniquement les rows
// où session_id ou session_label ont changé (ou qui n'ont pas encore de row).
func deltaSessionAssignments(ctx context.Context, db *sql.DB, newAssignments []domain.SessionAssignment) ([]domain.SessionAssignment, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT match_id,
		       COALESCE(CAST(session_id AS VARCHAR), ''),
		       COALESCE(session_label, '')
		FROM player_match_enrichment
	`)
	if err != nil {
		return nil, fmt.Errorf("deltaSessionAssignments: query: %w", err)
	}
	defer rows.Close()

	type curEntry struct{ sid, label string }
	cur := make(map[string]curEntry, len(newAssignments))
	for rows.Next() {
		var matchID, sid, label string
		if err := rows.Scan(&matchID, &sid, &label); err != nil {
			return nil, fmt.Errorf("deltaSessionAssignments: scan: %w", err)
		}
		cur[matchID] = curEntry{sid, label}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("deltaSessionAssignments: rows.Err: %w", err)
	}

	var changed []domain.SessionAssignment
	for _, a := range newAssignments {
		e, found := cur[a.MatchID]
		newSID := strconv.Itoa(a.SessionID)
		if !found || e.sid != newSID || e.label != a.SessionLabel {
			changed = append(changed, a)
		}
	}
	return changed, nil
}
