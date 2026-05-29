// Command backfill_first_joined_tz re-normalise first_joined_time / last_leave_time
// pour les matchs dont ces timestamps ont été stockés en heure locale (Europe/Paris)
// au lieu d'UTC — héritage d'un ancien chemin d'écriture.
//
// Règle VALIDÉE (preuve interne sur 964 matchs, 2026-05-29) : le décalage est
// toujours l'offset Europe/Paris exact (+3600s CET / +7200s CEST), déductible
// par match : offset_local = epoch(start_time as-UTC) − epoch(start_time_utc).
//
// Détection match décalé : MIN(first_joined des present_at_beginning non-bot)
// − start_utc > 120s. Correction : first_joined ET last_leave −= offset_local
// pour TOUS les participants du match. Idempotent (un match corrigé n'est plus
// détecté). Les matchs corrects (T0 apparent ≤ 120s) ne sont jamais touchés.
//
// Usage :
//
//	go run ./cmd/backfill_first_joined_tz --db <shared.duckdb>            # dry-run
//	go run ./cmd/backfill_first_joined_tz --db <shared.duckdb> --commit   # écrit
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"
)

const shiftThresholdSec = 120

type decaledMatch struct {
	matchID string
	offset  int64 // secondes (3600 ou 7200)
}

func main() {
	dbPath := flag.String("db", "", "chemin shared_matches_v2.duckdb")
	commit := flag.Bool("commit", false, "écrit en DB (défaut: dry-run lecture seule)")
	flag.Parse()
	if *dbPath == "" {
		fmt.Println("usage: backfill_first_joined_tz --db <shared.duckdb> [--commit]")
		os.Exit(2)
	}

	mode := "read_only"
	if *commit {
		mode = "read_write"
	}
	db, err := sql.Open("duckdb", *dbPath+"?access_mode="+mode)
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	defer db.Close()

	matches, err := loadDecaledMatches(db)
	if err != nil {
		fmt.Println("detect:", err)
		os.Exit(1)
	}

	var cet, cest, other int
	for _, m := range matches {
		switch m.offset {
		case 3600:
			cet++
		case 7200:
			cest++
		default:
			other++
		}
	}
	affected, _ := countAffectedRows(db, matches)

	fmt.Printf("=== Re-normalisation first_joined_time / last_leave_time ===\n\n")
	fmt.Printf("  Matchs décalés détectés : %d\n", len(matches))
	fmt.Printf("    offset +1h (CET)  : %d\n", cet)
	fmt.Printf("    offset +2h (CEST) : %d\n", cest)
	if other > 0 {
		fmt.Printf("    offset AUTRE      : %d  (ignorés, garde-fou)\n", other)
	}
	fmt.Printf("  Lignes participants à corriger : %d\n", affected)

	if !*commit {
		fmt.Println("\n[DRY-RUN] aucune écriture. Relancer avec --commit pour persister.")
		return
	}

	tx, err := db.Begin()
	if err != nil {
		fmt.Println("begin:", err)
		os.Exit(1)
	}
	stmt, err := tx.Prepare(`
		UPDATE match_participants
		SET first_joined_time = first_joined_time - to_seconds(?),
		    last_leave_time   = last_leave_time   - to_seconds(?)
		WHERE match_id = ? AND first_joined_time IS NOT NULL`)
	if err != nil {
		fmt.Println("prepare:", err)
		os.Exit(1)
	}
	var rowsWritten int64
	for _, m := range matches {
		if m.offset != 3600 && m.offset != 7200 {
			continue // garde-fou : ne corriger que les offsets horaires connus
		}
		res, err := stmt.Exec(m.offset, m.offset, m.matchID)
		if err != nil {
			_ = tx.Rollback()
			fmt.Printf("update %s: %v\n", m.matchID, err)
			os.Exit(1)
		}
		n, _ := res.RowsAffected()
		rowsWritten += n
	}
	if err := tx.Commit(); err != nil {
		fmt.Println("commit:", err)
		os.Exit(1)
	}
	fmt.Printf("\n[COMMIT] %d lignes corrigées sur %d matchs.\n", rowsWritten, len(matches))
}

// loadDecaledMatches identifie les matchs dont first_joined est décalé et
// retourne leur offset Europe/Paris (secondes).
func loadDecaledMatches(db *sql.DB) ([]decaledMatch, error) {
	rows, err := db.Query(`
		WITH det AS (
			SELECT r.match_id,
				CAST((epoch_ms(r.start_time AT TIME ZONE 'UTC') - epoch_ms(r.start_time_utc))/1000 AS BIGINT) AS offset_s,
				CAST((epoch_ms(MIN(p.first_joined_time)) - epoch_ms(r.start_time_utc))/1000 AS BIGINT) AS t0_apparent
			FROM match_participants p
			JOIN match_registry r USING (match_id)
			WHERE p.present_at_beginning = true
			  AND p.first_joined_time IS NOT NULL
			  AND p.xuid SIMILAR TO '[0-9]+'
			GROUP BY r.match_id, r.start_time, r.start_time_utc
		)
		SELECT match_id, offset_s
		FROM det
		WHERE t0_apparent > ? AND offset_s IN (3600, 7200)
		ORDER BY match_id`, shiftThresholdSec)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []decaledMatch
	for rows.Next() {
		var m decaledMatch
		if err := rows.Scan(&m.matchID, &m.offset); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// countAffectedRows compte les lignes participants (first_joined non-null) des
// matchs décalés — estimation du volume corrigé.
func countAffectedRows(db *sql.DB, matches []decaledMatch) (int64, error) {
	if len(matches) == 0 {
		return 0, nil
	}
	ids := make([]any, len(matches))
	ph := ""
	for i, m := range matches {
		ids[i] = m.matchID
		if i > 0 {
			ph += ","
		}
		ph += "?"
	}
	var n int64
	err := db.QueryRow(
		"SELECT COUNT(*) FROM match_participants WHERE first_joined_time IS NOT NULL AND match_id IN ("+ph+")",
		ids...,
	).Scan(&n)
	return n, err
}
