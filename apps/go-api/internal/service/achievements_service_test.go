package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// mockAchievementsRepo implémente port.AchievementsRepository.
type mockAchievementsRepo struct {
	rows []domain.PlayerAchievementRow
	err  error
}

func (m *mockAchievementsRepo) GetPlayerAchievements(_ context.Context) ([]domain.PlayerAchievementRow, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.rows, nil
}

// mockMetadataAchievementsRepo implémente port.MetadataAchievementsRepository.
type mockMetadataAchievementsRepo struct {
	defs    []domain.AchievementDefinitionRow
	err     error
	gotSlug string // capture le slug passé par le service (PMT-6)
}

func (m *mockMetadataAchievementsRepo) GetAchievementDefinitions(_ context.Context, titleSlug string) ([]domain.AchievementDefinitionRow, error) {
	m.gotSlug = titleSlug
	if m.err != nil {
		return nil, m.err
	}
	return m.defs, nil
}

// TestAchievementsService_PassesTitleSlugToRepo (PMT-6) : le service transmet son
// slug au filtre de définitions ; absence de slug → fallback halo_infinite.
func TestAchievementsService_PassesTitleSlugToRepo(t *testing.T) {
	meta := &mockMetadataAchievementsRepo{}
	svc := NewAchievementsService(&mockAchievementsRepo{}, meta).WithTitleSlug("synthetic_test_title")
	if _, err := svc.GetAchievementsPage(context.Background()); err != nil {
		t.Fatalf("GetAchievementsPage: %v", err)
	}
	if meta.gotSlug != "synthetic_test_title" {
		t.Errorf("slug passé au repo = %q, attendu synthetic_test_title", meta.gotSlug)
	}

	meta2 := &mockMetadataAchievementsRepo{}
	svc2 := NewAchievementsService(&mockAchievementsRepo{}, meta2)
	if _, err := svc2.GetAchievementsPage(context.Background()); err != nil {
		t.Fatalf("GetAchievementsPage(default): %v", err)
	}
	if meta2.gotSlug != "halo_infinite" {
		t.Errorf("slug par défaut = %q, attendu halo_infinite (fallback)", meta2.gotSlug)
	}
}

// intPtr est défini dans testhelpers_test.go.

func timePtr(t time.Time) *time.Time { return &t }

// TestAchievementsService_NominalCase : 5 définitions + 2 unlocked → summary correct,
// tri correct (locked d'abord par gamerscore DESC, puis unlocked par UnlockedAt ASC).
func TestAchievementsService_NominalCase(t *testing.T) {
	t1 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 4, 15, 14, 0, 0, 0, time.UTC)
	defs := []domain.AchievementDefinitionRow{
		{AchievementID: "a", NameEN: "A", Gamerscore: 10},
		{AchievementID: "b", NameEN: "B", Gamerscore: 50},
		{AchievementID: "c", NameEN: "C", Gamerscore: 30},
		{AchievementID: "d", NameEN: "D", Gamerscore: 20},
		{AchievementID: "e", NameEN: "E", Gamerscore: 100},
	}
	playerRows := []domain.PlayerAchievementRow{
		{AchievementID: "a", Unlocked: true, UnlockedAt: timePtr(t1)},
		{AchievementID: "c", Unlocked: true, UnlockedAt: timePtr(t2)},
		{AchievementID: "d", Unlocked: false, CurrentProgress: intPtr(5), TargetProgress: intPtr(10)},
	}
	svc := NewAchievementsService(
		&mockAchievementsRepo{rows: playerRows},
		&mockMetadataAchievementsRepo{defs: defs},
	).WithTitleSlug("halo_infinite")

	resp, err := svc.GetAchievementsPage(context.Background())
	if err != nil {
		t.Fatalf("GetAchievementsPage: %v", err)
	}
	if resp.Summary.TotalCount != 5 || resp.Summary.UnlockedCount != 2 {
		t.Errorf("summary count: total=%d unlocked=%d (attendu 5/2)",
			resp.Summary.TotalCount, resp.Summary.UnlockedCount)
	}
	if resp.Summary.TotalGamerscore != 210 || resp.Summary.EarnedGamerscore != 40 {
		t.Errorf("summary gamerscore: total=%d earned=%d (attendu 210/40)",
			resp.Summary.TotalGamerscore, resp.Summary.EarnedGamerscore)
	}
	if resp.Summary.CompletionPct != 40.0 {
		t.Errorf("completion pct: attendu 40.0, obtenu %v", resp.Summary.CompletionPct)
	}
	if len(resp.Achievements) != 5 {
		t.Fatalf("achievements: attendu 5, obtenu %d", len(resp.Achievements))
	}
	// Tri : locked en premier (gamerscore DESC) → e (100), b (50), d (20)
	// Puis unlocked (UnlockedAt ASC, le plus ancien en haut) → a (1 avril), c (15 avril)
	expected := []string{"e", "b", "d", "a", "c"}
	for i, want := range expected {
		if resp.Achievements[i].AchievementID != want {
			t.Errorf("position %d: attendu %s, obtenu %s",
				i, want, resp.Achievements[i].AchievementID)
		}
	}
	// Vérifier que la progression est attachée à 'd'
	for _, e := range resp.Achievements {
		if e.AchievementID == "d" {
			if e.CurrentProgress == nil || *e.CurrentProgress != 5 {
				t.Errorf("d.CurrentProgress: attendu 5")
			}
			if e.TargetProgress == nil || *e.TargetProgress != 10 {
				t.Errorf("d.TargetProgress: attendu 10")
			}
		}
	}
}

