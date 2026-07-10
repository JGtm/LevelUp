//go:build integration

// Package sync — writes_test.go : tests d'intégration des helpers d'écriture DuckDB.
package sync_test

import (
	"database/sql"
	"testing"

	"levelup/go-api/internal/domain"
	intsync "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/testutil"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestUpsertXUIDAlias(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	if err := intsync.UpsertXUIDAlias(t.Context(), db, "xuid-001", "PlayerOne"); err != nil {
		t.Fatalf("UpsertXUIDAlias: %v", err)
	}

	var gamertag string
	if err := db.QueryRow("SELECT gamertag FROM xuid_aliases WHERE xuid = ?", "xuid-001").Scan(&gamertag); err != nil {
		t.Fatal(err)
	}
	if gamertag != "PlayerOne" {
		t.Errorf("expected PlayerOne, got %s", gamertag)
	}

	// Upsert avec un nouveau gamertag → mise à jour
	if err := intsync.UpsertXUIDAlias(t.Context(), db, "xuid-001", "PlayerOneRenamed"); err != nil {
		t.Fatalf("UpsertXUIDAlias update: %v", err)
	}
	_ = db.QueryRow("SELECT gamertag FROM xuid_aliases WHERE xuid = ?", "xuid-001").Scan(&gamertag)
	if gamertag != "PlayerOneRenamed" {
		t.Errorf("expected PlayerOneRenamed after upsert, got %s", gamertag)
	}
}

func TestSetSyncMeta(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	if err := intsync.SetSyncMeta(t.Context(), db, "last_sync", "2025-01-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}

	var val string
	_ = db.QueryRow("SELECT value FROM sync_meta WHERE key = 'last_sync'").Scan(&val)
	if val != "2025-01-01T00:00:00Z" {
		t.Errorf("expected 2025-01-01T00:00:00Z, got %s", val)
	}

	// Upsert
	_ = intsync.SetSyncMeta(t.Context(), db, "last_sync", "2025-06-01T00:00:00Z")
	_ = db.QueryRow("SELECT value FROM sync_meta WHERE key = 'last_sync'").Scan(&val)
	if val != "2025-06-01T00:00:00Z" {
		t.Errorf("expected 2025-06-01T00:00:00Z after upsert, got %s", val)
	}
}

func TestUpsertPlayerEnrichment(t *testing.T) {
	db := testutil.NewInMemoryPlayer(t)

	// Insert new enrichment
	if err := intsync.UpsertPlayerEnrichment(t.Context(), db, "m1", "sig-abc"); err != nil {
		t.Fatalf("UpsertPlayerEnrichment: %v", err)
	}

	var sig string
	_ = db.QueryRow("SELECT teammates_signature FROM player_match_enrichment_latest WHERE match_id = 'm1'").Scan(&sig)
	if sig != "sig-abc" {
		t.Errorf("expected sig-abc, got %s", sig)
	}

	// Upsert with new signature
	if err := intsync.UpsertPlayerEnrichment(t.Context(), db, "m1", "sig-def"); err != nil {
		t.Fatalf("UpsertPlayerEnrichment update: %v", err)
	}
	_ = db.QueryRow("SELECT teammates_signature FROM player_match_enrichment_latest WHERE match_id = 'm1'").Scan(&sig)
	if sig != "sig-def" {
		t.Errorf("expected sig-def after upsert, got %s", sig)
	}

	// Upsert with empty string → should preserve existing via COALESCE
	if err := intsync.UpsertPlayerEnrichment(t.Context(), db, "m1", ""); err != nil {
		t.Fatalf("UpsertPlayerEnrichment empty: %v", err)
	}
	_ = db.QueryRow("SELECT teammates_signature FROM player_match_enrichment_latest WHERE match_id = 'm1'").Scan(&sig)
	if sig != "sig-def" {
		t.Errorf("expected sig-def preserved after empty upsert, got %s", sig)
	}
}

func TestInsertWeaponKills(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	// Seed a match for FK reference
	_, _ = db.Exec("INSERT INTO match_registry (match_id) VALUES ('m1')")

	wid1 := uint64(1234)
	wid2 := uint64(5678)
	rec := uint64(9999)
	delta := 150
	pidx := 3
	attrs := []intsync.WeaponKillRow{
		{TimeMS: 1000, WeaponID: &wid1, Confidence: "high", AttributionPath: "direct"},
		{TimeMS: 2000, WeaponID: &wid2, ReconciledAs: &rec, DeltaMS: &delta, Confidence: "medium", PlayerIndex: &pidx},
	}
	if err := intsync.InsertWeaponKills(t.Context(), db, "m1", "x1", attrs); err != nil {
		t.Fatalf("InsertWeaponKills: %v", err)
	}

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM weapon_kills WHERE match_id = 'm1' AND xuid = 'x1'").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 weapon_kills, got %d", count)
	}

	// Append-only #23046 (Phase 2) : la ré-insertion ne DELETE plus ; elle alloue une
	// nouvelle génération qui supersède via v_weapon_kills (dernière génération) → 1 row.
	attrs2 := []intsync.WeaponKillRow{
		{TimeMS: 3000, WeaponID: &wid1, Confidence: "low", AttributionPath: "swap"},
	}
	if err := intsync.InsertWeaponKills(t.Context(), db, "m1", "x1", attrs2); err != nil {
		t.Fatalf("InsertWeaponKills replace: %v", err)
	}
	_ = db.QueryRow("SELECT COUNT(*) FROM v_weapon_kills WHERE match_id = 'm1' AND xuid = 'x1'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 weapon_kill (v_weapon_kills dernière génération) after replace, got %d", count)
	}
}

