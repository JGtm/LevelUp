// Command snapshot_shared_social — affiche un dump COUNT(*) des tables critiques
// de shared_social.duckdb. Utilise pour
//
//	(a) baseline non-régression avant intervention WAL
//	(b) diff post-recovery pour vérifier 0 perte
//	(c) diagnostic occasionnel (likes/favoris/notifs)
//
// Usage : go run ./apps/go-api/cmd/snapshot_shared_social <shared_social_path>
package main

import (
	"database/sql"
	"fmt"
	"os"
	"sort"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: snapshot_shared_social <shared_social.duckdb path>")
		os.Exit(1)
	}
	dbPath := os.Args[1]
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'main' AND table_type = 'BASE TABLE'
		ORDER BY 1
	`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "list tables: %v\n", err)
		os.Exit(1)
	}
	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			continue
		}
		tables = append(tables, t)
	}
	rows.Close()
	sort.Strings(tables)

	fmt.Printf("# snapshot_shared_social db=%s tables=%d\n", dbPath, len(tables))
	for _, t := range tables {
		var n int64
		if err := db.QueryRow(`SELECT COUNT(*) FROM "` + t + `"`).Scan(&n); err != nil {
			fmt.Printf("%-40s ERROR %v\n", t, err)
			continue
		}
		fmt.Printf("%-40s %d\n", t, n)
	}
}