// TestAchievementsService_NewPlayer : 0 player rows → tous locked, earned=0.
func TestAchievementsService_NewPlayer(t *testing.T) {
	defs := []domain.AchievementDefinitionRow{
		{AchievementID: "a", NameEN: "A", Gamerscore: 10},
		{AchievementID: "b", NameEN: "B", Gamerscore: 50},
	}
	svc := NewAchievementsService(
		&mockAchievementsRepo{rows: nil},
		&mockMetadataAchievementsRepo{defs: defs},
	)

	resp, err := svc.GetAchievementsPage(context.Background())
	if err != nil {
		t.Fatalf("GetAchievementsPage: %v", err)
	}
	if resp.Summary.TotalCount != 2 || resp.Summary.UnlockedCount != 0 {
		t.Errorf("summary count: total=%d unlocked=%d (attendu 2/0)",
			resp.Summary.TotalCount, resp.Summary.UnlockedCount)
	}
	if resp.Summary.EarnedGamerscore != 0 || resp.Summary.CompletionPct != 0 {
		t.Errorf("summary doit être 0/0 pour un nouveau joueur")
	}
	for _, e := range resp.Achievements {
		if e.Unlocked {
			t.Errorf("achievement %s ne doit pas être unlocked", e.AchievementID)
		}
	}
}

// TestAchievementsService_EmptyDefinitions : aucune définition (backfill non lancé)
// → réponse vide valide, pas d'erreur.
func TestAchievementsService_EmptyDefinitions(t *testing.T) {
	svc := NewAchievementsService(
		&mockAchievementsRepo{rows: nil},
		&mockMetadataAchievementsRepo{defs: nil},
	)

	resp, err := svc.GetAchievementsPage(context.Background())
	if err != nil {
		t.Fatalf("GetAchievementsPage: %v", err)
	}
	if resp.Summary.TotalCount != 0 || len(resp.Achievements) != 0 {
		t.Errorf("réponse doit être vide quand pas de définitions")
	}
}

// TestAchievementsService_OrphanPlayerRow : player row sans définition correspondante
// → ignorée silencieusement (pas d'entrée dans la réponse, pas d'erreur).
func TestAchievementsService_OrphanPlayerRow(t *testing.T) {
	defs := []domain.AchievementDefinitionRow{
		{AchievementID: "a", NameEN: "A", Gamerscore: 10},
	}
	playerRows := []domain.PlayerAchievementRow{
		{AchievementID: "a", Unlocked: true, UnlockedAt: timePtr(time.Now().UTC())},
		{AchievementID: "ghost", Unlocked: true}, // orphelin : pas de définition
	}
	svc := NewAchievementsService(
		&mockAchievementsRepo{rows: playerRows},
		&mockMetadataAchievementsRepo{defs: defs},
	)

	resp, err := svc.GetAchievementsPage(context.Background())
	if err != nil {
		t.Fatalf("GetAchievementsPage: %v", err)
	}
	if len(resp.Achievements) != 1 {
		t.Fatalf("attendu 1 achievement (l'orphelin doit être ignoré), obtenu %d",
			len(resp.Achievements))
	}
	if resp.Achievements[0].AchievementID != "a" {
		t.Errorf("seul 'a' doit être présent")
	}
	if resp.Summary.UnlockedCount != 1 {
		t.Errorf("summary.UnlockedCount: attendu 1 (orphelin ignoré), obtenu %d",
			resp.Summary.UnlockedCount)
	}
}

