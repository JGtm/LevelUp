//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	_ "github.com/marcboeker/go-duckdb"
)

func main() {
	db, err := sql.Open("duckdb", "../../data/warehouse/shared_matches_v2.duckdb")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	rows, err := db.Query(`SELECT DISTINCT playlist_id, playlist_name, playlist_name_fr FROM match_registry WHERE playlist_id IS NOT NULL LIMIT 20`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, name, nameFR sql.NullString
		rows.Scan(&id, &name, &nameFR)
		fmt.Printf("id=%-40s  en=%-30s  fr=%s\n", id.String, name.String, nameFR.String)
	}
}
