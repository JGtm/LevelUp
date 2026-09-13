//go:build cgo

package main

// diag.go — diagnostic et réparation des index ART de `match_skill_rank`.
//
// LA RÈGLE DE COMPARAISON N'EST PAS ICI. Elle vit dans
// `internal/platform/duckdb/indexcheck` (scan forcé par expression de clé vs
// lookup indexé par colonnes nues), et la CARTE DES AXES avec elle
// (`indexcheck.MatchSkillRankAxes`). Cet outil et la sonde data-health
// périodique (`internal/scheduler/data_health_msr_index.go`) en sont les deux
// consommateurs : recopier l'une ou l'autre les ferait diverger en silence dès
// qu'une migration ajoute un index. Garde-rail :
// `internal/archlint/no_local_msr_axes_test.go`.
//
// CE QUI RESTE ICI est le propre de l'OUTIL : la décision des index à
// reconstruire et la réparation elle-même.
//
// DDL DE RÉPARATION : jamais recopiée. Elle est CAPTURÉE dans la base
// (`duckdb_indexes().sql`) avant le DROP, puis rejouée — c'est par construction
// l'index que les migrations ont posé (autorité :
// games/halo_infinite/migrations/steps_player_match_skill_rank.go). Une DDL
// recopiée dans un outil dérive en silence dès que la migration évolue.

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/platform/duckdb/indexcheck"
)

// maxSampledKeys borne le nombre de clés sondées par axe. L'axe du triplet porte
// ~1 clé par ligne (des milliers) ; au-delà de cette borne le diagnostic reste
// représentatif et le rapport dit explicitement qu'il a été tronqué. L'outil
// prend les PREMIÈRES clés (ordre déterministe), pas un tirage : un diagnostic
// manuel doit être reproductible d'une exécution à l'autre.
const maxSampledKeys = 5000

// msrAxes — la carte partagée, lue une fois (copie défensive côté paquet).
var msrAxes = indexcheck.MatchSkillRankAxes()

// diagOptions — les options de passe de CET outil.
func diagOptions() indexcheck.Options {
	return indexcheck.Options{
		Table:   indexcheck.MatchSkillRankTable,
		MaxKeys: maxSampledKeys,
		Sample:  false,
	}
}

// diagnoseAll passe les trois axes et retourne les rapports dans l'ordre.
func diagnoseAll(ctx context.Context, db *sql.DB) ([]indexcheck.Report, error) {
	return indexcheck.RunAll(ctx, db, msrAxes, diagOptions())
}

// indexesToRebuild — union (ordonnée) des index des axes en écart.
func indexesToRebuild(reports []indexcheck.Report) []string {
	return indexcheck.IndexesToRebuild(reports, msrAxes)
}

// captureIndexDDL relève, dans la base, la DDL des index de match_skill_rank.
// C'est l'autorité : l'instruction rejouée est celle que les migrations ont posée.
func captureIndexDDL(ctx context.Context, db *sql.DB) (map[string]string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT index_name, sql FROM duckdb_indexes()
		 WHERE table_name = 'match_skill_rank' AND sql IS NOT NULL ORDER BY index_name`)
	if err != nil {
		return nil, fmt.Errorf("duckdb_indexes(): %w", err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var name, ddl string
		if err := rows.Scan(&name, &ddl); err != nil {
			return nil, fmt.Errorf("duckdb_indexes() scan: %w", err)
		}
		out[name] = ddl
	}
	return out, rows.Err()
}

// repairIndexes reconstruit les index nommés : DROP puis CREATE avec la DDL
// capturée. Aucune ligne de données n'est touchée — DDL d'index uniquement,
// jamais de DELETE ni d'UPDATE (ce serait le vecteur ART lui-même).
func repairIndexes(ctx context.Context, db *sql.DB, names []string) error {
	ddls, err := captureIndexDDL(ctx, db)
	if err != nil {
		return err
	}
	for _, name := range names {
		ddl, ok := ddls[name]
		if !ok {
			return fmt.Errorf("DDL introuvable pour l'index %q dans duckdb_indexes() — "+
				"index absent de cette base ? autorité : "+
				"games/halo_infinite/migrations/steps_player_match_skill_rank.go", name)
		}
		if _, err := db.ExecContext(ctx, `DROP INDEX IF EXISTS `+name); err != nil {
			return fmt.Errorf("DROP INDEX %s: %w", name, err)
		}
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("CREATE INDEX %s (%s): %w", name, ddl, err)
		}
	}
	// CHECKPOINT : rend la DDL durable dans le fichier plutôt que dans le seul WAL.
	if _, err := db.ExecContext(ctx, `CHECKPOINT`); err != nil {
		return fmt.Errorf("CHECKPOINT après réparation: %w", err)
	}
	return nil
}

// existingIndexes liste les index présents sur match_skill_rank.
func existingIndexes(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT index_name FROM duckdb_indexes() WHERE table_name = 'match_skill_rank' ORDER BY index_name`)
	if err != nil {
		return nil, fmt.Errorf("duckdb_indexes(): %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("duckdb_indexes() scan: %w", err)
		}
		out = append(out, name)
	}
	return out, rows.Err()
}
