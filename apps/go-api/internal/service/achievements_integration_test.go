// Package service — achievements_integration_test.go : test bout-en-bout
// du service achievements avec deux DuckDB :memory: (player + metadata).
//
// Build tag `integration` — exclu du go test ./... par défaut. Lancer avec :
//   go test -tags=integration ./internal/service/ -run TestAchievementsIntegration
//
//go:build integration

package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
)

func openIntegrationPlayerDB(t *testing.T) *duckdb.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stats.duckdb")
	db, err := duckdb.OpenReadWrite(path)
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

func openIntegrationMetadataDB(t *testing.T) *duckdb.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "metadata.duckdb")
	db, err := duckdb.OpenReadWrite(path)
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

// TestAchievementsIntegration_FullFlow : peuple les deux DBs avec un dataset
// hétérogène (10 défs + 4 unlocked) et vérifie le merge cross-DB de bout en bout.
func TestAchievementsIntegration_FullFlow(t *testing.T) {
	playerDB := openIntegrationPlayerDB(t)
	metaDB := openIntegrationMetadataDB(t)
	ctx := context.Background()

	// 10 définitions avec gamerscores variés.
	defs := []struct {
		id     string
		nameEN string
		nameFR string
		score  int
	}{
		{"def01", "Boot Camp", "Camp d'entraînement", 5},
		{"def02", "Squad Up", "En escouade", 10},
		{"def03", "First Blood", "Premier sang", 15},
		{"def04", "Untouchable", "Intouchable", 20},
		{"def05", "Sharpshooter", "Tireur d'élite", 25},
		{"def06", "Killing Spree", "Tueur en série", 30},
		{"def07", "Demon", "Démon", 50},
		{"def08", "Legend", "Légende", 75},
		{"def09", "Onyx", "Onyx", 100},
		{"def10", "Champion", "Champion", 200},
	}
	// title_id 'halo_infinite' obligatoire — la query GetAchievementDefinitions
	// filtre dessus depuis la migration `add_title_id_to_xbox_achievement_definitions`.
	insertDef := `INSERT INTO xbox_achievement_definitions
		(achievement_id, name_en, name_fr, description_en, description_fr,
		 locked_desc_en, locked_desc_fr, gamerscore, image_url, is_secret,
		 rarity_category, rarity_percent, title_id, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`
	for _, d := range defs {
		if _, err := metaDB.Exec(ctx, insertDef,
			d.id, d.nameEN, d.nameFR,
			"Desc EN", "Desc FR",
			"Locked EN", "Verrouillé FR",
			d.score, "https://example.com/img.png", false,
			"Common", 25.0, "halo_infinite",
		); err != nil {
			t.Fatalf("insert def %s: %v", d.id, err)
		}
	}

	// 4 unlocked (def01, def03, def07, def10) + 1 in-progress (def05) + 5 locked vierges.
	t1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 2, 15, 14, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 3, 20, 9, 0, 0, 0, time.UTC)
	t4 := time.Date(2026, 4, 10, 18, 0, 0, 0, time.UTC)
	insertPlayer := `INSERT INTO player_achievements
		(achievement_id, unlocked, unlocked_at, current_progress, target_progress, fetched_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`
	if _, err := playerDB.Exec(ctx, insertPlayer, "def01", true, t1, nil, nil); err != nil {
		t.Fatalf("insert def01: %v", err)
	}
	if _, err := playerDB.Exec(ctx, insertPlayer, "def03", true, t2, nil, nil); err != nil {
		t.Fatalf("insert def03: %v", err)
	}
	if _, err := playerDB.Exec(ctx, insertPlayer, "def07", true, t3, nil, nil); err != nil {
		t.Fatalf("insert def07: %v", err)
	}
	if _, err := playerDB.Exec(ctx, insertPlayer, "def10", true, t4, nil, nil); err != nil {
		t.Fatalf("insert def10: %v", err)
	}
	current, target := 60, 100
	if _, err := playerDB.Exec(ctx, insertPlayer, "def05", false, nil, current, target); err != nil {
		t.Fatalf("insert def05: %v", err)
	}

	// Wirer le service comme en prod.
	pdb := &duckdb.PlayerDB{Player: playerDB, Metadata: metaDB}
	repo := duckdb.NewAchievementsRepo(pdb)
	metaRepo := duckdb.NewMetadataRepo(pdb)
	svc := NewAchievementsService(repo, metaRepo).WithTitleSlug("halo_infinite")

	resp, err := svc.GetAchievementsPage(ctx)
	if err != nil {
		t.Fatalf("GetAchievementsPage: %v", err)
	}

	// Summary : 10 total, 4 unlocked, gamerscore total = 530 (5+10+15+20+25+30+50+75+100+200)
	// Earned : 5 (def01) + 15 (def03) + 50 (def07) + 200 (def10) = 270
	// Pct : 4/10 = 40.0
	if resp.Summary.TotalCount != 10 {
		t.Errorf("TotalCount: attendu 10, obtenu %d", resp.Summary.TotalCount)
	}
	if resp.Summary.UnlockedCount != 4 {
		t.Errorf("UnlockedCount: attendu 4, obtenu %d", resp.Summary.UnlockedCount)
	}
	if resp.Summary.TotalGamerscore != 530 {
		t.Errorf("TotalGamerscore: attendu 530, obtenu %d", resp.Summary.TotalGamerscore)
	}
	if resp.Summary.EarnedGamerscore != 270 {
		t.Errorf("EarnedGamerscore: attendu 270, obtenu %d", resp.Summary.EarnedGamerscore)
	}
	if resp.Summary.CompletionPct != 40.0 {
		t.Errorf("CompletionPct: attendu 40.0, obtenu %v", resp.Summary.CompletionPct)
	}

	// Tri (cf. sortAchievementEntries dans achievements_service.go) : locked
	// en premier (gamerscore DESC, tie-break id ASC) — def09(100), def08(75),
	// def06(30), def05(25), def04(20), def02(10) — puis unlocked (UnlockedAt
	// ASC, plus récent en bas) — def01(janv), def03(févr), def07(mars), def10(avril).
	if len(resp.Achievements) != 10 {
		t.Fatalf("attendu 10 entrées, obtenu %d", len(resp.Achievements))
	}
	expected := []string{"def09", "def08", "def06", "def05", "def04", "def02", "def01", "def03", "def07", "def10"}
	for i, want := range expected {
		if resp.Achievements[i].AchievementID != want {
			t.Errorf("position %d: attendu %s, obtenu %s",
				i, want, resp.Achievements[i].AchievementID)
		}
	}

	// Vérifier que la progression de def05 est bien attachée
	for _, a := range resp.Achievements {
		if a.AchievementID == "def05" {
			if a.CurrentProgress == nil || *a.CurrentProgress != current {
				t.Errorf("def05 CurrentProgress: attendu %d", current)
			}
			if a.TargetProgress == nil || *a.TargetProgress != target {
				t.Errorf("def05 TargetProgress: attendu %d", target)
			}
		}
		// Vérifier que les noms bilingues sont préservés
		if a.AchievementID == "def01" && (a.NameEN != "Boot Camp" || a.NameFR != "Camp d'entraînement") {
			t.Errorf("def01 noms bilingues incorrects: EN=%q FR=%q", a.NameEN, a.NameFR)
		}
	}
}

