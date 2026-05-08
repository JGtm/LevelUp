//go:build cgo

// diag_expected_assists — corrélation exhaustive assists ~ toutes les variables disponibles.
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

func main() { //nolint:funlen
	const dbPath = "../../data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb"

	connector, err := duckdb.NewConnector(dbPath+"?access_mode=READ_ONLY", nil)
	if err != nil {
		log.Fatalf("connector: %v", err)
	}
	db := sql.OpenDB(connector)
	defer db.Close()

	ctx := context.Background()

	printSection("CORRÉLATIONS — assists ~ variables combat")
	runQuery(ctx, db, `
		SELECT
			COUNT(*) AS n,
			ROUND(corr(assists, kills),               4) AS r_kills,
			ROUND(corr(assists, deaths),              4) AS r_deaths,
			ROUND(corr(assists, damage_dealt),        4) AS r_damage_dealt,
			ROUND(corr(assists, damage_taken),        4) AS r_damage_taken,
			ROUND(corr(assists, time_played_seconds), 4) AS r_time_played,
			ROUND(corr(assists, avg_life_seconds),    4) AS r_avg_life,
			ROUND(corr(assists, personal_score),      4) AS r_personal_score
		FROM match_participants
		WHERE kills >= 0 AND time_played_seconds > 0
		  AND damage_dealt >= 0 AND NOT isnan(damage_dealt)
		  AND damage_taken >= 0 AND NOT isnan(damage_taken)
	`)

	printSection("CORRÉLATIONS — assists ~ tir et kills spécialisés")
	runQuery(ctx, db, `
		SELECT
			COUNT(*) AS n,
			ROUND(corr(assists, shots_fired),        4) AS r_shots_fired,
			ROUND(corr(assists, shots_hit),          4) AS r_shots_hit,
			ROUND(corr(assists, headshot_kills),     4) AS r_headshot_kills,
			ROUND(corr(assists, grenade_kills),      4) AS r_grenade_kills,
			ROUND(corr(assists, melee_kills),        4) AS r_melee_kills,
			ROUND(corr(assists, power_weapon_kills), 4) AS r_pw_kills,
			ROUND(corr(assists, max_killing_spree),  4) AS r_spree
		FROM match_participants
		WHERE kills >= 0
		  AND shots_fired > 0 AND NOT isnan(shots_fired)
		  AND shots_hit  >= 0 AND NOT isnan(shots_hit)
	`)

	printSection("R² individuels — top variables")
	runQuery(ctx, db, `
		SELECT
			ROUND(regr_r2(assists, kills),               4) AS r2_kills,
			ROUND(regr_r2(assists, damage_dealt),        4) AS r2_damage,
			ROUND(regr_r2(assists, personal_score),      4) AS r2_score,
			ROUND(regr_r2(assists, time_played_seconds), 4) AS r2_time,
			ROUND(regr_r2(assists, deaths_stddev),       4) AS r2_deaths_std,
			ROUND(regr_r2(assists, kills_stddev),        4) AS r2_kills_std,
			ROUND(regr_r2(assists, kills_expected),      4) AS r2_kills_exp
		FROM match_participants
		WHERE kills >= 0 AND time_played_seconds > 0
		  AND damage_dealt >= 0 AND NOT isnan(damage_dealt)
	`)

	printSection("CORRÉLATIONS ~ match_registry (durée, nb joueurs)")
	runQuery(ctx, db, `
		SELECT
			COUNT(*) AS n,
			ROUND(corr(p.assists, r.duration_seconds), 4) AS r_duration,
			ROUND(corr(p.assists, r.player_count),     4) AS r_player_count
		FROM match_participants p
		JOIN match_registry r ON r.match_id = p.match_id
		WHERE p.kills >= 0 AND p.time_played_seconds > 0
	`)

	printSection("VARIABLES DÉRIVÉES — ratios")
	runQuery(ctx, db, `
		SELECT
			ROUND(corr(assists, kills::FLOAT / NULLIF(time_played_seconds, 0)),           4) AS r_kill_rate,
			ROUND(corr(assists, assists::FLOAT / NULLIF(kills + assists, 0)),             4) AS r_assist_ratio,
			ROUND(corr(assists, damage_dealt::FLOAT / NULLIF(time_played_seconds, 0)),    4) AS r_dmg_rate,
			ROUND(corr(assists, time_played_seconds::FLOAT / NULLIF(avg_life_seconds,0)), 4) AS r_respawns
		FROM match_participants
		WHERE kills >= 0 AND time_played_seconds > 0
		  AND damage_dealt >= 0 AND NOT isnan(damage_dealt)
		  AND kills + assists > 0
	`)

	printSection("HISTORIQUE JOUEUR — avg assists prédit match actuel")
	runQuery(ctx, db, `
		WITH player_hist AS (
			SELECT
				xuid,
				AVG(assists)    AS hist_avg_assists,
				AVG(kills)      AS hist_avg_kills,
				STDDEV(assists) AS hist_std_assists,
				COUNT(*)        AS n_matches
			FROM match_participants
			GROUP BY xuid
		)
		SELECT
			COUNT(*) AS n,
			ROUND(corr(p.assists, h.hist_avg_assists),    4) AS r_hist_avg_assists,
			ROUND(corr(p.assists, h.hist_avg_kills),      4) AS r_hist_avg_kills,
			ROUND(corr(p.assists, h.hist_std_assists),    4) AS r_hist_std_assists,
			ROUND(regr_r2(p.assists, h.hist_avg_assists), 4) AS r2_hist_avg_assists,
			ROUND(regr_r2(p.assists, h.hist_avg_kills),   4) AS r2_hist_avg_kills
		FROM match_participants p
		JOIN player_hist h ON h.xuid = p.xuid
		WHERE p.kills >= 0 AND h.n_matches >= 10
	`)

	printSection("BASELINE CONTEXTUELLE — hist assists par joueur × mode")
	runQuery(ctx, db, `
		WITH mode_hist AS (
			SELECT
				p.xuid,
				r.game_variant_name AS mode,
				AVG(p.assists) AS hist_mode_avg,
				COUNT(*)       AS n_matches
			FROM match_participants p
			JOIN match_registry r ON r.match_id = p.match_id
			GROUP BY p.xuid, r.game_variant_name
		)
		SELECT
			COUNT(*) AS n,
			ROUND(corr(p.assists, h.hist_mode_avg),    4) AS r_hist_mode,
			ROUND(regr_r2(p.assists, h.hist_mode_avg), 4) AS r2_hist_mode,
			ROUND(regr_slope(p.assists,     h.hist_mode_avg), 4) AS slope,
			ROUND(regr_intercept(p.assists, h.hist_mode_avg), 4) AS intercept,
			ROUND(AVG(ABS(p.assists - h.hist_mode_avg)), 3) AS mae_naive
		FROM match_participants p
		JOIN match_registry r ON r.match_id = p.match_id
		JOIN mode_hist h ON h.xuid = p.xuid AND h.mode = r.game_variant_name
		WHERE p.kills >= 0 AND h.n_matches >= 5
	`)

	printSection("MULTIVAR POST-MATCH — combinaisons des meilleurs prédicteurs")
	runQuery(ctx, db, `
		SELECT
			COUNT(*) AS n,
			-- Paires
			ROUND(regr_r2(assists, personal_score + shots_hit),          4) AS r2_score_shotshit,
			ROUND(regr_r2(assists, personal_score + damage_dealt),       4) AS r2_score_dmg,
			ROUND(regr_r2(assists, personal_score + kills),              4) AS r2_score_kills,
			ROUND(regr_r2(assists, shots_hit + damage_dealt),            4) AS r2_shotshit_dmg,
			ROUND(regr_r2(assists, shots_hit + kills),                   4) AS r2_shotshit_kills,
			-- Triplets
			ROUND(regr_r2(assists, personal_score + shots_hit + kills),  4) AS r2_score_sh_kills,
			ROUND(regr_r2(assists, personal_score + shots_hit + damage_dealt), 4) AS r2_score_sh_dmg,
			ROUND(regr_r2(assists, shots_hit + damage_dealt + kills),    4) AS r2_sh_dmg_kills
		FROM match_participants
		WHERE kills >= 0 AND time_played_seconds > 0
		  AND damage_dealt >= 0 AND NOT isnan(damage_dealt)
		  AND shots_hit >= 0    AND NOT isnan(shots_hit)
	`)

	printSection("FORMULE FINALE — personal_score + shots_hit : slope + MAE")
	runQuery(ctx, db, `
		WITH coefs AS (
			SELECT
				regr_slope(assists,     personal_score + shots_hit) AS s,
				regr_intercept(assists, personal_score + shots_hit) AS b,
				regr_r2(assists,        personal_score + shots_hit) AS rr
			FROM match_participants
			WHERE kills >= 0
			  AND shots_hit >= 0 AND NOT isnan(shots_hit)
		),
		with_pred AS (
			SELECT
				p.assists,
				c.s * (p.personal_score + p.shots_hit) + c.b AS pred,
				c.s AS slope, c.b AS intercept, c.rr AS r2
			FROM match_participants p, coefs c
			WHERE p.kills >= 0
			  AND p.shots_hit >= 0 AND NOT isnan(p.shots_hit)
		)
		SELECT
			ROUND(ANY_VALUE(slope),     6) AS slope,
			ROUND(ANY_VALUE(intercept), 4) AS intercept,
			ROUND(ANY_VALUE(r2),        4) AS r2,
			ROUND(AVG(ABS(assists - pred)), 3) AS mae,
			ROUND(AVG(assists), 3)            AS avg_assists,
			ROUND(STDDEV(assists), 3)         AS std_assists
		FROM with_pred
	`)

	printSection("FORMULE PAR MODE (game_variant_name) — personal_score + shots_hit")
	runQuery(ctx, db, `
		SELECT
			r.game_variant_name AS mode,
			COUNT(*) AS n,
			ROUND(AVG(p.assists), 3) AS avg_assists,
			ROUND(regr_slope(p.assists,     p.personal_score + p.shots_hit), 6) AS slope,
			ROUND(regr_intercept(p.assists, p.personal_score + p.shots_hit), 4) AS intercept,
			ROUND(regr_r2(p.assists,        p.personal_score + p.shots_hit), 4) AS r2
		FROM match_participants p
		JOIN match_registry r ON r.match_id = p.match_id
		WHERE p.kills >= 0
		  AND p.shots_hit >= 0 AND NOT isnan(p.shots_hit)
		GROUP BY r.game_variant_name
		HAVING COUNT(*) >= 30
		ORDER BY COUNT(*) DESC
		LIMIT 20
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
		fmt.Printf("%-26s", c)
	}
	fmt.Println()
	for range cols {
		fmt.Printf("%-26s", "--------------------------")
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
			fmt.Printf("%-26v", v)
		}
		fmt.Println()
	}
}
