//go:build cgo

// query_medals_gap — liste les matchs sans médailles pour diagnostic.
package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"levelup/go-api/internal/analysis"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	db, err := sql.Open("duckdb", "../../data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb?access_mode=read_only")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT r.match_id,
		       ` + analysis.SQLStartTimeCanonical("r") + ` AS st,
		       COALESCE(r.is_firefight, FALSE) AS pve,
		       (SELECT COUNT(*) FROM match_participants p WHERE p.match_id = r.match_id) AS parts,
		       COALESCE(r.game_variant_name, '')  AS cat,
		       COALESCE(r.playlist_name, '')      AS playlist
		FROM match_registry r
		LEFT JOIN medals_earned m ON m.match_id = r.match_id
		WHERE m.match_id IS NULL
		  AND (SELECT COUNT(*) FROM match_participants p WHERE p.match_id = r.match_id) > 0
		ORDER BY st DESC NULLS LAST`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	now := time.Now()
	fmt.Printf("%-37s  %-10s  %-4s  %-4s  %-8s  %s\n",
		"MatchID", "Date", "PVE", "part", "AgeDays", "Category / Playlist")
	n := 0
	fresh := 0
	for rows.Next() {
		var mid, cat, playlist string
		var st sql.NullTime
		var pve bool
		var parts int
		if err := rows.Scan(&mid, &st, &pve, &parts, &cat, &playlist); err != nil {
			log.Fatal(err)
		}
		ageDays := -1.0
		age := "?"
		date := "?"
		if st.Valid {
			ageDays = now.Sub(st.Time).Hours() / 24
			age = fmt.Sprintf("%.0f", ageDays)
			date = st.Time.Format("2006-01-02")
		}
		if ageDays >= 0 && ageDays <= 90 {
			fresh++
		}
		fmt.Printf("%-37s  %-10s  %-4v  %-4d  %-8s  %s / %s\n", mid, date, pve, parts, age, cat, playlist)
		n++
	}
	fmt.Printf("\nTotal : %d matchs sans médailles  (dont %d <90j — potentiellement refetchables)\n", n, fresh)
}
