package migration

// steps_player_lusr_components.go — migration historique des 8 composantes LUSR
// (V2 PlayerProfile §1 — follow-up de V1 commit-4).
//
// Table ajoutée (par joueur) : lusr_component_history.
//
// Alimentée :
//   - Live : par sync.upsertLUSRRatings à chaque match raté (V2 commit-1).
//   - Backfill : par re-run de ComputeSkillRatingsBatch en mode force.
//
// Consommée :
//   - profile.loadLUSRComponentsBreakdown (Section B breakdown 8 composantes)
//   - campaign.axisValueExpression (axes lusr_component.*)
//   - progression/streaks (médianes personnelles pour streaks perf-based — V2 commit-4)
//
// PK composite (match_id, component_name) — 1 ligne par (match, composante).
// Schema indexé sur component_name pour les agrégats moyenne/top20% rapides.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "create_lusr_component_history",
		TargetDB:    TargetPlayer,
		Description: "Table lusr_component_history (V2 §1 — alimentation live + backfill)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS lusr_component_history (
					match_id        VARCHAR NOT NULL,
					component_name  VARCHAR NOT NULL,
					value           DOUBLE  NOT NULL,
					weight          DOUBLE  NOT NULL,
					computed_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					PRIMARY KEY (match_id, component_name)
				);
				CREATE INDEX IF NOT EXISTS idx_lch_component ON lusr_component_history(component_name);
				CREATE INDEX IF NOT EXISTS idx_lch_match ON lusr_component_history(match_id);
			`)
		},
	})
}
