//go:build cgo

// apply_tz_migration — ajoute start_time_utc/end_time_utc à match_registry et
// backfille la convention Paris/UTC via real_start_time.
//
// Usage : go run -tags cgo ./cmd/apply_tz_migration/
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	path := "../../data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	if _, err := os.Stat(path); err != nil {
		log.Fatalf("DB non trouvée : %s", path)
	}

	db, err := sql.Open("duckdb", path)
	if err != nil {
		log.Fatalf("open: %v", err)
	}
	defer db.Close()

	// DDL : ajout colonnes (idempotent)
	for _, ddl := range []string{
		"ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS start_time_utc TIMESTAMPTZ",
		"ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS end_time_utc   TIMESTAMPTZ",
	} {
		if _, err := db.Exec(ddl); err != nil {
			log.Fatalf("DDL: %v\n  SQL: %s", err, ddl)
		}
	}
	fmt.Println("DDL: colonnes ajoutées (ou déjà présentes)")

	// Backfill start_time_utc (re-calculé pour tous les matchs)
	// Convention détectée par match via real_start_time (= start_time + countdown film).
	// Si real_start_time = start_time exactement (film_match_start_ms absent) → fallback first_sync_at.
	res, err := db.Exec(`
		UPDATE match_registry SET
			start_time_utc = CASE
				WHEN real_start_time IS NOT NULL
				  AND EPOCH(real_start_time) != EPOCH(start_time)
				  AND ABS(EPOCH(real_start_time) - EPOCH(start_time)) < 1800
				  THEN start_time AT TIME ZONE 'UTC'
				WHEN real_start_time IS NOT NULL
				  AND EPOCH(real_start_time) != EPOCH(start_time)
				  THEN start_time AT TIME ZONE 'Europe/Paris'
				WHEN first_sync_at >= TIMESTAMP '2026-03-01'
				  THEN start_time AT TIME ZONE 'UTC'
				ELSE
				  start_time AT TIME ZONE 'Europe/Paris'
			END
		WHERE start_time IS NOT NULL
	`)
	if err != nil {
		log.Fatalf("backfill start_time_utc: %v", err)
	}
	n, _ := res.RowsAffected()
	fmt.Printf("Backfill start_time_utc : %d lignes mises à jour\n", n)

	// Backfill end_time_utc
	res, err = db.Exec(`
		UPDATE match_registry SET
			end_time_utc = start_time_utc + (duration_seconds * INTERVAL '1 second')
		WHERE end_time_utc IS NULL
		  AND start_time_utc IS NOT NULL
		  AND duration_seconds IS NOT NULL
	`)
	if err != nil {
		log.Fatalf("backfill end_time_utc: %v", err)
	}
	n, _ = res.RowsAffected()
	fmt.Printf("Backfill end_time_utc   : %d lignes mises à jour\n", n)

	// Vérification
	var total, withUtc, nullUtc int
	_ = db.QueryRow("SELECT COUNT(*) FROM match_registry").Scan(&total)
	_ = db.QueryRow("SELECT COUNT(*) FROM match_registry WHERE start_time_utc IS NOT NULL").Scan(&withUtc)
	_ = db.QueryRow("SELECT COUNT(*) FROM match_registry WHERE start_time_utc IS NULL AND start_time IS NOT NULL").Scan(&nullUtc)
	fmt.Printf("\nVérification : total=%d, avec start_time_utc=%d, sans (à corriger)=%d\n",
		total, withUtc, nullUtc)

	if nullUtc > 0 {
		fmt.Printf("ATTENTION : %d matchs sans start_time_utc malgré start_time non-null\n", nullUtc)
	} else {
		fmt.Println("OK : tous les matchs ont start_time_utc renseigné.")
	}
}
