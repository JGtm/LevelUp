package migration

// steps_metadata_template_tagging.go — extension du schéma `challenge_template`
// pour le tagging V1 PlayerProfile (cf. PLAN_PLAYER_PROFILE_ASCENSION.md §5.1).
//
// Ajoute 3 colonnes :
//   - lusr_components VARCHAR : CSV des composantes LUSR ciblées
//     (ex: "kills_vs_expected,deaths_vs_expected"). Permet le matching
//     profil → suggestions de défis (Section C du profil).
//   - radar_axes VARCHAR : CSV des axes narrative 6 ciblés (optionnel).
//   - is_long_term BOOLEAN : true si le template encourage la progression
//     durable (window_type=rolling_days OU last_n_matches threshold).
//
// Idempotente via addColumnIfMissing.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "add_template_tagging_columns",
		TargetDB:    TargetMetadata,
		Description: "Ajoute lusr_components, radar_axes, is_long_term à challenge_template pour le tagging V1 PlayerProfile.",
		ApplySchema: func(db *sql.DB) error {
			if err := addColumnIfMissing(db, "challenge_template", "lusr_components", "VARCHAR"); err != nil {
				return err
			}
			if err := addColumnIfMissing(db, "challenge_template", "radar_axes", "VARCHAR"); err != nil {
				return err
			}
			if err := addColumnIfMissing(db, "challenge_template", "is_long_term", "BOOLEAN DEFAULT FALSE"); err != nil {
				return err
			}
			return nil
		},
	})
}
