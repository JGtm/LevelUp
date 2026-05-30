package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// --- mock ---

type mockExplorerRepo struct {
	xuid     string
	xuidErr  error
	matches  []domain.CommonMatchRaw
	matchErr error
	kv       domain.KillerVictimAggregate
	kvErr    error
	// Encart target_profile : sample stats sur common_matches.
	participants    *domain.ParticipantStatsAggregate
	participantsErr error
	medals          *domain.MedalCountsAggregate
	medalsErr       error
}

func (m *mockExplorerRepo) ResolveXUIDByGamertag(_ context.Context, _ string) (string, error) {
	return m.xuid, m.xuidErr
}
func (m *mockExplorerRepo) GetCommonMatches(_ context.Context, _, _ string) ([]domain.CommonMatchRaw, error) {
	return m.matches, m.matchErr
}
func (m *mockExplorerRepo) GetKillerVictimBetween(_ context.Context, _, _ string) (domain.KillerVictimAggregate, error) {
	return m.kv, m.kvErr
}
func (m *mockExplorerRepo) GetParticipantStatsForMatches(_ context.Context, _ string, _ []string) (*domain.ParticipantStatsAggregate, error) {
	return m.participants, m.participantsErr
}
func (m *mockExplorerRepo) GetMedalCountsForMatches(_ context.Context, _ string, _ []string) (*domain.MedalCountsAggregate, error) {
	return m.medals, m.medalsErr
}

// --- tests ---

