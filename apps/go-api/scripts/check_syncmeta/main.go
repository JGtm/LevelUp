package main

import (
	"database/sql"
	"fmt"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	dbPath := `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration\data\titles\halo_infinite\players\Chocoboflor\stats.duckdb?access_mode=read_only`
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		fmt.Println("open err:", err)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT key, length(value) FROM sync_meta ORDER BY key")
	if err != nil {
		fmt.Println("query err:", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var k string
		var l int
		rows.Scan(&k, &l)
		fmt.Printf("  key=%-40s len=%d\n", k, l)
	}
	fmt.Println("done")
}
