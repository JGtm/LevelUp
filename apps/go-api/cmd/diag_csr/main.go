//go:build ignore

package main

import (
	"database/sql"
	"fmt"

	_ "github.com/duckdb/duckdb-go/v2"
)

const sharedPath = `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\titles\halo_infinite\warehouse\shared_matches_v2.duckdb`

func getXUID(playerPath string) string {
	db, err := sql.Open("duckdb", playerPath+"?access_mode=READ_ONLY")
	if err != nil {
		return ""
	}
	defer db.Close()
	var xuid string
	db.QueryRow(`SELECT value FROM sync_meta WHERE key = 'player_xuid' LIMIT 1`).Scan(&xuid)
	return xuid
}

func diagUnrankedDates(playerPath, gamertag string) {
	xuid := getXUID(playerPath)
	sdb, err := sql.Open("duckdb", sharedPath+"?access_mode=READ_ONLY")
	if err != nil {
		fmt.Println("shared open err:", err)
		return
	}
	defer sdb.Close()

	fmt.Printf("\n════ %s (xuid=%s) — matchs non-classés ════\n", gamertag, xuid)

	rows, err := sdb.Query(`
		SELECT YEAR(r.start_time) as yr, COUNT(*) as n,
		       MIN(r.start_time)::VARCHAR as first_match,
		       MAX(r.start_time)::VARCHAR as last_match
		FROM match_registry r
		JOIN match_participants mp ON r.match_id = mp.match_id AND mp.xuid = ?
		WHERE COALESCE(r.is_ranked, FALSE) = FALSE
		  AND COALESCE(r.is_firefight, FALSE) = FALSE
		  AND r.start_time IS NOT NULL
		  AND (r.duration_seconds IS NULL OR r.duration_seconds >= 30)
		GROUP BY YEAR(r.start_time)
		ORDER BY yr`, xuid)
	if err != nil {
		fmt.Println("dist query err:", err)
		return
	}
	defer rows.Close()
	fmt.Printf("  %-6s %6s  %-20s  %-20s\n", "année", "matchs", "premier", "dernier")
	total := 0
	for rows.Next() {
		var yr, n int
		var first, last string
		rows.Scan(&yr, &n, &first, &last)
		if len(first) > 19 {
			first = first[:19]
		}
		if len(last) > 19 {
			last = last[:19]
		}
		fmt.Printf("  %-6d %6d  %-20s  %-20s\n", yr, n, first, last)
		total += n
	}
	fmt.Printf("  TOTAL: %d matchs non-classés\n", total)
}

func diagUnrankedDatesXUID(xuid, gamertag string) {
	sdb, err := sql.Open("duckdb", sharedPath+"?access_mode=READ_ONLY")
	if err != nil {
		fmt.Println("shared open err:", err)
		return
	}
	defer sdb.Close()

	fmt.Printf("\n════ %s (xuid=%s) — matchs non-classés ════\n", gamertag, xuid)

	rows, err := sdb.Query(`
		SELECT YEAR(r.start_time) as yr, COUNT(*) as n,
		       MIN(r.start_time)::VARCHAR as first_match,
		       MAX(r.start_time)::VARCHAR as last_match
		FROM match_registry r
		JOIN match_participants mp ON r.match_id = mp.match_id AND mp.xuid = ?
		WHERE COALESCE(r.is_ranked, FALSE) = FALSE
		  AND COALESCE(r.is_firefight, FALSE) = FALSE
		  AND r.start_time IS NOT NULL
		  AND (r.duration_seconds IS NULL OR r.duration_seconds >= 30)
		GROUP BY YEAR(r.start_time)
		ORDER BY yr`, xuid)
	if err != nil {
		fmt.Println("dist query err:", err)
		return
	}
	defer rows.Close()
	fmt.Printf("  %-6s %6s  %-20s  %-20s\n", "année", "matchs", "premier", "dernier")
	total := 0
	for rows.Next() {
		var yr, n int
		var first, last string
		rows.Scan(&yr, &n, &first, &last)
		if len(first) > 19 {
			first = first[:19]
		}
		if len(last) > 19 {
			last = last[:19]
		}
		fmt.Printf("  %-6d %6d  %-20s  %-20s\n", yr, n, first, last)
		total += n
	}
	fmt.Printf("  TOTAL: %d matchs non-classés\n", total)
}

func main() {
	diagUnrankedDatesXUID("2533274858283686", "Madina97294")
	diagUnrankedDatesXUID("2533274823110022", "JGtm")
	diagUnrankedDatesXUID("2535469190789936", "Chocoboflor")
}
