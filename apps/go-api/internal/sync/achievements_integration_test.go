//go:build integration

// Package sync — achievements_integration_test.go : tests d'intégration DuckDB pour la sync achievements.
//
// Ces tests utilisent DuckDB in-memory. Ils sont exclus du build unitaire.
// Lancer avec : go test -tags integration ./internal/sync/...
package sync

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openMemForAchievements ouvre deux DuckDB in-memory avec les tables achievements.
func openMemForAchievements(t *testing.T) (metadataDB *sql.DB, playerDB *sql.DB) {
	t.Helper()

	metadataDB, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open metadataDB: %v", err)
	}
	t.Cleanup(func() { metadataDB.Close() })

	_, err = metadataDB.Exec(`
		CREATE TABLE IF NOT EXISTS xbox_achievement_definitions (
			achievement_id   VARCHAR PRIMARY KEY,
			name_en          VARCHAR NOT NULL DEFAULT '',
			name_fr          VARCHAR NOT NULL DEFAULT '',
			description_en   VARCHAR,
			description_fr   VARCHAR,
			locked_desc_en   VARCHAR,
			locked_desc_fr   VARCHAR,
			gamerscore       INTEGER NOT NULL DEFAULT 0,
			image_url        VARCHAR,
			is_secret        BOOLEAN NOT NULL DEFAULT FALSE,
			rarity_category  VARCHAR,
			rarity_percent   FLOAT,
			fetched_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("CREATE xbox_achievement_definitions: %v", err)
	}

	playerDB, err = sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open playerDB: %v", err)
	}
	t.Cleanup(func() { playerDB.Close() })

	_, err = playerDB.Exec(`
		CREATE TABLE IF NOT EXISTS player_achievements (
			achievement_id    VARCHAR PRIMARY KEY,
			unlocked          BOOLEAN NOT NULL DEFAULT FALSE,
			unlocked_at       TIMESTAMP,
			current_progress  INTEGER,
			target_progress   INTEGER,
			fetched_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("CREATE player_achievements: %v", err)
	}

	return metadataDB, playerDB
}

func TestSyncAchievements_Integration_UpsertInsertsRows(t *testing.T) {
	metadataDB, playerDB := openMemForAchievements(t)

	client := newMockXboxClient()
	client.responses["en-US"] = fixtureAchievementsEN
	client.responses["fr-FR"] = fixtureAchievementsFR

	if err := SyncAchievements(context.Background(), client, nil, metadataDB, playerDB, "xuid-test"); err != nil {
		t.Fatalf("SyncAchievements: %v", err)
	}

	// Vérifier 3 lignes dans xbox_achievement_definitions
	var count int
	if err := metadataDB.QueryRow("SELECT COUNT(*) FROM xbox_achievement_definitions").Scan(&count); err != nil {
		t.Fatalf("count metadata: %v", err)
	}
	if count != 3 {
		t.Errorf("attendu 3 lignes dans xbox_achievement_definitions, obtenu %d", count)
	}

	// Vérifier données bilingues
	var nameEN, nameFR string
	if err := metadataDB.QueryRow(
		"SELECT name_en, name_fr FROM xbox_achievement_definitions WHERE achievement_id = '1'",
	).Scan(&nameEN, &nameFR); err != nil {
		t.Fatalf("select ID 1: %v", err)
	}
	if nameEN != "First Steps" {
		t.Errorf("name_en attendu 'First Steps', obtenu %q", nameEN)
	}
	if nameFR != "Premiers pas" {
		t.Errorf("name_fr attendu 'Premiers pas', obtenu %q", nameFR)
	}

	// Vérifier player_achievements
	if err := playerDB.QueryRow("SELECT COUNT(*) FROM player_achievements").Scan(&count); err != nil {
		t.Fatalf("count player: %v", err)
	}
	if count != 3 {
		t.Errorf("attendu 3 lignes dans player_achievements, obtenu %d", count)
	}

	// ID "1" doit être unlocked
	var unlocked bool
	if err := playerDB.QueryRow(
		"SELECT unlocked FROM player_achievements WHERE achievement_id = '1'",
	).Scan(&unlocked); err != nil {
		t.Fatalf("select player achievement 1: %v", err)
	}
	if !unlocked {
		t.Error("achievement 1 attendu unlocked=true")
	}
}

