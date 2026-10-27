package duckdb

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/i18n"
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

	rows, err := repo.GetAchievementDefinitions(context.Background(), "halo_infinite")
	if err != nil {
		t.Fatalf("GetAchievementDefinitions: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("attendu 0 définitions, obtenu %d", len(rows))
	}
}

// insertAchievementText range un texte de succès dans asset_translations sous l'étiquette Halo
// principale de la langue (type `achievement` : nom + description ; `achievement_locked` : texte
// de succès verrouillé dans `name`).
func insertAchievementText(t *testing.T, db *DB, assetType, id string, l i18n.Locale, name string, desc any) {
	t.Helper()
	_, err := db.Exec(context.Background(),
		`INSERT INTO asset_translations (asset_id, asset_type, lang, name, description) VALUES (?, ?, ?, ?, ?)`,
		id, assetType, i18n.HaloTags(l)[0], name, desc)
	if err != nil {
		t.Fatalf("insert asset_translations %s/%s (%s): %v", assetType, id, l, err)
	}
}

// TestMetadataRepo_GetAchievementDefinitions_Populated : insère 2 définitions
// (une complète, une avec champs nullables vides) et vérifie les champs non localisés
// et les textes lus dans asset_translations (une ligne par langue).
func TestMetadataRepo_GetAchievementDefinitions_Populated(t *testing.T) {
	db := openAchievementsMetadataDB(t)
	ctx := context.Background()

	insert := `INSERT INTO xbox_achievement_definitions
		(achievement_id, gamerscore, image_url, is_secret,
		 rarity_category, rarity_percent, title_id, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

	if _, err := db.Exec(ctx, insert,
		"ach_full", 50, "https://example.com/full.png", false,
		"Rare", 12.5, "halo_infinite",
	); err != nil {
		t.Fatalf("insert ach_full: %v", err)
	}
	if _, err := db.Exec(ctx, insert,
		"ach_min", 10, nil, true,
		nil, nil, "halo_infinite",
	); err != nil {
		t.Fatalf("insert ach_min: %v", err)
	}
	insertAchievementText(t, db, AssetTypeAchievement, "ach_full", i18n.LocaleEN, "Full Name EN", "Description EN")
	insertAchievementText(t, db, AssetTypeAchievement, "ach_full", i18n.LocaleFR, "Nom Complet FR", "Description FR")
	insertAchievementText(t, db, AssetTypeAchievementLocked, "ach_full", i18n.LocaleEN, "Locked EN", nil)
	insertAchievementText(t, db, AssetTypeAchievementLocked, "ach_full", i18n.LocaleFR, "Verrouillé FR", nil)
	insertAchievementText(t, db, AssetTypeAchievement, "ach_min", i18n.LocaleEN, "Min EN", nil)

	repo := NewMetadataRepoFromDB(db)
	rows, err := repo.GetAchievementDefinitions(ctx, "halo_infinite")
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
	if full.Name[i18n.LocaleEN] != "Full Name EN" || full.Name[i18n.LocaleFR] != "Nom Complet FR" {
		t.Errorf("ach_full noms par langue incorrects")
	}
	if full.Description[i18n.LocaleEN] != "Description EN" || full.Description[i18n.LocaleFR] != "Description FR" {
		t.Errorf("ach_full descriptions incorrectes")
	}
	if full.LockedDesc[i18n.LocaleEN] != "Locked EN" || full.LockedDesc[i18n.LocaleFR] != "Verrouillé FR" {
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
	if min.Name[i18n.LocaleEN] != "Min EN" || min.Name[i18n.LocaleFR] != "" {
		t.Errorf("ach_min name incorrect: %q, %q", min.Name[i18n.LocaleEN], min.Name[i18n.LocaleFR])
	}
	if min.Description[i18n.LocaleEN] != "" || min.Description[i18n.LocaleFR] != "" {
		t.Errorf("ach_min description doit être vide")
	}
	if min.LockedDesc[i18n.LocaleEN] != "" || min.LockedDesc[i18n.LocaleFR] != "" {
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

// TestMetadataRepo_GetAchievementDefinitions_LangueDemandee : le texte choisi suit la langue de la
// requête (`Text.Pick`) : fr et en lisent leur ligne, une langue sans traduction (fr absent, ou une
// langue hors liste) se replie sur l'anglais, jamais sur un texte inventé.
func TestMetadataRepo_GetAchievementDefinitions_LangueDemandee(t *testing.T) {
	db := openAchievementsMetadataDB(t)
	ctx := context.Background()
	for _, id := range []string{"both", "en_only"} {
		if _, err := db.Exec(ctx, `INSERT INTO xbox_achievement_definitions (achievement_id, gamerscore, is_secret, title_id)
			VALUES (?, 5, FALSE, 'halo_infinite')`, id); err != nil {
			t.Fatalf("insert %s: %v", id, err)
		}
	}
	insertAchievementText(t, db, AssetTypeAchievement, "both", i18n.LocaleEN, "First Blood", "Get a kill")
	insertAchievementText(t, db, AssetTypeAchievement, "both", i18n.LocaleFR, "Premier sang", "Faire un frag")
	insertAchievementText(t, db, AssetTypeAchievementLocked, "both", i18n.LocaleEN, "Keep going", nil)
	insertAchievementText(t, db, AssetTypeAchievementLocked, "both", i18n.LocaleFR, "Continue", nil)
	insertAchievementText(t, db, AssetTypeAchievement, "en_only", i18n.LocaleEN, "Only English", "Only EN text")

	rows, err := NewMetadataRepoFromDB(db).GetAchievementDefinitions(ctx, "halo_infinite")
	if err != nil {
		t.Fatalf("GetAchievementDefinitions: %v", err)
	}
	byID := map[string]int{rows[0].AchievementID: 0, rows[1].AchievementID: 1}

	cases := []struct {
		name       string
		loc        i18n.Locale
		id         string
		wantName   string
		wantDesc   string
		wantLocked string
	}{
		{"fr lit sa ligne", i18n.LocaleFR, "both", "Premier sang", "Faire un frag", "Continue"},
		{"en lit sa ligne", i18n.LocaleEN, "both", "First Blood", "Get a kill", "Keep going"},
		{"langue hors liste : repli en", i18n.Locale("es"), "both", "First Blood", "Get a kill", "Keep going"},
		{"fr sans traduction : repli en", i18n.LocaleFR, "en_only", "Only English", "Only EN text", ""},
	}
	for _, c := range cases {
		r := rows[byID[c.id]]
		if got := r.Name.Pick(c.loc); got != c.wantName {
			t.Errorf("%s : nom = %q, attendu %q", c.name, got, c.wantName)
		}
		if got := r.Description.Pick(c.loc); got != c.wantDesc {
			t.Errorf("%s : description = %q, attendu %q", c.name, got, c.wantDesc)
		}
		if got := r.LockedDesc.PickOptional(c.loc); got != c.wantLocked {
			t.Errorf("%s : texte verrouillé = %q, attendu %q", c.name, got, c.wantLocked)
		}
	}
	if rows[byID["en_only"]].Name.Has(i18n.LocaleFR) || rows[byID["both"]].Name.Has(i18n.Locale("es")) {
		t.Errorf("aucune traduction ne doit être inventée pour une langue sans ligne")
	}
}

// TestMetadataRepo_GetAchievementDefinitions_TitleRouting (PMT-6 oracle b) : le
// filtre title_id route réellement par titre — seules les défs du slug demandé
// reviennent ; "" → défaut halo_infinite ; slug inconnu → slice vide.
func TestMetadataRepo_GetAchievementDefinitions_TitleRouting(t *testing.T) {
	db := openAchievementsMetadataDB(t)
	ctx := context.Background()
	insert := `INSERT INTO xbox_achievement_definitions
		(achievement_id, gamerscore, is_secret, title_id)
		VALUES (?, ?, ?, ?)`
	if _, err := db.Exec(ctx, insert, "hi_1", 10, false, "halo_infinite"); err != nil {
		t.Fatalf("insert hi: %v", err)
	}
	if _, err := db.Exec(ctx, insert, "synth_1", 20, false, "synthetic_test_title"); err != nil {
		t.Fatalf("insert synth: %v", err)
	}
	insertAchievementText(t, db, AssetTypeAchievement, "hi_1", i18n.LocaleEN, "HI One", nil)
	insertAchievementText(t, db, AssetTypeAchievement, "hi_1", i18n.LocaleFR, "HI Un", nil)
	insertAchievementText(t, db, AssetTypeAchievement, "synth_1", i18n.LocaleEN, "Synth One", nil)
	insertAchievementText(t, db, AssetTypeAchievement, "synth_1", i18n.LocaleFR, "Synth Un", nil)
	repo := NewMetadataRepoFromDB(db)

	onlyOne := func(slug, wantID, wantNameEN string) {
		t.Helper()
		got, err := repo.GetAchievementDefinitions(ctx, slug)
		if err != nil {
			t.Fatalf("GetAchievementDefinitions(%q): %v", slug, err)
		}
		if len(got) != 1 || got[0].AchievementID != wantID {
			t.Fatalf("slug %q → %d défs, attendu 1 (%s)", slug, len(got), wantID)
		}
		if got[0].Name[i18n.LocaleEN] != wantNameEN {
			t.Errorf("slug %q : nom en = %q, attendu %q", slug, got[0].Name[i18n.LocaleEN], wantNameEN)
		}
	}
	onlyOne("halo_infinite", "hi_1", "HI One")
	onlyOne("synthetic_test_title", "synth_1", "Synth One")
	onlyOne("", "hi_1", "HI One") // "" → défaut halo_infinite (parité historique)

	unknown, err := repo.GetAchievementDefinitions(ctx, "no_such_title")
	if err != nil {
		t.Fatalf("slug inconnu: %v", err)
	}
	if len(unknown) != 0 {
		t.Errorf("slug inconnu → %d défs, attendu 0", len(unknown))
	}
}
