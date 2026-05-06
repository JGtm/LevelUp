//go:build integration

// Package sync — writes_test.go : tests d'intégration des helpers d'écriture DuckDB.
package sync_test

import (
	"database/sql"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	intsync "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/testutil"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestInsertRegistryIfNotExists(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	mapName := "Recharge"
	plName := "Ranked Arena"
	row := intsync.MatchRegistryRow{
		MatchID:      "test-match-001",
		StartTime:    time.Now(),
		MapName:      &mapName,
		PlaylistName: &plName,
		FirstSyncBy:  "test-player",
	}
	if err := intsync.InsertRegistryIfNotExists(db, row); err != nil {
		t.Fatalf("InsertRegistryIfNotExists: %v", err)
	}

	// Vérifier l'insertion
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM match_registry WHERE match_id = ?", "test-match-001").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}

	// INSERT OR IGNORE → pas de doublon
	if err := intsync.InsertRegistryIfNotExists(db, row); err != nil {
		t.Fatalf("second insert should not fail: %v", err)
	}
	_ = db.QueryRow("SELECT COUNT(*) FROM match_registry WHERE match_id = ?", "test-match-001").Scan(&count)
	if count != 1 {
		t.Errorf("expected still 1 row after duplicate insert, got %d", count)
	}
}

func TestUpsertXUIDAlias(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	if err := intsync.UpsertXUIDAlias(db, "xuid-001", "PlayerOne"); err != nil {
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
	if err := intsync.UpsertXUIDAlias(db, "xuid-001", "PlayerOneRenamed"); err != nil {
		t.Fatalf("UpsertXUIDAlias update: %v", err)
	}
	_ = db.QueryRow("SELECT gamertag FROM xuid_aliases WHERE xuid = ?", "xuid-001").Scan(&gamertag)
	if gamertag != "PlayerOneRenamed" {
		t.Errorf("expected PlayerOneRenamed after upsert, got %s", gamertag)
	}
}

func TestInsertMedals(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	rows := []intsync.MedalRow{
		{MatchID: "m1", XUID: "x1", MedalNameID: 100, Count: 3},
		{MatchID: "m1", XUID: "x1", MedalNameID: 200, Count: 1},
	}
	if err := intsync.InsertMedals(db, rows); err != nil {
		t.Fatalf("InsertMedals: %v", err)
	}

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM medals_earned WHERE match_id = 'm1'").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 medals, got %d", count)
	}
}

func TestSetSyncMeta(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	if err := intsync.SetSyncMeta(db, "last_sync", "2025-01-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}

	var val string
	_ = db.QueryRow("SELECT value FROM sync_meta WHERE key = 'last_sync'").Scan(&val)
	if val != "2025-01-01T00:00:00Z" {
		t.Errorf("expected 2025-01-01T00:00:00Z, got %s", val)
	}

	// Upsert
	_ = intsync.SetSyncMeta(db, "last_sync", "2025-06-01T00:00:00Z")
	_ = db.QueryRow("SELECT value FROM sync_meta WHERE key = 'last_sync'").Scan(&val)
	if val != "2025-06-01T00:00:00Z" {
		t.Errorf("expected 2025-06-01T00:00:00Z after upsert, got %s", val)
	}
}

func TestInsertParticipants(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	teamMMR := 1500.0
	enemyMMR := 1480.0
	teamID0, teamID1 := 0, 1
	outcome2, outcome3 := 2, 3
	kills15, kills8 := 15, 8
	deaths10, deaths12 := 10, 12
	assists5, assists3 := 5, 3
	kda20, kda092 := 2.0, 0.92
	rows := []intsync.ParticipantRow{
		{
			MatchID:  "m1",
			XUID:     "x1",
			TeamID:   &teamID0,
			Outcome:  &outcome2,
			Kills:    &kills15,
			Deaths:   &deaths10,
			Assists:  &assists5,
			KDA:      &kda20,
			TeamMMR:  &teamMMR,
			EnemyMMR: &enemyMMR,
		},
		{
			MatchID: "m1",
			XUID:    "x2",
			TeamID:  &teamID1,
			Outcome: &outcome3,
			Kills:   &kills8,
			Deaths:  &deaths12,
			Assists: &assists3,
			KDA:     &kda092,
		},
	}
	if err := intsync.InsertParticipants(db, rows); err != nil {
		t.Fatalf("InsertParticipants: %v", err)
	}

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM match_participants WHERE match_id = 'm1'").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 participants, got %d", count)
	}

	// Verify MMR propagated
	var mmr float64
	_ = db.QueryRow("SELECT team_mmr FROM match_participants WHERE xuid = 'x1'").Scan(&mmr)
	if mmr != 1500.0 {
		t.Errorf("expected team_mmr=1500.0, got %f", mmr)
	}

	// Idempotence: re-insert should not fail or duplicate
	if err := intsync.InsertParticipants(db, rows); err != nil {
		t.Fatalf("second InsertParticipants should not fail: %v", err)
	}
	_ = db.QueryRow("SELECT COUNT(*) FROM match_participants WHERE match_id = 'm1'").Scan(&count)
	if count != 2 {
		t.Errorf("expected still 2 participants after duplicate insert, got %d", count)
	}

	// Empty slice should no-op
	if err := intsync.InsertParticipants(db, nil); err != nil {
		t.Fatalf("InsertParticipants(nil) should not fail: %v", err)
	}
}