func TestSyncAchievements_Integration_UpsertIdempotent(t *testing.T) {
	metadataDB, playerDB := openMemForAchievements(t)

	client := newMockXboxClient()
	client.responses["en-US"] = fixtureAchievementsEN
	client.responses["fr-FR"] = fixtureAchievementsFR

	// Deux syncs consécutives → toujours 3 lignes (pas de doublons)
	for i := 0; i < 2; i++ {
		if err := SyncAchievements(context.Background(), client, nil, metadataDB, playerDB, "xuid-test"); err != nil {
			t.Fatalf("SyncAchievements (appel %d): %v", i+1, err)
		}
	}

	var count int
	if err := metadataDB.QueryRow("SELECT COUNT(*) FROM xbox_achievement_definitions").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 3 {
		t.Errorf("attendu 3 lignes après double upsert, obtenu %d", count)
	}
}

func TestSyncAchievements_Integration_APICallsEN_FR(t *testing.T) {
	metadataDB, playerDB := openMemForAchievements(t)

	client := newMockXboxClient()
	client.responses["en-US"] = fixtureAchievementsEN
	client.responses["fr-FR"] = fixtureAchievementsFR

	if err := SyncAchievements(context.Background(), client, nil, metadataDB, playerDB, "xuid-test"); err != nil {
		t.Fatalf("SyncAchievements: %v", err)
	}

	// Les deux langs doivent avoir été appelées exactement une fois
	if client.callCount["en-US"] != 1 {
		t.Errorf("en-US attendu 1 appel, obtenu %d", client.callCount["en-US"])
	}
	if client.callCount["fr-FR"] != 1 {
		t.Errorf("fr-FR attendu 1 appel, obtenu %d", client.callCount["fr-FR"])
	}
}

func TestSyncAchievements_Integration_APIError_ReturnError(t *testing.T) {
	metadataDB, playerDB := openMemForAchievements(t)

	client := newMockXboxClient()
	client.err = context.DeadlineExceeded

	err := SyncAchievements(context.Background(), client, nil, metadataDB, playerDB, "xuid-test")
	if err == nil {
		t.Fatal("attendu une erreur, obtenu nil")
	}
}

func TestSyncAchievements_Integration_LockedDescBilingual(t *testing.T) {
	metadataDB, playerDB := openMemForAchievements(t)

	enData := []PlayerAchievementRaw{
		{ID: "99", Name: "Mystery", LockedDesc: "Complete the hidden challenge", Gamerscore: 20, IsSecret: true},
	}
	frData := []PlayerAchievementRaw{
		{ID: "99", Name: "Mystère", LockedDesc: "Terminez le défi caché"},
	}

	client := newMockXboxClient()
	client.responses["en-US"] = enData
	client.responses["fr-FR"] = frData

	if err := SyncAchievements(context.Background(), client, nil, metadataDB, playerDB, "xuid-test"); err != nil {
		t.Fatalf("SyncAchievements: %v", err)
	}

	var lockedEN, lockedFR string
	if err := metadataDB.QueryRow(
		"SELECT locked_desc_en, locked_desc_fr FROM xbox_achievement_definitions WHERE achievement_id = '99'",
	).Scan(&lockedEN, &lockedFR); err != nil {
		t.Fatalf("select ID 99: %v", err)
	}
	if lockedEN != "Complete the hidden challenge" {
		t.Errorf("locked_desc_en inattendu: %q", lockedEN)
	}
	if lockedFR != "Terminez le défi caché" {
		t.Errorf("locked_desc_fr inattendu: %q", lockedFR)
	}
}
