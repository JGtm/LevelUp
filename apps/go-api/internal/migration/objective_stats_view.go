package migration

// objective_stats_view.go — DDL canonique de la vue match_objective_stats_latest
// (lecture append-only ADR 0026 : derniere version par (match_id, xuid), QUALIFY
// written_at DESC, id DESC).
//
// SOURCE UNIQUE du DDL — consommee par :
//   - games/halo_infinite/migrations/steps_shared_objective_stats.go (creation + extension) ;
//   - ops/snapshot_read.go (reconstruction du schema shared snapshot :memory:) ;
//   - les fixtures de test shared (sync/sync_pipeline_fixture_test.go,
//     platform/duckdb/player_repos_test.go).
//
// Contexte (2026-07-26, regression v7.2 scoreboard_empty) : le DDL existait en 4
// copies et une 5e allait naitre dans la lecture snapshot — regle projet « <= 2
// copies d'un meme pattern ». Garde-rail :
// archlint/no_inline_objective_latest_view_test.go interdit tout DDL inline de
// cette vue hors de ce fichier.

import "fmt"

// MatchObjectiveStatsLatestViewSQL retourne le DDL CREATE OR REPLACE VIEW de
// match_objective_stats_latest. tableRef est la relation source :
// "match_objective_stats" en production ; les fixtures de test peuvent la
// qualifier ("shared.match_objective_stats"). Pas de point-virgule final : le
// caller peut inserer le DDL dans un script multi-statements (ExecScript).
func MatchObjectiveStatsLatestViewSQL(tableRef string) string {
	return fmt.Sprintf(`CREATE OR REPLACE VIEW match_objective_stats_latest AS
	SELECT *
	FROM %s
	QUALIFY ROW_NUMBER() OVER (
		PARTITION BY match_id, xuid
		ORDER BY written_at DESC, id DESC
	) = 1`, tableRef)
}
