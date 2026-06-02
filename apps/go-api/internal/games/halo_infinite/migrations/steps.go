// Package migrations regroupe les migrations DDL APPARTENANT à Halo Infinite
// (Phase 1.5.1 voie B, ADR 0025). Elles vivaient dans internal/migration/
// (couplées au registre global via init()) ; elles en sortent ici, fournies au
// runner via migration.SetTitleStepsProvider(StepsFor).
//
// Construites avec les helpers exportés du package migration (TableExists,
// AddColumnIfMissing, ExecScript, …). L'ordre d'exécution reste imposé par
// migration.canonicalOrder (order.go) — déplacer un step ici ne le réordonne
// pas.
//
// État transition : Steps() se remplit au fur et à mesure des déplacements
// (b3). Vide = aucun step encore migré (le registre global legacy fournit
// tout, comportement inchangé).
package migrations

import (
	"database/sql"

	"levelup/go-api/internal/migration"
)

// Steps retourne toutes les migrations title-owned de Halo Infinite, tous
// targets confondus. Se remplit au fur et à mesure des déplacements (b3) ;
// chaque entrée doit être listée dans migration.canonicalOrder (vérifié par
// order_audit_test.go).
func Steps() []migration.Migration {
	return []migration.Migration{
		// Déplacé depuis internal/migration/steps_shared_pve.go (b3 pilote).
		{
			Name:        "add_pve_schema",
			TargetDB:    migration.TargetSharedPvE,
			Description: "Table pve_match_stats pour stats Firefight",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS pve_match_stats (
						match_id            VARCHAR NOT NULL,
						xuid                VARCHAR NOT NULL,
						waves_completed     INTEGER,
						boss_kills          INTEGER,
						grunt_kills         INTEGER,
						elite_kills         INTEGER,
						jackal_kills        INTEGER,
						brute_kills         INTEGER,
						hunter_kills        INTEGER,
						skimmer_kills       INTEGER,
						crawler_kills       INTEGER,
						soldier_kills       INTEGER,
						knight_kills        INTEGER,
						warden_kills        INTEGER,
						total_kills         INTEGER,
						deaths              INTEGER,
						created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (match_id, xuid)
					);
					CREATE INDEX IF NOT EXISTS idx_pve_match ON pve_match_stats(match_id);
					CREATE INDEX IF NOT EXISTS idx_pve_xuid ON pve_match_stats(xuid);
				`)
			},
		},
		// Déplacés depuis internal/migration/steps_shared_*.go (b3 batch Shared).
		{
			Name:        "shared_add_t0_quality",
			TargetDB:    migration.TargetShared,
			Description: "Colonne t0_quality sur match_registry + repurpose real_start_time en début gameplay UTC (Match Timeline T0 Phase 2)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS t0_quality VARCHAR;
				`)
			},
		},
		{
			Name:        "shared_add_participation_info_booleans",
			TargetDB:    migration.TargetShared,
			Description: "ParticipationInfo booleans sur match_participants pour LUSR v2 §9 quit penalty",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS present_at_beginning BOOLEAN;
					ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS present_at_completion BOOLEAN;
					ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS joined_in_progress BOOLEAN;
					ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS left_in_progress BOOLEAN;
				`)
			},
		},
		{
			Name:        "shared_add_participation_timestamps",
			TargetDB:    migration.TargetShared,
			Description: "ParticipationInfo timestamps (FirstJoinedTime, LastLeaveTime) sur match_participants pour LUSR v2 quit ordering",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS first_joined_time TIMESTAMPTZ;
					ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS last_leave_time TIMESTAMPTZ;
				`)
			},
		},
		{
			Name:        "add_shared_match_csrs",
			TargetDB:    migration.TargetShared,
			Description: "Table shared.match_csrs : CSR par-match par-joueur (capture all participants)",
			ApplySchema: func(db *sql.DB) error {
				return migration.ExecScript(db, `
					CREATE TABLE IF NOT EXISTS match_csrs (
						match_id                     VARCHAR NOT NULL,
						xuid                         VARCHAR NOT NULL,
						rating_type                  VARCHAR NOT NULL DEFAULT 'CSR',
						rating_value                 FLOAT,
						tier                         VARCHAR,
						sub_tier                     SMALLINT DEFAULT 0,
						tier_label                   VARCHAR,
						rating_delta                 FLOAT,
						measurement_matches_remaining INTEGER DEFAULT 0,
						season_id                    VARCHAR,
						created_at                   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						updated_at                   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (match_id, xuid)
					);
					CREATE INDEX IF NOT EXISTS idx_match_csrs_xuid    ON match_csrs(xuid);
					CREATE INDEX IF NOT EXISTS idx_match_csrs_season  ON match_csrs(season_id);
					CREATE INDEX IF NOT EXISTS idx_match_csrs_match   ON match_csrs(match_id);
				`)
			},
		},
		// Déplacé depuis internal/migration/steps_shared_seed_tier_boundaries_v2.go.
		{
			Name:        "shared_seed_tier_boundaries_v2",
			TargetDB:    migration.TargetShared,
			Description: "LUSR v2 Phase 3e v2 — seed des tier_boundary_* (Bronze..Onyx) dans lusr_hyperparams_v2",
			ApplySchema: func(db *sql.DB) error {
				return nil // table déjà créée par shared_create_skill_v2_tables (reste dans le registre global)
			},
			ApplyBackfill: func(db *sql.DB) error {
				boundaries := []struct {
					name  string
					value float64
				}{
					{"tier_boundary_bronze", 0.0},
					{"tier_boundary_silver", 21.0},
					{"tier_boundary_gold", 22.0},
					{"tier_boundary_platinum", 25.0},
					{"tier_boundary_diamond", 25.8},
					{"tier_boundary_onyx", 27.0},
				}
				groups := []string{"arena_slayer", "arena_objectif", "btb", "chaos"}
				for _, g := range groups {
					for _, b := range boundaries {
						if _, err := db.ExecContext(migration.BootCtx(), `
							INSERT INTO lusr_hyperparams_v2 (playlist_group, name, value, source)
							SELECT ?, ?, ?, 'phase_3e_v2_default'
							WHERE NOT EXISTS (
								SELECT 1 FROM lusr_hyperparams_v2
								WHERE playlist_group = ? AND name = ?
							)`,
							g, b.name, b.value, g, b.name); err != nil {
							return err
						}
					}
				}
				return nil
			},
		},
	}
}

// StepsFor filtre Steps() par target — c'est la fonction enregistrée comme
// provider via migration.SetTitleStepsProvider.
func StepsFor(target migration.TargetDB) []migration.Migration {
	all := Steps()
	if len(all) == 0 {
		return nil
	}
	out := make([]migration.Migration, 0, len(all))
	for _, m := range all {
		if m.TargetDB == target {
			out = append(out, m)
		}
	}
	return out
}
