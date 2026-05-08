package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// --- mock ---

type mockMatchViewRepo struct {
	meta      *domain.MatchMetaRaw
	metaErr   error
	stats     *domain.PlayerMatchStatsRaw
	statsErr  error
	enrich    *domain.MatchEnrichmentRaw
	enrichErr error
	board     []domain.ScoreboardRaw
	boardErr  error
	medals    []domain.MedalRaw
	medalsErr error
	events    []domain.EventRaw
	eventsErr error
	weapons   []domain.WeaponKillRaw
	weaponErr error
	kvPairs   []domain.KVPairRaw
	kvErr     error
}

func (m *mockMatchViewRepo) GetMatchMeta(_ context.Context, _ string) (*domain.MatchMetaRaw, error) {
	return m.meta, m.metaErr
}
func (m *mockMatchViewRepo) GetPlayerMatchStats(_ context.Context, _, _ string) (*domain.PlayerMatchStatsRaw, error) {
	return m.stats, m.statsErr
}
func (m *mockMatchViewRepo) GetMatchEnrichment(_ context.Context, _ string) (*domain.MatchEnrichmentRaw, error) {
	return m.enrich, m.enrichErr
}
func (m *mockMatchViewRepo) GetMatchScoreboard(_ context.Context, _ string) ([]domain.ScoreboardRaw, error) {
	return m.board, m.boardErr
}
func (m *mockMatchViewRepo) GetMatchObjectiveScore(_ context.Context, _, _ string) (int, error) {
	return 0, nil
}
func (m *mockMatchViewRepo) GetMatchMedals(_ context.Context, _, _ string) ([]domain.MedalRaw, error) {
	return m.medals, m.medalsErr
}
func (m *mockMatchViewRepo) GetMatchEvents(_ context.Context, _ string) ([]domain.EventRaw, error) {
	return m.events, m.eventsErr
}
func (m *mockMatchViewRepo) GetMatchWeaponKills(_ context.Context, _, _ string) ([]domain.WeaponKillRaw, error) {
	return m.weapons, m.weaponErr
}
func (m *mockMatchViewRepo) GetMatchKVPairs(_ context.Context, _ string) ([]domain.KVPairRaw, error) {
	return m.kvPairs, m.kvErr
}
func (m *mockMatchViewRepo) GetMatchNeighbors(_ context.Context, _, _ string) (*domain.MatchNeighbors, error) {
	return nil, nil
}
func (m *mockMatchViewRepo) GetMatchNeighborsFiltered(_ context.Context, _, _ string, _ *domain.MatchFilterSpec) (*domain.MatchNeighbors, error) {
	return nil, nil
}
func (m *mockMatchViewRepo) GetMatchEncounters(_ context.Context, _, _ string) ([]domain.EncounterRaw, error) {
	return nil, nil
}
func (m *mockMatchViewRepo) GetMatchEncounterStats(_ context.Context, _, _ string) ([]domain.EncounterStatsRaw, error) {
	return nil, nil
}
func (m *mockMatchViewRepo) GetMatchSkillRank(_ context.Context, _ string) (*domain.SkillRankRaw, error) {
	return nil, nil
}
func (m *mockMatchViewRepo) GetMatchMedia(_ context.Context, _ string) ([]domain.MediaAssocRaw, error) {
	return nil, nil
}
func (m *mockMatchViewRepo) GetMatchExpectedStats(_ context.Context, _, _ string) (*domain.ExpectedStatsRaw, error) {
	return nil, nil
}
func (m *mockMatchViewRepo) GetMatchBulkMedals(_ context.Context, _ string) ([]domain.BulkMedalRaw, error) {
	return nil, nil
}
func (m *mockMatchViewRepo) GetMatchBulkWeaponKills(_ context.Context, _ string) ([]domain.BulkWeaponKillRaw, error) {
	return nil, nil
}
func (m *mockMatchViewRepo) GetHistoryForAvg(_ context.Context, _ string) ([]domain.MatchHistAvgRow, error) {
	return nil, nil
}
func (m *mockMatchViewRepo) GetPlayerAssistsModel(_ context.Context, _ string) (*domain.PlayerAssistsModel, error) {
	return nil, nil
}

// --- tests ---

func TestMatchViewService_GetMatchView_OK(t *testing.T) {
	now := time.Now()
	mapN, pairN, plN := aquariusMap, slayerMode, sessionCategoryRanked
	dur := 600.0
	repo := &mockMatchViewRepo{
		meta: &domain.MatchMetaRaw{
			MatchID:         "m1",
			StartTime:       &now,
			MapName:         &mapN,
			PairName:        &pairN,
			PlaylistName:    &plN,
			DurationSeconds: &dur,
		},
		stats: &domain.PlayerMatchStatsRaw{
			OutcomeCode: 2, Kills: 15, Deaths: 5, Assists: 3,
		},
		enrich: &domain.MatchEnrichmentRaw{IsWithFriends: true},
		board: []domain.ScoreboardRaw{
			{XUID: "xuid1", Gamertag: "Player1", Kills: 15, Deaths: 5, Assists: 3},
		},
		medals:  []domain.MedalRaw{{MedalID: 1, Count: 2, Label: "Double Kill"}},
		events:  []domain.EventRaw{{EventType: "kill"}},
		weapons: []domain.WeaponKillRaw{{WeaponID: 100, WeaponLabel: "BR75", Kills: 8}},
		kvPairs: []domain.KVPairRaw{{KillerXUID: "x1", VictimXUID: "x2", KillCount: 3}},
	}
	svc := NewMatchViewService(repo, "xuid1")

	resp, err := svc.GetMatchView(context.Background(), "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Header.MatchID != "m1" {
		t.Errorf("MatchID = %q, want m1", resp.Header.MatchID)
	}
}

