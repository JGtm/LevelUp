package migration

// steps_shared_add_participation_timestamps.go — Phase 3-quit timestamps
// (suite de Mini-Phase 0.5).
//
// Capture les 2 timestamps absolus retournés par l'API Halo dans
// `ParticipationInfo` :
//   - FirstJoinedTime : moment où le joueur a rejoint le match
//   - LastLeaveTime   : moment où le joueur a quitté (null = encore présent à la fin)
//
// Combinés avec les 4 booleans existants, ils permettent :
//   - d'ordonner précisément les quitters (1er parti = LastLeaveTime ASC) au
//     lieu du proxy time_played_seconds
//   - de vérifier "a-t-il quitté avant la fin ?" via
//     LastLeaveTime IS NOT NULL AND LastLeaveTime < match_registry.end_time
//
// Les colonnes sont NULL pour tous les matchs déjà syncés (pas de backfill
// automatique au boot ; un cmd dédié `backfill_quit_timestamps` re-fetche
// les JSONs si l'API les a encore).

import "database/sql"

func init() {
	Register(Migration{
		Name:        "shared_add_participation_timestamps",
		TargetDB:    TargetShared,
		Description: "ParticipationInfo timestamps (FirstJoinedTime, LastLeaveTime) sur match_participants pour LUSR v2 quit ordering",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS first_joined_time TIMESTAMPTZ;
				ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS last_leave_time TIMESTAMPTZ;
			`)
		},
	})
}
