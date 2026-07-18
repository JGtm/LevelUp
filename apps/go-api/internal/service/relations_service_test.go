package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

type mockRelationsRepo struct {
	rows      []domain.RelationRawRow
	err       error
	gotScope  []string // capture le scope reçu (assertions Phase 2)
	scopeSeen bool

	// Phase 3a : heatmap + timelines par rival.
	heatmapRows    []domain.RelationHeatmapRawRow
	heatmapTopN    int
	timelineByXUID map[string][]domain.RelationDuelRawRow
	timelineLimit  int

	// Carte Noyau dur : WR perso (lift) + frise forme récente.
	coreEngagement domain.CoreEngagement
	gotCoreXUIDs   []string

	// Carte binôme : forme récente du top-allié.
	topAllyForm    []string
	gotTopAllyXUID string

	// Carte bête noire : forme récente CONTRE le top-nemesis.
	topNemesisForm    []string
	gotTopNemesisXUID string

	// Contexte CSR de la bête noire (lot relations-G).
	csrByXUID   map[string]*domain.RelationCSR
	csrErr      error
	gotCSRXUIDs []string
}

func (m *mockRelationsRepo) GetRelations(_ context.Context, scope []string) ([]domain.RelationRawRow, error) {
	m.gotScope = scope
	m.scopeSeen = true
	return m.rows, m.err
}

func (m *mockRelationsRepo) GetRelationsHeatmap(_ context.Context, _ []string, topN int) ([]domain.RelationHeatmapRawRow, error) {
	m.heatmapTopN = topN
	return m.heatmapRows, m.err
}

func (m *mockRelationsRepo) GetRivalTimeline(_ context.Context, rivalXUID string, _ []string, limit int) ([]domain.RelationDuelRawRow, error) {
	m.timelineLimit = limit
	if m.timelineByXUID == nil {
		return nil, m.err
	}
	return m.timelineByXUID[rivalXUID], m.err
}

func (m *mockRelationsRepo) GetCoreEngagement(_ context.Context, coreXUIDs []string, _ []string, _ int) (domain.CoreEngagement, error) {
	m.gotCoreXUIDs = coreXUIDs
	return m.coreEngagement, nil
}

func (m *mockRelationsRepo) GetRelationRecentForm(_ context.Context, xuid string, _ []string, _ int) ([]string, error) {
	m.gotTopAllyXUID = xuid
	return m.topAllyForm, nil
}

func (m *mockRelationsRepo) GetRelationEnemyRecentForm(_ context.Context, xuid string, _ []string, _ int) ([]string, error) {
	m.gotTopNemesisXUID = xuid
	return m.topNemesisForm, nil
}

func (m *mockRelationsRepo) GetLatestCSR(_ context.Context, xuid string) (*domain.RelationCSR, error) {
	m.gotCSRXUIDs = append(m.gotCSRXUIDs, xuid)
	if m.csrErr != nil {
		return nil, m.csrErr
	}
	if m.csrByXUID == nil {
		return nil, nil
	}
	return m.csrByXUID[xuid], nil
}

// mockFiltersService renvoie un scope fixe et capture l'input reçu.
type mockFiltersService struct {
	ids      []string
	err      error
	gotInput domain.FilterContextInput
	called   bool
}

func (m *mockFiltersService) Resolve(_ context.Context, _ domain.FilterContextInput) (domain.FilterContextResolved, error) {
	return domain.FilterContextResolved{}, nil
}

func (m *mockFiltersService) ResolveMatchIDs(_ context.Context, in domain.FilterContextInput) ([]string, error) {
	m.called = true
	m.gotInput = in
	return m.ids, m.err
}

func fixedNow() func() time.Time {
	t := time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)
	return func() time.Time { return t }
}

