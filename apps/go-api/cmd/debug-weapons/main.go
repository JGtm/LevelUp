// Diagnostic temporaire — vérifie l'état de weapon_kills pour un match.
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/marcboeker/go-duckdb/v2"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: debug-weapons <match-id>")
		os.Exit(1)
	}
	matchID := os.Args[1]

	dbPath := `c:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\titles\halo_infinite\warehouse\shared_matches_v2.duckdb`
	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		fmt.Printf("open: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Schema de weapon_kills
	rows, err := db.Query(`SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'weapon_kills' ORDER BY ordinal_position`)
	if err != nil {
		fmt.Printf("query columns: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("weapon_kills schema:")
	for rows.Next() {
		var name, dtype string
		_ = rows.Scan(&name, &dtype)
		fmt.Printf("  %s : %s\n", name, dtype)
	}
	rows.Close()

	// Total rows
	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM weapon_kills`).Scan(&total); err == nil {
		fmt.Printf("\ntotal rows in weapon_kills: %d\n", total)
	} else {
		fmt.Printf("count failed: %v\n", err)
	}

	// Pour ce match
	var matchCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM weapon_kills WHERE match_id = ?`, matchID).Scan(&matchCount); err == nil {
		fmt.Printf("rows for match %s: %d\n", matchID, matchCount)
	}

	// Distinct match_ids
	var distinctMatches int
	if err := db.QueryRow(`SELECT COUNT(DISTINCT match_id) FROM weapon_kills`).Scan(&distinctMatches); err == nil {
		fmt.Printf("distinct match_ids in weapon_kills: %d\n", distinctMatches)
	}

	// Top 5 matches
	rows2, err := db.Query(`SELECT match_id, COUNT(*) FROM weapon_kills GROUP BY match_id ORDER BY COUNT(*) DESC LIMIT 5`)
	if err == nil {
		fmt.Println("\ntop 5 matches by weapon_kills row count:")
		for rows2.Next() {
			var mid string
			var n int
			_ = rows2.Scan(&mid, &n)
			fmt.Printf("  %s : %d rows\n", mid, n)
		}
		rows2.Close()
	}

	// View v_weapon_kills exists?
	var viewExists int
	if err := db.QueryRow(`SELECT COUNT(*) FROM information_schema.views WHERE table_name = 'v_weapon_kills'`).Scan(&viewExists); err == nil {
		fmt.Printf("\nv_weapon_kills view exists: %v\n", viewExists > 0)
	}
}
