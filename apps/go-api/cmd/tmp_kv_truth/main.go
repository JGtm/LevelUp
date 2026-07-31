//go:build ignore

// tmp_kv_truth : dump la vérité-terrain (killer_victim_pairs + medals_earned + participants)
// pour le match 000d5950-83d9-423f-ab55-d068a7237b9f, à croiser avec le décodage chunk_27.
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"
)

const matchID = "000d5950-83d9-423f-ab55-d068a7237b9f"

func main() {
	dbPath := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}
	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	defer db.Close()

	// Colonnes killer_victim_pairs
	fmt.Println("=== killer_victim_pairs columns ===")
	dumpCols(db, "killer_victim_pairs")

	fmt.Println("\n=== killer_victim_pairs (match) — with time_ms ===")
	rows, err := db.Query(`SELECT killer_xuid, victim_xuid, kill_count, COALESCE(time_ms,-1) FROM killer_victim_pairs WHERE match_id = ? ORDER BY time_ms`, matchID)
	if err != nil {
		fmt.Println("kvp query:", err)
	} else {
		total := 0
		for rows.Next() {
			var k, v string
			var c, t int
			rows.Scan(&k, &v, &c, &t)
			fmt.Printf("  t=%-8d killer=%s victim=%s count=%d\n", t, k, v, c)
			total += c
		}
		rows.Close()
		fmt.Printf("  TOTAL pairs kills=%d\n", total)
	}

	fmt.Println("\n=== killer_victim_pairs aggregate (killer->victim totals) ===")
	rowsA, _ := db.Query(`SELECT killer_xuid, victim_xuid, SUM(kill_count) FROM killer_victim_pairs WHERE match_id = ? GROUP BY 1,2 ORDER BY 3 DESC`, matchID)
	if rowsA != nil {
		for rowsA.Next() {
			var k, v string
			var c int
			rowsA.Scan(&k, &v, &c)
			fmt.Printf("  killer=%s victim=%s total=%d\n", k, v, c)
		}
		rowsA.Close()
	}

	fmt.Println("\n=== participants columns ===")
	dumpCols(db, "match_participants")

	fmt.Println("\n=== participants (xuid -> team, kills, deaths) ===")
	rows2, err := db.Query(`SELECT xuid, COALESCE(CAST(team_id AS VARCHAR),'?'), kills, deaths FROM match_participants WHERE match_id = ? ORDER BY team_id, kills DESC`, matchID)
	if err != nil {
		fmt.Println("participants query:", err)
	} else {
		for rows2.Next() {
			var x, t string
			var k, d int
			rows2.Scan(&x, &t, &k, &d)
			fmt.Printf("  xuid=%s team=%s kills=%d deaths=%d\n", x, t, k, d)
		}
		rows2.Close()
	}

	fmt.Println("\n=== medals_earned columns ===")
	dumpCols(db, "medals_earned")

	fmt.Println("\n=== medals_earned (match) ===")
	rows3, err := db.Query(`SELECT xuid, medal_name_id, count FROM medals_earned WHERE match_id = ? ORDER BY xuid, count DESC`, matchID)
	if err != nil {
		fmt.Println("medals query:", err)
	} else {
		for rows3.Next() {
			var x string
			var mid int64
			var c int
			rows3.Scan(&x, &mid, &c)
			fmt.Printf("  xuid=%s medal_name_id=%d count=%d\n", x, mid, c)
		}
		rows3.Close()
	}
}

func dumpCols(db *sql.DB, table string) {
	rows, err := db.Query(`SELECT column_name, data_type FROM information_schema.columns WHERE table_name = ? ORDER BY ordinal_position`, table)
	if err != nil {
		fmt.Println("  cols err:", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var n, t string
		rows.Scan(&n, &t)
		fmt.Printf("  %-24s %s\n", n, t)
	}
}