func TestGetRelationsPage_Empty(t *testing.T) {
	svc := NewRelationsService(&mockRelationsRepo{}).withNow(fixedNow())
	page, err := svc.GetRelationsPage(context.Background(), domain.FilterContextInput{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if page.Overview.DistinctPlayers != 0 {
		t.Fatalf("distinct=%d want 0", page.Overview.DistinctPlayers)
	}
	if page.Relations == nil {
		t.Fatal("relations must be non-nil empty slice, not null")
	}
	if len(page.Relations) != 0 {
		t.Fatalf("relations len=%d want 0", len(page.Relations))
	}
	if page.Overview.TopAlly != nil || page.Overview.TopNemesis != nil {
		t.Fatal("top refs must be nil on empty")
	}
}

func TestGetRelationsPage_Error(t *testing.T) {
	svc := NewRelationsService(&mockRelationsRepo{err: errors.New("boom")})
	if _, err := svc.GetRelationsPage(context.Background(), domain.FilterContextInput{}); err == nil {
		t.Fatal("expected error propagation")
	}
}

func TestGetRelationsPage_Enriched(t *testing.T) {
	now := time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)
	old := now.AddDate(0, -8, 0)
	rows := []domain.RelationRawRow{
		{
			XUID: "x1", Gamertag: "Ally", TotalMatches: 15,
			TeammateCount: 15, TeammateWins: 11, TeammateLosses: 4,
			KillsDealt: 10, DeathsSuffered: 5, FirstSeen: old, LastSeen: now,
		},
		{
			XUID: "x2", Gamertag: "Nemesis", TotalMatches: 12,
			EnemyCount: 12, EnemyWins: 3, EnemyLosses: 9,
			KillsDealt: 4, DeathsSuffered: 20, FirstSeen: old, LastSeen: now,
		},
		{
			XUID: "x3", Gamertag: "Mix", TotalMatches: 4,
			TeammateCount: 2, EnemyCount: 2, FirstSeen: now, LastSeen: now,
		},
	}
	svc := NewRelationsService(&mockRelationsRepo{rows: rows}).withNow(func() time.Time { return now })
	page, err := svc.GetRelationsPage(context.Background(), domain.FilterContextInput{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if page.Overview.DistinctPlayers != 3 {
		t.Fatalf("distinct=%d want 3", page.Overview.DistinctPlayers)
	}
	if page.Overview.AlliesCount != 2 { // Ally + Mix
		t.Fatalf("allies=%d want 2", page.Overview.AlliesCount)
	}
	if page.Overview.RivalsCount != 2 { // Nemesis + Mix
		t.Fatalf("rivals=%d want 2", page.Overview.RivalsCount)
	}

	// top_ally : Ally (teammate 15 >= 8, win rate 11/15 ~0.733)
	if page.Overview.TopAlly == nil || page.Overview.TopAlly.Gamertag != "Ally" {
		t.Fatalf("top ally=%v want Ally", page.Overview.TopAlly)
	}
	// top_nemesis : Nemesis (enemy 12 >= 8, win rate 3/12 = 0.25)
	if page.Overview.TopNemesis == nil || page.Overview.TopNemesis.Gamertag != "Nemesis" {
		t.Fatalf("top nemesis=%v want Nemesis", page.Overview.TopNemesis)
	}

	// Categories
	byGT := map[string]domain.RelationInsight{}
	for _, r := range page.Relations {
		byGT[r.Gamertag] = r
	}
	if byGT["Ally"].Category != "ally" {
		t.Fatalf("Ally category=%q", byGT["Ally"].Category)
	}
	if byGT["Nemesis"].Category != "enemy" {
		t.Fatalf("Nemesis category=%q", byGT["Nemesis"].Category)
	}
	if byGT["Mix"].Category != "mixed" {
		t.Fatalf("Mix category=%q", byGT["Mix"].Category)
	}

	// Win rates
	if byGT["Ally"].TeammateWinRate == nil || *byGT["Ally"].TeammateWinRate < 0.73 {
		t.Fatalf("Ally teammate win rate=%v", byGT["Ally"].TeammateWinRate)
	}
	if byGT["Ally"].EnemyWinRate != nil {
		t.Fatal("Ally enemy win rate must be nil")
	}
	// Duel ratio Ally = 10/5 = 2.0
	if byGT["Ally"].DuelRatio == nil || *byGT["Ally"].DuelRatio != 2.0 {
		t.Fatalf("Ally duel ratio=%v want 2.0", byGT["Ally"].DuelRatio)
	}
	// Mix duel ratio nil (0 kills 0 deaths)
	if byGT["Mix"].DuelRatio != nil {
		t.Fatalf("Mix duel ratio=%v want nil", byGT["Mix"].DuelRatio)
	}
	// first_seen_at / last_seen_at present
	if byGT["Ally"].FirstSeenAt == nil || byGT["Ally"].LastSeenAt == nil {
		t.Fatal("Ally timestamps must be set")
	}
}

// top_ally_recent_form / top_nemesis_recent_form : chaque frise « Derniers matchs »
// est lue pour le BON joueur (binôme via GetRelationRecentForm = à ses côtés ; bête
// noire via GetRelationEnemyRecentForm = face à lui) et posée sur l'overview.
func TestGetRelationsPage_RecentForms(t *testing.T) {
	now := time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)
	repo := &mockRelationsRepo{
		rows:           nemesisRows(now),
		topAllyForm:    []string{"win", "loss", "win"},
		topNemesisForm: []string{"loss", "loss", "win"},
	}
	svc := NewRelationsService(repo).withNow(func() time.Time { return now })
	page, err := svc.GetRelationsPage(context.Background(), domain.FilterContextInput{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	// Binôme : forme lue pour x1 (Ally) → top_ally_recent_form.
	if repo.gotTopAllyXUID != "x1" {
		t.Fatalf("top-ally form lu pour %q want x1", repo.gotTopAllyXUID)
	}
	if got := page.Overview.TopAllyRecentForm; len(got) != 3 || got[0] != "win" {
		t.Fatalf("top_ally_recent_form=%v want [win loss win]", got)
	}
	// Bête noire : forme lue pour x2 (Nemesis) via le miroir ennemi → top_nemesis_recent_form.
	if repo.gotTopNemesisXUID != "x2" {
		t.Fatalf("top-nemesis form lu pour %q want x2", repo.gotTopNemesisXUID)
	}
	if got := page.Overview.TopNemesisRecentForm; len(got) != 3 || got[0] != "loss" {
		t.Fatalf("top_nemesis_recent_form=%v want [loss loss win]", got)
	}
}

// ─── Lot relations-G : contexte CSR de la bête noire (best-effort) ──────────

// nemesisRows : un allié (x1) + une bête noire nette (x2, enemy WR 3/12) — la
// bête noire retenue par SelectTopNemesis.
func nemesisRows(now time.Time) []domain.RelationRawRow {
	old := now.AddDate(0, -8, 0)
	return []domain.RelationRawRow{
		{
			XUID: "x1", Gamertag: "Ally", TotalMatches: 15,
			TeammateCount: 15, TeammateWins: 11, TeammateLosses: 4,
			KillsDealt: 10, DeathsSuffered: 5, FirstSeen: old, LastSeen: now,
		},
		{
			XUID: "x2", Gamertag: "Nemesis", TotalMatches: 12,
			EnemyCount: 12, EnemyWins: 3, EnemyLosses: 9,
			KillsDealt: 4, DeathsSuffered: 20, FirstSeen: old, LastSeen: now,
		},
	}
}

func strptr(s string) *string { return &s }
func intptr(n int) *int       { return &n }
func f64ptr(f float64) *float64 { return &f }

// Bête noire AVEC ligne CSR dans match_csrs_latest → CSR peuplé sur top_nemesis
// (et lu par le xuid de la bête noire), top_ally NON enrichi.
func TestGetRelationsPage_NemesisCSR_Populated(t *testing.T) {
	now := time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)
	want := &domain.RelationCSR{Tier: strptr("Onyx"), SubTier: intptr(0), RatingValue: f64ptr(1523)}
	repo := &mockRelationsRepo{
		rows:      nemesisRows(now),
		csrByXUID: map[string]*domain.RelationCSR{"x2": want},
	}
	svc := NewRelationsService(repo).withNow(func() time.Time { return now })

	page, err := svc.GetRelationsPage(context.Background(), domain.FilterContextInput{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if page.Overview.TopNemesis == nil || page.Overview.TopNemesis.Gamertag != "Nemesis" {
		t.Fatalf("top nemesis=%v want Nemesis", page.Overview.TopNemesis)
	}
	csr := page.Overview.TopNemesis.CSR
	if csr == nil || csr.Tier == nil || *csr.Tier != "Onyx" {
		t.Fatalf("nemesis CSR=%+v want tier Onyx", csr)
	}
	if csr.RatingValue == nil || *csr.RatingValue != 1523 {
		t.Fatalf("nemesis CSR rating=%v want 1523", csr.RatingValue)
	}
	// Lu par le XUID de la bête noire (x2), pas de l'allié.
	if len(repo.gotCSRXUIDs) != 1 || repo.gotCSRXUIDs[0] != "x2" {
		t.Fatalf("GetLatestCSR called with %v want [x2]", repo.gotCSRXUIDs)
	}
	// L'allié n'est jamais enrichi (CSR réservé à la bête noire).
	if page.Overview.TopAlly == nil || page.Overview.TopAlly.CSR != nil {
		t.Fatalf("top ally CSR=%v want nil", page.Overview.TopAlly)
	}
}

// Bête noire SANS ligne CSR (relation social / non collectée) → CSR nil :
// dégradation gracieuse, rien à afficher, aucune erreur.
func TestGetRelationsPage_NemesisCSR_Absent(t *testing.T) {
	now := time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)
	repo := &mockRelationsRepo{
		rows:      nemesisRows(now),
		csrByXUID: map[string]*domain.RelationCSR{}, // aucune entrée pour x2
	}
	svc := NewRelationsService(repo).withNow(func() time.Time { return now })

	page, err := svc.GetRelationsPage(context.Background(), domain.FilterContextInput{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if page.Overview.TopNemesis == nil {
		t.Fatal("top nemesis must be set")
	}
	if page.Overview.TopNemesis.CSR != nil {
		t.Fatalf("nemesis CSR=%v want nil (graceful degradation)", page.Overview.TopNemesis.CSR)
	}
}

// Lecture CSR en échec → l'aperçu est renvoyé SANS CSR, l'erreur n'est PAS propagée
// (best-effort strict : un échec de l'enrichissement ne casse jamais /relations).
func TestGetRelationsPage_NemesisCSR_BestEffortOnError(t *testing.T) {
	now := time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)
	repo := &mockRelationsRepo{
		rows:   nemesisRows(now),
		csrErr: errors.New("csr read boom"),
	}
	svc := NewRelationsService(repo).withNow(func() time.Time { return now })

	page, err := svc.GetRelationsPage(context.Background(), domain.FilterContextInput{})
	if err != nil {
		t.Fatalf("CSR read error must NOT propagate, got: %v", err)
	}
	if page.Overview.TopNemesis == nil || page.Overview.TopNemesis.CSR != nil {
		t.Fatalf("nemesis CSR=%v want nil on read error", page.Overview.TopNemesis)
	}
}

// Aucune bête noire (que des alliés) → GetLatestCSR n'est jamais appelé.
func TestGetRelationsPage_NemesisCSR_NoNemesis_NoLookup(t *testing.T) {
	now := time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)
	old := now.AddDate(0, -8, 0)
	repo := &mockRelationsRepo{
		rows: []domain.RelationRawRow{
			{
				XUID: "x1", Gamertag: "Ally", TotalMatches: 15,
				TeammateCount: 15, TeammateWins: 11, TeammateLosses: 4,
				KillsDealt: 10, DeathsSuffered: 5, FirstSeen: old, LastSeen: now,
			},
		},
	}
	svc := NewRelationsService(repo).withNow(func() time.Time { return now })

	page, err := svc.GetRelationsPage(context.Background(), domain.FilterContextInput{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if page.Overview.TopNemesis != nil {
		t.Fatalf("top nemesis=%v want nil", page.Overview.TopNemesis)
	}
	if len(repo.gotCSRXUIDs) != 0 {
		t.Fatalf("GetLatestCSR called %v want no call", repo.gotCSRXUIDs)
	}
}

// ─── Phase 2 : segmentation serveur (scope match_id) ────────────────────────

// Input trivial (tout) → pas de résolution de scope, scope nil passé au repo.
func TestGetRelationsPage_TrivialInput_NoScope(t *testing.T) {
	repo := &mockRelationsRepo{}
	fs := &mockFiltersService{ids: []string{"x"}}
	svc := NewRelationsService(repo).WithFilters(fs).withNow(fixedNow())

	if _, err := svc.GetRelationsPage(context.Background(), domain.FilterContextInput{}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if fs.called {
		t.Fatal("FiltersService must NOT be called for a trivial input")
	}
	if !repo.scopeSeen || repo.gotScope != nil {
		t.Fatalf("repo scope=%v want nil (Phase 1 path)", repo.gotScope)
	}
}

// Input actif (cascade) → résolution → scope non-nil passé au repo.
func TestGetRelationsPage_ActiveInput_ScopeResolved(t *testing.T) {
	repo := &mockRelationsRepo{}
	fs := &mockFiltersService{ids: []string{"m1", "m2"}}
	svc := NewRelationsService(repo).WithFilters(fs).withNow(fixedNow())

	in := domain.FilterContextInput{Cascade: domain.CascadeFilter{Playlists: []string{"Ranked Arena"}}}
	if _, err := svc.GetRelationsPage(context.Background(), in); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !fs.called {
		t.Fatal("FiltersService must be called for an active input")
	}
	if len(repo.gotScope) != 2 || repo.gotScope[0] != "m1" {
		t.Fatalf("repo scope=%v want [m1 m2]", repo.gotScope)
	}
}

// Vue solo/escouade seule → input actif → résolution.
func TestGetRelationsPage_MatchContextActivatesScope(t *testing.T) {
	repo := &mockRelationsRepo{}
	fs := &mockFiltersService{ids: []string{"m1"}}
	svc := NewRelationsService(repo).WithFilters(fs).withNow(fixedNow())

	in := domain.FilterContextInput{MatchContext: domain.MatchContextSquad}
	if _, err := svc.GetRelationsPage(context.Background(), in); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !fs.called || fs.gotInput.MatchContext != domain.MatchContextSquad {
		t.Fatalf("expected squad context forwarded, got %+v called=%v", fs.gotInput.MatchContext, fs.called)
	}
}

// Période bornée → input actif.
func TestGetRelationsPage_PeriodActivatesScope(t *testing.T) {
	repo := &mockRelationsRepo{}
	fs := &mockFiltersService{ids: []string{"m1"}}
	svc := NewRelationsService(repo).WithFilters(fs).withNow(fixedNow())
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	in := domain.FilterContextInput{Period: domain.PeriodInput{StartDate: &start}}
	if _, err := svc.GetRelationsPage(context.Background(), in); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !fs.called {
		t.Fatal("period filter must activate scope resolution")
	}
}

// Résolution renvoyant 0 match → scope vide (non-nil) passé au repo (qui
// court-circuite) → page vide.
func TestGetRelationsPage_EmptyResolution_PassesEmptyScope(t *testing.T) {
	repo := &mockRelationsRepo{}
	fs := &mockFiltersService{ids: nil} // population vide
	svc := NewRelationsService(repo).WithFilters(fs).withNow(fixedNow())

	in := domain.FilterContextInput{MatchContext: domain.MatchContextSolo}
	if _, err := svc.GetRelationsPage(context.Background(), in); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if repo.gotScope == nil {
		t.Fatal("empty resolution must pass a non-nil empty scope (not nil = tout)")
	}
	if len(repo.gotScope) != 0 {
		t.Fatalf("scope=%v want empty", repo.gotScope)
	}
}

// Erreur de résolution propagée.
func TestGetRelationsPage_ResolveError(t *testing.T) {
	fs := &mockFiltersService{err: errors.New("resolve boom")}
	svc := NewRelationsService(&mockRelationsRepo{}).WithFilters(fs).withNow(fixedNow())
	in := domain.FilterContextInput{Cascade: domain.CascadeFilter{Modes: []string{"Slayer"}}}
	if _, err := svc.GetRelationsPage(context.Background(), in); err == nil {
		t.Fatal("expected resolve error propagation")
	}
}

// Sans FiltersService injecté, un input actif reste inerte (scope nil) :
// dégradation gracieuse, jamais de panic.
func TestGetRelationsPage_NoFiltersService_ScopeNil(t *testing.T) {
	repo := &mockRelationsRepo{}
	svc := NewRelationsService(repo).withNow(fixedNow())
	in := domain.FilterContextInput{
		Cascade:  domain.CascadeFilter{Playlists: []string{"X"}},
		Sessions: domain.SessionsFilter{PickedSessionLabel: strPtr("s")},
	}
	if _, err := svc.GetRelationsPage(context.Background(), in); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if repo.gotScope != nil {
		t.Fatalf("scope=%v want nil (no filters service)", repo.gotScope)
	}
}
