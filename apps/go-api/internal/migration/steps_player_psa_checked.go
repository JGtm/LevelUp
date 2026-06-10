package migration

// steps_player_psa_checked.go — marqueur terminal de la convergence PSA.
//
// Ajoute `psa_checked_at` à player_match_enrichment : NULL = extraction des
// PersonalScores jamais tentée pour ce match ; non-NULL = JSON match fetché et
// extraction tentée (même si 0 award extractible). Sans ce marqueur, la
// convergence PSA (convergePSA, cf. internal/sync/convergence.go) re-fetcherait
// indéfiniment les matchs sans PersonalScores.
//
// Contexte : gate invariants 2026-06-10 — psa_missing confirmé pour les matchs
// delta-skippés (seul le traitement per-match écrivait les PSA).
//
// Les DBs neuves ont la colonne au bootstrap via playerSchemaSQL
// (sync/schema.go) — migration no-op sur DB neuve, additive sur DB existante.

import "database/sql"

func init() {
	Register(Migration{
		Name:     "player_match_enrichment_psa_checked_v1",
		TargetDB: TargetPlayer,
		Description: "Ajout de la colonne psa_checked_at à player_match_enrichment" +
			" (marqueur terminal de la convergence des personal_score_awards).",
		ApplySchema: func(db *sql.DB) error {
			_, err := db.ExecContext(bootCtx(),
				`ALTER TABLE player_match_enrichment ADD COLUMN IF NOT EXISTS psa_checked_at TIMESTAMP`,
			)
			return err
		},
	})
}
