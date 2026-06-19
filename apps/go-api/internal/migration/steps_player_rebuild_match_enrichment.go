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

	// Capture les CREATE INDEX des indexes secondaires (hors PK) AVANT le swap :
	// le DROP TABLE les supprime, on les rejoue a l'identique apres. Dynamique
	// pour ne RIEN perdre — la table a gagne idx_pme_engagement_history /
	// _paces / _session apres 2026-05 (migration engagement), absents de la
	// version initiale de ce rebuild.
	indexDDL, err := loadSecondaryIndexDDL(ctx, db, "player_match_enrichment")
	if err != nil {
		return fmt.Errorf("rebuild_pme_runtime: capture indexes: %w", err)
	}

	// Swap CTAS en TRANSACTION unique avec garde anti-perte : on refuse de
	// detruire l'original si le rebuild n'a pas EXACTEMENT le meme nombre de
	// rows. Un crash entre DROP et RENAME -> rollback integral, table intacte
	// (aucune perte possible). Meme niveau de surete que RebuildMatchParticipantsART.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("rebuild_pme_runtime: begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS player_match_enrichment__rebuilt`); err != nil {
		return fmt.Errorf("rebuild_pme_runtime: drop stale __rebuilt: %w", err)
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(
		`CREATE TABLE player_match_enrichment__rebuilt AS SELECT %s FROM player_match_enrichment`, colList)); err != nil {
		return fmt.Errorf("rebuild_pme_runtime: create __rebuilt: %w", err)
	}
	var rebuilt int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM player_match_enrichment__rebuilt`).Scan(&rebuilt); err != nil {
		return fmt.Errorf("rebuild_pme_runtime: count __rebuilt: %w", err)
	}
	if rebuilt != before {
		return fmt.Errorf("rebuild_pme_runtime: swap abandonne, rebuilt=%d != before=%d (rollback, aucune perte de rows)", rebuilt, before)
	}
	for _, stmt := range []string{
		`DROP TABLE player_match_enrichment`,
		`ALTER TABLE player_match_enrichment__rebuilt RENAME TO player_match_enrichment`,
		`ALTER TABLE player_match_enrichment ADD PRIMARY KEY (match_id)`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("rebuild_pme_runtime: swap step (%s): %w", firstWords(stmt, 3), err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("rebuild_pme_runtime: commit swap: %w", err)
	}
	committed = true

	// Recree les indexes secondaires (hors transaction).
	for _, ddl := range indexDDL {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("rebuild_pme_runtime: recreate index (%s): %w", firstWords(ddl, 3), err)
		}
	}

	var after int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM player_match_enrichment`).Scan(&after); err != nil {
		return fmt.Errorf("rebuild_pme_runtime: count after: %w", err)
	}

	slog.InfoContext(ctx, "rebuild_player_match_enrichment_runtime: table rebuilt (ART corruption defeated)",
		"rows_before_rebuild", before,
		"rows_after_rebuild", after,
		"columns_preserved", len(cols),
		"indexes_recreated", len(indexDDL),
	)
	return nil
}

// loadSecondaryIndexDDL retourne les CREATE INDEX des indexes NON-PK d'une
// table (via duckdb_indexes()), pour les rejouer apres un rebuild swap.
func loadSecondaryIndexDDL(ctx context.Context, db *sql.DB, tableName string) ([]string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT sql FROM duckdb_indexes() WHERE table_name = ? AND is_primary = false AND sql IS NOT NULL`,
		tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var ddl string
		if err := rows.Scan(&ddl); err != nil {
			return nil, err
		}
		out = append(out, ddl)
	}
	return out, rows.Err()
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