func TestExplorerService_GetCommonMatches_OK(t *testing.T) {
	now := time.Now()
	tid1, tid2 := 0, 0
	repo := &mockExplorerRepo{
		xuid: "other-xuid",
		matches: []domain.CommonMatchRaw{
			{
				MatchID:       "m1",
				StartTime:     now,
				MapUI:         "Aquarius",
				ModeUI:        "Slayer",
				Player1TeamID: &tid1,
				Player2TeamID: &tid2,
			},
		},
	}
	svc := NewExplorerService(repo, "my-xuid")

	resp, err := svc.GetCommonMatches(context.Background(), "OtherPlayer", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TargetGamertag != "OtherPlayer" {
		t.Errorf("TargetGamertag = %q, want OtherPlayer", resp.TargetGamertag)
	}
	if resp.TargetXUID != "other-xuid" {
		t.Errorf("TargetXUID = %q, want other-xuid", resp.TargetXUID)
	}
	if resp.Total != 1 {
		t.Errorf("Total = %d, want 1", resp.Total)
	}
	if !resp.CommonMatches[0].WereTeammates {
		t.Error("expected WereTeammates = true (same team ID)")
	}
}

func TestExplorerService_GetCommonMatches_Empty(t *testing.T) {
	repo := &mockExplorerRepo{
		xuid:    "other-xuid",
		matches: []domain.CommonMatchRaw{},
	}
	svc := NewExplorerService(repo, "my-xuid")

	resp, err := svc.GetCommonMatches(context.Background(), "Player", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("Total = %d, want 0", resp.Total)
	}
}

func TestExplorerService_GetCommonMatches_ResolveError(t *testing.T) {
	repo := &mockExplorerRepo{xuidErr: errors.New("not found")}
	svc := NewExplorerService(repo, "my-xuid")

	_, err := svc.GetCommonMatches(context.Background(), "Unknown", 1)
	if err == nil {
		t.Error("expected error")
	}
}

func TestExplorerService_GetCommonMatches_QueryError(t *testing.T) {
	repo := &mockExplorerRepo{xuid: "other", matchErr: errors.New("db fail")}
	svc := NewExplorerService(repo, "my-xuid")

	_, err := svc.GetCommonMatches(context.Background(), "Player", 1)
	if err == nil {
		t.Error("expected error")
	}
}

// TestExplorerService_GetCommonMatches_WithStats — vérifie kills/deaths/kda + OutcomeLabel.
func TestExplorerService_GetCommonMatches_WithStats(t *testing.T) {
	now := time.Now()
	tid1, tid2 := 0, 1 // équipes différentes
	repo := &mockExplorerRepo{
		xuid: "enemy-xuid",
		matches: []domain.CommonMatchRaw{
			{
				MatchID:        "golden-match-1",
				StartTime:      now,
				MapUI:          "Recharge",
				ModeUI:         "Slayer",
				Player1TeamID:  &tid1,
				Player2TeamID:  &tid2,
				Player1Outcome: 2, // WIN
				Player1Kills:   18,
				Player1Deaths:  7,
				Player1KDA:     2.57,
			},
		},
	}
	svc := NewExplorerService(repo, "my-xuid")

	resp, err := svc.GetCommonMatches(context.Background(), "EnemyPlayer", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TotalCount != 1 {
		t.Fatalf("TotalCount = %d, want 1", resp.TotalCount)
	}
	m := resp.CommonMatches[0]
	if m.Kills != 18 {
		t.Errorf("Kills = %d, want 18", m.Kills)
	}
	if m.Deaths != 7 {
		t.Errorf("Deaths = %d, want 7", m.Deaths)
	}
	if m.KDA == 0.0 {
		t.Error("KDA = 0.0 : kills/deaths/kda ne sont pas propagés depuis match_participants")
	}
	if m.WereTeammates {
		t.Error("WereTeammates attendu false (équipes différentes)")
	}
	if m.PlayerOutcome != 2 {
		t.Errorf("PlayerOutcome = %d, want 2 (WIN)", m.PlayerOutcome)
	}
	if m.OutcomeLabel == "" {
		t.Error("OutcomeLabel vide — doit être résolu via outcomeLabel()")
	}
}

func TestExplorerService_GetCommonMatches_DifferentTeams(t *testing.T) {
	now := time.Now()
	tid1, tid2 := 0, 1
	repo := &mockExplorerRepo{
		xuid: "other",
		matches: []domain.CommonMatchRaw{
			{MatchID: "m1", StartTime: now, Player1TeamID: &tid1, Player2TeamID: &tid2},
		},
	}
	svc := NewExplorerService(repo, "my-xuid")

	resp, err := svc.GetCommonMatches(context.Background(), "Enemy", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CommonMatches[0].WereTeammates {
		t.Error("expected WereTeammates = false (different teams)")
	}
}

// TestExplorerService_GetCommonMatches_Pagination vérifie la pagination à 20 éléments.
func TestExplorerService_GetCommonMatches_Pagination(t *testing.T) {
	now := time.Now()
	tid1, tid2 := 0, 0
	// 25 matchs — page 1 = 20, page 2 = 5
	matches := make([]domain.CommonMatchRaw, 25)
	for i := range matches {
		matches[i] = domain.CommonMatchRaw{
			MatchID:       fmt.Sprintf("m%d", i),
			StartTime:     now,
			Player1TeamID: &tid1,
			Player2TeamID: &tid2,
		}
	}
	repo := &mockExplorerRepo{xuid: "other", matches: matches}
	svc := NewExplorerService(repo, "my-xuid")

	p1, err := svc.GetCommonMatches(context.Background(), "Player", 1)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if len(p1.CommonMatches) != 20 {
		t.Errorf("page 1 items = %d, want 20", len(p1.CommonMatches))
	}
	if p1.TotalCount != 25 {
		t.Errorf("TotalCount = %d, want 25", p1.TotalCount)
	}
	if p1.Page != 1 {
		t.Errorf("Page = %d, want 1", p1.Page)
	}

	p2, err := svc.GetCommonMatches(context.Background(), "Player", 2)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}
	if len(p2.CommonMatches) != 5 {
		t.Errorf("page 2 items = %d, want 5", len(p2.CommonMatches))
	}
}

// TestExplorerService_GetCommonMatches_AllyPlusBadge vérifie que le badge ally_plus
// est émis quand le win rate dépasse le seuil.
func TestExplorerService_GetCommonMatches_AllyPlusBadge(t *testing.T) {
	now := time.Now()
	tid := 0
	// 4 matchs en équipe, 3 victoires → winrate 0.75 > 0.70
	matches := []domain.CommonMatchRaw{
		{MatchID: "a1", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
		{MatchID: "a2", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
		{MatchID: "a3", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
		{MatchID: "a4", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 3},
	}
	repo := &mockExplorerRepo{xuid: "ally-xuid", matches: matches}
	svc := NewExplorerService(repo, "my-xuid")

	resp, err := svc.GetCommonMatches(context.Background(), "AllyPlayer", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var hasAllyPlus bool
	for _, b := range resp.Badges {
		if b.Kind == "ally_plus" {
			hasAllyPlus = true
		}
	}
	if !hasAllyPlus {
		t.Error("badge ally_plus attendu (winrate 75% > 70%)")
	}
}

// --- mocks pour les target_profile providers ---

// mockLocalIdentityResolver simule la résolution d'identité locale : renvoie
// `identity` (nil = cible non suivie). Aucun fetch live.
type mockLocalIdentityResolver struct {
	identity *domain.HomeSpartanIdentityRow
	called   bool
}

func (m *mockLocalIdentityResolver) LocalSpartanIdentity(_ context.Context, _ string) *domain.HomeSpartanIdentityRow {
	m.called = true
	return m.identity
}

type mockRemoteStatsProvider struct {
	stats  *domain.NormalizedPlayerStats
	err    error
	called bool
}

func (m *mockRemoteStatsProvider) FetchRemoteStats(_ context.Context, _, _ string) (*domain.NormalizedPlayerStats, error) {
	m.called = true
	return m.stats, m.err
}

// ctxAuth construit un ctx avec ou sans tokens Halo.
func ctxAuth(hasAuth bool, userXUID string) context.Context {
	ctx := context.Background()
	if hasAuth {
		return ctxkeys.WithHaloAuth(ctx, &domain.HaloTokens{SpartanToken: "spartan-tok"}, userXUID)
	}
	return ctxkeys.WithHaloAuth(ctx, nil, userXUID)
}

// TestExplorerService_TargetProfile_LocalTargetAllSources couvre le cas heureux
// d'une cible LOCALE avec tokens : identité (locale) + carrière (live) + sample
// remplis. Aucune privacy (supprimée).
func TestExplorerService_TargetProfile_LocalTargetAllSources(t *testing.T) {
	now := time.Now()
	tid := 0
	matches := []domain.CommonMatchRaw{
		{MatchID: "m1", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
		{MatchID: "m2", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 3},
	}
	repo := &mockExplorerRepo{
		xuid:    "target-xuid",
		matches: matches,
		participants: &domain.ParticipantStatsAggregate{
			Kills: 20, Deaths: 10, Assists: 8,
			Wins: 1, Losses: 1, Draws: 0,
			ShotsFired: 200, ShotsHit: 100,
			DamageDealt: 4500, DamageTaken: 3000,
		},
		medals: &domain.MedalCountsAggregate{Total: 42, Unique: 8},
	}
	idRes := &mockLocalIdentityResolver{
		identity: &domain.HomeSpartanIdentityRow{RankNumber: 76, CurrentXP: 47820},
	}
	remoteProv := &mockRemoteStatsProvider{
		stats: &domain.NormalizedPlayerStats{KDA: 1.5, WinRate: 0.6, Accuracy: 0.45},
	}

	svc := NewExplorerService(repo, "my-xuid").
		WithTargetProfileProviders(idRes, remoteProv, "halo_infinite")

	resp, err := svc.GetCommonMatches(ctxAuth(true, "my-xuid"), "TargetPlayer", 1)
	if err != nil {
		t.Fatalf("GetCommonMatches: %v", err)
	}
	tp := resp.TargetProfile
	if tp == nil {
		t.Fatal("TargetProfile attendu non-nil")
	}
	if !tp.AuthAvailable {
		t.Error("AuthAvailable doit être true avec tokens")
	}
	if tp.Identity == nil || tp.Identity.RankNumber != 76 {
		t.Errorf("Identity attendue rank=76, got %+v", tp.Identity)
	}
	if !idRes.called {
		t.Error("LocalSpartanIdentity devait être appelé")
	}
	if tp.CareerStats == nil || tp.CareerStats.KDA != 1.5 {
		t.Errorf("CareerStats attendues KDA=1.5, got %+v", tp.CareerStats)
	}
	if tp.SampleStats == nil || tp.SampleStats.SampleSize != 2 {
		t.Errorf("SampleStats attendues SampleSize=2, got %+v", tp.SampleStats)
	}
	if tp.PrivacyWarning != nil {
		t.Errorf("PrivacyWarning doit toujours être nil (privacy supprimée), got %+v", tp.PrivacyWarning)
	}
}

// TestExplorerService_TargetProfile_NoTokens : sans tokens, l'identité LOCALE
// reste résolue (indépendante de l'auth — c'est la logique inversée), seule la
// carrière (live) est skip. Sample présent, privacy toujours nil.
func TestExplorerService_TargetProfile_NoTokens(t *testing.T) {
	now := time.Now()
	tid := 0
	matches := []domain.CommonMatchRaw{
		{MatchID: "m1", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
	}
	repo := &mockExplorerRepo{
		xuid:    "target-xuid",
		matches: matches,
		participants: &domain.ParticipantStatsAggregate{
			Kills: 5, Deaths: 3,
		},
	}
	idRes := &mockLocalIdentityResolver{
		identity: &domain.HomeSpartanIdentityRow{RankNumber: 12},
	}
	remoteProv := &mockRemoteStatsProvider{
		stats: &domain.NormalizedPlayerStats{KDA: 1.5},
	}

	svc := NewExplorerService(repo, "my-xuid").
		WithTargetProfileProviders(idRes, remoteProv, "halo_infinite")

	resp, err := svc.GetCommonMatches(ctxAuth(false, "my-xuid"), "TargetPlayer", 1)
	if err != nil {
		t.Fatalf("GetCommonMatches: %v", err)
	}
	tp := resp.TargetProfile
	if tp == nil {
		t.Fatal("TargetProfile attendu non-nil")
	}
	if tp.AuthAvailable {
		t.Error("AuthAvailable doit être false sans tokens")
	}
	// Identité local-first : résolue même sans tokens.
	if !idRes.called || tp.Identity == nil || tp.Identity.RankNumber != 12 {
		t.Errorf("Identity locale attendue rank=12 même sans tokens, got %+v", tp.Identity)
	}
	if remoteProv.called {
		t.Error("FetchRemoteStats (carrière live) NE doit PAS être appelé sans tokens")
	}
	if tp.CareerStats != nil {
		t.Errorf("CareerStats attendues nil sans tokens, got %+v", tp.CareerStats)
	}
	if tp.PrivacyWarning != nil {
		t.Errorf("PrivacyWarning toujours nil, got %+v", tp.PrivacyWarning)
	}
	if tp.SampleStats == nil || tp.SampleStats.SampleSize != 1 {
		t.Errorf("SampleStats attendues SampleSize=1, got %+v", tp.SampleStats)
	}
}

// TestExplorerService_TargetProfile_CareerFetchError : si le fetch carrière
// échoue, la carrière reste nil mais l'identité (locale) + sample restent
// affichés. Pas de privacy.
func TestExplorerService_TargetProfile_CareerFetchError(t *testing.T) {
	repo := &mockExplorerRepo{xuid: "target", matches: nil}
	idRes := &mockLocalIdentityResolver{
		identity: &domain.HomeSpartanIdentityRow{RankNumber: 50},
	}
	remoteProv := &mockRemoteStatsProvider{
		stats: nil,
		err:   errors.New("waypoint 404"),
	}

	svc := NewExplorerService(repo, "my-xuid").
		WithTargetProfileProviders(idRes, remoteProv, "halo_infinite")

	resp, err := svc.GetCommonMatches(ctxAuth(true, "my-xuid"), "SomePlayer", 1)
	if err != nil {
		t.Fatalf("GetCommonMatches: %v", err)
	}
	tp := resp.TargetProfile
	if tp == nil {
		t.Fatal("TargetProfile attendu non-nil")
	}
	if tp.Identity == nil || tp.Identity.RankNumber != 50 {
		t.Errorf("Identity locale doit rester affichée, got %+v", tp.Identity)
	}
	if tp.CareerStats != nil {
		t.Errorf("CareerStats attendues nil quand FetchRemoteStats échoue, got %+v", tp.CareerStats)
	}
	if tp.PrivacyWarning != nil {
		t.Errorf("PrivacyWarning toujours nil, got %+v", tp.PrivacyWarning)
	}
}

// TestExplorerService_TargetProfile_NoProviders couvre le cas où aucune
// dépendance n'est wirée : TargetProfile reste non-nil mais seules les données
// locales (sample) sont actives.
func TestExplorerService_TargetProfile_NoProviders(t *testing.T) {
	now := time.Now()
	tid := 0
	matches := []domain.CommonMatchRaw{
		{MatchID: "m1", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
	}
	repo := &mockExplorerRepo{
		xuid:    "target",
		matches: matches,
		participants: &domain.ParticipantStatsAggregate{
			Kills: 7, Deaths: 4,
		},
	}
	svc := NewExplorerService(repo, "my-xuid")

	resp, err := svc.GetCommonMatches(ctxAuth(true, "my-xuid"), "Player", 1)
	if err != nil {
		t.Fatalf("GetCommonMatches: %v", err)
	}
	tp := resp.TargetProfile
	if tp == nil {
		t.Fatal("TargetProfile attendu non-nil même sans providers (transport AuthAvailable)")
	}
	if tp.Identity != nil || tp.CareerStats != nil || tp.PrivacyWarning != nil {
		t.Errorf("identity/career/privacy attendus nil sans providers, got %+v", tp)
	}
	if tp.SampleStats == nil || tp.SampleStats.Kills != 7 {
		t.Errorf("SampleStats attendues kills=7 sans providers, got %+v", tp.SampleStats)
	}
}
