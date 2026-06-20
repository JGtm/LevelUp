// Package migration — steps_player_lusr_components_append_only.go :
// éradication ART de la table lusr_component_history (player DB, V2 §1) — 2026-06-20.
//
// **Pourquoi** : l'ancien schéma avait PK (match_id, component_name), ce qui forçait
// writeLUSRComponentHistory à écrire en `INSERT ... ON CONFLICT (match_id, component_name)
// DO UPDATE` — donc un delete+insert interne sur l'index ART, déclencheur du bug DuckDB
// amont #23046 ("Failed to delete all rows from index" → DB FATAL invalidated).
// lusr_component_history est la table SŒUR de match_skill_rank (même pipeline LUSR,
// même horloge) ; match_skill_rank a déjà été migrée en append-only (phase 2.B).
//
// **Stratégie append-only** : N versions par (match_id, component_name). PK technique
// `id BIGINT` (séquence lch_seq) = unicité physique sans contrainte fonctionnelle.
// `computed_at` sert d'horloge. La vue `lusr_component_history_latest` filtre la
// dernière version par (match_id, component_name) via window function. Toutes les
// écritures futures sont de simples INSERT (writer loaders + persister). Le bug ART
// devient impossible par construction.
//
// **Index secondaires conservés** (idx_lch_component / idx_lch_match) : append-only =
// INSERT pur, jamais de delete-from-index → ces index ne sont pas une surface ART
// (même raisonnement que les idx_msr_* sur match_skill_rank append-only).
//
// **Placement** : ce fichier s'enregistre juste APRÈS create_lusr_component_history
// (steps_player_lusr_components.go) → le rebuild s'applique dès le 1er boot (la table
// existe déjà), sans attendre la rotation suivante.
//
// **Idempotence** : check columnExists(id) → no-op si déjà appliquée.

package migration

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

func init() {
	Register(Migration{
		Name:        "player_append_only_lusr_component_history_v1",
		TargetDB:    TargetPlayer,
		Description: "Rebuild lusr_component_history en append-only (id PK + vue latest) — élimine ON CONFLICT/ART par construction",
		ApplySchema: applyAppendOnlyLUSRComponentHistory,
	})
}

func applyAppendOnlyLUSRComponentHistory(db *sql.DB) error {
	ctx := bootCtx()

	hasTable, err := tableExists(db, "lusr_component_history")
	if err != nil {
		return fmt.Errorf("append-only lch: check table: %w", err)
	}
	if !hasTable {
		// Table pas encore créée (player DB neuve, ordre create-then-migrate).
		// No-op : create_lusr_component_history la crée, et comme ce fichier
		// s'enregistre juste après, le rebuild s'applique dans la même passe.
		return nil
	}

	// Idempotence : si la colonne `id` existe déjà, on a déjà migré.
	hasIDCol, err := columnExists(db, "lusr_component_history", "id")
	if err != nil {
		return fmt.Errorf("append-only lch: check id column: %w", err)
	}
	if hasIDCol {
		return nil
	}

	cols, err := loadTableColumns(ctx, db, "lusr_component_history")
	if err != nil {
		return fmt.Errorf("append-only lch: enumerate columns: %w", err)
	}
	if len(cols) == 0 {
		return nil
	}
	colList := strings.Join(cols, ", ")

	var before int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM lusr_component_history`).Scan(&before); err != nil {
		return fmt.Errorf("append-only lch: count before: %w", err)
	}

	stmts := []string{
		`CREATE SEQUENCE IF NOT EXISTS lch_seq START 1`,
		`DROP TABLE IF EXISTS lusr_component_history__appendonly`,
		// CTAS : id généré + colonnes existantes (dont computed_at, préservé tel
		// quel pour les rows existantes — leur horloge réelle est conservée).
		fmt.Sprintf(`
			CREATE TABLE lusr_component_history__appendonly AS
			SELECT
				nextval('lch_seq') AS id,
				%s
			FROM lusr_component_history
		`, colList),
		`DROP TABLE lusr_component_history`,
		`ALTER TABLE lusr_component_history__appendonly RENAME TO lusr_component_history`,
		`ALTER TABLE lusr_component_history ADD PRIMARY KEY (id)`,
		`ALTER TABLE lusr_component_history ALTER COLUMN id SET DEFAULT nextval('lch_seq')`,
		// Restaure le DEFAULT de computed_at (perdu par CTAS) : le writer persister
		// (player_persister.go) omet computed_at et compte sur ce DEFAULT.
		`ALTER TABLE lusr_component_history ALTER COLUMN computed_at SET DEFAULT now()`,
		`CREATE INDEX IF NOT EXISTS idx_lch_component ON lusr_component_history(component_name)`,
		`CREATE INDEX IF NOT EXISTS idx_lch_match ON lusr_component_history(match_id)`,
		`CREATE OR REPLACE VIEW lusr_component_history_latest AS
			SELECT * FROM lusr_component_history
			QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, component_name ORDER BY computed_at DESC, id DESC) = 1`,
	}
	for _, sqlStmt := range stmts {
		if _, err := db.ExecContext(ctx, sqlStmt); err != nil {
			return fmt.Errorf("append-only lch: step (%s): %w",
				firstWords(sqlStmt, 3), err)
		}
	}

	var after int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM lusr_component_history`).Scan(&after); err != nil {
		return fmt.Errorf("append-only lch: count after: %w", err)
	}

	slog.InfoContext(ctx, "append-only lusr_component_history: migration appliquée (ART eradication)",
		"rows_before", before,
		"rows_after", after,
		"columns_preserved", len(cols),
	)
	return nil
}
