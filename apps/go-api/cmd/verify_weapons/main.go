//go:build cgo

package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	sharedDBPath := flag.String("db", "../../data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb", "Path to shared DB")
	flag.Parse()

	db, err := sql.Open("duckdb", *sharedDBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	type Result struct {
		TotalRows    int
		WithWeaponID int
		NullWeaponID int
	}

	var res Result
	row := db.QueryRow(`
		SELECT
		  COUNT(*) as total_rows,
		  COUNT(CASE WHEN weapon_id IS NOT NULL THEN 1 END) as with_weapon_id,
		  COUNT(CASE WHEN weapon_id IS NULL THEN 1 END) as null_weapon_id
		FROM weapon_kills
		WHERE match_id IN (
		  SELECT DISTINCT mp.match_id
		  FROM match_participants mp
		  JOIN match_registry mr ON mp.match_id = mr.match_id
		  WHERE mp.xuid IN ('2535469190789936', '2533274823110022', '2533274858283686')
		    AND mr.start_time > CURRENT_DATE - INTERVAL 28 DAY
		)
	`)

	if err := row.Scan(&res.TotalRows, &res.WithWeaponID, &res.NullWeaponID); err != nil {
		fmt.Fprintf(os.Stderr, "scan: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("=== weapon_kills vérification post-backfill ===\n")
	fmt.Printf("Total rows             : %d\n", res.TotalRows)
	fmt.Printf("With weapon_id ✓       : %d (%.1f%%)\n", res.WithWeaponID, 100*float64(res.WithWeaponID)/float64(res.TotalRows))
	fmt.Printf("NULL weapon_id         : %d\n", res.NullWeaponID)
	fmt.Println()

	if res.WithWeaponID == 0 {
		fmt.Println("⚠️  PROBLÈME : aucun weapon_id peuplé !")
		os.Exit(1)
	}

	if res.NullWeaponID > 0 {
		fmt.Printf("⚠️  %d lignes ont weapon_id=NULL (fire events non reconnus ?)\n", res.NullWeaponID)
	}

	fmt.Println("✅ weapon_kills peuplés avec succès")

	// Sample rows
	fmt.Println("\nSample weapon_kills (derniers insérés):")
	rows, _ := db.Query(`
		SELECT wk.match_id, wk.xuid, wk.weapon_id, wk.time_ms, wk.delta_ms, wk.confidence, wk.attribution_path
		FROM weapon_kills wk
		WHERE wk.match_id IN (
		  SELECT DISTINCT mp.match_id
		  FROM match_participants mp
		  JOIN match_registry mr ON mp.match_id = mr.match_id
		  WHERE mp.xuid IN ('2535469190789936', '2533274823110022', '2533274858283686')
		    AND mr.start_time > CURRENT_DATE - INTERVAL 28 DAY
		)
		  AND wk.weapon_id IS NOT NULL
		ORDER BY wk.time_ms DESC
		LIMIT 5
	`)
	defer rows.Close()

	for rows.Next() {
		var matchID, xuid, wid, tms, dms, conf, path interface{}
		rows.Scan(&matchID, &xuid, &wid, &tms, &dms, &conf, &path)
		fmt.Printf("  %v | %v | wid=%v | t=%vms | confidence=%v\n", matchID, xuid, wid, tms, conf)
	}
}
