package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// fakeSquadLoader mock de SquadV2Loader pour les tests.
type fakeSquadLoader struct {
	mu         sync.Mutex
	rowsByGT   map[string][]canonical.PlayerMatchRow
	errByGT    map[string]error
	calls      []string // gamertags appeles dans l'ordre
	delayPerGT time.Duration
}

func (f *fakeSquadLoader) LoadFor(
	_ context.Context,
	_, gamertag string,
	_ port.PlayerMatchFilters,
) ([]canonical.PlayerMatchRow, error) {
	f.mu.Lock()
	f.calls = append(f.calls, gamertag)
	f.mu.Unlock()
	if f.delayPerGT > 0 {
		time.Sleep(f.delayPerGT)
	}
	if err, ok := f.errByGT[gamertag]; ok {
		return nil, err
	}
	return f.rowsByGT[gamertag], nil
}

// row construit une PlayerMatchRow minimale avec match_id + start_time.
func row(matchID string, startedAt time.Time, outcome canonical.Outcome) canonical.PlayerMatchRow {
	return canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			MatchID:      matchID,
			StartedAtUTC: startedAt,
			Outcome:      outcome,
			Map: &canonical.AssetReference{
				Kind:         "map",
				ID:           "bazaar",
				DefaultLabel: "Bazaar",
			},
		},
		Self: canonical.MatchParticipant{
			Outcome: outcome,
		},
	}
}

func TestSquadServiceV2_GetSquadPage_OneTeammateIntersection(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main": {
				row("m1", t0, canonical.OutcomeWin),
				row("m2", t0.Add(time.Hour), canonical.OutcomeLoss),
				row("m3", t0.Add(2*time.Hour), canonical.OutcomeWin),
			},
			"friend1": {
				row("m1", t0, canonical.OutcomeWin),
				row("m3", t0.Add(2*time.Hour), canonical.OutcomeWin),
				row("m_solo", t0.Add(3*time.Hour), canonical.OutcomeWin),
			},
		},
	}
	svc := NewSquadServiceV2(loader)
	resp, err := svc.GetSquadPage(context.Background(), "halo_infinite", "main",
		[]string{"friend1"}, temporal.Period1Y)
	if err != nil {
		t.Fatalf("GetSquadPage: %v", err)
	}
	// m1 et m3 sont communs, m2 et m_solo non.
	if resp.SharedMatchesCount != 2 {
		t.Errorf("want 2 shared matches, got %d", resp.SharedMatchesCount)
	}
	// Ordre DESC : m3 d'abord
	if resp.SharedMatches[0].MatchID != "m3" {
		t.Errorf("first shared match should be m3 (most recent), got %s",
			resp.SharedMatches[0].MatchID)
	}
	// Players hydraté pour les 2 joueurs
	for _, sm := range resp.SharedMatches {
		if _, ok := sm.Players["main"]; !ok {
			t.Errorf("match %s missing main player", sm.MatchID)
		}
		if _, ok := sm.Players["friend1"]; !ok {
			t.Errorf("match %s missing friend1", sm.MatchID)
		}
	}
}

func TestSquadServiceV2_GetSquadPage_ThreeTeammates(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main": {row("m1", t0, canonical.OutcomeWin), row("m2", t0.Add(time.Hour), canonical.OutcomeWin)},
			"f1":   {row("m1", t0, canonical.OutcomeWin), row("m2", t0.Add(time.Hour), canonical.OutcomeWin)},
			"f2":   {row("m1", t0, canonical.OutcomeWin), row("m2", t0.Add(time.Hour), canonical.OutcomeWin)},
			"f3":   {row("m1", t0, canonical.OutcomeWin)}, // m2 absent chez f3
		},
	}
	svc := NewSquadServiceV2(loader)
	resp, err := svc.GetSquadPage(context.Background(), "halo_infinite", "main",
		[]string{"f1", "f2", "f3"}, temporal.PeriodAll)
	if err != nil {
		t.Fatalf("GetSquadPage: %v", err)
	}
	// Seul m1 est commun aux 4 joueurs.
	if resp.SharedMatchesCount != 1 {
		t.Errorf("want 1 shared match (m1), got %d", resp.SharedMatchesCount)
	}
	if len(resp.SharedMatches[0].Players) != 4 {
		t.Errorf("want 4 players in m1, got %d", len(resp.SharedMatches[0].Players))
	}
}

func TestSquadServiceV2_GetSquadPage_NoIntersection(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main":    {row("m1", t0, canonical.OutcomeWin)},
			"friend1": {row("m2", t0, canonical.OutcomeWin)}, // pas de match commun
		},
	}
	svc := NewSquadServiceV2(loader)
	resp, err := svc.GetSquadPage(context.Background(), "halo_infinite", "main",
		[]string{"friend1"}, temporal.PeriodAll)
	if err != nil {
		t.Fatalf("GetSquadPage: %v", err)
	}
	if resp.SharedMatchesCount != 0 {
		t.Errorf("want 0 shared matches, got %d", resp.SharedMatchesCount)
	}
}

