// Package sync — achievements_test.go : tests unitaires pour la sync des achievements.
//
// Tests purs (pas de DuckDB) : merge bilingue, parseAchievementItem, warmAchievementImages.
// Le mock XboxAchievementsClient vit dans achievements_mocks_integration_test.go (tag integration).
// Pour les tests d'intégration DuckDB, voir achievements_integration_test.go.
package sync

import (
	"context"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Fixtures partagées (utilisées aussi par achievements_integration_test.go)
// ---------------------------------------------------------------------------

var fixtureAchievementsEN = []PlayerAchievementRaw{
	{ID: "1", Name: "First Steps", Description: "Win your first match", Gamerscore: 5, ImageURL: "https://img/1.png", Unlocked: true, UnlockedAt: time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)},
	{ID: "2", Name: "Sharpshooter", Description: "Kill 10 enemies with precision weapons", Gamerscore: 10, IsSecret: true},
	{ID: "3", Name: "Legendary", Description: "Complete the campaign on Legendary", Gamerscore: 50, CurrentProgress: 2, TargetProgress: 4},
}

var fixtureAchievementsFR = []PlayerAchievementRaw{
	{ID: "1", Name: "Premiers pas", Description: "Gagnez votre premier match"},
	{ID: "2", Name: "Tireur d'élite", Description: "Tuez 10 ennemis avec des armes de précision"},
	// ID "3" absent en FR → name_fr sera vide
}

// ---------------------------------------------------------------------------
// Tests unitaires (merge bilingue)
// ---------------------------------------------------------------------------

func TestMergeAchievements_BilingualMerge(t *testing.T) {
	merged := mergeAchievements(fixtureAchievementsEN, fixtureAchievementsFR)

	if len(merged) != 3 {
		t.Fatalf("attendu 3 achievements, obtenu %d", len(merged))
	}

	// ID "1" : les deux langues présentes
	a1 := findAchievementByID(merged, "1")
	if a1 == nil {
		t.Fatal("achievement ID 1 manquant")
	}
	if a1.NameEN != "First Steps" {
		t.Errorf("NameEN attendu 'First Steps', obtenu %q", a1.NameEN)
	}
	if a1.NameFR != "Premiers pas" {
		t.Errorf("NameFR attendu 'Premiers pas', obtenu %q", a1.NameFR)
	}
	if !a1.Unlocked {
		t.Error("Unlocked attendu true pour ID 1")
	}

	// ID "3" : absent en FR → name_fr vide
	a3 := findAchievementByID(merged, "3")
	if a3 == nil {
		t.Fatal("achievement ID 3 manquant")
	}
	if a3.NameFR != "" {
		t.Errorf("NameFR attendu '' pour ID 3, obtenu %q", a3.NameFR)
	}
	if a3.CurrentProgress != 2 || a3.TargetProgress != 4 {
		t.Errorf("progression ID 3 attendue (2/4), obtenu (%d/%d)", a3.CurrentProgress, a3.TargetProgress)
	}
}

func TestMergeAchievements_EmptyEN_ReturnsEmpty(t *testing.T) {
	result := mergeAchievements(nil, fixtureAchievementsFR)
	if len(result) != 0 {
		t.Errorf("attendu 0 résultats quand EN est vide, obtenu %d", len(result))
	}
}

func TestMergeAchievements_EmptyFR_KeepsEN(t *testing.T) {
	result := mergeAchievements(fixtureAchievementsEN, nil)
	if len(result) != len(fixtureAchievementsEN) {
		t.Errorf("attendu %d résultats, obtenu %d", len(fixtureAchievementsEN), len(result))
	}
	for _, a := range result {
		if a.NameFR != "" {
			t.Errorf("NameFR attendu vide, obtenu %q pour ID %s", a.NameFR, a.AchievementID)
		}
	}
}

func TestMergeAchievements_IsSecretPreserved(t *testing.T) {
	merged := mergeAchievements(fixtureAchievementsEN, fixtureAchievementsFR)
	a2 := findAchievementByID(merged, "2")
	if a2 == nil {
		t.Fatal("achievement ID 2 manquant")
	}
	if !a2.IsSecret {
		t.Error("IsSecret attendu true pour ID 2")
	}
}

