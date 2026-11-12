package profilerepo

// player.go — les lectures du profil qui portent sur la base du joueur (stats.duckdb) : note LUSR et sa
// série, awards, composantes de performance.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync/skill"
)

// LatestLUSR lit le dernier rating LUSR + count total.
func (r *Repo) LatestLUSR(ctx context.Context) (port.ProfileLUSR, error) {
	var (
		lusr        port.ProfileLUSR
		rating      sql.NullFloat64
		deviation   sql.NullFloat64
		lastMatchAt sql.NullTime
	)
	// Lecture via la vue _latest (1 row/match, dédup append-only) ordonnée par
	// written_at : match_skill_rank.start_time est 100% NULL sur les rows LUSR
	// (aucun writer ne le renseigne — seul CSR l'a), donc ORDER BY start_time
	// renvoyait une row arbitraire. written_at (NOT NULL) donne le rating le plus
	// récemment écrit = LUSR courant. lastMatchAt = COALESCE(start_time, written_at).
	err := r.pdb.Player.QueryRow(ctx, `
			SELECT rating_value, rating_deviation, COALESCE(start_time, written_at)
			FROM match_skill_rank_latest
			WHERE rating_type = 'LUSR' AND rating_value IS NOT NULL
			ORDER BY written_at DESC
			LIMIT 1
		`).Scan(&rating, &deviation, &lastMatchAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return lusr, fmt.Errorf("query latest rating: %w", err)
	}
	if rating.Valid {
		lusr.Mu = rating.Float64
	}
	if deviation.Valid {
		lusr.Sigma = deviation.Float64
	}
	if lastMatchAt.Valid {
		t := lastMatchAt.Time
		lusr.LastMatchAt = &t
	}

	if err := r.pdb.Player.QueryRow(ctx, `
			SELECT COUNT(*) FROM match_skill_rank_latest
			WHERE rating_type = 'LUSR' AND rating_value IS NOT NULL
		`).Scan(&lusr.MatchesCount); err != nil {
		return lusr, fmt.Errorf("query count: %w", err)
	}
	return lusr, nil
}

// LUSRSeriesPoints retourne les ratings LUSR DATÉS (ASC) sur la fenêtre — même source ART-safe que
// LUSRSeries (vue `_latest`, règle n°2) mais en conservant le timestamp `COALESCE(start_time, written_at)`
// pour dater chaque point de la sparkline. Les rows LUSR ont `start_time` souvent NULL : written_at sert de
// proxy temporel.
func (r *Repo) LUSRSeriesPoints(ctx context.Context, since, until time.Time) ([]port.ProfileLUSRPoint, error) {
	rows, err := r.pdb.Player.Query(ctx, `
			SELECT rating_value, COALESCE(start_time, written_at)
			FROM match_skill_rank_latest
			WHERE rating_type = 'LUSR' AND rating_value IS NOT NULL
			  AND COALESCE(start_time, written_at) >= ? AND COALESCE(start_time, written_at) <= ?
			ORDER BY COALESCE(start_time, written_at) ASC
		`, since, until)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []port.ProfileLUSRPoint
	for rows.Next() {
		var (
			v  sql.NullFloat64
			at sql.NullTime
		)
		if err := rows.Scan(&v, &at); err != nil {
			return nil, err
		}
		if v.Valid && at.Valid {
			out = append(out, port.ProfileLUSRPoint{At: at.Time, Value: v.Float64})
		}
	}
	return out, rows.Err()
}

// LUSRSeries retourne la série de μ ordonnée chronologiquement (ASC).
func (r *Repo) LUSRSeries(ctx context.Context, since, until time.Time) ([]float64, error) {
	// Vue _latest + COALESCE(start_time, written_at) : les rows LUSR ont start_time NULL ; sans COALESCE le
	// filtre temporel excluait TOUT → série μ vide (« pas de tendance »). written_at sert de proxy temporel
	// pour l'historique ; start_time est préféré quand présent.
	rows, err := r.pdb.Player.Query(ctx, `
			SELECT rating_value
			FROM match_skill_rank_latest
			WHERE rating_type = 'LUSR' AND rating_value IS NOT NULL
			  AND COALESCE(start_time, written_at) >= ? AND COALESCE(start_time, written_at) <= ?
			ORDER BY COALESCE(start_time, written_at) ASC
		`, since, until)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []float64
	for rows.Next() {
		var v sql.NullFloat64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		if v.Valid {
			out = append(out, v.Float64)
		}
	}
	return out, rows.Err()
}

