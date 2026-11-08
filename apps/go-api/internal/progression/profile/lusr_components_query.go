package profile

import (
	"context"
	"fmt"

	"levelup/go-api/internal/platform/duckdb"
)

// lusrComponentsBaseSQL agrège les composantes LUSR ; %s reçoit le filtre des matchs exclus
// (vide, ou `WHERE match_id NOT IN (?, ...)`).
const lusrComponentsBaseSQL = `
	WITH ranked AS (
		SELECT
			component_name,
			value,
			ROW_NUMBER() OVER (PARTITION BY component_name ORDER BY computed_at DESC) AS rk_desc,
			ROW_NUMBER() OVER (PARTITION BY component_name ORDER BY computed_at ASC)  AS rk_asc,
			COUNT(*)    OVER (PARTITION BY component_name)                            AS n
		FROM lusr_component_history_latest
		%s
	)
	SELECT
		component_name,
		AVG(value)                                         AS current_avg,
		QUANTILE_CONT(value, 0.8)                          AS top20,
		AVG(CASE WHEN n >= 20 AND rk_desc <= 10 THEN value END) AS last_avg,
		AVG(CASE WHEN n >= 20 AND rk_asc  <= 10 THEN value END) AS first_avg,
		MAX(n)                                             AS n
	FROM ranked
	GROUP BY component_name
`

// lusrComponentsQuery construit la requête des composantes et ses arguments. Les matchs
// exclus sont passés en paramètres constants (jamais en sous-requête, ADR 0036) : la
// fenêtre glissante et le comptage des 20 matchs de la tendance portent sur les matchs
// retenus seulement. Liste vide : aucune clause.
func lusrComponentsQuery(excluded []string) (string, []any) {
	if len(excluded) == 0 {
		return fmt.Sprintf(lusrComponentsBaseSQL, ""), nil
	}
	clause := "WHERE match_id NOT IN (" + duckdb.Placeholders(len(excluded)) + ")"
	return fmt.Sprintf(lusrComponentsBaseSQL, clause), duckdb.ToAnySlice(excluded)
}

// loadExcludedMatchIDs lit les identifiants des matchs exclus du joueur. Une erreur est
// remontée avec son contexte : continuer sans elle ferait peser des matchs exclus.
func (s *Service) loadExcludedMatchIDs(ctx context.Context) ([]string, error) {
	rows, err := s.db.Query(ctx,
		`SELECT match_id FROM player_match_enrichment_latest WHERE COALESCE(is_excluded, FALSE)`)
	if err != nil {
		return nil, fmt.Errorf("profile: lecture des matchs exclus: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("profile: lecture des matchs exclus (scan): %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("profile: lecture des matchs exclus (lignes): %w", err)
	}
	return ids, nil
}
