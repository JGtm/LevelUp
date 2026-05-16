package migration

// steps_player_perf_chain.go — migration pour le découplage du performance_score
// par chaîne de playlist (cf. plan "comme le lusr, découpler le score de perf").
//
// Ajoute la colonne `performance_chain` à player_match_enrichment (idempotent
// via ADD COLUMN IF NOT EXISTS, supporté DuckDB ≥ 0.10). Les DBs neuves ont déjà
// la colonne au bootstrap via playerSchemaSQL (sync/schema.go) — cette migration
// est donc no-op sur DB neuve, et l'ajoute sur les DBs pré-existantes.
//
// Le backfill du contenu (peuplement par chaîne) est piloté par
// batchComputePerformanceScores (cf. internal/sync/performance.go) déclenché
// via `levelup backfill --force-performance-scores` post-merge.

import "database/sql"

func init() {
	Register(Migration{
		Name:     "player_match_enrichment_performance_chain_v1",
		TargetDB: TargetPlayer,
		Description: "Ajout de la colonne performance_chain à player_match_enrichment" +
			" (découplage du score de performance par chaîne de playlist, 6 chaînes).",
		ApplySchema: func(db *sql.DB) error {
			_, err := db.Exec(
				`ALTER TABLE player_match_enrichment ADD COLUMN IF NOT EXISTS performance_chain VARCHAR`,
			)
			return err
		},
	})
}