// TestAchievementsIntegration_OrphanPlayerRow : ligne player sans définition
// correspondante (cas d'achievement supprimé côté Xbox) → ignorée silencieusement.
func TestAchievementsIntegration_OrphanPlayerRow(t *testing.T) {
	playerDB := openIntegrationPlayerDB(t)
	metaDB := openIntegrationMetadataDB(t)
	ctx := context.Background()

	// 1 définition seulement.
	if _, err := metaDB.Exec(ctx,
		`INSERT INTO xbox_achievement_definitions
		 (achievement_id, name_en, name_fr, description_en, description_fr,
		  locked_desc_en, locked_desc_fr, gamerscore, image_url, is_secret,
		  rarity_category, rarity_percent, title_id, fetched_at)
		 VALUES ('alive', 'Alive', 'Vivant', '', '', '', '', 10, '', false, '', 0, 'halo_infinite', CURRENT_TIMESTAMP)`,
	); err != nil {
		t.Fatalf("insert def alive: %v", err)
	}

	// 2 player rows : 1 valide (alive) + 1 orphelin (ghost).
	t1 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	for _, p := range []struct {
		id       string
		unlocked bool
		at       *time.Time
	}{
		{"alive", true, &t1},
		{"ghost", true, &t1},
	} {
		if _, err := playerDB.Exec(ctx,
			`INSERT INTO player_achievements
			 (achievement_id, unlocked, unlocked_at, current_progress, target_progress, fetched_at)
			 VALUES (?, ?, ?, NULL, NULL, CURRENT_TIMESTAMP)`,
			p.id, p.unlocked, p.at,
		); err != nil {
			t.Fatalf("insert player %s: %v", p.id, err)
		}
	}

	pdb := &duckdb.PlayerDB{Player: playerDB, Metadata: metaDB}
	svc := NewAchievementsService(
		duckdb.NewAchievementsRepo(pdb),
		duckdb.NewMetadataRepo(pdb),
	)

	resp, err := svc.GetAchievementsPage(ctx)
	if err != nil {
		t.Fatalf("GetAchievementsPage: %v", err)
	}
	if len(resp.Achievements) != 1 {
		t.Fatalf("orphelin doit être ignoré : attendu 1 entrée, obtenu %d", len(resp.Achievements))
	}
	if resp.Achievements[0].AchievementID != "alive" {
		t.Errorf("seul 'alive' doit être présent")
	}
	if resp.Summary.UnlockedCount != 1 {
		t.Errorf("UnlockedCount: attendu 1 (orphelin ignoré), obtenu %d", resp.Summary.UnlockedCount)
	}
}