func TestMergeAchievements_UnlockedAtPreserved(t *testing.T) {
	merged := mergeAchievements(fixtureAchievementsEN, fixtureAchievementsFR)
	a1 := findAchievementByID(merged, "1")
	if a1 == nil {
		t.Fatal("achievement ID 1 manquant")
	}
	expected := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	if !a1.UnlockedAt.Equal(expected) {
		t.Errorf("UnlockedAt attendu %v, obtenu %v", expected, a1.UnlockedAt)
	}
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func findAchievementByID(achievements []PlayerAchievement, id string) *PlayerAchievement {
	for i := range achievements {
		if achievements[i].AchievementID == id {
			return &achievements[i]
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Tests parseAchievementItem
// ---------------------------------------------------------------------------

func TestParseAchievementItem_GamerscoreFromRewards(t *testing.T) {
	item := xboxAchievementItem{
		ID:            "a1",
		Name:          "Test",
		ProgressState: "NotStarted",
		Rewards: []struct {
			Name        interface{} `json:"name"`
			Description interface{} `json:"description"`
			Value       string      `json:"value"`
			Type        string      `json:"type"`
			MediaAsset  interface{} `json:"mediaAsset"`
		}{
			{Type: "Gamerscore", Value: "25"},
		},
	}
	raw := parseAchievementItem(item)
	if raw.Gamerscore != 25 {
		t.Errorf("Gamerscore attendu 25, obtenu %d", raw.Gamerscore)
	}
	if raw.Unlocked {
		t.Error("Unlocked attendu false pour ProgressState NotStarted")
	}
}

func TestParseAchievementItem_UnlockedFromProgressState(t *testing.T) {
	item := xboxAchievementItem{
		ID:            "a2",
		ProgressState: "Achieved",
		Progression: struct {
			Requirements []struct {
				ID                    string `json:"id"`
				Current               string `json:"current"`
				Target                string `json:"target"`
				OperationType         string `json:"operationType"`
				ValueType             string `json:"valueType"`
				RuleParticipationType string `json:"ruleParticipationType"`
			} `json:"requirements"`
			TimeUnlocked string `json:"timeUnlocked"`
		}{
			TimeUnlocked: "2024-03-15T12:00:00Z",
		},
	}
	raw := parseAchievementItem(item)
	if !raw.Unlocked {
		t.Error("Unlocked attendu true pour ProgressState Achieved")
	}
	expectedUnlock := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)
	if !raw.UnlockedAt.Equal(expectedUnlock) {
		t.Errorf("UnlockedAt attendu %v, obtenu %v", expectedUnlock, raw.UnlockedAt)
	}
}

func TestParseAchievementItem_ImageFromMediaAssets(t *testing.T) {
	item := xboxAchievementItem{
		ID:            "a3",
		ProgressState: "NotStarted",
		MediaAssets: []struct {
			Name string `json:"name"`
			Type string `json:"type"`
			URL  string `json:"url"`
		}{
			{Name: "badge", Type: "Icon", URL: "https://img.xbox.com/a3.png"},
		},
	}
	raw := parseAchievementItem(item)
	if raw.ImageURL != "https://img.xbox.com/a3.png" {
		t.Errorf("ImageURL attendu 'https://img.xbox.com/a3.png', obtenu %q", raw.ImageURL)
	}
}

func TestParseAchievementItem_ProgressionRequirement(t *testing.T) {
	item := xboxAchievementItem{
		ID:            "a4",
		ProgressState: "InProgress",
		Progression: struct {
			Requirements []struct {
				ID                    string `json:"id"`
				Current               string `json:"current"`
				Target                string `json:"target"`
				OperationType         string `json:"operationType"`
				ValueType             string `json:"valueType"`
				RuleParticipationType string `json:"ruleParticipationType"`
			} `json:"requirements"`
			TimeUnlocked string `json:"timeUnlocked"`
		}{
			Requirements: []struct {
				ID                    string `json:"id"`
				Current               string `json:"current"`
				Target                string `json:"target"`
				OperationType         string `json:"operationType"`
				ValueType             string `json:"valueType"`
				RuleParticipationType string `json:"ruleParticipationType"`
			}{
				{Current: "7", Target: "10"},
			},
		},
	}
	raw := parseAchievementItem(item)
	if raw.CurrentProgress != 7 {
		t.Errorf("CurrentProgress attendu 7, obtenu %d", raw.CurrentProgress)
	}
	if raw.TargetProgress != 10 {
		t.Errorf("TargetProgress attendu 10, obtenu %d", raw.TargetProgress)
	}
}

func TestParseAchievementItem_RarityPreserved(t *testing.T) {
	item := xboxAchievementItem{
		ID:            "a5",
		ProgressState: "NotStarted",
		IsSecret:      true,
		Rarity: struct {
			CurrentCategory   string  `json:"currentCategory"`
			CurrentPercentage float64 `json:"currentPercentage"`
		}{
			CurrentCategory:   "Rare",
			CurrentPercentage: 4.5,
		},
	}
	raw := parseAchievementItem(item)
	if raw.RarityCategory != "Rare" {
		t.Errorf("RarityCategory attendu 'Rare', obtenu %q", raw.RarityCategory)
	}
	if raw.RarityPercent != 4.5 {
		t.Errorf("RarityPercent attendu 4.5, obtenu %f", raw.RarityPercent)
	}
	if !raw.IsSecret {
		t.Error("IsSecret attendu true")
	}
}

func TestParseAchievementItem_TimeUnlocked_ZeroValue_Ignored(t *testing.T) {
	item := xboxAchievementItem{
		ID:            "a6",
		ProgressState: "Achieved",
		Progression: struct {
			Requirements []struct {
				ID                    string `json:"id"`
				Current               string `json:"current"`
				Target                string `json:"target"`
				OperationType         string `json:"operationType"`
				ValueType             string `json:"valueType"`
				RuleParticipationType string `json:"ruleParticipationType"`
			} `json:"requirements"`
			TimeUnlocked string `json:"timeUnlocked"`
		}{
			TimeUnlocked: "0001-01-01T00:00:00Z", // valeur sentinelle Xbox
		},
	}
	raw := parseAchievementItem(item)
	if !raw.UnlockedAt.IsZero() {
		t.Errorf("UnlockedAt devrait être zéro pour la valeur sentinelle Xbox, obtenu %v", raw.UnlockedAt)
	}
}

// ---------------------------------------------------------------------------
// Tests warmAchievementImages
// ---------------------------------------------------------------------------

func TestWarmAchievementImages_EmptyImageURL_Skipped(t *testing.T) {
	// Aucun achievement n'a d'image → refs vide → goroutine non lancée.
	// On passe nil resolver : pas de panic car len(refs)==0 fait un early return.
	achievements := []PlayerAchievement{
		{AchievementID: "1", ImageURL: ""},
		{AchievementID: "2", ImageURL: ""},
	}
	warmAchievementImages(context.Background(), nil, achievements)
}

func TestWarmAchievementImages_NilAchievements_NoPanic(t *testing.T) {
	warmAchievementImages(context.Background(), nil, nil)
}
