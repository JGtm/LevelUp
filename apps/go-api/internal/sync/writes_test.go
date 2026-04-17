//go:build integration

// Package sync — writes_test.go : tests d'intégration des helpers d'écriture DuckDB.
package sync_test

import (
	"testing"
	"time"

	intsync "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/testutil"
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
