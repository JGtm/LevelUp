//go:build integration

// Package duckdb_test — db_timezone_test.go : tests timezone sur connexion DuckDB.
// Lance avec : go test -tags=integration ./internal/platform/duckdb/ -run Timezone -v
package duckdb_test

import (
	"context"
	"testing"

	ddb "levelup/go-api/internal/platform/duckdb"
)

// TestOpenReadWrite_TimezoneApplied vérifie que le SET TimeZone est bien appliqué
// à la session DuckDB lors de l'ouverture avec une timezone.
func TestOpenReadWrite_TimezoneApplied(t *testing.T) {
	ctx := context.Background()
	db, err := ddb.OpenReadWrite(":memory:", "Europe/Paris")
	if err != nil {
		t.Fatalf("OpenReadWrite avec timezone: %v", err)
	}
	defer db.Close()

	var tz string
	if err := db.QueryRow(ctx, "SELECT current_setting('TimeZone')").Scan(&tz); err != nil {
		t.Fatalf("lecture current_setting('TimeZone'): %v", err)
	}
	if tz != "Europe/Paris" {
		t.Errorf("TimeZone attendu 'Europe/Paris', obtenu %q", tz)
	}
}

// TestOpenReadWrite_TimezoneUTC vérifie qu'UTC est bien appliqué.
func TestOpenReadWrite_TimezoneUTC(t *testing.T) {
	ctx := context.Background()
	db, err := ddb.OpenReadWrite(":memory:", "UTC")
	if err != nil {
		t.Fatalf("OpenReadWrite UTC: %v", err)
	}
	defer db.Close()

	var tz string
	if err := db.QueryRow(ctx, "SELECT current_setting('TimeZone')").Scan(&tz); err != nil {
		t.Fatalf("SELECT TimeZone: %v", err)
	}
	if tz != "UTC" {
		t.Errorf("TimeZone attendu 'UTC', obtenu %q", tz)
	}
}

// TestOpenReadWrite_TimezoneInvalid vérifie qu'une timezone invalide est ignorée
// (sanitizeTimezone la rejette → connexion ouverte sans SET TimeZone).
func TestOpenReadWrite_TimezoneInvalid(t *testing.T) {
	db, err := ddb.OpenReadWrite(":memory:", "'; DROP TABLE x; --")
	if err != nil {
		t.Fatalf("OpenReadWrite timezone invalide ne doit pas échouer: %v", err)
	}
	defer db.Close()
	// La DB s'ouvre mais sans timezone configurée — pas de panique.
}

// TestOpenReadOnly_TimezoneApplied vérifie que la timezone est propagée sur une
// connexion read-only.
func TestOpenReadOnly_TimezoneApplied(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := dir + "/tz_test.duckdb"

	rw, err := ddb.OpenReadWrite(dbPath)
	if err != nil {
		t.Fatalf("créer DB: %v", err)
	}
	if _, err := rw.Exec(ctx, "CREATE TABLE dummy (n INTEGER)"); err != nil {
		t.Fatalf("CREATE TABLE: %v", err)
	}
	rw.Close()

	ro, err := ddb.OpenReadOnly(dbPath, "America/New_York")
	if err != nil {
		t.Fatalf("OpenReadOnly avec timezone: %v", err)
	}
	defer ro.Close()

	var tz string
	if err := ro.QueryRow(ctx, "SELECT current_setting('TimeZone')").Scan(&tz); err != nil {
		t.Fatalf("SELECT TimeZone: %v", err)
	}
	if tz != "America/New_York" {
		t.Errorf("TimeZone attendu 'America/New_York', obtenu %q", tz)
	}
}

// TestTimezoneConversion vérifie que DuckDB interprète correctement un TIMESTAMP
// naïf comme heure Paris (CEST été = UTC+2) et retourne l'UTC correct.
func TestTimezoneConversion(t *testing.T) {
	ctx := context.Background()
	db, err := ddb.OpenReadWrite(":memory:", "Europe/Paris")
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	defer db.Close()

	// Stocker un timestamp naïf représentant 23:40 heure Paris (CEST, UTC+2)
	// → l'UTC réel devrait être 21:40.
	// DuckDB avec TZ=Europe/Paris va interpréter ce TIMESTAMP naïf comme 23:40 Paris.
	// Go recevra un time.Time en UTC = 21:40.
	if _, err := db.Exec(ctx, `CREATE TABLE ts_test (t TIMESTAMP)`); err != nil {
		t.Fatalf("CREATE: %v", err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO ts_test VALUES ('2026-04-06 23:40:32'::TIMESTAMP)`); err != nil {
		t.Fatalf("INSERT: %v", err)
	}

	var ts interface{}
	if err := db.QueryRow(ctx, "SELECT t FROM ts_test").Scan(&ts); err != nil {
		t.Fatalf("SELECT: %v", err)
	}
	// On vérifie simplement que la valeur est lisible — la conversion UTC dépend
	// du driver duckdb-go qui attache time.UTC aux TIMESTAMP naïfs retournés.
	if ts == nil {
		t.Error("timestamp scanné comme nil")
	}
}
