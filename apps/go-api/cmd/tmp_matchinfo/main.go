// tmp_matchinfo — THROWAWAY : récupère date/map/mode du match (préfixe cache film)
// + participants + médailles, pour préparer l'observation in-game / la lecture CE.
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"
)

const root = "c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration"

func main() {
	prefix := "000d5950"
	if len(os.Args) > 1 {
		prefix = os.Args[1]
	}
	shared := root + "/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb"

	db, err := sql.Open("duckdb", shared+"?access_mode=read_only")
	if err != nil {
		fmt.Println("open error:", err)
		os.Exit(1)
	}
	defer db.Close()

	// 1) registre du match
	var matchID, pairName string
	var startUTC, startLocal, mapID, playlistID sql.NullString
	row := db.QueryRow(`SELECT match_id, start_time, start_time_utc, pair_name, map_id, playlist_id
		FROM match_registry WHERE match_id LIKE ? || '%' LIMIT 1`, prefix)
	if err := row.Scan(&matchID, &startLocal, &startUTC, &pairName, &mapID, &playlistID); err != nil {
		fmt.Println("registry scan error:", err)
		// fallback : afficher quelques match_id pour caler le préfixe
		rs, _ := db.Query(`SELECT match_id FROM match_registry LIMIT 5`)
		if rs != nil {
			fmt.Println("exemples match_id :")
			for rs.Next() {
				var m string
				rs.Scan(&m)
				fmt.Println("  ", m)
			}
			rs.Close()
		}
		os.Exit(1)
	}
	fmt.Println("=== MATCH ===")
	fmt.Printf("  match_id   : %s\n", matchID)
	fmt.Printf("  start_local: %s\n", startLocal.String)
	fmt.Printf("  start_utc  : %s\n", startUTC.String)
	fmt.Printf("  pair_name  : %s\n", pairName)
	fmt.Printf("  map_id     : %s\n", mapID.String)
	fmt.Printf("  playlist_id: %s\n", playlistID.String)

	// 2) participants — dump générique (colonnes inconnues)
	fmt.Println("\n=== PARTICIPANTS (colonnes) ===")
	cols := dumpColumns(db, "match_participants")
	dumpRows(db, "SELECT * FROM match_participants WHERE match_id = '"+matchID+"'", cols)

	// 3) médailles — schéma réel (medal_name_id, created_at)
	fmt.Println("\n=== MEDALS (medal_name_id, created_at) ===")
	mcols := dumpColumns(db, "medals_earned")
	dumpRows(db, "SELECT * FROM medals_earned WHERE match_id = '"+matchID+"'", mcols)
}

func dumpColumns(db *sql.DB, table string) []string {
	rs, err := db.Query("SELECT column_name FROM information_schema.columns WHERE table_name = ? ORDER BY ordinal_position", table)
	if err != nil {
		fmt.Println("  colonnes error:", err)
		return nil
	}
	defer rs.Close()
	var out []string
	for rs.Next() {
		var c string
		rs.Scan(&c)
		out = append(out, c)
	}
	fmt.Printf("  %v\n", out)
	return out
}

func dumpRows(db *sql.DB, query string, cols []string) {
	rs, err := db.Query(query)
	if err != nil {
		fmt.Println("  query error:", err)
		return
	}
	defer rs.Close()
	colNames, _ := rs.Columns()
	n := 0
	for rs.Next() {
		vals := make([]any, len(colNames))
		ptrs := make([]any, len(colNames))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		rs.Scan(ptrs...)
		parts := ""
		for i, c := range colNames {
			v := vals[i]
			if v == nil {
				continue
			}
			s := fmt.Sprintf("%v", v)
			if len(s) > 0 && s != "0" {
				parts += fmt.Sprintf("%s=%s  ", c, s)
			}
		}
		fmt.Printf("  [%d] %s\n", n, parts)
		n++
		if n >= 20 {
			break
		}
	}
	if n == 0 {
		fmt.Println("  (aucune ligne)")
	}
}
