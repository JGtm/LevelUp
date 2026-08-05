package main

// kvquery.go — vérification croisée : killer_victim_pairs + held weapon pour les
// kills oracle. But : confirmer quel kill est le Frag Parfait Sidekick et quelle
// arme la DB / le held-weapon attend par kill, pour comparer à la couverture.

import (
	"context"
	"database/sql"
	"fmt"
)

func dumpKillerVictim(ctx context.Context, full string) {
	db, err := openRO()
	if err != nil {
		fmt.Println("kv: open:", err)
		return
	}
	defer db.Close()

	fmt.Printf("killer_victim_pairs colonnes: %v\n", columnsOf(ctx, db, "killer_victim_pairs"))
	dumpJGtmKills(ctx, db, full)

	fmt.Printf("\nhighlight_events colonnes: %v\n", columnsOf(ctx, db, "highlight_events"))
	dumpJGtmHighlights(ctx, db, full)
}

func columnsOf(ctx context.Context, db *sql.DB, table string) []string {
	rows, err := db.QueryContext(ctx,
		`SELECT column_name FROM information_schema.columns WHERE table_name = ? ORDER BY ordinal_position`, table)
	if err != nil {
		return []string{"<err:" + err.Error() + ">"}
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var c string
		if rows.Scan(&c) == nil {
			out = append(out, c)
		}
	}
	return out
}

func dumpJGtmKills(ctx context.Context, db *sql.DB, full string) {
	rows, err := db.QueryContext(ctx, `
		SELECT time_ms, victim_xuid
		FROM killer_victim_pairs
		WHERE match_id = ? AND CAST(killer_xuid AS VARCHAR) LIKE ?
		ORDER BY time_ms`, full, "%"+jgtmXUID+"%")
	if err != nil {
		fmt.Println("kv: query:", err)
		return
	}
	defer rows.Close()
	fmt.Println("kills JGtm (killer_victim_pairs):")
	n := 0
	for rows.Next() {
		var t sql.NullInt64
		var v sql.NullString
		if rows.Scan(&t, &v) == nil {
			fmt.Printf("  time=%d victim=%s\n", t.Int64, v.String)
			n++
		}
	}
	fmt.Printf("  (%d kills)\n", n)
}

func dumpJGtmHighlights(ctx context.Context, db *sql.DB, full string) {
	rows, err := db.QueryContext(ctx, `
		SELECT time_ms, event_type
		FROM highlight_events
		WHERE match_id = ? AND CAST(xuid AS VARCHAR) LIKE ?
		  AND LOWER(COALESCE(event_type,'')) LIKE '%kill%'
		ORDER BY time_ms`, full, "%"+jgtmXUID+"%")
	if err != nil {
		fmt.Println("hl: query:", err)
		return
	}
	defer rows.Close()
	fmt.Println("kill-events JGtm (highlight_events):")
	for rows.Next() {
		var t sql.NullInt64
		var et sql.NullString
		if rows.Scan(&t, &et) == nil {
			fmt.Printf("  time=%d event=%s\n", t.Int64, et.String)
		}
	}
}
