package migration

// steps_player_drop_challenge_art_indexes.go — éradication ART table `challenge`
// (player DB, Prestige) — 2026-06-19.
//
// Trois index secondaires de `challenge` portent sur des colonnes MUTÉES par des
// UPDATE → surface ART (cf. crash canonique match_skill_rank, qui a prouvé qu'une
// player DB mono-writer peut FATAL-invalider sur un UPDATE/DELETE touchant un index ART) :
//   - idx_ch_user_status(user_id, status)  : PrestigeChallengeRepo.UpdateStatus mute `status`.
//   - idx_ch_arc(arc_id, position)         : DetachFromArc mute `arc_id` (UPDATE ... SET arc_id=NULL).
//   - idx_challenge_campaign(campaign_id)  : campaign_repo mute `campaign_id`.
// La table challenge est petite (challenges Prestige par joueur) → ces index n'apportent
// aucun gain mesurable et sont une pure surface de corruption. PK (id) gardée (jamais mutée).
// idx_ch_metric(metric) reste : `metric` n'est jamais muté.
//
// Idempotent (DROP INDEX IF EXISTS). Les migrations sources ne recréent plus ces index.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "drop_challenge_mutated_art_indexes_v1",
		TargetDB:    TargetPlayer,
		Description: "Retire idx_ch_user_status/idx_ch_arc/idx_challenge_campaign (colonnes status/arc_id/campaign_id mutées = surface ART player DB)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				DROP INDEX IF EXISTS idx_ch_user_status;
				DROP INDEX IF EXISTS idx_ch_arc;
				DROP INDEX IF EXISTS idx_challenge_campaign;
			`)
		},
	})
}
