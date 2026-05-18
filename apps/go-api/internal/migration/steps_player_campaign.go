package migration

// steps_player_campaign.go — migration ImprovementCampaign (V1 PlayerProfile §4.5).
//
// Tables ajoutées (par joueur) :
//   - improvement_campaign : campagne d'amélioration volontaire sur 1 axe
//
// Colonnes ajoutées (par joueur) :
//   - challenge.campaign_id : lien optionnel vers une campagne active
//
// Idempotent via CREATE TABLE IF NOT EXISTS / ALTER TABLE IF NOT EXISTS.
//
// Réf : .ai/PLAN_PLAYER_PROFILE_ASCENSION.md §4.5

import "database/sql"

func init() {
	Register(Migration{
		Name:        "create_improvement_campaign_schema",
		TargetDB:    TargetPlayer,
		Description: "Table improvement_campaign + challenge.campaign_id (V1 PlayerProfile §4.5)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS improvement_campaign (
					id                       VARCHAR PRIMARY KEY,
					user_id                  VARCHAR NOT NULL,
					title_slug               VARCHAR NOT NULL,
					axis                     VARCHAR NOT NULL,
					axis_kind                VARCHAR NOT NULL,
					started_at               TIMESTAMP NOT NULL,
					ended_at                 TIMESTAMP,
					status                   VARCHAR NOT NULL DEFAULT 'active',
					playlist_group           VARCHAR NOT NULL DEFAULT 'all',
					snapshot_value           DOUBLE NOT NULL,
					snapshot_sample          INTEGER NOT NULL DEFAULT 0,
					current_value_raw        DOUBLE,
					current_value_lowess     DOUBLE,
					matches_since_start      INTEGER NOT NULL DEFAULT 0,
					last_evaluated_at        TIMESTAMP,
					mann_whitney_p           DOUBLE,
					progression_confirmed    BOOLEAN NOT NULL DEFAULT FALSE,
					auto_closure_suggested   BOOLEAN NOT NULL DEFAULT FALSE,
					auto_closure_reason      VARCHAR
				);
				CREATE INDEX IF NOT EXISTS idx_campaign_user_title ON improvement_campaign(user_id, title_slug, status);

				ALTER TABLE challenge ADD COLUMN IF NOT EXISTS campaign_id VARCHAR;
				CREATE INDEX IF NOT EXISTS idx_challenge_campaign ON challenge(campaign_id);
			`)
		},
	})
}
