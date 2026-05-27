package migration

// steps_shared_add_participation_info.go — Mini-Phase 0.5 du chantier LUSR v2.
//
// Capture des 4 booleans `ParticipationInfo` retournés par l'API Halo
// (`PresentAtBeginning`, `PresentAtCompletion`, `JoinedInProgress`,
// `LeftInProgress`). Le wrapper Go `internal/openspartan/models.go` les
// parse déjà depuis le JSON ; le mapper et le sync les jetaient.
//
// Une fois capturés, ces booleans permettront au LUSR v2 Phase 3
// (quit penalty TS2 §9) de distinguer un quitter d'un late-joiner :
//   - quit_real = PresentAtBeginning && !PresentAtCompletion
//   - late_join = JoinedInProgress
//
// Les colonnes sont NULL pour tous les matchs déjà syncés (pas de backfill
// possible sans re-fetcher les JSON depuis l'API). Les futurs matchs et les
// matchs re-syncés via delta auront la donnée.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "shared_add_participation_info_booleans",
		TargetDB:    TargetShared,
		Description: "ParticipationInfo booleans sur match_participants pour LUSR v2 §9 quit penalty",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS present_at_beginning BOOLEAN;
				ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS present_at_completion BOOLEAN;
				ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS joined_in_progress BOOLEAN;
				ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS left_in_progress BOOLEAN;
			`)
		},
	})
}
