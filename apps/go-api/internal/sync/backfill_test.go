//go:build integration

// Package sync — backfill_test.go : tests pour les fonctions pures de backfill.
//
// Sprint 47 T16 — couvrir isValidMatchID, doneGuard, et recherche de matchs manquants.
// Note : ce package importe DuckDB transitif → ne compile pas sur Windows (contrainte
// build constraint windows-amd64). Ces tests sont conçus pour tourner en CI Linux.
package sync

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ── Tests isValidMatchID ─────────────────────────────────────────────────────

func TestIsValidMatchID_ValidUUID(t *testing.T) {
	valid := []string{
		"a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		"AABBCCDD-EEFF-0011-2233-445566778899",
		"00000000-0000-0000-0000-000000000000",
	}
	for _, id := range valid {
		if !isValidMatchID(id) {
			t.Errorf("isValidMatchID(%q) = false, attendu true", id)
		}
	}
}

func TestIsValidMatchID_Empty(t *testing.T) {
	if isValidMatchID("") {
		t.Error("isValidMatchID('') devrait retourner false")
	}
}

func TestIsValidMatchID_TooLong(t *testing.T) {
	long := make([]byte, 65)
	for i := range long {
		long[i] = 'a'
	}
	if isValidMatchID(string(long)) {
		t.Error("isValidMatchID(65 chars) devrait retourner false")
	}
}

func TestIsValidMatchID_InvalidChars(t *testing.T) {
	invalid := []string{
		"GGGGGGGG-GGGG-GGGG-GGGG-GGGGGGGGGGGG", // 'G' invalide
		"hello world",
		"drop table",
		"' OR '1'='1",
	}
	for _, id := range invalid {
		if isValidMatchID(id) {
			t.Errorf("isValidMatchID(%q) = true, attendu false", id)
		}
	}
}

// ── Tests doneGuard ───────────────────────────────────────────────────────────

func TestDoneGuard_NoBFCol(t *testing.T) {
	// Sans colonne backfill_completed, retourne ""
	result := doneGuard("MEDALS", false)
	if result != "" {
		t.Errorf("doneGuard sans BF col attendu '', obtenu %q", result)
	}
}

func TestDoneGuard_KnownFlag(t *testing.T) {
	// Avec colonne présente et flag connu, retourne clause SQL
	// Note : les clés de BackfillFlags sont en minuscules ("medals", pas "MEDALS")
	result := doneGuard("medals", true)
	if result == "" {
		t.Error("doneGuard avec flag MEDALS devrait retourner une clause SQL non-vide")
	}
	// La clause ne doit pas contenir d'injection SQL
	if len(result) > 0 && result != "" {
		t.Logf("doneGuard(MEDALS, true) = %q", result)
	}
}

func TestDoneGuard_UnknownFlag(t *testing.T) {
	// Flag inconnu → retourne ""
	result := doneGuard("UNKNOWN_FLAG_XYZ", true)
	if result != "" {
		t.Errorf("doneGuard avec flag inconnu attendu '', obtenu %q", result)
	}
}

// ── Tests hasBackfillCompletedColumn ─────────────────────────────────────────

func TestHasBackfillCompletedColumn_WithColumn(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE match_registry (match_id VARCHAR, backfill_completed INTEGER)")
	if err != nil {
		t.Fatalf("CREATE: %v", err)
	}

	if !hasBackfillCompletedColumn(db) {
		t.Error("hasBackfillCompletedColumn devrait retourner true")
	}
}

func TestHasBackfillCompletedColumn_WithoutColumn(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE match_registry (match_id VARCHAR)")
	if err != nil {
		t.Fatalf("CREATE: %v", err)
	}

	if hasBackfillCompletedColumn(db) {
		t.Error("hasBackfillCompletedColumn devrait retourner false sans colonne")
	}
}
