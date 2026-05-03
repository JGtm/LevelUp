//go:build cgo

// apply_tz_migration — re-backfill start_time_utc / end_time_utc via la TZ de
// session (= celle utilisée par le sync à l'écriture). Outil one-shot pour
// rattraper les matchs déjà stockés. La logique est identique à la migration
// fix_start_time_utc_via_session_tz enregistrée dans steps_shared.go.
//
// Usage : go run -tags cgo ./cmd/apply_tz_migration/
//
// Le cast `start_time::TIMESTAMPTZ` interprète les bytes naïfs avec la session
// TZ courante. Comme le sync écrit toujours sous la session TZ user (e.g.
// Europe/Paris), le cast est l'inverse exact de la conversion d'écriture →
// résultat UTC correct, sans heuristique.
package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"os"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

func main() {
	path := "../../data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb"
	tz := "Europe/Paris"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	if len(os.Args) > 2 {
		tz = os.Args[2]
	}

	if _, err := os.Stat(path); err != nil {
		log.Fatalf("DB non trouvée : %s", path)
	}

	// Ouverture avec session TimeZone explicite — doit matcher celle utilisée
	// par le pool DB du serveur lors des syncs précédents (cf. app_settings.json).
	connector, err := duckdb.NewConnector(path, func(execer driver.ExecerContext) error {
		_, e := execer.ExecContext(context.Background(), "SET TimeZone='"+tz+"'", nil)
		return e
	})
	if err != nil {
		log.Fatalf("connector: %v", err)
	}
	db := sql.OpenDB(connector)
	defer db.Close()
	fmt.Printf("Session TimeZone = %q\n", tz)

	// DDL idempotent (au cas où les colonnes n'existeraient pas — rare, normalement
	// ajoutées par add_start_time_utc_to_match_registry).
	for _, ddl := range []string{
		"ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS start_time_utc TIMESTAMPTZ",
		"ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS end_time_utc   TIMESTAMPTZ",
	} {
		if _, err := db.Exec(ddl); err != nil {
			log.Fatalf("DDL: %v\n  SQL: %s", err, ddl)
		}
	}

	res, err := db.Exec(`
		UPDATE match_registry SET
			start_time_utc = start_time::TIMESTAMPTZ,
			end_time_utc   = end_time::TIMESTAMPTZ
		WHERE start_time IS NOT NULL
	`)
	if err != nil {
		log.Fatalf("backfill start_time_utc: %v", err)
	}
	n, _ := res.RowsAffected()
	fmt.Printf("Backfill start_time_utc / end_time_utc : %d lignes\n", n)

	res, err = db.Exec(`
		UPDATE match_registry SET
			end_time_utc = start_time_utc + (duration_seconds * INTERVAL '1 second')
		WHERE end_time_utc IS NULL
		  AND start_time_utc IS NOT NULL
		  AND duration_seconds IS NOT NULL
	`)
	if err != nil {
		log.Fatalf("backfill end_time_utc fallback: %v", err)
	}
	n, _ = res.RowsAffected()
	fmt.Printf("Fallback end_time_utc (depuis duration) : %d lignes\n", n)

	var total, withUtc, nullUtc int
	_ = db.QueryRow("SELECT COUNT(*) FROM match_registry").Scan(&total)
	_ = db.QueryRow("SELECT COUNT(*) FROM match_registry WHERE start_time_utc IS NOT NULL").Scan(&withUtc)
	_ = db.QueryRow("SELECT COUNT(*) FROM match_registry WHERE start_time_utc IS NULL AND start_time IS NOT NULL").Scan(&nullUtc)
	fmt.Printf("\ntotal=%d, avec start_time_utc=%d, sans (KO)=%d\n", total, withUtc, nullUtc)
	if nullUtc > 0 {
		fmt.Printf("ATTENTION : %d matchs sans start_time_utc malgré start_time non-null\n", nullUtc)
	} else {
		fmt.Println("OK : tous les matchs ont start_time_utc renseigné.")
	}
}
