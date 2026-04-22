//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	dbPath := "../../data/warehouse/metadata.duckdb"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		fmt.Println("open error:", err)
		os.Exit(1)
	}
	defer db.Close()

	// Tables
	rows, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema='main' ORDER BY 1")
	if err != nil {
		fmt.Println("tables error:", err)
		os.Exit(1)
	}
	fmt.Println("=== Tables ===")
	for rows.Next() {
		var t string
		rows.Scan(&t)
		fmt.Println(" ", t)
	}
	rows.Close()

	// battlepass_item_definitions
	var total, nullCount int
	db.QueryRow("SELECT COUNT(*) FROM battlepass_item_definitions").Scan(&total)
	db.QueryRow("SELECT COUNT(*) FROM battlepass_item_definitions WHERE display_path IS NULL OR display_path = ''").Scan(&nullCount)
	fmt.Printf("\nbattlepass_item_definitions: total=%d, display_path NULL/empty=%d\n", total, nullCount)

	rows2, _ := db.Query("SELECT inventory_item_path, display_path FROM battlepass_item_definitions WHERE display_path IS NOT NULL AND display_path != '' LIMIT 3")
	if rows2 != nil {
		for rows2.Next() {
			var p, d string
			rows2.Scan(&p, &d)
			fmt.Printf("  item=%s\n  display=%s\n", p, d)
		}
		rows2.Close()
	}

	// battlepass_item_translations
	var ttotal int
	err = db.QueryRow("SELECT COUNT(*) FROM battlepass_item_translations").Scan(&ttotal)
	if err != nil {
		fmt.Println("battlepass_item_translations error:", err)
	} else {
		fmt.Printf("\nbattlepass_item_translations: total=%d\n", ttotal)
	}
}
