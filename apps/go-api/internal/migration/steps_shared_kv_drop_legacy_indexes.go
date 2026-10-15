// Package migration — steps_shared_kv_drop_legacy_indexes.go : `killer_victim_pairs` sans index.
//
// Les bases nees avant le schema Go portent trois index ART herites de la migration des donnees
// de reference (`idx_kv_match`, `idx_kv_killer`, `idx_kv_victim` ; `docs/archive/MIGRATION_REFDATA.md`).
// Aucun code du depot ne les cree, et la completion combat (`persist.EventsCompletionPersister`)
// reecrit la table par match en DELETE puis INSERT : sur une table indexee, ce DELETE est le
// declencheur du bug DuckDB ART #23645 (ADR 0019). Ils sont retires. (L entretien
// `repair-highlight-phantoms`, lui, reecrit par echange de table : `ReplaceKillerVictimPairs`.)
//
// AUCUN LECTEUR N EN DEPEND : les lectures filtrent par `match_id` (egalite ou liste) ou par
// xuid, et DuckDB les sert par balayage a un cout du meme ordre (mesure du 2026-10-15 sur une copie
// de 532 618 lignes, en ms par requete, avec puis sans : `match_id =` 0,38 / 0,49 ; liste de 200
// matchs 2,4 / 1,9 ; tueur ou victime 2,2 / 2,4).
//
// Idempotente (`DROP INDEX IF EXISTS`). Garde-rail : aucun index ne se cree sur la table
// (`TestKillerVictimPairsSansIndex`, `internal/sync/no_art_killer_victim_pairs_test.go`).
package migration

import "database/sql"

func init() {
	Register(Migration{
		Name:        "shared_killer_victim_pairs_drop_legacy_indexes_v1",
		TargetDB:    TargetShared,
		Description: "Retire les index ART herites de killer_victim_pairs (idx_kv_match, idx_kv_killer, idx_kv_victim) : la table se reecrit par DELETE + INSERT",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				DROP INDEX IF EXISTS idx_kv_match;
				DROP INDEX IF EXISTS idx_kv_killer;
				DROP INDEX IF EXISTS idx_kv_victim;
			`)
		},
	})
}
