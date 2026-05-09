package duckdb

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/migration"
)

func openAchievementsPlayerDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stats.duckdb")
	db, err := OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite player: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_ = migration.All()
	if err := migration.RunForDB(db.SQLDb(), migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB(TargetPlayer): %v", err)
	}
	return db
}

func openAchievementsMetadataDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "metadata.duckdb")
	db, err := OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite metadata: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_ = migration.All()
	if err := migration.RunForDB(db.SQLDb(), migration.TargetMetadata); err != nil {
		t.Fatalf("RunForDB(TargetMetadata): %v", err)
	}
	return db
}

// TestAchievementsRepo_GetPlayerAchievements_Empty : un joueur sans données
// player_achievements (jamais syncé) reçoit un slice vide, pas une erreur.
func TestAchievementsRepo_GetPlayerAchievements_Empty(t *testing.T) {
	db := openAchievementsPlayerDB(t)
	pdb := &PlayerDB{Player: db}
	repo := NewAchievementsRepo(pdb)

	rows, err := repo.GetPlayerAchievements(context.Background())
	if err != nil {
		t.Fatalf("GetPlayerAchievements: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("attendu 0 rows, obtenu %d", len(rows))
	}
}

// TestAchievementsRepo_GetPlayerAchievements_Populated : insère 3 lignes (1 unlocked
// avec date, 1 locked avec progression, 1 locked sans progression) et vérifie
// scan complet + tri par achievement_id.
func TestAchievementsRepo_GetPlayerAchievements_Populated(t *testing.T) {
	db := openAchievementsPlayerDB(t)
	ctx := context.Background()

	unlockedAt := time.Date(2026, 4, 1, 12, 30, 0, 0, time.UTC)
	current, target := 5, 10

	insert := `INSERT INTO player_achievements
		(achievement_id, unlocked, unlocked_at, current_progress, target_progress, fetched_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

	if _, err := db.Exec(ctx, insert, "ach_z", true, unlockedAt, nil, nil); err != nil {
		t.Fatalf("insert ach_z: %v", err)
	}
	if _, err := db.Exec(ctx, insert, "ach_a", false, nil, current, target); err != nil {
		t.Fatalf("insert ach_a: %v", err)
	}
	if _, err := db.Exec(ctx, insert, "ach_m", false, nil, nil, nil); err != nil {
		t.Fatalf("insert ach_m: %v", err)
	}

	pdb := &PlayerDB{Player: db}
	rows, err := NewAchievementsRepo(pdb).GetPlayerAchievements(ctx)
	if err != nil {
		t.Fatalf("GetPlayerAchievements: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("attendu 3 rows, obtenu %d", len(rows))
	}
	// Tri attendu : ach_a, ach_m, ach_z
	if rows[0].AchievementID != "ach_a" || rows[1].AchievementID != "ach_m" || rows[2].AchievementID != "ach_z" {
		t.Errorf("tri incorrect: %s, %s, %s", rows[0].AchievementID, rows[1].AchievementID, rows[2].AchievementID)
	}
	// ach_a : locked + progression
	if rows[0].Unlocked || rows[0].UnlockedAt != nil {
		t.Errorf("ach_a doit être locked sans unlocked_at")
	}
	if rows[0].CurrentProgress == nil || *rows[0].CurrentProgress != current {
		t.Errorf("ach_a current_progress: attendu %d", current)
	}
	if rows[0].TargetProgress == nil || *rows[0].TargetProgress != target {
		t.Errorf("ach_a target_progress: attendu %d", target)
	}
	// ach_m : locked sans progression
	if rows[1].CurrentProgress != nil || rows[1].TargetProgress != nil {
		t.Errorf("ach_m doit avoir current/target=nil")
	}
	// ach_z : unlocked avec date
	if !rows[2].Unlocked {
		t.Errorf("ach_z doit être unlocked")
	}
	if rows[2].UnlockedAt == nil || !rows[2].UnlockedAt.Equal(unlockedAt) {
		t.Errorf("ach_z unlocked_at: attendu %v, obtenu %v", unlockedAt, rows[2].UnlockedAt)
	}
}

// TestMetadataRepo_GetAchievementDefinitions_Empty : table vide → slice vide.
func TestMetadataRepo_GetAchievementDefinitions_Empty(t *testing.T) {
	db := openAchievementsMetadataDB(t)
	repo := NewMetadataRepoFromDB(db)

	rows, err := repo.GetAchievementDefinitions(context.Background())
	if err != nil {
		t.Fatalf("GetAchievementDefinitions: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("attendu 0 définitions, obtenu %d", len(rows))
	}
}

// TestMetadataRepo_GetAchievementDefinitions_Populated : insère 2 définitions
// (une complète, une avec champs nullables vides) et vérifie le scan bilingue.
func TestMetadataRepo_GetAchievementDefinitions_Populated(t *testing.T) {
	db := openAchievementsMetadataDB(t)
	ctx := context.Background()

	insert := `INSERT INTO xbox_achievement_definitions
		(achievement_id, name_en, name_fr, description_en, description_fr,
		 locked_desc_en, locked_desc_fr, gamerscore, image_url, is_secret,
		 rarity_category, rarity_percent, title_id, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

	if _, err := db.Exec(ctx, insert,
		"ach_full", "Full Name EN", "Nom Complet FR",
		"Description EN", "Description FR",
		"Locked EN", "Verrouillé FR",
		50, "https://example.com/full.png", false,
		"Rare", 12.5, "halo_infinite",
	); err != nil {
		t.Fatalf("insert ach_full: %v", err)
	}
	if _, err := db.Exec(ctx, insert,
		"ach_min", "Min EN", "",
		nil, nil, nil, nil,
		10, nil, true,
		nil, nil, "halo_infinite",
	); err != nil {
		t.Fatalf("insert ach_min: %v", err)
	}

	repo := NewMetadataRepoFromDB(db)
	rows, err := repo.GetAchievementDefinitions(ctx)
	if err != nil {
		t.Fatalf("GetAchievementDefinitions: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("attendu 2 définitions, obtenu %d", len(rows))
	}
	// Tri attendu : ach_full, ach_min
	if rows[0].AchievementID != "ach_full" || rows[1].AchievementID != "ach_min" {
		t.Errorf("tri incorrect")
	}
	// ach_full : tous les champs peuplés
	full := rows[0]
	if full.NameEN != "Full Name EN" || full.NameFR != "Nom Complet FR" {
		t.Errorf("ach_full noms bilingues incorrects")
	}
	if full.DescriptionEN != "Description EN" || full.DescriptionFR != "Description FR" {
		t.Errorf("ach_full descriptions incorrectes")
	}
	if full.LockedDescEN != "Locked EN" || full.LockedDescFR != "Verrouillé FR" {
		t.Errorf("ach_full locked desc incorrects")
	}
	if full.Gamerscore != 50 || full.ImageURL != "https://example.com/full.png" {
		t.Errorf("ach_full gamerscore/image incorrects")
	}
	if full.IsSecret {
		t.Errorf("ach_full ne doit pas être secret")
	}
	if full.RarityCategory != "Rare" || full.RarityPercent != 12.5 {
		t.Errorf("ach_full rarity incorrect: %s, %v", full.RarityCategory, full.RarityPercent)
	}
	// ach_min : nullables → strings vides, percent à 0
	min := rows[1]
	if min.NameEN != "Min EN" || min.NameFR != "" {
		t.Errorf("ach_min name incorrect: %q, %q", min.NameEN, min.NameFR)
	}
	if min.DescriptionEN != "" || min.DescriptionFR != "" {
		t.Errorf("ach_min description doit être vide")
	}
	if min.LockedDescEN != "" || min.LockedDescFR != "" {
		t.Errorf("ach_min locked desc doit être vide")
	}
	if min.ImageURL != "" {
		t.Errorf("ach_min image_url doit être vide")
	}
	if !min.IsSecret {
		t.Errorf("ach_min doit être secret")
	}
	if min.RarityCategory != "" || min.RarityPercent != 0 {
		t.Errorf("ach_min rarity doit être nulle")
	}
}
