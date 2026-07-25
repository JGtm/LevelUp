//go:build integration

// Tests d'intégration des repos Prestige (Phase 2).
// Vérifie le round-trip insert/get sur stats.duckdb, shared_social.duckdb, metadata.duckdb.

package prestige

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/prestige"
)

// setupPrestigeDB ouvre une DB temp et applique les migrations du target.
//
// Retourne un *duckdb.DB du package duckdb-root (avec pool) prêt pour les repos.
func setupPrestigeDB(t *testing.T, target migration.TargetDB) *duckdb.DB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, string(target)+".duckdb")

	// Créer la DB et appliquer migrations via sql.DB direct.
	raw, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if err := migration.RunForDB(raw, target); err != nil {
		raw.Close()
		t.Fatalf("RunForDB(%s): %v", target, err)
	}
	raw.Close()

	// Ouvrir via le wrapper duckdb.OpenReadWrite (cache + pool).
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		os.RemoveAll(dir)
	})
	return db
}

// ─────────── ChallengeRepo ───────────

func TestPrestigeChallengeRepo_CreateGet_Roundtrip(t *testing.T) {
	db := setupPrestigeDB(t, migration.TargetPlayer)
	repo := NewPrestigeChallengeRepo(db)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	committed := now.Add(time.Minute)
	c := prestige.Challenge{
		ID:          "ch_test_001",
		UserID:      "user_42",
		TitleSlug:   "halo_infinite",
		ArcID:       "arc_slayer",
		Position:    1,
		TemplateID:  "tpl_kda",
		Metric:      "FieldKDA",
		Target:      1.5,
		WindowType:  prestige.WindowSession,
		WindowValue: "3",
		Cadence:     prestige.CadenceWeekly,
		EvalType:    prestige.EvalThreshold,
		Mode:        prestige.ModeLibre,
		Tier:        prestige.TierHeroic,
		DataTier:    prestige.DataFull,
		Label:       "Test challenge",
		Status:      prestige.StatusActive,
		CreatedAt:   now,
		CommittedAt: &committed,
		Source:      prestige.ChallengeSourceCoach,
	}
	if err := repo.Create(ctx, c); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := repo.Get(ctx, c.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != c.ID || got.Target != c.Target || got.Tier != c.Tier {
		t.Errorf("roundtrip mismatch: got %+v want %+v", got, c)
	}
	if got.Source != prestige.ChallengeSourceCoach {
		t.Errorf("source roundtrip: got %q want %q", got.Source, prestige.ChallengeSourceCoach)
	}
}

func TestPrestigeChallengeRepo_UpdateStatus_SetsTimestamp(t *testing.T) {
	db := setupPrestigeDB(t, migration.TargetPlayer)
	repo := NewPrestigeChallengeRepo(db)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	c := prestige.Challenge{
		ID: "ch_status", UserID: "u1", TitleSlug: "halo_infinite",
		Metric: "FieldKDA", Target: 1.0, WindowType: prestige.WindowSession,
		Cadence: prestige.CadenceFree, EvalType: prestige.EvalThreshold,
		Mode: prestige.ModeLibre, DataTier: prestige.DataFull,
		Status: prestige.StatusActive, CreatedAt: now,
	}
	if err := repo.Create(ctx, c); err != nil {
		t.Fatalf("Create: %v", err)
	}
	completedAt := now.Add(time.Hour)
	if err := repo.UpdateStatus(ctx, c.ID, prestige.StatusCompleted, completedAt); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	got, _ := repo.Get(ctx, c.ID)
	if got.Status != prestige.StatusCompleted {
		t.Errorf("status got %s want completed", got.Status)
	}
	if got.CompletedAt == nil || !got.CompletedAt.Equal(completedAt) {
		t.Errorf("completed_at not set: %v", got.CompletedAt)
	}
}

func TestPrestigeChallengeRepo_List_FilterByStatus(t *testing.T) {
	db := setupPrestigeDB(t, migration.TargetPlayer)
	repo := NewPrestigeChallengeRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	for i, status := range []prestige.ChallengeStatus{
		prestige.StatusActive, prestige.StatusActive, prestige.StatusCompleted,
	} {
		c := prestige.Challenge{
			ID:     "ch_" + string(rune('a'+i)),
			UserID: "u1", TitleSlug: "halo_infinite",
			Metric: "FieldKDA", Target: 1.0,
			WindowType: prestige.WindowSession, Cadence: prestige.CadenceFree,
			EvalType: prestige.EvalThreshold, Mode: prestige.ModeLibre,
			DataTier: prestige.DataFull, Status: status, CreatedAt: now,
		}
		if err := repo.Create(ctx, c); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	active := prestige.StatusActive
	list, err := repo.List(ctx, prestige.ChallengeFilter{
		UserID: "u1", TitleSlug: "halo_infinite", Status: &active,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d active, want 2", len(list))
	}
}

func TestPrestigeChallengeRepo_List_FilterByMetric(t *testing.T) {
	db := setupPrestigeDB(t, migration.TargetPlayer)
	repo := NewPrestigeChallengeRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	for i, metric := range []string{"FieldKDA", "FieldKills", "FieldKDA"} {
		c := prestige.Challenge{
			ID:     "ch_" + string(rune('a'+i)),
			UserID: "u1", TitleSlug: "halo_infinite",
			Metric: metric, Target: 1.0,
			WindowType: prestige.WindowSession, Cadence: prestige.CadenceFree,
			EvalType: prestige.EvalThreshold, Mode: prestige.ModeLibre,
			DataTier: prestige.DataFull, Status: prestige.StatusActive, CreatedAt: now,
		}
		if err := repo.Create(ctx, c); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	kda := "FieldKDA"
	list, err := repo.List(ctx, prestige.ChallengeFilter{
		UserID: "u1", TitleSlug: "halo_infinite", Metric: &kda,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d FieldKDA, want 2", len(list))
	}
	for _, c := range list {
		if c.Metric != "FieldKDA" {
			t.Errorf("unexpected metric %q in filtered list", c.Metric)
		}
	}
}

func seedArcWithChallenges(t *testing.T, db *duckdb.DB, arcID string, challengeIDs ...string) (*PrestigeChallengeRepo, *PrestigeArcRepo) {
	t.Helper()
	chRepo := NewPrestigeChallengeRepo(db)
	arcRepo := NewPrestigeArcRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	if err := arcRepo.Create(ctx, prestige.Arc{
		ID: arcID, UserID: "u1", TitleSlug: "halo_infinite", Title: "T", CreatedAt: now,
	}); err != nil {
		t.Fatalf("arc Create: %v", err)
	}
	for _, id := range challengeIDs {
		if err := chRepo.Create(ctx, prestige.Challenge{
			ID: id, UserID: "u1", TitleSlug: "halo_infinite", ArcID: arcID,
			Metric: "FieldKDA", Target: 1.0,
			WindowType: prestige.WindowSession, Cadence: prestige.CadenceFree,
			EvalType: prestige.EvalThreshold, Mode: prestige.ModeLibre,
			DataTier: prestige.DataFull, Status: prestige.StatusActive, CreatedAt: now,
		}); err != nil {
			t.Fatalf("challenge Create: %v", err)
		}
	}
	return chRepo, arcRepo
}

func TestPrestigeChallengeRepo_DetachFromArc(t *testing.T) {
	db := setupPrestigeDB(t, migration.TargetPlayer)
	chRepo, _ := seedArcWithChallenges(t, db, "arc1", "ch1", "ch2")
	ctx := context.Background()

	if err := chRepo.DetachFromArc(ctx, "arc1"); err != nil {
		t.Fatalf("DetachFromArc: %v", err)
	}
	arcID := "arc1"
	inArc, _ := chRepo.List(ctx, prestige.ChallengeFilter{UserID: "u1", TitleSlug: "halo_infinite", ArcID: &arcID})
	if len(inArc) != 0 {
		t.Errorf("after detach: %d défis encore liés, want 0", len(inArc))
	}
	all, _ := chRepo.List(ctx, prestige.ChallengeFilter{UserID: "u1", TitleSlug: "halo_infinite"})
	if len(all) != 2 {
		t.Errorf("after detach: défis supprimés à tort, got %d want 2", len(all))
	}
}

func TestPrestigeChallengeRepo_DeleteByArc(t *testing.T) {
	db := setupPrestigeDB(t, migration.TargetPlayer)
	chRepo, _ := seedArcWithChallenges(t, db, "arc1", "ch1", "ch2")
	ctx := context.Background()

	if err := chRepo.DeleteByArc(ctx, "arc1"); err != nil {
		t.Fatalf("DeleteByArc: %v", err)
	}
	all, _ := chRepo.List(ctx, prestige.ChallengeFilter{UserID: "u1", TitleSlug: "halo_infinite"})
	if len(all) != 0 {
		t.Errorf("after DeleteByArc: %d défis restants, want 0", len(all))
	}
}

func TestPrestigeArcRepo_Delete(t *testing.T) {
	db := setupPrestigeDB(t, migration.TargetPlayer)
	_, arcRepo := seedArcWithChallenges(t, db, "arc1")
	ctx := context.Background()

	if err := arcRepo.Delete(ctx, "arc1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := arcRepo.Get(ctx, "arc1"); !errors.Is(err, prestige.ErrArcNotFound) {
		t.Errorf("expected ErrArcNotFound after delete, got %v", err)
	}
}

// ─────────── PrestigeRepo (events + leaderboard) ───────────

func TestPrestigeSocialRepo_EmitEvent_BumpsTotal(t *testing.T) {
	db := setupPrestigeDB(t, migration.TargetSharedSocial)
	repo := NewPrestigeSocialRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	for i, pp := range []int{50, 75, 125} {
		ev := prestige.PrestigeEvent{
			ID:     "pe_" + string(rune('a'+i)),
			UserID: "u1", TitleSlug: "halo_infinite",
			SourceType: prestige.SourceChallenge, SourceID: "ch_x",
			PPAmount: pp, Tier: prestige.TierHeroic,
			CreatedAt: now,
		}
		if err := repo.EmitEvent(ctx, ev); err != nil {
			t.Fatalf("EmitEvent: %v", err)
		}
	}

	up, err := repo.GetUserPrestige(ctx, "u1", "halo_infinite")
	if err != nil {
		t.Fatalf("GetUserPrestige: %v", err)
	}
	if up.TotalPP != 250 {
		t.Errorf("got total_pp=%d want 250", up.TotalPP)
	}
}

func TestPrestigeSocialRepo_Leaderboard_PerTitle(t *testing.T) {
	db := setupPrestigeDB(t, migration.TargetSharedSocial)
	repo := NewPrestigeSocialRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	users := map[string]int{"u1": 1000, "u2": 500, "u3": 1500}
	for uid, pp := range users {
		if err := repo.UpsertUserPrestige(ctx, prestige.UserPrestige{
			UserID: uid, TitleSlug: "halo_infinite",
			TotalPP: pp, CurrentLevel: 0, UpdatedAt: now,
		}); err != nil {
			t.Fatalf("Upsert: %v", err)
		}
	}
	// PP d'un SECOND titre pour le même joueur : ils ne doivent JAMAIS s'ajouter
	// au classement d'Halo Infinite (indépendance stricte par titre, décision
	// produit 2026-07-18). Sans ces lignes, la suppression de la branche
	// cross-titre (V721-14a / D-08) ne serait pas observable.
	if err := repo.UpsertUserPrestige(ctx, prestige.UserPrestige{
		UserID: "u1", TitleSlug: "halo_5",
		TotalPP: 700, CurrentLevel: 0, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("Upsert halo_5: %v", err)
	}

	board, err := repo.GetLeaderboard(ctx, []string{"u1", "u2", "u3"}, "halo_infinite", time.Time{})
	if err != nil {
		t.Fatalf("GetLeaderboard: %v", err)
	}
	if len(board) != 3 {
		t.Fatalf("got %d entries want 3", len(board))
	}
	if board[0].UserID != "u3" || board[0].TotalPP != 1500 {
		t.Errorf("expected u3 first with 1500, got %s with %d", board[0].UserID, board[0].TotalPP)
	}
	for _, e := range board {
		if e.TitleSlug != "halo_infinite" {
			t.Errorf("entrée %q hors titre demandé: title_slug=%q", e.UserID, e.TitleSlug)
		}
		if e.UserID == "u1" && e.TotalPP != 1000 {
			t.Errorf("FUSION CROSS-TITRE : u1 = %d PP (attendu 1000, halo_5 exclu)", e.TotalPP)
		}
	}
}

// TestPrestigeSocialRepo_Leaderboard_RejectsEmptyTitle — MORSURE de D-08.
// Un slug vide doit être REFUSÉ, jamais dégradé en « tous titres confondus ».
// Si la branche cross-titre supprimée le 2026-07-25 revenait (sous n'importe
// quelle forme : *string nil, slug vide, flag), ce test la verrait : il
// recevrait un classement au lieu d'une erreur.
func TestPrestigeSocialRepo_Leaderboard_RejectsEmptyTitle(t *testing.T) {
	db := setupPrestigeDB(t, migration.TargetSharedSocial)
	repo := NewPrestigeSocialRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	for _, slug := range []string{"halo_infinite", "halo_5"} {
		if err := repo.UpsertUserPrestige(ctx, prestige.UserPrestige{
			UserID: "u1", TitleSlug: slug, TotalPP: 300, CurrentLevel: 0, UpdatedAt: now,
		}); err != nil {
			t.Fatalf("Upsert %s: %v", slug, err)
		}
	}

	board, err := repo.GetLeaderboard(ctx, []string{"u1"}, "", time.Time{})
	if err == nil {
		t.Fatalf("GetLeaderboard(title_slug=\"\") a rendu %d entrée(s) au lieu d'une erreur — "+
			"la voie cross-titre est de retour", len(board))
	}
	if !errors.Is(err, prestige.ErrInvalidInput) {
		t.Errorf("GetLeaderboard(title_slug=\"\") : erreur %v, attendu ErrInvalidInput", err)
	}
	if board != nil {
		t.Errorf("GetLeaderboard(title_slug=\"\") doit rendre nil, got %v", board)
	}
}

// ─────────── TemplateRepo ───────────

func TestPrestigeTemplateRepo_ReplaceListGet(t *testing.T) {
	db := setupPrestigeDB(t, migration.TargetMetadata)
	repo := NewPrestigeTemplateRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	templates := []prestige.Template{
		{
			ID: "tpl_1", TitleSlug: "halo_infinite", Metric: "FieldKDA",
			WindowType: prestige.WindowSession, WindowValue: "1",
			Cadence: prestige.CadenceDaily, EvalType: prestige.EvalThreshold,
			ModeFilter: "universal",
			LabelEN:    "Stay sharp", LabelFR: "Reste affûté",
			NormalTarget: 1.0, HeroicTarget: 1.3, LegendaryTarget: 1.6, MythicTarget: 2.0,
			SchemaVersion: 1, UpdatedAt: now,
		},
		{
			ID: "tpl_2", TitleSlug: "halo_infinite", Metric: "FieldWinRate",
			WindowType: prestige.WindowSession, WindowValue: "3",
			Cadence: prestige.CadenceWeekly, EvalType: prestige.EvalThreshold,
			ModeFilter: "pvp",
			LabelEN:    "Winning streak", LabelFR: "Série de victoires",
			NormalTarget: 50, HeroicTarget: 60, LegendaryTarget: 70, MythicTarget: 80,
			SchemaVersion: 1, UpdatedAt: now,
		},
	}
	if err := repo.Replace(ctx, "halo_infinite", templates); err != nil {
		t.Fatalf("Replace: %v", err)
	}

	list, err := repo.ListByTitle(ctx, "halo_infinite")
	if err != nil {
		t.Fatalf("ListByTitle: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d templates want 2", len(list))
	}
	got, err := repo.GetByID(ctx, "tpl_1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.LabelFR != "Reste affûté" {
		t.Errorf("got label %q", got.LabelFR)
	}
}

func TestPrestigeTemplateRepo_Suggest_ExcludesIDs(t *testing.T) {
	db := setupPrestigeDB(t, migration.TargetMetadata)
	repo := NewPrestigeTemplateRepo(db)
	ctx := context.Background()
	now := time.Now().UTC()

	tpls := make([]prestige.Template, 5)
	for i := 0; i < 5; i++ {
		tpls[i] = prestige.Template{
			ID:        "tpl_" + string(rune('a'+i)),
			TitleSlug: "halo_infinite", Metric: "FieldKDA",
			WindowType: prestige.WindowSession, Cadence: prestige.CadenceDaily,
			EvalType: prestige.EvalThreshold, ModeFilter: "universal",
			LabelEN: "L", LabelFR: "L",
			NormalTarget: 1, HeroicTarget: 2, LegendaryTarget: 3, MythicTarget: 4,
			SchemaVersion: 1, UpdatedAt: now,
		}
	}
	if err := repo.Replace(ctx, "halo_infinite", tpls); err != nil {
		t.Fatalf("Replace: %v", err)
	}

	got, err := repo.Suggest(ctx, "halo_infinite", []string{"tpl_a", "tpl_b"}, 2)
	if err != nil {
		t.Fatalf("Suggest: %v", err)
	}
	for _, t2 := range got {
		if t2.ID == "tpl_a" || t2.ID == "tpl_b" {
			t.Errorf("expected exclusion, got %s", t2.ID)
		}
	}
}
