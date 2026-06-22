package migration

// steps_player_drop_coach_proposal_status_index.go — éradication ART table
// `coach_proposal` (player DB, coach_advisor ADR 0020) — 2026-06-20.
//
// idx_coach_proposal_user_status(user_id, title_slug, status) porte sur `status`,
// muté par MarkAccepted/MarkDismissed/MarkSuperseded/MarkObsoleted (UPDATE SET status)
// → surface ART DuckDB #23046 (vecteur UPDATE sur colonne indexée ; prouvé par le crash
// canonique match_skill_rank, qui a démontré qu'une player DB mono-writer peut FATAL-
// invalider sur un UPDATE touchant un index ART). La table coach_proposal est minuscule
// (quelques propositions par joueur) → l'index n'apporte aucun gain mesurable ; un scan
// complet est instantané.
//
// idx_coach_proposal_metric_axis(user_id, title_slug, source_metric, radar_axis) reste :
// ni source_metric ni radar_axis ne sont jamais mutés. PK (id) gardée (jamais mutée).
//
// Idempotent (DROP INDEX IF EXISTS). create_coach_proposal_player_schema ne recrée plus
// cet index ; le garde-fou metadata_art_surface_guard_test interdit sa réintroduction.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "drop_coach_proposal_status_art_index_v1",
		TargetDB:    TargetPlayer,
		Description: "Retire idx_coach_proposal_user_status (colonne status mutée par MarkAccepted/Dismissed/Superseded/Obsoleted = surface ART player DB)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `DROP INDEX IF EXISTS idx_coach_proposal_user_status;`)
		},
	})
}
