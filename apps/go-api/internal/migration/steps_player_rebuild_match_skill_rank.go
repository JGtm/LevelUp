package migration

// steps_player_rebuild_match_skill_rank.go — rebuild match_skill_rank via
// swap CTAS pour défaire la corruption d'index ART qui faussait les
// DELETE/UPSERT LUSR (Phase 4.5 smoke test follow-up 2026-05-24).
//
// Contexte : observation prod 2026-05-24 (smoke test Phase 4 cycle 2) —
// FATAL Error sur PostSyncLUSRPersister.Upsert : "Invalid Input Error:
// Failed to delete all rows from index. Only deleted 0 out of N rows"
// lors du DELETE WHERE match_id IN(...) AND rating_type='LUSR'. La table
// player_match_enrichment avait été rebuilt mais match_skill_rank non,
// donc l'ART de match_skill_rank gardait des entrées orphelines qui
// bloquaient le DELETE batch.
//
// Stratégie identique aux autres rebuilds (RebuildPlayerMatchEnrichmentART,
// RebuildMatchParticipantsART) : swap CTAS sans sentinel, re-rune au runtime
// via le CLI force_rebuild_art. Recrée les 2 indexes secondaires
// (idx_msr_rating_type sur rating_type, idx_msr_playlist sur playlist_group)
// après le swap.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

// RebuildMatchSkillRankART exécute le rebuild swap CTAS de la table
// match_skill_rank pour défaire la corruption d'index ART (Phase 4.5
// follow-up 2026-05-24).
//
// IDEMPOTENTE — peut être rappelée à chaque détection de corruption.
// Préserve les rows + la PK (match_id) + les 2 indexes secondaires
// (idx_msr_rating_type, idx_msr_playlist).
//
// Pré-condition : `db` ouvert en RW EXCLUSIF (caller doit s'assurer
// qu'aucun autre process ne tient le fichier — l'app serveur arrêtée).
//
// No-op gracieux si la table est absente (player DB jamais initialisée
// ou pre-migration add_skill_rating_table).
func RebuildMatchSkillRankART(ctx context.Context, db *sql.DB) error {
	hasTable, err := tableExists(db, "match_skill_rank")
	if err != nil {
		return fmt.Errorf("rebuild_msr_runtime: check table: %w", err)
	}
	if !hasTable {
		return nil
	}

	cols, err := loadTableColumns(ctx, db, "match_skill_rank")
	if err != nil {
		return fmt.Errorf("rebuild_msr_runtime: enumerate columns: %w", err)
	}
	if len(cols) == 0 {
		return nil
	}
	colList := strings.Join(cols, ", ")

	var before int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_skill_rank`).Scan(&before); err != nil {
		return fmt.Errorf("rebuild_msr_runtime: count before: %w", err)
	}

	stmts := []string{
		`DROP TABLE IF EXISTS match_skill_rank__rebuilt`,
		fmt.Sprintf(`CREATE TABLE match_skill_rank__rebuilt AS SELECT %s FROM match_skill_rank`, colList),
		`DROP TABLE match_skill_rank`,
		`ALTER TABLE match_skill_rank__rebuilt RENAME TO match_skill_rank`,
		`ALTER TABLE match_skill_rank ADD PRIMARY KEY (match_id)`,
		`CREATE INDEX IF NOT EXISTS idx_msr_rating_type ON match_skill_rank(rating_type)`,
		`CREATE INDEX IF NOT EXISTS idx_msr_playlist ON match_skill_rank(playlist_group)`,
	}
	for _, sqlStmt := range stmts {
		if _, err := db.ExecContext(ctx, sqlStmt); err != nil {
			return fmt.Errorf("rebuild_msr_runtime: swap step (%s): %w",
				firstWords(sqlStmt, 3), err)
		}
	}

	var after int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_skill_rank`).Scan(&after); err != nil {
		return fmt.Errorf("rebuild_msr_runtime: count after: %w", err)
	}

	slog.InfoContext(ctx, "rebuild_match_skill_rank_runtime: table rebuilt (ART corruption defeated)",
		"rows_before_rebuild", before,
		"rows_after_rebuild", after,
		"columns_preserved", len(cols),
	)
	return nil
}