func TestMatchViewService_GetMatchView_MetaError(t *testing.T) {
	repo := &mockMatchViewRepo{metaErr: errors.New("not found")}
	svc := NewMatchViewService(repo, "xuid1")

	_, err := svc.GetMatchView(context.Background(), "m1")
	if err == nil {
		t.Error("expected error when meta fails")
	}
}

func TestMatchViewService_GetMatchView_GracefulDegradation(t *testing.T) {
	now := time.Now()
	repo := &mockMatchViewRepo{
		meta:      &domain.MatchMetaRaw{MatchID: "m1", StartTime: &now},
		statsErr:  errors.New("no stats"),
		enrichErr: errors.New("no enrich"),
		boardErr:  errors.New("no board"),
		medalsErr: errors.New("no medals"),
		eventsErr: errors.New("no events"),
		weaponErr: errors.New("no weapons"),
		kvErr:     errors.New("no kv"),
	}
	svc := NewMatchViewService(repo, "xuid1")

	resp, err := svc.GetMatchView(context.Background(), "m1")
	if err != nil {
		t.Fatalf("expected graceful degradation, got error: %v", err)
	}
	if resp.Header.MatchID != "m1" {
		t.Errorf("MatchID = %q, want m1", resp.Header.MatchID)
	}
}

func TestMatchViewService_GetMatchView_NilMeta(t *testing.T) {
	repo := &mockMatchViewRepo{meta: nil}
	svc := NewMatchViewService(repo, "xuid1")

	// nil meta (no error) should still return a response
	resp, err := svc.GetMatchView(context.Background(), "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = resp
}

// --- Tests Phase 2b : GetMatchNeighborsFiltered ---

// mockNeighborsRepo : capture le spec passé pour vérifier la propagation
// jusqu'au repo. Retourne des neighbors fixes.
type mockNeighborsRepo struct {
	mockMatchViewRepo
	receivedSpec  *domain.MatchFilterSpec
	receivedXUID  string
	receivedMatch string
	returnNil     bool
}

func (m *mockNeighborsRepo) GetMatchNeighborsFiltered(_ context.Context, xuid, matchID string, spec *domain.MatchFilterSpec) (*domain.MatchNeighbors, error) {
	m.receivedSpec = spec
	m.receivedXUID = xuid
	m.receivedMatch = matchID
	if m.returnNil {
		return nil, nil
	}
	prev := "p"
	next := "n"
	return &domain.MatchNeighbors{
		PreviousMatchID: &prev,
		NextMatchID:     &next,
		CurrentIndex:    1,
		TotalMatches:    3,
	}, nil
}

func TestMatchViewService_GetMatchNeighborsFiltered_PropagateSpec(t *testing.T) {
	repo := &mockNeighborsRepo{}
	svc := NewMatchViewService(repo, "xuid-me")

	pl := "Ranked Arena"
	out := "win"
	spec := &domain.MatchFilterSpec{PlaylistName: &pl, Outcome: &out}
	resp, err := svc.GetMatchNeighborsFiltered(context.Background(), "m1", spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.receivedXUID != "xuid-me" {
		t.Errorf("xuid = %q, want xuid-me", repo.receivedXUID)
	}
	if repo.receivedMatch != "m1" {
		t.Errorf("match = %q, want m1", repo.receivedMatch)
	}
	if repo.receivedSpec == nil || repo.receivedSpec.PlaylistName == nil || *repo.receivedSpec.PlaylistName != "Ranked Arena" {
		t.Errorf("spec not propagated correctly : %+v", repo.receivedSpec)
	}
	// AppliedFilters echo doit être présent quand spec non vide
	if resp.AppliedFilters == nil || resp.AppliedFilters.PlaylistName == nil {
		t.Errorf("AppliedFilters echo missing : %+v", resp.AppliedFilters)
	}
	if resp.CurrentIndex != 1 || resp.TotalMatches != 3 {
		t.Errorf("neighbors mal renvoyés : %+v", resp)
	}
}

func TestMatchViewService_GetMatchNeighborsFiltered_EmptySpec_NoAppliedFilters(t *testing.T) {
	repo := &mockNeighborsRepo{}
	svc := NewMatchViewService(repo, "xuid-me")

	resp, err := svc.GetMatchNeighborsFiltered(context.Background(), "m1", &domain.MatchFilterSpec{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AppliedFilters != nil {
		t.Errorf("AppliedFilters doit être nil pour spec vide, got %+v", resp.AppliedFilters)
	}
}

func TestMatchViewService_GetMatchNeighborsFiltered_NilSpec(t *testing.T) {
	repo := &mockNeighborsRepo{}
	svc := NewMatchViewService(repo, "xuid-me")

	_, err := svc.GetMatchNeighborsFiltered(context.Background(), "m1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Le spec nil est passé tel quel au repo (qui doit le déléguer à GetMatchNeighbors)
	if repo.receivedSpec != nil {
		t.Errorf("spec nil doit être propagé tel quel, got %+v", repo.receivedSpec)
	}
}

func TestMatchViewService_GetMatchNeighborsFiltered_NilResult(t *testing.T) {
	repo := &mockNeighborsRepo{returnNil: true}
	svc := NewMatchViewService(repo, "xuid-me")

	resp, err := svc.GetMatchNeighborsFiltered(context.Background(), "m1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.PreviousMatchID != nil || resp.NextMatchID != nil {
		t.Errorf("résultat nil doit retourner MatchNeighbors zéro, got %+v", resp)
	}
}
