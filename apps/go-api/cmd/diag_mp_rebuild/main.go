//go:build ignore

// Inspection read-only avant rebuild match_participants : confirme la
// corruption ART + capture colonnes / PK / vues dépendantes / index.
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	dbPath := os.Args[1]
	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	defer db.Close()

	q := func(label, query string) {
		fmt.Println("\n===", label, "===")
		rows, err := db.Query(query)
		if err != nil {
			fmt.Println("  ERR:", err)
			return
		}
		defer rows.Close()
		cols, _ := rows.Columns()
		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			rows.Scan(ptrs...)
			fmt.Printf("  %v\n", vals)
		}
	}

	// 0. Test ciblé du match_id flaggé par art_guard au boot.
	q("TARGETED de3cec8b (index vs scan)",
		`SELECT
		   (SELECT COUNT(*) FROM match_participants WHERE match_id = 'de3cec8b-edf1-4edc-ad87-830369e0a358') AS via_index,
		   (SELECT COUNT(*) FROM match_participants WHERE match_id || '' = 'de3cec8b-edf1-4edc-ad87-830369e0a358') AS via_scan`)

	// 1. Confirme la corruption : COUNT via index vs via table-scan, sur un sample.
	q("ART probe (index vs scan) — divergence si counts != ",
		`WITH s AS (SELECT match_id FROM match_participants WHERE match_id IS NOT NULL ORDER BY random() LIMIT 8)
		 SELECT s.match_id,
		   (SELECT COUNT(*) FROM match_participants p WHERE p.match_id = s.match_id) AS via_index,
		   (SELECT COUNT(*) FROM match_participants p WHERE p.match_id || '' = s.match_id) AS via_scan
		 FROM s`)

	q("total rows (scan)", `SELECT COUNT(*) FROM match_participants WHERE match_id || '' IS NOT NULL`)
	q("columns (ordinal)", `SELECT ordinal_position, column_name, data_type FROM information_schema.columns WHERE table_name='match_participants' AND table_schema='main' ORDER BY ordinal_position`)
	q("primary key cols", `SELECT kcu.ordinal_position, kcu.column_name FROM information_schema.key_column_usage kcu JOIN information_schema.table_constraints tc ON tc.constraint_name=kcu.constraint_name WHERE tc.constraint_type='PRIMARY KEY' AND kcu.table_name='match_participants' ORDER BY kcu.ordinal_position`)
	q("indexes on match_participants (DDL)", `SELECT index_name, sql FROM duckdb_indexes() WHERE table_name='match_participants'`)
	q("ALL views (name + sql) — repère ceux référençant match_participants", `SELECT view_name, sql FROM duckdb_views() WHERE internal=false ORDER BY view_name`)
}