func TestSquadServiceV2_GetSquadPage_TeammateCapabilityMissing(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main": {row("m1", t0, canonical.OutcomeWin)},
		},
		errByGT: map[string]error{
			"friend1": games.ErrCapabilityNotSupported,
		},
	}
	svc := NewSquadServiceV2(loader)
	resp, err := svc.GetSquadPage(context.Background(), "halo_infinite", "main",
		[]string{"friend1"}, temporal.PeriodAll)
	if err != nil {
		t.Fatalf("GetSquadPage should not error on teammate capability missing: %v", err)
	}
	// CapabilityGap signalé pour friend1
	if len(resp.Capabilities) != 1 {
		t.Fatalf("want 1 capability gap, got %d", len(resp.Capabilities))
	}
	if resp.Capabilities[0].CapabilityKey != string(games.CapMatchHistory) {
		t.Errorf("want CapabilityKey=match.history, got %s", resp.Capabilities[0].CapabilityKey)
	}
	// L'intersection avec friend1 absent = matchs du main seul (pas d'intersect avec un set vide)
	// Note : friend1 est exclu de perPlayer, donc l'intersection ne porte que sur main.
	// m1 est present chez main -> 1 match dans SharedMatches.
	if resp.SharedMatchesCount != 1 {
		t.Errorf("with friend1 excluded, should keep main matches only, got %d",
			resp.SharedMatchesCount)
	}
}

func TestSquadServiceV2_GetSquadPage_MainPlayerCapabilityMissing(t *testing.T) {
	t.Parallel()
	loader := &fakeSquadLoader{
		errByGT: map[string]error{
			"main": games.ErrCapabilityNotSupported,
		},
	}
	svc := NewSquadServiceV2(loader)
	resp, err := svc.GetSquadPage(context.Background(), "halo_infinite", "main",
		[]string{"friend1"}, temporal.PeriodAll)
	if err != nil {
		t.Fatalf("expected nil error (gap signaled, not panic): %v", err)
	}
	// SharedMatches vide
	if resp.SharedMatchesCount != 0 {
		t.Errorf("main capability missing -> should have 0 shared matches, got %d",
			resp.SharedMatchesCount)
	}
	if len(resp.Capabilities) == 0 {
		t.Error("should signal CapabilityGap for main player")
	}
}

func TestSquadServiceV2_GetSquadPage_RejectsInvalidInput(t *testing.T) {
	t.Parallel()
	svc := NewSquadServiceV2(&fakeSquadLoader{})

	if _, err := svc.GetSquadPage(context.Background(), "halo_infinite", "",
		nil, temporal.PeriodAll); err == nil {
		t.Error("empty mainGT should error")
	}
	if _, err := svc.GetSquadPage(context.Background(), "halo_infinite", "main",
		[]string{"f1", "f2", "f3", "f4"}, temporal.PeriodAll); err == nil {
		t.Error(">3 teammates should error")
	}
}

func TestSquadServiceV2_GetSquadPage_LoaderRunsInParallel(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main": {row("m1", t0, canonical.OutcomeWin)},
			"f1":   {row("m1", t0, canonical.OutcomeWin)},
			"f2":   {row("m1", t0, canonical.OutcomeWin)},
		},
		delayPerGT: 50 * time.Millisecond,
	}
	svc := NewSquadServiceV2(loader)
	start := time.Now()
	if _, err := svc.GetSquadPage(context.Background(), "halo_infinite", "main",
		[]string{"f1", "f2"}, temporal.PeriodAll); err != nil {
		t.Fatalf("GetSquadPage: %v", err)
	}
	// 3 joueurs × 50ms en sequentiel = 150ms. En parallele : ~50ms.
	// On laisse une marge confortable pour CI lente.
	elapsed := time.Since(start)
	if elapsed >= 130*time.Millisecond {
		t.Errorf("loader appeared to run sequentially (%v >= 130ms)", elapsed)
	}
}

func TestSquadServiceV2_GetSquadPage_LoaderError_NotCapability(t *testing.T) {
	t.Parallel()
	loader := &fakeSquadLoader{
		errByGT: map[string]error{
			"f1": errors.New("disk full"),
		},
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main": {row("m1", time.Now(), canonical.OutcomeWin)},
		},
	}
	svc := NewSquadServiceV2(loader)
	_, err := svc.GetSquadPage(context.Background(), "halo_infinite", "main",
		[]string{"f1"}, temporal.PeriodAll)
	if err == nil {
		t.Error("non-capability errors should propagate, got nil")
	}
}

func TestSquadServiceV2_GetSquadPage_OrderDeterministicOnEqualTimes(t *testing.T) {
	t.Parallel()
	// Deux matchs avec exactement la meme StartedAt -> tri par MatchID alphabetique.
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main": {row("m_b", t0, canonical.OutcomeWin), row("m_a", t0, canonical.OutcomeWin)},
			"f1":   {row("m_b", t0, canonical.OutcomeWin), row("m_a", t0, canonical.OutcomeWin)},
		},
	}
	svc := NewSquadServiceV2(loader)
	resp, err := svc.GetSquadPage(context.Background(), "halo_infinite", "main",
		[]string{"f1"}, temporal.PeriodAll)
	if err != nil {
		t.Fatalf("GetSquadPage: %v", err)
	}
	if resp.SharedMatches[0].MatchID != "m_a" {
		t.Errorf("equal times: alphabetical fallback expected, got first=%s",
			resp.SharedMatches[0].MatchID)
	}
}
