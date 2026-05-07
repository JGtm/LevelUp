// Command inspect_engagement — diagnostic one-shot pour comprendre pourquoi
// batchRecomputeCoefficients retourne modes_updated=0.
//
// Usage : go run ./cmd/inspect_engagement <player_db_path>
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: inspect_engagement <player_db_path>")
		os.Exit(1)
	}
	dbPath := os.Args[1]
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println("=== player_match_enrichment columns ===")
	rows, err := db.Query("DESCRIBE player_match_enrichment")
	if err != nil {
		fmt.Fprintf(os.Stderr, "describe: %v\n", err)
		os.Exit(1)
	}
	for rows.Next() {
		var name, typ string
		var nullable, key, dflt, extra sql.NullString
		_ = rows.Scan(&name, &typ, &nullable, &key, &dflt, &extra)
		fmt.Printf("  %-50s %s\n", name, typ)
	}
	rows.Close()

	fmt.Println("\n=== row counts ===")
	queries := []string{
		"SELECT COUNT(*) FROM player_match_enrichment",
		"SELECT COUNT(*) FROM player_match_enrichment WHERE engagement_score IS NOT NULL",
		"SELECT COUNT(*) FROM player_match_enrichment WHERE engagement_pace_player IS NOT NULL",
		"SELECT COUNT(*) FROM player_match_enrichment WHERE engagement_pace_team IS NOT NULL",
		"SELECT COUNT(*) FROM engagement_coefficients",
	}
	for _, q := range queries {
		var n int
		if err := db.QueryRow(q).Scan(&n); err != nil {
			fmt.Printf("  %s -> ERR %v\n", q, err)
			continue
		}
		fmt.Printf("  %s -> %d\n", q, n)
	}

	fmt.Println("\n=== engagement_coefficients rows ===")
	rows, err = db.Query("SELECT * FROM engagement_coefficients")
	if err == nil {
		cols, _ := rows.Columns()
		fmt.Printf("  cols: %v\n", cols)
		for rows.Next() {
			values := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range values {
				ptrs[i] = &values[i]
			}
			_ = rows.Scan(ptrs...)
			fmt.Printf("  %v\n", values)
		}
		rows.Close()
	}

	fmt.Println("\n=== sample paces (1ères 5 lignes avec pace_player non NULL) ===")
	rows, err = db.Query(`
		SELECT match_id, engagement_pace_player, engagement_pace_team, engagement_pace_lobby, engagement_player_activity
		FROM player_match_enrichment
		WHERE engagement_pace_player IS NOT NULL
		LIMIT 5
	`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sample: %v\n", err)
	} else {
		for rows.Next() {
			var mid string
			var p, t, l sql.NullFloat64
			var act sql.NullInt64
			_ = rows.Scan(&mid, &p, &t, &l, &act)
			fmt.Printf("  %s p=%v t=%v l=%v act=%v\n", mid, p, t, l, act)
		}
		rows.Close()
	}
}
