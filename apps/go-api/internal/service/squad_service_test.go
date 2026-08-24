package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// --- mock ---

type mockSquadRepo struct {
	topRows   []domain.TopTeammateRow
	topErr    error
	squadRows []domain.SquadMatchRow
	// squadRowsByTeammate (optionnel) : matchs communs par XUID de coéquipier.
	// Permet de tester l'intersection multi-coéquipiers (chaque coéquipier a un set
	// distinct). Si nil, LoadSquadMatches retombe sur squadRows (rétro-compatible).
	squadRowsByTeammate map[string][]domain.SquadMatchRow
	squadErr            error
	tmRows              []domain.TeammateMatchRow
	tmErr               error
	impactRows          []domain.ImpactEventRow
	impactErr           error
	kvPairs             []domain.KVPairRaw
	kvErr               error
	heatmapRows         []domain.SynthesisHeatmapRow
	heatmapErr          error
	synthRows           []legacymatch.SynthesisMatchRow
	synthErr            error
	allyRows            []domain.AllyParticipant
	allyErr             error
	// LookupXUIDByGamertag : lookup attendu (gamertag normalisÃ© en lowercase â†’ xuid).
	// Si vide, retourne ("", false, nil) â€” comportement par dÃ©faut.
	lookupAliases map[string]string
	lookupErr     error
	// assetFR : traductions FR par type d'asset ("map"|"playlist"|"pair") →
	// (asset_id → libellé FR). nil → comportement historique (aucune trad).
	assetFR map[string]map[string]string
	// modeFR : mode_name_tr FR (mode EN normalisé → FR). nil → aucune trad.
	modeFR map[string]string
}

func (m *mockSquadRepo) LoadTopTeammates(_ context.Context, _ string) ([]domain.TopTeammateRow, error) {
	return m.topRows, m.topErr
}
func (m *mockSquadRepo) LookupXUIDByGamertag(_ context.Context, gamertag string) (string, bool, error) {
	if m.lookupErr != nil {
		return "", false, m.lookupErr
	}
	if x, ok := m.lookupAliases[strings.ToLower(gamertag)]; ok {
		return x, true, nil
	}
	return "", false, nil
}
func (m *mockSquadRepo) LoadSquadMatches(_ context.Context, _, teammateXUID string) ([]domain.SquadMatchRow, error) {
	if m.squadErr != nil {
		return nil, m.squadErr
	}
	if m.squadRowsByTeammate != nil {
		return m.squadRowsByTeammate[teammateXUID], nil
	}
	return m.squadRows, nil
}
func (m *mockSquadRepo) LoadTeammateMatches(_ context.Context, _, _ string) ([]domain.TeammateMatchRow, error) {
	return m.tmRows, m.tmErr
}
func (m *mockSquadRepo) LoadImpactEvents(_ context.Context, _ []string) ([]domain.ImpactEventRow, error) {
	return m.impactRows, m.impactErr
}
func (m *mockSquadRepo) LoadKVPairs(_ context.Context, _ []string) ([]domain.KVPairRaw, error) {
	return m.kvPairs, m.kvErr
}
func (m *mockSquadRepo) LoadSquadAssistPairs(_ context.Context, _, _ []string) ([]domain.SquadAssistPairRaw, int, error) {
	return nil, 0, nil
}
func (m *mockSquadRepo) LoadMainTeamParticipants(_ context.Context, _ string, _ []string) ([]domain.AllyParticipant, error) {
	return m.allyRows, m.allyErr
}
func (m *mockSquadRepo) LoadSynthesisHeatmap(_ context.Context, _ string) ([]domain.SynthesisHeatmapRow, error) {
	return m.heatmapRows, m.heatmapErr
}
func (m *mockSquadRepo) LoadAssetTranslationsFR(_ context.Context, assetType string, _ []string) (map[string]string, error) {
	if m.assetFR == nil {
		return nil, nil
	}
	return m.assetFR[assetType], nil
}
func (m *mockSquadRepo) LoadModeTranslationsFR(_ context.Context, _ []string) (map[string]string, error) {
	return m.modeFR, nil
}
func (m *mockSquadRepo) LoadMapStatsForSquad(_ context.Context, _ string, _, _ []string) (map[string]domain.MapSquadStats, error) {
	return nil, nil
}
func (m *mockSquadRepo) LoadSynthesisMatches(_ context.Context, _ string) ([]legacymatch.SynthesisMatchRow, error) {
	return m.synthRows, m.synthErr
}

