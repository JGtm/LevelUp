package migration

// steps_player_rebuild_match_enrichment.go — rebuild player_match_enrichment
// via swap CTAS pour défaire la corruption d'index ART qui faussait les
// UPSERTs (Phase 4.1 follow-up 2026-05-23).
//
// Contexte : observation prod 2026-05-23 — FATAL Error sur
// `WriteSessionAssignments(match_id) Invalid Input Error: Failed to delete
// all rows from index. Only deleted 0 out of 1 rows` lors du post-sync
// `recalculateSessionsInline`. Symptôme identique au bug ART sur
// shared.match_participants (steps_shared_rebuild_match_participants.go)
// mais cette fois sur la player DB (`stats.duckdb`) et la table
// `player_match_enrichment` (PK simple match_id, 8 colonnes).
//
// Stratégie identique au rebuild shared : swap CTAS sans sentinel pour
// permettre des re-runs au runtime via le CLI force_rebuild_art (ou plus
// tard un mécanisme auto-heal périodique).

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// RebuildPlayerMatchEnrichmentART garantit que player_match_enrichment est
// append-only (id PK + stage + written_at + vue _latest), en déléguant à la
// migration idempotente applyAppendOnlyMatchEnrichment.
//
// Append-only #23046 (2026-06-21) : l'ANCIEN rebuild swap re-posait
// `ADD PRIMARY KEY (match_id)` ET rejouait dynamiquement les ex-index ART
// (idx_pme_session / idx_pme_engagement_history / idx_pme_engagement_paces via
// loadSecondaryIndexDDL) → il RÉINTRODUISAIT le vecteur DuckDB #23046 dès qu'il
// était invoqué (exposé via cmd/rebuild_pme_art, cmd/force_rebuild_art,
// levelup rebuild-pme-art). On délègue désormais :
//   - table legacy (id absent)      → swap CTAS vers append-only (id seq + stage),
//   - table déjà append-only        → refresh vue _latest + idx_pme_match_lookup.
//
// Plus aucune PK(match_id) ni index muté → aucune corruption ART possible.
//
// IDEMPOTENTE. No-op gracieux si la table est absente.
// Pré-condition : `db` ouvert en RW EXCLUSIF (app serveur arrêtée).
func RebuildPlayerMatchEnrichmentART(ctx context.Context, db *sql.DB) error {
	if err := applyAppendOnlyMatchEnrichment(db); err != nil {
		return fmt.Errorf("rebuild_pme_runtime: %w", err)
	}
	slog.InfoContext(ctx, "rebuild_player_match_enrichment_runtime: append-only assuré (ART éradiqué, plus de PK match_id)")
	return nil
}

// loadTableColumns énumère les colonnes d'une table via PRAGMA table_info.
// Helper générique réutilisable (variante de loadMatchParticipantsColumns
// qui est spécifique).
func loadTableColumns(ctx context.Context, db *sql.DB, tableName string) ([]string, error) {
	rows, err := db.QueryContext(ctx,
		fmt.Sprintf(`PRAGMA table_info('%s')`, tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var cid int
		var name, typ string
		var nn bool
		var dflt *string
		var pk bool
		if err := rows.Scan(&cid, &name, &typ, &nn, &dflt, &pk); err != nil {
			return nil, err
		}
		cols = append(cols, name)
	}
	return cols, rows.Err()
}
