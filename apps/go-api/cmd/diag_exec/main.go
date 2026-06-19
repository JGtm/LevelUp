// cmd/diag_exec — exécuteur SQL RW one-shot pour merges/maintenance locale.
// Usage: go run ./cmd/diag_exec <db_path> "<sql>"
// Ouvre la DB en READ_WRITE et exécute le SQL (plusieurs statements séparés par ;).
// Outil throwaway (non embarqué dans l'image).
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: diag_exec <db_path> <sql...>")
		os.Exit(2)
	}
	dbPath := os.Args[1]
	sqlText := strings.Join(os.Args[2:], " ")

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	ctx := context.Background()
	for _, stmt := range strings.Split(sqlText, ";") {
		s := strings.TrimSpace(stmt)
		if s == "" {
			continue
		}
		// SELECT/WITH/PRAGMA → afficher les lignes ; sinon Exec.
		up := strings.ToUpper(s)
		if strings.HasPrefix(up, "SELECT") || strings.HasPrefix(up, "WITH") || strings.HasPrefix(up, "PRAGMA") {
			if err := runQuery(ctx, db, s); err != nil {
				fmt.Fprintf(os.Stderr, "query error on [%s]: %v\n", firstN(s, 60), err)
				os.Exit(1)
			}
			continue
		}
		res, err := db.ExecContext(ctx, s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "exec error on [%s]: %v\n", firstN(s, 60), err)
			os.Exit(1)
		}
		if n, e := res.RowsAffected(); e == nil {
			fmt.Printf("OK (%d rows): %s\n", n, firstN(s, 60))
		} else {
			fmt.Printf("OK: %s\n", firstN(s, 60))
		}
	}
}

func runQuery(ctx context.Context, db *sql.DB, q string) error {
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return err
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	fmt.Println(strings.Join(cols, " | "))
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		cells := make([]string, len(cols))
		for i, v := range vals {
			cells[i] = fmt.Sprintf("%v", v)
		}
		fmt.Println(strings.Join(cells, " | "))
	}
	return rows.Err()
}

func firstN(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
