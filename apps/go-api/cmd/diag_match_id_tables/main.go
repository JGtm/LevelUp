//go:build cgo

// diag_match_id_tables — énumère toutes les tables de chaque DB et liste
// celles qui ont une colonne `match_id`. Sert à vérifier qu'un cleanup ne
// laisse pas d'orphelin dans une table qu'on aurait oubliée.
//
// Usage : go run -tags cgo ./cmd/diag_match_id_tables [<match_id_à_compter>]
package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"os"
	"path/filepath"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

const tz = "Europe/Paris"

func main() {
	probeID := ""
	if len(os.Args) > 1 {
		probeID = os.Args[1]
	}

	dataRoot := "../../data/titles/halo_infinite"
	dbs := []struct{ label, path string }{
		{"shared", filepath.Join(dataRoot, "warehouse", "shared_matches_v2.duckdb")},
		{"shared_pve", filepath.Join(dataRoot, "warehouse", "shared_pve.duckdb")},
		{"shared_social", filepath.Join(dataRoot, "warehouse", "shared_social.duckdb")},
	}
	playersDir := filepath.Join(dataRoot, "players")
	entries, _ := os.ReadDir(playersDir)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dbs = append(dbs, struct{ label, path string }{
			"player[" + e.Name() + "]",
			filepath.Join(playersDir, e.Name(), "stats.duckdb"),
		})
	}

	for _, d := range dbs {
		if _, err := os.Stat(d.path); err != nil {
			continue
		}
		fmt.Printf("=== %s ===\n", d.label)
		listTablesWithMatchID(d.path, probeID)
		fmt.Println()
	}
}

func listTablesWithMatchID(path, probeID string) {
	connector, err := duckdb.NewConnector(path+"?access_mode=READ_ONLY", func(execer driver.ExecerContext) error {
		_, e := execer.ExecContext(context.Background(), "SET TimeZone='"+tz+"'", nil)
		return e
	})
	if err != nil {
		log.Printf("connector(%s): %v", path, err)
		return
	}
	db := sql.OpenDB(connector)
	defer db.Close()

	// Liste les tables/vues ayant une colonne match_id (toutes schemas confondus)
	rows, err := db.Query(`
		SELECT c.table_schema, c.table_name, COALESCE(t.table_type, 'UNKNOWN')
		FROM information_schema.columns c
		LEFT JOIN information_schema.tables t
		    ON t.table_schema = c.table_schema AND t.table_name = c.table_name
		WHERE c.column_name = 'match_id'
		ORDER BY c.table_schema, c.table_name`)
	if err != nil {
		log.Printf("query schemas(%s): %v", path, err)
		return
	}
	defer rows.Close()

	type tbl struct{ schema, name, kind string }
	var tables []tbl
	for rows.Next() {
		var t tbl
		_ = rows.Scan(&t.schema, &t.name, &t.kind)
		tables = append(tables, t)
	}

	if len(tables) == 0 {
		fmt.Println("  (aucune table avec colonne match_id)")
		return
	}
	for _, t := range tables {
		full := t.schema + "." + t.name
		if probeID != "" {
			var n int
			err := db.QueryRow("SELECT COUNT(*) FROM "+full+" WHERE match_id = ?", probeID).Scan(&n)
			if err != nil {
				fmt.Printf("  %s [%s] : ERR %v\n", full, t.kind, err)
				continue
			}
			marker := " "
			if n > 0 {
				marker = "⚠"
			}
			fmt.Printf("  %s %s [%s] : %d ligne(s) avec match_id=%s\n", marker, full, t.kind, n, probeID)
		} else {
			fmt.Printf("  - %s [%s]\n", full, t.kind)
		}
	}
}