func TestUpsertPlayerEnrichment(t *testing.T) {
	db := testutil.NewInMemoryPlayer(t)

	// Insert new enrichment
	if err := intsync.UpsertPlayerEnrichment(db, "m1", "sig-abc"); err != nil {
		t.Fatalf("UpsertPlayerEnrichment: %v", err)
	}

	var sig string
	_ = db.QueryRow("SELECT teammates_signature FROM player_match_enrichment WHERE match_id = 'm1'").Scan(&sig)
	if sig != "sig-abc" {
		t.Errorf("expected sig-abc, got %s", sig)
	}

	// Upsert with new signature
	if err := intsync.UpsertPlayerEnrichment(db, "m1", "sig-def"); err != nil {
		t.Fatalf("UpsertPlayerEnrichment update: %v", err)
	}
	_ = db.QueryRow("SELECT teammates_signature FROM player_match_enrichment WHERE match_id = 'm1'").Scan(&sig)
	if sig != "sig-def" {
		t.Errorf("expected sig-def after upsert, got %s", sig)
	}

	// Upsert with empty string → should preserve existing via COALESCE
	if err := intsync.UpsertPlayerEnrichment(db, "m1", ""); err != nil {
		t.Fatalf("UpsertPlayerEnrichment empty: %v", err)
	}
	_ = db.QueryRow("SELECT teammates_signature FROM player_match_enrichment WHERE match_id = 'm1'").Scan(&sig)
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
	if err := intsync.InsertWeaponKills(db, "m1", "x1", attrs); err != nil {
		t.Fatalf("InsertWeaponKills: %v", err)
	}

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM weapon_kills WHERE match_id = 'm1' AND xuid = 'x1'").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 weapon_kills, got %d", count)
	}

	// Re-insert should replace (DELETE + INSERT)
	attrs2 := []intsync.WeaponKillRow{
		{TimeMS: 3000, WeaponID: &wid1, Confidence: "low", AttributionPath: "swap"},
	}
	if err := intsync.InsertWeaponKills(db, "m1", "x1", attrs2); err != nil {
		t.Fatalf("InsertWeaponKills replace: %v", err)
	}
	_ = db.QueryRow("SELECT COUNT(*) FROM weapon_kills WHERE match_id = 'm1' AND xuid = 'x1'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 weapon_kill after replace, got %d", count)
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

	if err := intsync.MarkWeaponKillsDone(db, "m1", false); err != nil {
		t.Fatalf("MarkWeaponKillsDone: %v", err)
	}

	var bits int
	_ = db.QueryRow("SELECT backfill_completed FROM match_registry WHERE match_id = 'm1'").Scan(&bits)
	if bits&intsync.MBitWeaponKills == 0 {
		t.Errorf("expected MBitWeaponKills set, got bits=%d", bits)
	}

	// Mark no-film variant
	_, _ = db.Exec("INSERT INTO match_registry (match_id) VALUES ('m2')")
	if err := intsync.MarkWeaponKillsDone(db, "m2", true); err != nil {
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
	n, err := intsync.WriteSessionAssignments(db, assignments)
	if err != nil {
		t.Fatalf("WriteSessionAssignments: %v", err)
	}
	if n != 3 {
		t.Errorf("expected 3 rows updated, got %d", n)
	}

	// Vérifier les valeurs écrites.
	var sid, slabel string
	_ = db.QueryRow("SELECT session_id, session_label FROM player_match_enrichment WHERE match_id = 'm1'").Scan(&sid, &slabel)
	if sid != "1" || slabel != "Session 1" {
		t.Errorf("m1: got session_id=%q session_label=%q, want 1 / 'Session 1'", sid, slabel)
	}
	_ = db.QueryRow("SELECT session_id, session_label FROM player_match_enrichment WHERE match_id = 'm3'").Scan(&sid, &slabel)
	if sid != "2" || slabel != "Session 2" {
		t.Errorf("m3: got session_id=%q session_label=%q, want 2 / 'Session 2'", sid, slabel)
	}
}

func TestWriteSessionAssignments_EmptySlice(t *testing.T) {
	db := testutil.NewInMemoryPlayer(t)
	n, err := intsync.WriteSessionAssignments(db, nil)
	if err != nil {
		t.Fatalf("WriteSessionAssignments(nil): %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}

func TestWriteSessionAssignments_MissingMatchID_Zero(t *testing.T) {
	db := testutil.NewInMemoryPlayer(t)

	// Aucune ligne dans la table — l'UPDATE ne doit pas échouer, juste 0 lignes affectées.
	assignments := []domain.SessionAssignment{
		{MatchID: "nonexistent", SessionID: 1, SessionLabel: "S1"},
	}
	n, err := intsync.WriteSessionAssignments(db, assignments)
	if err != nil {
		t.Fatalf("WriteSessionAssignments: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 rows updated for missing match_id, got %d", n)
	}
}
