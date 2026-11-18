// Package migration — steps_shared_registry_kickoff_source.go : la source du coup d'envoi d'un
// match, rangée à côté de sa valeur (ADR 0049).
//
// `match_registry.real_start_source` dit d'où vient `real_start_time` : `film` (le coup d'envoi
// publié par le document de rejeu), `api` (l'estimation par les arrivées des joueurs), NULL
// (inconnu). Les deux colonnes s'écrivent ensemble, par l'insertion de la synchronisation ou par
// `persist.KickoffPersister`, jamais séparément.
//
// AJOUT DE COLONNE SEULEMENT : `match_registry` porte une clé primaire et des index, sous lesquels
// DuckDB refuse `ALTER TABLE ... DROP COLUMN` ; ADD COLUMN est accepté. La colonne n'est pas
// indexée : elle s'écrit par un `UPDATE ... WHERE match_id = ?` par match, la forme autorisée par
// `no_art_patterns_test.go`. Les lignes existantes restent NULL jusqu'au rattrapage de démarrage
// (`sync/kickoff.ApplyRevision`), que la révision `timeline.KickoffRev` déclenche.
package migration

import (
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Name:        "shared_registry_real_start_source_v1",
		TargetDB:    TargetShared,
		Description: "Colonne real_start_source sur match_registry (source du coup d'envoi : film, api, NULL = inconnu ; ADR 0049)",
		ApplySchema: applyRealStartSource,
	})
}

// applyRealStartSource ajoute la colonne. Idempotente (`IF NOT EXISTS`). Garde `match_registry`
// absente : la table est créée par `create_base_shared_schema`, ordonnée avant.
func applyRealStartSource(db *sql.DB) error {
	exists, err := tableExists(db, "match_registry")
	if err != nil {
		return fmt.Errorf("real_start_source: check table: %w", err)
	}
	if !exists {
		return nil
	}
	return execScript(db, `
		ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS real_start_source VARCHAR;
	`)
}