// AwardTotals agrège personal_score_awards_latest du joueur par award_name sur les matchs donnés.
// Cross-DB (ADR 0016) : les identifiants des matchs de la fenêtre viennent du SharedReader (WindowMatchIDs),
// l'agrégat se lit sur la base du joueur avec WHERE match_id IN (...).
func (r *Repo) AwardTotals(ctx context.Context, userID string, matchIDs []string) ([]port.ProfileAwardTotal, error) {
	placeholders := make([]string, len(matchIDs))
	args := make([]any, 0, len(matchIDs)+1)
	args = append(args, userID)
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	rows, err := r.pdb.Player.Query(ctx, `
			SELECT psa.award_name, SUM(psa.award_count)
			FROM personal_score_awards_latest psa
			WHERE psa.xuid = ?
			  AND psa.match_id IN (`+strings.Join(placeholders, ",")+`)
			GROUP BY psa.award_name
		`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []port.ProfileAwardTotal
	for rows.Next() {
		var name string
		var total sql.NullInt64
		if err := rows.Scan(&name, &total); err != nil {
			return out, err
		}
		if !total.Valid || total.Int64 <= 0 {
			continue
		}
		out = append(out, port.ProfileAwardTotal{Name: name, Total: total.Int64})
	}
	return out, nil
}

// lusrComponentsBaseSQL agrège les composantes LUSR ; %s reçoit le filtre des matchs exclus (vide, ou
// `WHERE match_id NOT IN (?, ...)`).
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

// lusrComponentsQuery construit la requête des composantes et ses arguments. Les matchs exclus sont passés
// en paramètres constants (jamais en sous-requête, ADR 0036) : la fenêtre glissante et le comptage des 20
// matchs de la tendance portent sur les matchs retenus seulement. Liste vide : aucune clause.
func lusrComponentsQuery(excluded []string) (string, []any) {
	if len(excluded) == 0 {
		return fmt.Sprintf(lusrComponentsBaseSQL, ""), nil
	}
	clause := "WHERE match_id NOT IN (" + duckdb.Placeholders(len(excluded)) + ")"
	return fmt.Sprintf(lusrComponentsBaseSQL, clause), duckdb.ToAnySlice(excluded)
}

// excludedMatchIDs rend, triés, les identifiants des matchs exclus du joueur. La lecture est celle de
// skill.LoadExcludedMatchIDs (source unique, vue _latest) : une base sans la vue ne porte aucune exclusion,
// toute autre erreur est remontée avec son contexte, car continuer sans elle ferait peser des matchs exclus.
func (r *Repo) excludedMatchIDs(ctx context.Context) ([]string, error) {
	set, err := skill.LoadExcludedMatchIDs(ctx, r.pdb.Player.SQLDb())
	if err != nil {
		return nil, fmt.Errorf("profile: lecture des matchs exclus: %w", err)
	}
	return slices.Sorted(maps.Keys(set)), nil
}

// LUSRComponents agrège, pour chaque composante LUSR, la moyenne, le seuil du quintile haut et les moyennes
// des 10 derniers et des 10 plus anciens matchs (≥ 20 matchs requis pour la tendance). Source : table
// `lusr_component_history`, alimentée live par le LUSR v2 et, au rattrapage, par son rejeu complet. Une
// seule requête CTE batchée groupe tout pour les 8 composantes. Si la requête échoue, la liste est vide
// silencieusement (l'interface dégrade) ; la lecture des exclusions, elle, remonte.
func (r *Repo) LUSRComponents(ctx context.Context) ([]port.ProfileComponentAggregate, error) {
	excluded, err := r.excludedMatchIDs(ctx)
	if err != nil {
		return nil, err
	}
	query, args := lusrComponentsQuery(excluded)
	rows, err := r.pdb.Player.Query(ctx, query, args...)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()
	var out []port.ProfileComponentAggregate
	for rows.Next() {
		var name string
		var avg, top20, lastAvg, firstAvg sql.NullFloat64
		var n int
		if err := rows.Scan(&name, &avg, &top20, &lastAvg, &firstAvg, &n); err != nil {
			return out, err
		}
		out = append(out, port.ProfileComponentAggregate{
			Name: name, Avg: nullFloat(avg), Top20: nullFloat(top20),
			LastAvg: nullFloat(lastAvg), FirstAvg: nullFloat(firstAvg), N: n,
		})
	}
	return out, rows.Err()
}
