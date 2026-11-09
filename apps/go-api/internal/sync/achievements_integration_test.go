//go:build integration

// Package sync — achievements_integration_test.go : tests d'intégration DuckDB pour la sync achievements.
//
// Ces tests utilisent DuckDB in-memory. Ils sont exclus du build unitaire.
// Lancer avec : go test -tags integration ./internal/sync/...
package sync

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/migration"
	ddb "levelup/go-api/internal/platform/duckdb"
)

// openMemForAchievements ouvre deux DuckDB in-memory avec les tables achievements.
// Utilise les migrations TargetMetadata/TargetPlayer pour rester en sync avec la
// prod — un DDL inline divergerait silencieusement à chaque nouvelle migration
// (ex: ajout xbox_title_id, service_config_id) et casserait les tests
// d'intégration.
func openMemForAchievements(t *testing.T) (metadataDB *sql.DB, playerDB *sql.DB) {
	t.Helper()
	_ = migration.All()

	metadataDB, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open metadataDB: %v", err)
	}
	t.Cleanup(func() { metadataDB.Close() })
	if err := migration.RunForDB(metadataDB, migration.TargetMetadata); err != nil {
		t.Fatalf("RunForDB(TargetMetadata): %v", err)
	}

	playerDB, err = sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open playerDB: %v", err)
	}
	t.Cleanup(func() { playerDB.Close() })
	if err := migration.RunForDB(playerDB, migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB(TargetPlayer): %v", err)
	}

	return metadataDB, playerDB
}

// assetText lit le nom et la description d'un actif dans asset_translations pour une langue ;
// ("", "") quand la ligne n'existe pas.
func assetText(t *testing.T, db *sql.DB, assetType, id string, l i18n.Locale) (name, desc string) {
	t.Helper()
	var d sql.NullString
	err := db.QueryRow(
		`SELECT name, description FROM asset_translations WHERE asset_type = ? AND asset_id = ? AND lang = ?`,
		assetType, id, i18n.HaloTags(l)[0]).Scan(&name, &d)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ""
	}
	if err != nil {
		t.Fatalf("asset_translations %s/%s (%s): %v", assetType, id, l, err)
	}
	return name, d.String
}

// countAssetRows compte les lignes de asset_translations d'un type.
func countAssetRows(t *testing.T, db *sql.DB, assetType string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM asset_translations WHERE asset_type = ?`, assetType).Scan(&n); err != nil {
		t.Fatalf("count asset_translations %s: %v", assetType, err)
	}
	return n
}

func TestSyncAchievements_Integration_UpsertInsertsRows(t *testing.T) {
	metadataDB, playerDB := openMemForAchievements(t)

	client := newMockXboxClient()
	client.responses["en-US"] = fixtureAchievementsEN
	client.responses["fr-FR"] = fixtureAchievementsFR

	if err := SyncAchievements(context.Background(), client, nil, metadataDB, playerDB, "xuid-test", "halo_infinite"); err != nil {
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

	// Vérifier les textes par langue (asset_translations, une ligne par langue)
	nameEN, descEN := assetText(t, metadataDB, ddb.AssetTypeAchievement, "1", i18n.LocaleEN)
	nameFR, descFR := assetText(t, metadataDB, ddb.AssetTypeAchievement, "1", i18n.LocaleFR)
	if nameEN != "First Steps" || descEN != "Win your first match" {
		t.Errorf("texte en inattendu: %q / %q", nameEN, descEN)
	}
	if nameFR != "Premiers pas" || descFR != "Gagnez votre premier match" {
		t.Errorf("texte fr inattendu: %q / %q", nameFR, descFR)
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
		if err := SyncAchievements(context.Background(), client, nil, metadataDB, playerDB, "xuid-test", "halo_infinite"); err != nil {
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

	if err := SyncAchievements(context.Background(), client, nil, metadataDB, playerDB, "xuid-test", "halo_infinite"); err != nil {
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

	err := SyncAchievements(context.Background(), client, nil, metadataDB, playerDB, "xuid-test", "halo_infinite")
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

	if err := SyncAchievements(context.Background(), client, nil, metadataDB, playerDB, "xuid-test", "halo_infinite"); err != nil {
		t.Fatalf("SyncAchievements: %v", err)
	}

	lockedEN, _ := assetText(t, metadataDB, ddb.AssetTypeAchievementLocked, "99", i18n.LocaleEN)
	lockedFR, _ := assetText(t, metadataDB, ddb.AssetTypeAchievementLocked, "99", i18n.LocaleFR)
	if lockedEN != "Complete the hidden challenge" {
		t.Errorf("texte verrouillé en inattendu: %q", lockedEN)
	}
	if lockedFR != "Terminez le défi caché" {
		t.Errorf("texte verrouillé fr inattendu: %q", lockedFR)
	}
}

// Écriture à la synchro : une ligne par langue pour laquelle le texte existe, rien d'inventé pour
// une langue absente de l'API (id "3" sans fr), et une seconde synchro met les lignes à jour sur
// place sans doublon (SELECT-puis-UPDATE-ou-INSERT).
func TestSyncAchievements_Integration_TextesParLangue(t *testing.T) {
	metadataDB, playerDB := openMemForAchievements(t)

	client := newMockXboxClient()
	client.responses["en-US"] = fixtureAchievementsEN
	client.responses["fr-FR"] = fixtureAchievementsFR
	run := func() {
		t.Helper()
		if err := SyncAchievements(context.Background(), client, nil, metadataDB, playerDB, "xuid-test", "halo_infinite"); err != nil {
			t.Fatalf("SyncAchievements: %v", err)
		}
	}
	run()

	// 3 succès en en, 2 en fr (id 3 absent côté fr).
	if n := countAssetRows(t, metadataDB, ddb.AssetTypeAchievement); n != 5 {
		t.Errorf("asset_translations achievement = %d lignes, attendu 5", n)
	}
	if name, _ := assetText(t, metadataDB, ddb.AssetTypeAchievement, "3", i18n.LocaleFR); name != "" {
		t.Errorf("aucune ligne fr attendue pour l'id 3, obtenu %q", name)
	}
	if name, _ := assetText(t, metadataDB, ddb.AssetTypeAchievement, "3", i18n.LocaleEN); name != "Legendary" {
		t.Errorf("nom en de l'id 3 = %q", name)
	}

	// Le texte fr change côté API : la ligne est mise à jour, pas dupliquée.
	client.responses["fr-FR"] = []PlayerAchievementRaw{
		{ID: "1", Name: "Premiers pas (révisé)", Description: "Gagnez votre premier match"},
		{ID: "2", Name: "Tireur d'élite", Description: "Tuez 10 ennemis avec des armes de précision"},
		{ID: "3", Name: "Légendaire", Description: "Terminez la campagne en Légendaire"},
	}
	run()
	if n := countAssetRows(t, metadataDB, ddb.AssetTypeAchievement); n != 6 {
		t.Errorf("après la 2e synchro : %d lignes, attendu 6 (3 en + 3 fr, sans doublon)", n)
	}
	if name, _ := assetText(t, metadataDB, ddb.AssetTypeAchievement, "1", i18n.LocaleFR); name != "Premiers pas (révisé)" {
		t.Errorf("nom fr de l'id 1 non mis à jour : %q", name)
	}
	if name, desc := assetText(t, metadataDB, ddb.AssetTypeAchievement, "3", i18n.LocaleFR); name != "Légendaire" || desc != "Terminez la campagne en Légendaire" {
		t.Errorf("texte fr de l'id 3 = %q / %q", name, desc)
	}

}
