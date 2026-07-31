// tmp_kvdump — THROWAWAY : exporte en CSV, pour un match, killer_victim_pairs et les stats
// participants (shots_fired, kills, deaths) depuis shared_matches_v2 (read-only).
//
// Usage : go run ./cmd/tmp_kvdump <matchIdPrefix> <outDir>
package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
)

const sharedDB = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: tmp_kvdump <matchIdPrefix> <outDir>")
		os.Exit(2)
	}
	ctx := context.Background()
	db, err := sql.Open("duckdb", sharedDB+"?access_mode=read_only")
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	defer db.Close()
	var full string
	if err := db.QueryRowContext(ctx,
		`SELECT match_id FROM match_registry WHERE match_id LIKE ? ORDER BY match_id LIMIT 1`,
		os.Args[1]+"%").Scan(&full); err != nil {
		fmt.Println("match:", err)
		os.Exit(1)
	}
	fmt.Println("match:", full)
	out := os.Args[2]

	cols(ctx, db, "killer_victim_pairs")
	cols(ctx, db, "match_participants")

	dump(ctx, db, filepath.Join(out, "kv_"+os.Args[1]+".csv"), `
		SELECT * FROM killer_victim_pairs WHERE match_id = ? ORDER BY 1`, full)
	dump(ctx, db, filepath.Join(out, "participants_"+os.Args[1]+".csv"), `
		SELECT * FROM match_participants WHERE match_id = ?`, full)
}

func cols(ctx context.Context, db *sql.DB, t string) {
	rows, err := db.QueryContext(ctx,
		`SELECT column_name FROM information_schema.columns WHERE table_name = ? ORDER BY ordinal_position`, t)
	if err != nil {
		fmt.Println("cols:", err)
		return
	}
	defer rows.Close()
	fmt.Print(t, " : ")
	for rows.Next() {
		var c string
		_ = rows.Scan(&c)
		fmt.Print(c, " ")
	}
	fmt.Println()
}

func dump(ctx context.Context, db *sql.DB, path, q string, args ...any) {
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		fmt.Println("query:", err)
		return
	}
	defer rows.Close()
	names, _ := rows.Columns()
	f, err := os.Create(path)
	if err != nil {
		fmt.Println("create:", err)
		return
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write(names)
	n := 0
	for rows.Next() {
		vals := make([]any, len(names))
		ptrs := make([]any, len(names))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			fmt.Println("scan:", err)
			return
		}
		rec := make([]string, len(names))
		for i, v := range vals {
			if v == nil {
				rec[i] = ""
			} else {
				rec[i] = fmt.Sprintf("%v", v)
			}
		}
		_ = w.Write(rec)
		n++
	}
	fmt.Printf("%s : %d lignes\n", path, n)
}
