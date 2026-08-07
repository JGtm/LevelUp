package migration

// steps_shared_objective_score.go — table v3 (film) pour la TIMELINE de score
// per-équipe des modes à objectif (Strongholds, KOTH).
//
// Net-new : aucune table existante touchée. Produite par
// internal/analysis/objectivescore.DecodeScoreTimeline depuis les chunks film
// (paquet TYPE_2, ancre bit-level 0x7B6). 1 ligne = 1 point de timeline
// (granularité ~20s du snapshot type-2). PK (match_id, time_ms) → idempotent
// par DELETE-then-INSERT par match (cf. ObjectiveScoreRepo, miroir
// ObjectiveEventsRepo).
//
// source/confidence tracent la provenance + la précision : Strongholds calibré
// sur le final DB ('approx'), KOTH variante B (meters/5, exact bas-score) ou A
// (points, approx). Voir objectivescore/decode.go pour les faits validés.
//
// Écriture HORS chemin live (backfill diagnostic sérialisé, MaxOpenConns(1)) →
// pas de pression concurrente ART (cf. .ai/PLAN_WEAPON_ATTRIBUTION_V3.md §10).

import "database/sql"

func init() {
	Register(Migration{
		Name:        "shared_objective_score_v1",
		TargetDB:    TargetShared,
		Description: "Table match_objective_score_timeline (timeline score per-équipe Strongholds/KOTH, v3 film)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS match_objective_score_timeline (
					match_id     VARCHAR NOT NULL,
					time_ms      INTEGER NOT NULL,
					team_0_score INTEGER,
					team_1_score INTEGER,
					source       VARCHAR,
					confidence   VARCHAR,
					written_at   TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
					PRIMARY KEY (match_id, time_ms)
				);
			`)
		},
	})
}
