package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	db, err := sql.Open("duckdb", "../../data/titles/halo_infinite/warehouse/metadata.duckdb?access_mode=read_only")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer db.Close()

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM map_images_registry WHERE title_id='halo_infinite' AND TRIM(local_path) != ''`).Scan(&n); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Total entries: %d\n", n)

	rows, err := db.Query(`SELECT map_id, local_path FROM map_images_registry WHERE map_id IN ('105f5d84-8de1-4908-af3a-1c4f3bf9d642','3e1e4cec-4f2c-44c6-b8d2-96b85c66c702')`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer rows.Close()
	for rows.Next() {
		var id, lp string
		_ = rows.Scan(&id, &lp)
		fmt.Printf("%s -> %s\n", id, lp)
	}
}
