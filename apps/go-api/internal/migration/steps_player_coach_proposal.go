package migration

// steps_player_coach_proposal.go — migration coach_advisor (Phase 3, ADR 0020).
//
// Table `coach_proposal` dans stats.duckdb (par joueur).
//
// Schéma exact défini dans l'ADR 0020 §"Schéma DuckDB" :
//   - Status : 'pending' (defaut) | 'accepted' | 'dismissed' | 'superseded' |
//     'obsoleted' | 'stale'
//   - expires_at est NULLABLE (pas d'expiration par âge — cf. ADR 0020 §3).
//     Cleanup piloté par supersession + obsolescence sur completion.
//
// Index :
//   - (user_id, title_slug, status) : query principale pour HTTP GET pending
//   - (user_id, title_slug, source_metric, radar_axis) : query supersession

import "database/sql"

func init() {
	Register(Migration{
		Name:        "create_coach_proposal_player_schema",
		TargetDB:    TargetPlayer,
		Description: "Table coach_proposal pour le pont coach_advisor → Prestige (ADR 0020)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS coach_proposal (
					id                    VARCHAR PRIMARY KEY,
					user_id               VARCHAR NOT NULL,
					title_slug            VARCHAR NOT NULL,
					kind                  VARCHAR NOT NULL,
					template_id           VARCHAR,
					challenges_spec_json  VARCHAR,
					suggested_tier        VARCHAR,
					source_signal         VARCHAR NOT NULL,
					source_metric         VARCHAR,
					radar_axis            VARCHAR,
					strength              DOUBLE,
					origin                VARCHAR NOT NULL,
					reason_key_en         VARCHAR,
					reason_key_fr         VARCHAR,
					reason_params         VARCHAR,
					status                VARCHAR NOT NULL DEFAULT 'pending',
					created_at            TIMESTAMP NOT NULL,
					expires_at            TIMESTAMP,
					resolved_at           TIMESTAMP,
					resolved_ref          VARCHAR,
					superseded_by         VARCHAR,
					superseded_at         TIMESTAMP,
					obsoleted_at          TIMESTAMP
				);
				CREATE INDEX IF NOT EXISTS idx_coach_proposal_user_status
					ON coach_proposal(user_id, title_slug, status);
				CREATE INDEX IF NOT EXISTS idx_coach_proposal_metric_axis
					ON coach_proposal(user_id, title_slug, source_metric, radar_axis);
			`)
		},
	})
}
