// Outil de diagnostic temporaire — investigate_matches.
// Usage : go run ./cmd/investigate_matches/
package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/duckdb/duckdb-go/v2"
)

const sharedDB = `../../data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`

var matchIDs = []string{
	"20fcbfe4-5a35-4992-a4b8-4bb7d92b62b6",
	"bc0bdda3-4116-4d08-913d-628285633197",
}

func main() {
	db, err := sql.Open("duckdb", sharedDB+"?access_mode=read_only")
	if err != nil {
		log.Fatal("open:", err)
	}
	defer db.Close()

	for _, mid := range matchIDs {
		fmt.Printf("\n=== %s ===\n", mid)
		investigateMatch(db, mid)
	}
}

func investigateMatch(db *sql.DB, matchID string) {
	// 1. match_registry row complet
	rows, err := db.Query(`
		SELECT
			match_id,
			start_time,
			start_time_utc,
			duration_seconds,
			playable_duration_seconds,
			map_name,
			pair_name,
			playlist_name,
			is_firefight,
			is_ranked,
			map_id,
			game_variant_name,
			playlist_id,
			team_0_score,
			team_1_score,
			pair_name_fr,
			pair_id,
			game_variant_id
		FROM match_registry
		WHERE match_id = ?`, matchID)
	if err != nil {
		fmt.Printf("  [match_registry] ERREUR QUERY: %v\n", err)
		return
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	found := false
	for rows.Next() {
		found = true
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			fmt.Printf("  [match_registry] ERREUR SCAN: %v\n", err)
			continue
		}
		for i, col := range cols {
			fmt.Printf("  %-30s = %v  (%T)\n", col, vals[i], vals[i])
		}
	}
	if !found {
		fmt.Println("  [match_registry] INTROUVABLE dans shared_matches_v2.duckdb")
	}

	// 2. Count dans match_participants
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM match_participants WHERE match_id = ?`, matchID).Scan(&count)
	if err != nil {
		fmt.Printf("  [match_participants] ERREUR: %v\n", err)
	} else {
		fmt.Printf("  [match_participants] %d participant(s)\n", count)
	}

	// 3. Vérifier la Q13 simulée (problème de timezone ?)
	var id, mapName, pairName sql.NullString
	var startTime interface{}
	var durSec sql.NullFloat64
	var isFF bool
	err = db.QueryRow(`
		SELECT
			match_id,
			COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC') AS start_time,
			duration_seconds,
			map_name,
			pair_name,
			is_firefight
		FROM match_registry WHERE match_id = ?
	`, matchID).Scan(&id, &startTime, &durSec, &mapName, &pairName, &isFF)
	if err != nil {
		fmt.Printf("  [Q13 simulation] ERREUR: %v\n", err)
	} else {
		fmt.Printf("  [Q13 simulation] OK — map=%v pair=%v start=%v dur=%v\n",
			mapName.String, pairName.String, startTime, durSec.Float64)
	}

	// 4. Tester Q12 scoreboard (la vraie avec COALESCE/WHERE)
	sbRows, err := db.Query(`
		WITH me_perfect AS (
			SELECT xuid, COALESCE(SUM(count), 0) AS perfect_kills
			FROM medals_earned
			WHERE match_id = ? AND medal_name_id = 1512363953
			GROUP BY xuid
		),
		top_weapons AS (
			SELECT xuid, wid AS top_weapon_id
			FROM (
				SELECT xuid, COALESCE(reconciled_as, weapon_id) AS wid, COUNT(*) AS wk,
					   ROW_NUMBER() OVER (PARTITION BY xuid ORDER BY COUNT(*) DESC) AS rn
				FROM weapon_kills
				WHERE match_id = ? AND COALESCE(reconciled_as, weapon_id) NOT IN (0, 1, 2)
				GROUP BY xuid, COALESCE(reconciled_as, weapon_id)
			) t WHERE rn = 1
		)
		SELECT COUNT(*) FROM match_participants p
		LEFT JOIN me_perfect m ON p.xuid = m.xuid
		LEFT JOIN top_weapons w ON p.xuid = w.xuid
		WHERE p.match_id = ?
		  AND NOT (
			COALESCE(p.kills, 0) = 0
			AND COALESCE(p.deaths, 0) = 0
			AND COALESCE(p.assists, 0) = 0
			AND COALESCE(p.personal_score, 0) = 0
			AND (p.kills IS NOT NULL OR p.deaths IS NOT NULL
				 OR p.assists IS NOT NULL OR p.personal_score IS NOT NULL)
		  )
	`, matchID, matchID, matchID)
	if err != nil {
		fmt.Printf("  [Q12 scoreboard] ERREUR: %v\n", err)
	} else {
		defer sbRows.Close()
		if sbRows.Next() {
			var cnt int
			_ = sbRows.Scan(&cnt)
			fmt.Printf("  [Q12 scoreboard] %d ligne(s) retournées\n", cnt)
		}
	}

	// 5. Vérifier si le match est dans highlight_events
	var evCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM highlight_events WHERE match_id = ?`, matchID).Scan(&evCount)
	if err != nil {
		fmt.Printf("  [highlight_events] ERREUR: %v\n", err)
	} else {
		fmt.Printf("  [highlight_events] %d event(s)\n", evCount)
	}
}