func TestMarkWeaponKillsDone(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	// Create minimal match_registry schema (same as openPveDB)
	_, err = db.Exec("CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, backfill_completed INTEGER DEFAULT 0)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Seed a match
	_, _ = db.Exec("INSERT INTO match_registry (match_id) VALUES ('m1')")

	if err := intsync.MarkWeaponKillsDone(t.Context(), db, "m1", false); err != nil {
		t.Fatalf("MarkWeaponKillsDone: %v", err)
	}

	var bits int
	_ = db.QueryRow("SELECT backfill_completed FROM match_registry WHERE match_id = 'm1'").Scan(&bits)
	if bits&intsync.MBitWeaponKills == 0 {
		t.Errorf("expected MBitWeaponKills set, got bits=%d", bits)
	}

	// Mark no-film variant
	_, _ = db.Exec("INSERT INTO match_registry (match_id) VALUES ('m2')")
	if err := intsync.MarkWeaponKillsDone(t.Context(), db, "m2", true); err != nil {
		t.Fatalf("MarkWeaponKillsDone noFilm: %v", err)
	}
	_ = db.QueryRow("SELECT backfill_completed FROM match_registry WHERE match_id = 'm2'").Scan(&bits)
	if bits&intsync.MBitWeaponKillsNoFilm == 0 {
		t.Errorf("expected MBitWeaponKillsNoFilm set, got bits=%d", bits)
	}
}

func TestWriteSessionAssignments_UpdatesRows(t *testing.T) {
	db := testutil.NewInMemoryPlayer(t)

	// Seed deux matchs dans player_match_enrichment.
	_, err := db.Exec(`INSERT INTO player_match_enrichment (match_id) VALUES ('m1'), ('m2'), ('m3')`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	assignments := []domain.SessionAssignment{
		{MatchID: "m1", SessionID: 1, SessionLabel: "Session 1"},
		{MatchID: "m2", SessionID: 1, SessionLabel: "Session 1"},
		{MatchID: "m3", SessionID: 2, SessionLabel: "Session 2"},
	}
	n, err := intsync.WriteSessionAssignments(t.Context(), db, assignments)
	if err != nil {
		t.Fatalf("WriteSessionAssignments: %v", err)
	}
	if n != 3 {
		t.Errorf("expected 3 rows updated, got %d", n)
	}

	// Vérifier les valeurs écrites.
	var sid, slabel string
	_ = db.QueryRow("SELECT session_id, session_label FROM player_match_enrichment_latest WHERE match_id = 'm1'").Scan(&sid, &slabel)
	if sid != "1" || slabel != "Session 1" {
		t.Errorf("m1: got session_id=%q session_label=%q, want 1 / 'Session 1'", sid, slabel)
	}
	_ = db.QueryRow("SELECT session_id, session_label FROM player_match_enrichment_latest WHERE match_id = 'm3'").Scan(&sid, &slabel)
	if sid != "2" || slabel != "Session 2" {
		t.Errorf("m3: got session_id=%q session_label=%q, want 2 / 'Session 2'", sid, slabel)
	}
}

func TestWriteSessionAssignments_EmptySlice(t *testing.T) {
	db := testutil.NewInMemoryPlayer(t)
	n, err := intsync.WriteSessionAssignments(t.Context(), db, nil)
	if err != nil {
		t.Fatalf("WriteSessionAssignments(nil): %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}

// TestWriteSessionAssignments_NewMatchInsertsRow : append-only #23046 — un match
// sans row pré-existante reçoit désormais sa row session-stage (INSERT pur, plus
// d'UPDATE no-op). Le writer retourne 1 et la session est lisible via la vue merge.
func TestWriteSessionAssignments_NewMatchInsertsRow(t *testing.T) {
	db := testutil.NewInMemoryPlayer(t)

	assignments := []domain.SessionAssignment{
		{MatchID: "newmatch", SessionID: 1, SessionLabel: "S1"},
	}
	n, err := intsync.WriteSessionAssignments(t.Context(), db, assignments)
	if err != nil {
		t.Fatalf("WriteSessionAssignments: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 row inserted (append-only), got %d", n)
	}
	var sid int
	var label string
	if err := db.QueryRow(
		`SELECT CAST(session_id AS INT), session_label FROM player_match_enrichment_latest WHERE match_id='newmatch'`,
	).Scan(&sid, &label); err != nil {
		t.Fatalf("read _latest: %v", err)
	}
	if sid != 1 || label != "S1" {
		t.Errorf("session merge = (%d,%q), want (1,S1)", sid, label)
	}
}
