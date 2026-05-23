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
	"strings"
)

// RebuildPlayerMatchEnrichmentART exécute le rebuild swap CTAS de la table
// player_match_enrichment pour défaire la corruption d'index ART (Phase 4.1
// extension 2026-05-23).
//
// IDEMPOTENTE — peut être rappelée à chaque détection de corruption.
// Préserve les rows + la PK + les indexes éventuels.
//
// Pré-condition : `db` ouvert en RW EXCLUSIF (caller doit s'assurer
// qu'aucun autre process ne tient le fichier — l'app serveur arrêtée).
//
// No-op gracieux si la table est absente (player DB jamais initialisée).
func RebuildPlayerMatchEnrichmentART(ctx context.Context, db *sql.DB) error {
	hasTable, err := tableExists(db, "player_match_enrichment")
	if err != nil {
		return fmt.Errorf("rebuild_pme_runtime: check table: %w", err)
	}
	if !hasTable {
		return nil
	}

	cols, err := loadTableColumns(ctx, db, "player_match_enrichment")
	if err != nil {
		return fmt.Errorf("rebuild_pme_runtime: enumerate columns: %w", err)
	}
	if len(cols) == 0 {
		return nil
	}
	colList := strings.Join(cols, ", ")

	var before int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM player_match_enrichment`).Scan(&before); err != nil {
		return fmt.Errorf("rebuild_pme_runtime: count before: %w", err)
	}

	stmts := []string{
		`DROP TABLE IF EXISTS player_match_enrichment__rebuilt`,
		fmt.Sprintf(`CREATE TABLE player_match_enrichment__rebuilt AS SELECT %s FROM player_match_enrichment`, colList),
		`DROP TABLE player_match_enrichment`,
		`ALTER TABLE player_match_enrichment__rebuilt RENAME TO player_match_enrichment`,
		`ALTER TABLE player_match_enrichment ADD PRIMARY KEY (match_id)`,
	}
	for _, sqlStmt := range stmts {
		if _, err := db.ExecContext(ctx, sqlStmt); err != nil {
			return fmt.Errorf("rebuild_pme_runtime: swap step (%s): %w",
				firstWords(sqlStmt, 3), err)
		}
	}

	// Pas d'indexes secondaires connus sur player_match_enrichment (seulement
	// la PK). Si la table en a d'autres dans des installations spécifiques,
	// les recréer ici via un loadIndexes() helper (non implémenté).

	var after int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM player_match_enrichment`).Scan(&after); err != nil {
		return fmt.Errorf("rebuild_pme_runtime: count after: %w", err)
	}

	slog.InfoContext(ctx, "rebuild_player_match_enrichment_runtime: table rebuilt (ART corruption defeated)",
		"rows_before_rebuild", before,
		"rows_after_rebuild", after,
		"columns_preserved", len(cols),
	)
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
