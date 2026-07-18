package migrations

import (
	"database/sql"

	"levelup/go-api/internal/migration"
)

// steps_player_prestige_source.go — plumbing du champ `source` (origine d'un défi)
// jusqu'à la télémétrie de calage coach (ADR 0020).
//
// Ajoute deux colonnes VARCHAR sur la player DB (stats.duckdb) :
//   - challenge.source           : origine persistée du défi ("user" | "pilot_mode" |
//     "coach"), écrite UNE FOIS à la création, JAMAIS mutée.
//   - prestige_telemetry.source  : recopie de l'origine sur chaque événement de
//     télémétrie (created + transitions), pour agréger
//     taux d'acceptation/complétion par origine.
//
// Anti-ART (#23046) : les deux colonnes sont non indexées et non mutées —
// challenge.source est figée à l'INSERT (comme title_slug/metric), prestige_telemetry
// est une table append-only INSERT-only (pas de vue _latest : chaque événement est une
// ligne distincte, il n'y a pas de re-INSERT d'une même ligne logique). Aucune surface
// d'index sur colonne mutée n'est créée.
//
// Backfill : les lignes historiques (créées avant cette migration) gardent source NULL —
// l'endpoint diag les agrège sous "unknown". Pas de backfill inventé.
//
// Ordre (canonicalOrder) : APRÈS create_prestige_player_schema (créateur des tables
// challenge + prestige_telemetry) et create_progression_player_schema.
func playerPrestigeSourceSteps() []migration.Migration {
	return []migration.Migration{
		{
			Name:        "prestige_add_source_columns_v1",
			TargetDB:    migration.TargetPlayer,
			Description: "Ajoute challenge.source + prestige_telemetry.source (origine du défi user/pilot_mode/coach) pour le calage coach (ADR 0020). NULL sur l'historique -> agrégé 'unknown'.",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE challenge ADD COLUMN IF NOT EXISTS source VARCHAR;
					ALTER TABLE prestige_telemetry ADD COLUMN IF NOT EXISTS source VARCHAR;
				`)
			},
		},
	}
}
