// Package migration — steps_shared_registry_kickoff_source.go : la source du coup d'envoi d'un
// match, rangée à côté de sa valeur (ADR 0050).
//
// `match_registry.real_start_source` dit d'où vient `real_start_time` : `film` (le coup d'envoi
// publié par le document de rejeu), `api` (l'estimation par les arrivées des joueurs), NULL
// (inconnu). Les deux colonnes s'écrivent ensemble, par l'insertion de la synchronisation ou par
// `persist.KickoffPersister`, jamais séparément.
//
// AJOUT DE COLONNE SEULEMENT : `match_registry` porte une clé primaire et des index, sous lesquels
// DuckDB refuse `ALTER TABLE ... DROP COLUMN` ; ADD COLUMN est accepté. La colonne n'est pas
// indexée : elle s'écrit par un `UPDATE ... WHERE match_id = ?` par match, la forme autorisée par
// `no_art_patterns_test.go`.
//
// LA COLONNE NAÎT JUSTE. Un coup d'envoi déjà reporté depuis un rejeu porte `t0_quality =
// 'film_movement'` : ces lignes reçoivent `real_start_source = 'film'` dans la même étape, par un UPDATE
// paramétré par match (jamais un UPDATE en masse). Sans cela, le premier rattrapage
// (`sync/kickoff.ApplyRevision`) ne saurait pas qu'un match dont le rejeu a disparu porte un coup
// d'envoi du film, et le remplacerait par l'estimation. Les autres lignes restent NULL jusqu'à ce
// rattrapage, que la révision `timeline.KickoffRev` déclenche.
//
// LA VUE `v_match_full` (`SELECT mr.*`, la seule vue étoilée sur le registre) FIGE SON ÉTOILE À SA
// CRÉATION : l'étape la recrée par son DDL canonique (VMatchFullViewSQL, comme
// `refresh_views_after_team_rounds`) quand elle existe, pour qu'elle expose la colonne.
package migration

import (
	"context"
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Name:        "shared_registry_real_start_source_v1",
		TargetDB:    TargetShared,
		Description: "Colonne real_start_source sur match_registry (source du coup d'envoi : film, api, NULL = inconnu ; ADR 0050), amorcee a film la ou t0_quality = film_movement, puis v_match_full recreee",
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
	if err := execScript(db, `
		ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS real_start_source VARCHAR;
	`); err != nil {
		return err
	}
	if err := seedFilmSource(db); err != nil {
		return err
	}
	return refreshMatchFullView(db)
}

// refreshMatchFullView recrée `v_match_full` quand la base la porte (une base qui ne l'a pas encore la
// recevra de l'étape qui la crée, avec toutes ses colonnes).
func refreshMatchFullView(db *sql.DB) error {
	var n int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM information_schema.tables
		WHERE table_name = 'v_match_full' AND table_type = 'VIEW'`).Scan(&n); err != nil {
		return fmt.Errorf("real_start_source: check v_match_full: %w", err)
	}
	if n == 0 {
		return nil
	}
	if err := execScript(db, VMatchFullViewSQL); err != nil {
		return fmt.Errorf("real_start_source: recreate v_match_full: %w", err)
	}
	return nil
}

// seedFilmSource pose `real_start_source = 'film'` sur les matchs dont le coup d'envoi vient déjà d'un
// rejeu (`t0_quality = 'film_movement'`). Une base sans `t0_quality` (titre qui ne l'a jamais reçue,
// base neuve) n'a rien à amorcer.
func seedFilmSource(db *sql.DB) error {
	has, err := columnExists(db, "match_registry", "t0_quality")
	if err != nil {
		return fmt.Errorf("real_start_source: check t0_quality: %w", err)
	}
	if !has {
		return nil
	}
	ctx := context.Background()
	rows, err := db.QueryContext(ctx, `SELECT match_id FROM match_registry
		WHERE t0_quality = 'film_movement' AND real_start_source IS NULL ORDER BY match_id`)
	if err != nil {
		return fmt.Errorf("real_start_source: lecture des coups d'envoi du film: %w", err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return fmt.Errorf("real_start_source: lecture (scan): %w", err)
		}
		ids = append(ids, id)
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("real_start_source: lecture: %w", err)
	}
	return updateEachSource(ctx, db, ids)
}

// updateEachSource écrit la source `film` match par match, dans une transaction.
func updateEachSource(ctx context.Context, db *sql.DB, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("real_start_source: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx,
			`UPDATE match_registry SET real_start_source = ? WHERE match_id = ?`, "film", id); err != nil {
			return fmt.Errorf("real_start_source: amorce %s: %w", id, err)
		}
	}
	return tx.Commit()
}
