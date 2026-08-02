//go:build ignore

// Rebuild isolé de match_participants (défait la corruption ART filter-pushdown)
// via CTAS swap, en transaction (rollback auto si erreur). Capture les DDL des
// vues dépendantes + index depuis la DB elle-même (pas de recopie manuelle).
//
// Backup préalable obligatoire (fait côté shell). Usage :
//
//	go run main.go <shared_matches_v2.duckdb>
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"
)

const targetMatch = "de3cec8b-edf1-4edc-ad87-830369e0a358"

// Vues dépendant (directement ou transitivement) de match_participants, dans
// l'ordre de recréation.
//
// BASCULE DU 2026-08-02 : v_killer_victim_full n'existe plus (deux LEFT JOIN morts), et
// match_kill_events_latest entre dans la liste — v_gamertag_lookup la lit désormais, donc elle
// doit être recréée AVANT lui. Cet outil porte `//go:build ignore` : AUCUN test ne le protège,
// cette liste est le seul filet. Une vue oubliée ici n'est pas recréée après le CTAS, et elle
// manque ensuite silencieusement.
var dependentViews = []string{
	"match_kill_events_latest", "v_gamertag_lookup", "mv_player_matches",
}

func main() {
	dbPath := os.Args[1]
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	defer db.Close()

	scalar := func(qq string, args ...any) int {
		var n int
		if err := db.QueryRow(qq, args...).Scan(&n); err != nil {
			fmt.Println("  scalar ERR:", err, "q=", qq)
			os.Exit(1)
		}
		return n
	}

	// État avant.
	beforeScan := scalar(`SELECT COUNT(*) FROM match_participants WHERE match_id || '' IS NOT NULL`)
	beforeIdxTarget := scalar(`SELECT COUNT(*) FROM match_participants WHERE match_id = ?`, targetMatch)
	beforeScanTarget := scalar(`SELECT COUNT(*) FROM match_participants WHERE match_id || '' = ?`, targetMatch)
	fmt.Printf("AVANT : total_scan=%d  target_index=%d  target_scan=%d\n", beforeScan, beforeIdxTarget, beforeScanTarget)

	// Capture DDL des index de match_participants.
	idxSQL := captureSQL(db, `SELECT sql FROM duckdb_indexes() WHERE table_name='match_participants' ORDER BY index_name`)
	fmt.Printf("indexes capturés: %d\n", len(idxSQL))

	// Capture DDL des vues dépendantes (dans l'ordre voulu).
	viewSQL := make([]string, 0, len(dependentViews))
	for _, v := range dependentViews {
		var s string
		if err := db.QueryRow(`SELECT sql FROM duckdb_views() WHERE view_name = ? AND internal = false LIMIT 1`, v).Scan(&s); err != nil {
			fmt.Println("capture view", v, "ERR:", err)
			os.Exit(1)
		}
		viewSQL = append(viewSQL, s)
	}
	fmt.Printf("vues capturées: %d\n", len(viewSQL))

	// --- Transaction de swap ---
	tx, err := db.Begin()
	if err != nil {
		fmt.Println("begin:", err)
		os.Exit(1)
	}
	exec := func(qq string) {
		if _, err := tx.Exec(qq); err != nil {
			fmt.Println("EXEC ERR (rollback):", err, "\n  q=", trunc(qq))
			_ = tx.Rollback()
			os.Exit(1)
		}
	}

	exec(`CREATE TABLE match_participants__rebuilt AS SELECT * FROM match_participants`)
	// Vérifie le count rebuilt == scan avant, dans la tx.
	var rebuiltCount int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM match_participants__rebuilt`).Scan(&rebuiltCount); err != nil {
		fmt.Println("count rebuilt ERR:", err)
		_ = tx.Rollback()
		os.Exit(1)
	}
	if rebuiltCount != beforeScan {
		fmt.Printf("ABORT: rebuilt=%d != scan=%d → rollback\n", rebuiltCount, beforeScan)
		_ = tx.Rollback()
		os.Exit(1)
	}
	exec(`DROP TABLE match_participants CASCADE`)
	exec(`ALTER TABLE match_participants__rebuilt RENAME TO match_participants`)
	exec(`ALTER TABLE match_participants ADD PRIMARY KEY (match_id, xuid)`)
	for _, s := range idxSQL {
		exec(s)
	}
	for _, s := range viewSQL {
		exec(s)
	}
	if err := tx.Commit(); err != nil {
		fmt.Println("commit:", err)
		os.Exit(1)
	}

	// État après.
	afterScan := scalar(`SELECT COUNT(*) FROM match_participants WHERE match_id || '' IS NOT NULL`)
	afterIdxTarget := scalar(`SELECT COUNT(*) FROM match_participants WHERE match_id = ?`, targetMatch)
	afterScanTarget := scalar(`SELECT COUNT(*) FROM match_participants WHERE match_id || '' = ?`, targetMatch)
	fmt.Printf("APRÈS : total_scan=%d  target_index=%d  target_scan=%d\n", afterScan, afterIdxTarget, afterScanTarget)
	if afterScan == beforeScan && afterIdxTarget == afterScanTarget && afterIdxTarget == beforeScanTarget {
		fmt.Println("OK: rebuild réussi, divergence résolue, aucune perte de rows.")
	} else {
		fmt.Println("ATTENTION: vérifier — valeurs inattendues.")
	}
}

func captureSQL(db *sql.DB, q string) []string {
	rows, err := db.Query(q)
	if err != nil {
		fmt.Println("capture ERR:", err)
		os.Exit(1)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err == nil && s != "" {
			out = append(out, s)
		}
	}
	return out
}

func trunc(s string) string {
	if len(s) > 200 {
		return s[:200] + "…"
	}
	return s
}
