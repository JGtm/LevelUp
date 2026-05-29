package migration

// steps_shared_add_t0_quality.go — Phase 2 du refactor Match Timeline T0.
//
// Repurpose de la colonne existante `real_start_time` (jusqu'ici inutile :
// offset 0ms pour tous les matchs, calculée via start_time + (duration -
// playable_duration) — formule cassée pour Ranked Arena). Elle stocke désormais
// le DÉBUT RÉEL DU GAMEPLAY en UTC = start_time_utc + T0 (countdown pré-match).
//
// T0 est dérivable à la lecture :
//   t0_ms = epoch_ms(real_start_time AT TIME ZONE 'UTC') - epoch_ms(start_time_utc)
//
// La nouvelle colonne `t0_quality` trace la fiabilité du calcul (ok /
// single_source / spread_high / no_data / negative / suspicious_high — cf.
// analysis/timeline.T0Quality). Les rejets (negative, suspicious_high, no_data)
// laissent real_start_time NULL → fallback runtime T0=0.
//
// Pas de backfill automatique au boot : le cmd `backfill_t0` calcule et écrit.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "shared_add_t0_quality",
		TargetDB:    TargetShared,
		Description: "Colonne t0_quality sur match_registry + repurpose real_start_time en début gameplay UTC (Match Timeline T0 Phase 2)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS t0_quality VARCHAR;
			`)
		},
	})
}
