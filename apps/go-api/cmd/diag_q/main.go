// cmd/diag_q — generic read-only DuckDB query runner for diagnostics.
//
// Usage: go run ./cmd/diag_q <db_path> <sql...>
// Opens the DB in read_only mode (safe while no writer holds the lock) and
// prints the result set as TSV with a header row. Throwaway diag tool.
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
		fmt.Fprintln(os.Stderr, "usage: diag_q <db_path> <sql...>")
		os.Exit(2)
	}
	dbPath := os.Args[1]
	query := strings.Join(os.Args[2:], " ")

	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	rows, err := db.QueryContext(context.Background(), query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "query: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	fmt.Println(strings.Join(cols, "\t"))

	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	n := 0
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			fmt.Fprintf(os.Stderr, "scan: %v\n", err)
			os.Exit(1)
		}
		cells := make([]string, len(cols))
		for i, v := range vals {
			cells[i] = render(v)
		}
		fmt.Println(strings.Join(cells, "\t"))
		n++
	}
	if err := rows.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "rows.Err: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "(%d rows)\n", n)
}

func render(v any) string {
	switch t := v.(type) {
	case nil:
		return "NULL"
	case []byte:
		return string(t)
	default:
		return fmt.Sprintf("%v", t)
	}
}
