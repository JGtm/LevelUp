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
)

func init() {
	Register(Migration{
		Name:        "player_append_only_lusr_component_history_v1",
		TargetDB:    TargetPlayer,
		Description: "Rebuild lusr_component_history en append-only (id PK + vue latest) — élimine ON CONFLICT/ART par construction",
		ApplySchema: applyAppendOnlyLUSRComponentHistory,
	})
}

// applyAppendOnlyLUSRComponentHistory délègue au helper commun (cf. append_only_rebuild.go).
// Particularité : pas de written_at synthétique — `computed_at` (préservé tel quel
// par le CTAS) sert d'horloge. Le PostSwap restaure son DEFAULT (perdu par le CTAS),
// sur lequel le writer persister compte.
func applyAppendOnlyLUSRComponentHistory(db *sql.DB) error {
	return applyAppendOnlyRebuild(db, appendOnlyRebuild{
		Table: "lusr_component_history",
		IDSeq: "lch_seq",
		// SyntheticCols vide : computed_at d'origine préservé, pas de written_at.
		PostSwap: []string{
			`ALTER TABLE lusr_component_history ALTER COLUMN computed_at SET DEFAULT now()`,
			`CREATE INDEX IF NOT EXISTS idx_lch_component ON lusr_component_history(component_name)`,
			`CREATE INDEX IF NOT EXISTS idx_lch_match ON lusr_component_history(match_id)`,
		},
		ViewSQL: `CREATE OR REPLACE VIEW lusr_component_history_latest AS
			SELECT * FROM lusr_component_history
			QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, component_name ORDER BY computed_at DESC, id DESC) = 1`,
	})
}