// --- tests SquadService ---

func TestSquadService_GetSquadPage_NoTeammate(t *testing.T) {
	repo := &mockSquadRepo{
		topRows: []domain.TopTeammateRow{
			{XUID: "x1", Gamertag: "Ally", GamesTogether: 20, WinsTogether: 12, WinRate: 0.6},
		},
	}
	svc := NewSquadService(repo).WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test")

	resp, err := svc.GetSquadPage(context.Background(), "player-xuid", "PlayerGT", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.TopTeammates) != 1 {
		t.Errorf("TopTeammates = %d, want 1", len(resp.TopTeammates))
	}
}

func TestSquadService_GetSquadPage_WithTeammate(t *testing.T) {
	now := time.Now()
	repo := &mockSquadRepo{
		topRows: []domain.TopTeammateRow{
			{XUID: "tm1", Gamertag: "AllyGT", GamesTogether: 10, WinsTogether: 5, WinRate: 0.5},
		},
		squadRows: []domain.SquadMatchRow{
			{MatchID: "m1", StartTime: now, Kills: 10, Deaths: 5, Assists: 3, IsWithFriends: true, Outcome: 2, TimePlayedSecs: 600},
		},
		tmRows: []domain.TeammateMatchRow{
			{MatchID: "m1", StartTime: now, Kills: 8, Deaths: 6, Assists: 2, Outcome: 2, TimePlayedSecs: 600},
		},
	}
	svc := NewSquadService(repo).WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test")

	resp, err := svc.GetSquadPage(context.Background(), "player-xuid", "PlayerGT", "tm1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestSquadService_GetSquadPage_TopError(t *testing.T) {
	repo := &mockSquadRepo{topErr: errors.New("fail")}
	svc := NewSquadService(repo).WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test")

	_, err := svc.GetSquadPage(context.Background(), "x", "gt", "")
	if err == nil {
		t.Error("expected error")
	}
}

func TestSquadService_GetSquadPage_SquadMatchesError(t *testing.T) {
	repo := &mockSquadRepo{
		topRows:  []domain.TopTeammateRow{{XUID: "tm1", Gamertag: "Ally"}},
		squadErr: errors.New("fail"),
	}
	svc := NewSquadService(repo).WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test")

	_, err := svc.GetSquadPage(context.Background(), "x", "gt", "tm1")
	if err == nil {
		t.Error("expected error")
	}
}

func TestSquadService_GetSynthesisPage_OK(t *testing.T) {
	now := time.Now()
	repo := &mockSquadRepo{
		synthRows: []legacymatch.SynthesisMatchRow{
			{MatchID: "m1", StartTime: now, Outcome: 2, Kills: 10, Deaths: 5, IsWithFriends: true},
			{MatchID: "m2", StartTime: now, Outcome: 3, Kills: 5, Deaths: 10, IsWithFriends: false},
		},
	}
	svc := NewSquadService(repo).WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test")

	resp, err := svc.GetSynthesisPage(context.Background(), "player-xuid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.TotalMatches != 2 {
		t.Errorf("TotalMatches = %d, want 2", resp.TotalMatches)
	}
}

func TestSquadService_GetSynthesisPage_Error(t *testing.T) {
	repo := &mockSquadRepo{synthErr: errors.New("fail")}
	svc := NewSquadService(repo).WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test")

	_, err := svc.GetSynthesisPage(context.Background(), "xuid")
	if err == nil {
		t.Error("expected error")
	}
}

// (Les tests TeammatesService + safeDiv ont migré vers le sous-package teammates,
// K3b : cf. teammates/from_squad_service_test.go.)

func TestRound2(t *testing.T) {
	if round2(1.555) != 1.56 {
		t.Errorf("round2(1.555) = %f, want 1.56", round2(1.555))
	}
	if round2(1.0) != 1.0 {
		t.Errorf("round2(1.0) = %f, want 1.0", round2(1.0))
	}
}
