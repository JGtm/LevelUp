package migrations

import (
	"database/sql"

	"levelup/go-api/internal/migration"
)

// sharedObjectiveStatsSteps — table match_objective_stats (shared-core, schéma commun
// inter-titres). Stats objectifs par joueur par match des modes à objectif Halo Infinite :
// CTF (CaptureTheFlagStats), Zones (ZonesStats — Strongholds ET KOTH), Oddball
// (OddballStats). 1 ligne par (match_id, xuid) ; blocs mutuellement exclusifs par mode →
// table LARGE nullable (les colonnes du mode absent restent NULL). Agrégats équipe/lobby
// calculés à la LECTURE (SUM par team_id via JOIN match_participants) — pas de pré-agrégat.
//
// CRÉÉE DIRECTEMENT en forme APPEND-ONLY (id PK séquence + written_at + vue _latest) —
// PAS via ApplyAppendOnlyRebuild (recette de CONVERSION d'une table mutable existante).
// Modèle : create_world_csr_leaderboard_snapshots (steps.go). ART-safe par construction
// (#23046) : écritures = INSERT pur (persist.persistObjectiveStats), lecture via la vue
// match_objective_stats_latest UNIQUEMENT (une lecture brute sert des lignes périmées,
// ADR 0026). Seul index = match_id (jamais muté) — parité idx_pve_match.
//
// Halo 5 : les colonnes objectif ne sont pas exposées (capability match.objective.stats =
// not_exposed) — la table reste vide pour ce titre au lancement. Colonnes verrouillées sur
// payload réel GetMatchStats (P0, PLAN_V72_OBJECTIVE_STATS.md § « P0 — Mapping figé »).
func sharedObjectiveStatsSteps() []migration.Migration {
	return []migration.Migration{
		{
			Name:        "shared_create_objective_stats",
			TargetDB:    migration.TargetShared,
			Description: "Table match_objective_stats (append-only : id PK seq + written_at + vue _latest + index match_id) — stats objectifs CTF/Zones/Oddball par joueur/match, INSERT-only ART-safe",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE SEQUENCE IF NOT EXISTS match_objective_stats_seq START 1;

					CREATE TABLE IF NOT EXISTS match_objective_stats (
						id                                    BIGINT PRIMARY KEY DEFAULT nextval('match_objective_stats_seq'),
						match_id                              VARCHAR NOT NULL,
						xuid                                  VARCHAR NOT NULL,
						-- CTF (CaptureTheFlagStats)
						flag_captures                         INTEGER,
						flag_capture_assists                  INTEGER,
						flag_grabs                            INTEGER,
						flag_secures                          INTEGER,
						flag_steals                           INTEGER,
						flag_returns                          INTEGER,
						flag_carriers_killed                  INTEGER,
						flag_returners_killed                 INTEGER,
						kills_as_flag_carrier                 INTEGER,
						kills_as_flag_returner                INTEGER,
						time_as_flag_carrier_seconds          DOUBLE,
						-- Zones (ZonesStats — Strongholds + KOTH ; zone_scoring_ticks>0 distingue KOTH)
						zone_captures                         INTEGER,
						zone_secures                          INTEGER,
						zone_offensive_kills                  INTEGER,
						zone_defensive_kills                  INTEGER,
						zone_scoring_ticks                    INTEGER,
						time_in_zones_seconds                 DOUBLE,
						-- Oddball (OddballStats)
						kills_as_skull_carrier                INTEGER,
						skull_carriers_killed                 INTEGER,
						skull_grabs                           INTEGER,
						skull_scoring_ticks                   INTEGER,
						time_as_skull_carrier_seconds         DOUBLE,
						longest_time_as_skull_carrier_seconds DOUBLE,
						written_at                            TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					);

					CREATE INDEX IF NOT EXISTS idx_match_objective_stats_match
						ON match_objective_stats(match_id);

					CREATE OR REPLACE VIEW match_objective_stats_latest AS
						SELECT *
						FROM match_objective_stats
						QUALIFY ROW_NUMBER() OVER (
							PARTITION BY match_id, xuid
							ORDER BY written_at DESC, id DESC
						) = 1;
				`)
			},
		},
	}
}