// TestAchievementsService_Categories : la catégorie est résolue via le mapping
// statique du titre (fallback halo_infinite sans WithTitleSlug), nom inconnu →
// other, titre sans mapping → catégorie vide.
func TestAchievementsService_Categories(t *testing.T) {
	defs := []domain.AchievementDefinitionRow{
		{AchievementID: "a", NameEN: "Clocking In", Gamerscore: 10},
		{AchievementID: "b", NameEN: "Zeta", Gamerscore: 10},
		{AchievementID: "c", NameEN: "Get the Popcorn", Gamerscore: 10},
		{AchievementID: "d", NameEN: "Some Future DLC Achievement", Gamerscore: 10},
	}
	// Sans WithTitleSlug → fallback halo_infinite.
	svc := NewAchievementsService(
		&mockAchievementsRepo{rows: nil},
		&mockMetadataAchievementsRepo{defs: defs},
	)
	resp, err := svc.GetAchievementsPage(context.Background())
	if err != nil {
		t.Fatalf("GetAchievementsPage: %v", err)
	}
	want := map[string]domain.AchievementCategory{
		"a": domain.AchievementCategoryMultiplayer,
		"b": domain.AchievementCategoryCampaign,
		"c": domain.AchievementCategoryOther,
		"d": domain.AchievementCategoryOther, // inconnu → other
	}
	for _, e := range resp.Achievements {
		if e.Category != want[e.AchievementID] {
			t.Errorf("achievement %s: catégorie %q, attendu %q",
				e.AchievementID, e.Category, want[e.AchievementID])
		}
	}

	// Titre sans mapping → catégorie vide sur toutes les entrées.
	svcNoMapping := NewAchievementsService(
		&mockAchievementsRepo{rows: nil},
		&mockMetadataAchievementsRepo{defs: defs},
	).WithTitleSlug("halo_5")
	resp, err = svcNoMapping.GetAchievementsPage(context.Background())
	if err != nil {
		t.Fatalf("GetAchievementsPage (halo_5): %v", err)
	}
	for _, e := range resp.Achievements {
		if e.Category != "" {
			t.Errorf("titre sans mapping: achievement %s a la catégorie %q, attendu vide",
				e.AchievementID, e.Category)
		}
	}
}

// TestAchievementsService_RepoError : erreur repo propagée au caller.
func TestAchievementsService_RepoError(t *testing.T) {
	svc := NewAchievementsService(
		&mockAchievementsRepo{err: errors.New("db boom")},
		&mockMetadataAchievementsRepo{defs: nil},
	)

	_, err := svc.GetAchievementsPage(context.Background())
	if err == nil {
		t.Fatalf("attendu une erreur quand le repo échoue")
	}
}

// TestAchievementsService_MetadataError : erreur metadata repo propagée.
func TestAchievementsService_MetadataError(t *testing.T) {
	svc := NewAchievementsService(
		&mockAchievementsRepo{rows: nil},
		&mockMetadataAchievementsRepo{err: errors.New("metadata boom")},
	)

	_, err := svc.GetAchievementsPage(context.Background())
	if err == nil {
		t.Fatalf("attendu une erreur quand le metadata repo échoue")
	}
}

// TestComputeCompletionPct : arrondi à 0.1.
func TestComputeCompletionPct(t *testing.T) {
	tests := []struct {
		unlocked, total int
		want            float64
	}{
		{0, 0, 0},
		{0, 100, 0},
		{1, 3, 33.3},
		{2, 3, 66.7},
		{50, 100, 50.0},
		{100, 100, 100.0},
	}
	for _, tc := range tests {
		got := computeCompletionPct(tc.unlocked, tc.total)
		if got != tc.want {
			t.Errorf("computeCompletionPct(%d, %d) = %v, attendu %v",
				tc.unlocked, tc.total, got, tc.want)
		}
	}
}
