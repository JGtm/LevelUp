//go:build ignore

package main

import (
	"database/sql"
	"fmt"

	_ "github.com/duckdb/duckdb-go/v2"
)

const sharedPath = `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\titles\halo_infinite\warehouse\shared_matches_v2.duckdb`

var players = []struct {
	XUID     string
	Gamertag string
}{
	{"2533274858283686", "Madina97294"},
	{"2533274823110022", "JGtm"},
	{"2535469190789936", "Chocoboflor"},
}

func main() {
	db, err := sql.Open("duckdb", sharedPath+"?access_mode=READ_ONLY")
	if err != nil {
		fmt.Println("open err:", err)
		return
	}
	defer db.Close()

	fmt.Printf("\n%-15s %6s %7s %7s %7s %7s %7s %7s\n",
		"joueur", "matchs", "kpm_avg", "kpm_med", "kpm_p25", "kpm_p75", "dpm_avg", "dpm_med")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────")

	for _, p := range players {
		var gamertag string
		var nMatchs int
		var kpmAvg, kpmMed, kpmP25, kpmP75, dpmAvg, dpmMed float64

		err := db.QueryRow(`
			SELECT
				COALESCE(xa.gamertag, ?) as gamertag,
				COUNT(*) as n_matchs,
				ROUND(AVG(mp.kills::DOUBLE / mp.time_played_seconds * 60), 3) as kpm_avg,
				ROUND(MEDIAN(mp.kills::DOUBLE / mp.time_played_seconds * 60), 3) as kpm_med,
				ROUND(PERCENTILE_CONT(0.25) WITHIN GROUP (ORDER BY mp.kills::DOUBLE / mp.time_played_seconds * 60), 3) as kpm_p25,
				ROUND(PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY mp.kills::DOUBLE / mp.time_played_seconds * 60), 3) as kpm_p75,
				ROUND(AVG(mp.deaths::DOUBLE / mp.time_played_seconds * 60), 3) as dpm_avg,
				ROUND(MEDIAN(mp.deaths::DOUBLE / mp.time_played_seconds * 60), 3) as dpm_med
			FROM match_participants mp
			JOIN match_registry mr ON mr.match_id = mp.match_id
			LEFT JOIN xuid_aliases xa ON xa.xuid = mp.xuid
			WHERE mp.xuid = ?
			  AND COALESCE(mr.is_ranked, FALSE) = FALSE
			  AND COALESCE(mr.is_firefight, FALSE) = FALSE
			  AND mr.start_time IS NOT NULL
			  AND (mr.duration_seconds IS NULL OR mr.duration_seconds >= 30)
			  AND mp.time_played_seconds > 30
			GROUP BY xa.gamertag`,
			p.Gamertag, p.XUID,
		).Scan(&gamertag, &nMatchs, &kpmAvg, &kpmMed, &kpmP25, &kpmP75, &dpmAvg, &dpmMed)

		if err != nil {
			fmt.Printf("%-15s  err: %v\n", p.Gamertag, err)
			continue
		}

		fmt.Printf("%-15s %6d %7.3f %7.3f %7.3f %7.3f %7.3f %7.3f\n",
			gamertag, nMatchs, kpmAvg, kpmMed, kpmP25, kpmP75, dpmAvg, dpmMed)
	}

	// Distribution KPM par décile pour Madina
	fmt.Printf("\n── Déciles KPM Madina97294 (xuid=2533274858283686) ──\n")
	rows, err := db.Query(`
		SELECT
			NTILE(10) OVER (ORDER BY mp.kills::DOUBLE / mp.time_played_seconds * 60) as decile,
			ROUND(MIN(mp.kills::DOUBLE / mp.time_played_seconds * 60), 3) as kpm_min,
			ROUND(MAX(mp.kills::DOUBLE / mp.time_played_seconds * 60), 3) as kpm_max,
			COUNT(*) as n
		FROM match_participants mp
		JOIN match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.xuid = '2533274858283686'
		  AND COALESCE(mr.is_ranked, FALSE) = FALSE
		  AND COALESCE(mr.is_firefight, FALSE) = FALSE
		  AND mr.start_time IS NOT NULL
		  AND (mr.duration_seconds IS NULL OR mr.duration_seconds >= 30)
		  AND mp.time_played_seconds > 30
		GROUP BY decile
		ORDER BY decile`)
	if err != nil {
		fmt.Println("decile err:", err)
		return
	}
	defer rows.Close()
	fmt.Printf("  %6s %8s %8s %5s\n", "décile", "kpm_min", "kpm_max", "n")
	for rows.Next() {
		var decile, n int
		var kmin, kmax float64
		rows.Scan(&decile, &kmin, &kmax, &n)
		fmt.Printf("  %6d %8.3f %8.3f %5d\n", decile, kmin, kmax, n)
	}
}
