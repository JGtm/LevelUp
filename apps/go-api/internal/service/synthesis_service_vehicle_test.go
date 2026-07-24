// Package service — synthesis_service_vehicle_test.go : branche PAR TITRE de la
// source des compteurs « véhicules détruits » / « vol à la tire » (I1). Infinite =
// personal_score_awards (inchangé) ; Halo 5 = commendations natives (repo dédié qui
// PRIME sur les awards, vides pour ce titre). Best-effort si le repo H5 échoue.
package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

type mockAwardsRepo struct {
	rows []port.PersonalScoreAwardRow
	err  error
}

func (m *mockAwardsRepo) LoadPersonalScoreAwards(_ context.Context, _ string, _ port.PersonalScoreAwardsFilters) ([]port.PersonalScoreAwardRow, error) {
	return m.rows, m.err
}

type mockVehicleRepo struct {
	stats port.VehicleDestructionStats
	err   error
	calls int
}

func (m *mockVehicleRepo) LoadVehicleDestructionStats(_ context.Context, _ string, _ []string, _ string) (port.VehicleDestructionStats, error) {
	m.calls++
	return m.stats, m.err
}

func oneCanonRow(matchID string) []canonical.PlayerMatchRow {
	return []canonical.PlayerMatchRow{{Summary: canonical.MatchSummary{MatchID: matchID}}}
}

// Infinite : pas de repo véhicule → les compteurs viennent des awards (inchangé).
func TestApplyFunStats_Infinite_AwardsOnly(t *testing.T) {
	awards := &mockAwardsRepo{rows: []port.PersonalScoreAwardRow{
		{AwardName: "destroyed_wraith", Total: 3},
		{AwardName: "hijacked_banshee", Total: 2},
		{AwardName: "betrayed_player", Total: 1},
	}}
	svc := NewSynthesisService(&mockSynthesisRepo{}).
		WithPlayerMatchesRepo(&mockSynthesisPlayerMatches{}, "halo_infinite", "GT").
		WithPersonalScoreAwardsRepo(awards, "xuidI")

	ds := &domain.SynthesisDetailedStats{}
	svc.applyFunStatsToDetailedStats(context.Background(), ds, oneCanonRow("m1"))

	if ds.TotalVehiclesDestroyed != 3 || ds.TotalHijacks != 2 || ds.TotalBetrayals != 1 {
		t.Fatalf("Infinite awards path: got vehicles=%d hijacks=%d betrayals=%d, want 3/2/1",
			ds.TotalVehiclesDestroyed, ds.TotalHijacks, ds.TotalBetrayals)
	}
}

// Halo 5 : awards vides (ErrCapabilityNotSupported) + repo véhicule → override.
func TestApplyFunStats_H5_VehicleRepoOverrides(t *testing.T) {
	awards := &mockAwardsRepo{err: errors.New("capability not supported")}
	vehicle := &mockVehicleRepo{stats: port.VehicleDestructionStats{VehiclesDestroyed: 7, Hijacks: 5}}
	svc := NewSynthesisService(&mockSynthesisRepo{}).
		WithPlayerMatchesRepo(&mockSynthesisPlayerMatches{}, "halo_5", "GTH5").
		WithPersonalScoreAwardsRepo(awards, "xuidH5").
		WithVehicleDestructionStatsRepo(vehicle)

	ds := &domain.SynthesisDetailedStats{}
	svc.applyFunStatsToDetailedStats(context.Background(), ds, oneCanonRow("m1"))

	if vehicle.calls != 1 {
		t.Fatalf("repo véhicule appelé %d fois, want 1", vehicle.calls)
	}
	if ds.TotalVehiclesDestroyed != 7 || ds.TotalHijacks != 5 {
		t.Fatalf("H5 override: got vehicles=%d hijacks=%d, want 7/5",
			ds.TotalVehiclesDestroyed, ds.TotalHijacks)
	}
}

// Best-effort : repo véhicule en erreur → on ne clobber PAS la valeur des awards.
func TestApplyFunStats_H5_VehicleRepoError_KeepsAwards(t *testing.T) {
	awards := &mockAwardsRepo{rows: []port.PersonalScoreAwardRow{{AwardName: "destroyed_ghost", Total: 4}}}
	vehicle := &mockVehicleRepo{err: errors.New("shared reader down")}
	svc := NewSynthesisService(&mockSynthesisRepo{}).
		WithPlayerMatchesRepo(&mockSynthesisPlayerMatches{}, "halo_5", "GTH5").
		WithPersonalScoreAwardsRepo(awards, "xuidH5").
		WithVehicleDestructionStatsRepo(vehicle)

	ds := &domain.SynthesisDetailedStats{}
	svc.applyFunStatsToDetailedStats(context.Background(), ds, oneCanonRow("m1"))

	if ds.TotalVehiclesDestroyed != 4 {
		t.Fatalf("erreur repo véhicule → attendu la valeur awards (4), got %d", ds.TotalVehiclesDestroyed)
	}
}
