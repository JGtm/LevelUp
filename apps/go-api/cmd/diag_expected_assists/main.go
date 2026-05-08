//go:build cgo

// diag_expected_assists — exploration statistique pour approximer expected_assists.
//
// L'API officielle fournit kills_expected et deaths_expected mais pas assists_expected.
// Ce diag analyse les corrélations et régression pour identifier une formule.
//
// Usage : go run -tags cgo ./cmd/diag_expected_assists
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

// validKE = filtre propre excluant NaN/Inf sur kills_expected et deaths_expected
const validKE = `kills_expected > 0
		  AND kills_expected < 1e9
		  AND NOT isnan(kills_expected)
		  AND deaths_expected > 0
		  AND deaths_expected < 1e9
		  AND NOT isnan(deaths_expected)`

func main() {
	const dbPath = "../../data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb"

	connector, err := duckdb.NewConnector(dbPath+"?access_mode=READ_ONLY", nil)
	if err != nil {
		log.Fatalf("connector: %v", err)
	}
	db := sql.OpenDB(connector)
	defer db.Close()

	ctx := context.Background()

	printSection("STATISTIQUES DE BASE (sans NaN)")
	runQuery(ctx, db, `
		SELECT
			COUNT(*) AS n_rows,
			ROUND(AVG(assists), 3)          AS avg_assists,
			ROUND(STDDEV(assists), 3)        AS std_assists,
			ROUND(AVG(kills_expected), 3)   AS avg_kills_exp,
			ROUND(AVG(deaths_expected), 3)  AS avg_deaths_exp,
			ROUND(AVG(kills_stddev), 3)     AS avg_kills_stddev,
			ROUND(AVG(deaths_stddev), 3)    AS avg_deaths_stddev,
			ROUND(AVG(team_mmr), 3)         AS avg_team_mmr,
			ROUND(AVG(enemy_mmr), 3)        AS avg_enemy_mmr
		FROM match_participants
		WHERE `+validKE)

	printSection("CORRÉLATIONS avec assists (sans NaN)")
	runQuery(ctx, db, `
		SELECT
			ROUND(corr(assists, kills_expected),  4) AS r_kills_exp,
			ROUND(corr(assists, deaths_expected), 4) AS r_deaths_exp,
			ROUND(corr(assists, kills_stddev),    4) AS r_kills_stddev,
			ROUND(corr(assists, deaths_stddev),   4) AS r_deaths_stddev,
			ROUND(corr(assists, team_mmr),        4) AS r_team_mmr,
			ROUND(corr(assists, enemy_mmr),       4) AS r_enemy_mmr,
			ROUND(corr(assists, kills),           4) AS r_actual_kills,
			ROUND(corr(assists, deaths),          4) AS r_actual_deaths,
			ROUND(corr(assists, damage_dealt),    4) AS r_damage_dealt,
			ROUND(corr(assists, shots_fired),     4) AS r_shots_fired,
			ROUND(corr(assists, time_played_seconds), 4) AS r_time_played,
			ROUND(corr(assists, ln(kills_expected)), 4) AS r_ln_kills_exp
		FROM match_participants
		WHERE `+validKE)

	printSection("RÉGRESSION LINÉAIRE : assists ~ kills_expected")
	runQuery(ctx, db, `
		SELECT
			ROUND(regr_slope(assists, kills_expected),     4) AS slope,
			ROUND(regr_intercept(assists, kills_expected), 4) AS intercept,
			ROUND(regr_r2(assists, kills_expected),        4) AS r2
		FROM match_participants
		WHERE `+validKE)

	printSection("RÉGRESSION LOG : assists ~ ln(kills_expected)")
	runQuery(ctx, db, `
		SELECT
			ROUND(regr_slope(assists, ln(kills_expected)),     4) AS slope,
			ROUND(regr_intercept(assists, ln(kills_expected)), 4) AS intercept,
			ROUND(regr_r2(assists, ln(kills_expected)),        4) AS r2
		FROM match_participants
		WHERE `+validKE)

	printSection("RÉGRESSION : assists ~ deaths_expected")
	runQuery(ctx, db, `
		SELECT
			ROUND(regr_slope(assists, deaths_expected),     4) AS slope,
			ROUND(regr_intercept(assists, deaths_expected), 4) AS intercept,
			ROUND(regr_r2(assists, deaths_expected),        4) AS r2
		FROM match_participants
		WHERE `+validKE)

	printSection("RÉGRESSION : assists ~ kills_stddev")
	runQuery(ctx, db, `
		SELECT
			ROUND(regr_slope(assists, kills_stddev),     4) AS slope,
			ROUND(regr_intercept(assists, kills_stddev), 4) AS intercept,
			ROUND(regr_r2(assists, kills_stddev),        4) AS r2
		FROM match_participants
		WHERE kills_stddev > 0
		  AND NOT isnan(kills_stddev)
	`)

	printSection("RATIO assists/kills_expected par bucket kills_expected")
	runQuery(ctx, db, `
		WITH bucketed AS (
			SELECT
				ROUND(kills_expected, 0)::INT AS ke_bucket,
				assists,
				kills_expected
			FROM match_participants
			WHERE kills_expected BETWEEN 1 AND 30
			  AND `+validKE+`
		)
		SELECT
			ke_bucket,
			COUNT(*) AS n,
			ROUND(AVG(assists), 3)                             AS avg_assists,
			ROUND(AVG(assists / kills_expected), 4)            AS avg_ratio,
			ROUND(regr_slope(assists, kills_expected), 4)      AS slope_lin,
			ROUND(regr_slope(assists, ln(kills_expected)), 4)  AS slope_log
		FROM bucketed
		GROUP BY ke_bucket
		ORDER BY ke_bucket
		LIMIT 30
	`)

	printSection("STRATIFICATION PAR TAILLE D'ÉQUIPE")
	runQuery(ctx, db, `
		WITH teams AS (
			SELECT
				p.match_id,
				p.team_id,
				COUNT(*) AS team_size
			FROM match_participants p
			GROUP BY p.match_id, p.team_id
		),
		joined AS (
			SELECT
				p.assists,
				p.kills_expected,
				p.deaths_expected,
				t.team_size
			FROM match_participants p
			JOIN teams t ON t.match_id = p.match_id AND t.team_id = p.team_id
			WHERE `+validKE+`
		)
		SELECT
			team_size,
			COUNT(*) AS n,
			ROUND(AVG(assists), 3)                              AS avg_assists,
			ROUND(AVG(kills_expected), 3)                       AS avg_ke,
			ROUND(corr(assists, kills_expected), 4)             AS r_ke,
			ROUND(corr(assists, ln(kills_expected)), 4)         AS r_ln_ke,
			ROUND(regr_slope(assists, ln(kills_expected)), 4)   AS slope_log,
			ROUND(regr_r2(assists, ln(kills_expected)), 4)      AS r2_log
		FROM joined
		GROUP BY team_size
		ORDER BY team_size
	`)

	printSection("RÉSIDU log ~ deaths_expected + kills_stddev")
	runQuery(ctx, db, `
		WITH reg AS (
			SELECT
				regr_slope(assists, ln(kills_expected))     AS s,
				regr_intercept(assists, ln(kills_expected)) AS b
			FROM match_participants
			WHERE `+validKE+`
		),
		residuals AS (
			SELECT
				p.assists - (r.s * ln(p.kills_expected) + r.b) AS resid,
				p.deaths_expected,
				p.kills_stddev,
				p.deaths_stddev,
				p.team_mmr,
				p.enemy_mmr,
				p.time_played_seconds
			FROM match_participants p, reg r
			WHERE `+validKE+`
		)
		SELECT
			ROUND(corr(resid, deaths_expected),    4) AS r_deaths_exp,
			ROUND(corr(resid, kills_stddev),       4) AS r_kills_std,
			ROUND(corr(resid, deaths_stddev),      4) AS r_deaths_std,
			ROUND(corr(resid, team_mmr),           4) AS r_team_mmr,
			ROUND(corr(resid, enemy_mmr),          4) AS r_enemy_mmr,
			ROUND(corr(resid, time_played_seconds),4) AS r_time_played
		FROM residuals
	`)

	printSection("FORMULA CANDIDATES : comparaison R² sur données valides")
	runQuery(ctx, db, `
		WITH base AS (
			SELECT
				assists,
				kills_expected,
				deaths_expected,
				kills_stddev,
				deaths_stddev,
				-- Formule 1 : linéaire kills_expected
				0.218 * kills_expected + 1.4728                                   AS f1,
				-- Formule 2 : log kills_expected
				1.7996 * ln(kills_expected) - 0.2745                              AS f2,
				-- Formule 3 : kills_stddev seul
				1.03 * kills_stddev - 1.0912                                      AS f3,
				-- Formule 4 : kills_stddev + deaths_stddev (à coefficients égaux)
				0.6 * kills_stddev + 0.5 * deaths_stddev - 1.5                   AS f4
			FROM match_participants
			WHERE `+validKE+`
			  AND kills_stddev > 0
			  AND NOT isnan(kills_stddev)
		)
		SELECT
			ROUND(corr(assists, f1), 4)  AS r_f1_lin_ke,
			ROUND(corr(assists, f2), 4)  AS r_f2_log_ke,
			ROUND(corr(assists, f3), 4)  AS r_f3_kstd,
			ROUND(corr(assists, f4), 4)  AS r_f4_kstd_dstd,
			ROUND(regr_r2(assists, f1), 4) AS r2_f1,
			ROUND(regr_r2(assists, f2), 4) AS r2_f2,
			ROUND(regr_r2(assists, f3), 4) AS r2_f3,
			ROUND(regr_r2(assists, f4), 4) AS r2_f4
		FROM base
	`)

	printSection("RÉGRESSION : assists ~ kills_stddev + deaths_stddev (multivar approx)")
	runQuery(ctx, db, `
		SELECT
			ROUND(regr_slope(assists, kills_stddev + deaths_stddev),     4) AS slope_sum,
			ROUND(regr_intercept(assists, kills_stddev + deaths_stddev), 4) AS intercept_sum,
			ROUND(regr_r2(assists, kills_stddev + deaths_stddev),        4) AS r2_sum,
			ROUND(regr_slope(assists, kills_stddev),                     4) AS slope_ks,
			ROUND(regr_slope(assists, deaths_stddev),                    4) AS slope_ds
		FROM match_participants
		WHERE kills_stddev > 0
		  AND NOT isnan(kills_stddev)
		  AND NOT isnan(deaths_stddev)
	`)

	printSection("STRATIFICATION BUCKET MODE (par team_size)")
	runQuery(ctx, db, `
		WITH teams AS (
			SELECT match_id, team_id, COUNT(*) AS team_size
			FROM match_participants
			GROUP BY match_id, team_id
		),
		bucketed AS (
			SELECT
				p.assists, p.kills_expected, p.kills_stddev, p.deaths_stddev,
				CASE
					WHEN t.team_size <= 4  THEN 'arena'
					WHEN t.team_size <= 8  THEN 'mid'
					ELSE                        'btb'
				END AS mode_bucket
			FROM match_participants p
			JOIN teams t ON t.match_id = p.match_id AND t.team_id = p.team_id
			WHERE `+validKE+`
			  AND kills_stddev > 0 AND NOT isnan(kills_stddev)
		)
		SELECT
			mode_bucket,
			COUNT(*) AS n,
			ROUND(AVG(assists), 3)                                           AS avg_assists,
			ROUND(regr_slope(assists, kills_stddev + deaths_stddev), 4)      AS slope_std_sum,
			ROUND(regr_intercept(assists, kills_stddev + deaths_stddev), 4)  AS intcpt_std_sum,
			ROUND(regr_r2(assists, kills_stddev + deaths_stddev), 4)         AS r2_std_sum,
			ROUND(regr_r2(assists, kills_expected), 4)                       AS r2_ke_lin
		FROM bucketed
		GROUP BY mode_bucket
		ORDER BY mode_bucket
	`)
}

func printSection(title string) {
	fmt.Printf("\n=== %s ===\n", title)
}

func runQuery(ctx context.Context, db *sql.DB, query string) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		fmt.Printf("ERREUR: %v\n", err)
		return
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	for _, c := range cols {
		fmt.Printf("%-22s", c)
	}
	fmt.Println()
	for range cols {
		fmt.Printf("%-22s", "----------------------")
	}
	fmt.Println()

	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			fmt.Printf("scan: %v\n", err)
			continue
		}
		for _, v := range vals {
			fmt.Printf("%-22v", v)
		}
		fmt.Println()
	}
}
