package service

import (
	"context"
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/domain"
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

// LoadHighlightEvents stub — retourne nil + ErrCapabilityNotSupported par
// defaut pour eviter les charts dependants (les tests existants ne les
// testent pas).
func (f *fakeSquadLoader) LoadHighlightEvents(
	_ context.Context,
	_ string,
	_ port.HighlightEventFilters,
) ([]canonical.HighlightEvent, error) {
	return nil, games.ErrCapabilityNotSupported
}

// LoadWeaponKills stub — meme pattern que LoadHighlightEvents.
func (f *fakeSquadLoader) LoadWeaponKills(
	_ context.Context,
	_ string,
	_ port.WeaponKillFilters,
) ([]port.WeaponKillRow, error) {
	return nil, games.ErrCapabilityNotSupported
}

// LoadKillMechanics stub — mécaniques natives h5 non simulées ici (nil = aucune).
func (f *fakeSquadLoader) LoadKillMechanics(
	_ context.Context,
	_ string,
	_ port.WeaponKillFilters,
) ([]port.KillMechanicsRow, error) {
	return nil, nil
}

// LoadMedals stub — meme pattern que LoadHighlightEvents.
func (f *fakeSquadLoader) LoadMedals(
	_ context.Context,
	_ string,
	_ port.MedalsByXUIDFilters,
) ([]port.MedalRow, error) {
	return nil, games.ErrCapabilityNotSupported
}

// LoadEmblemURLs stub — retourne nil (pas d'emblemes dans les tests unitaires).
func (f *fakeSquadLoader) LoadEmblemURLs(
	_ context.Context,
	_ string,
	_ []string,
) map[string]string {
	return nil
}

// LoadMapStatsForSquad stub — retourne nil par defaut (les tests existants
// ne dependent pas de l'historique squad). Un test dedie peut shadower.
func (f *fakeSquadLoader) LoadMapStatsForSquad(
	_ context.Context,
	_, _ string,
	_ []string,
) (map[string]domain.MapSquadStats, error) {
	return nil, nil
}

func (f *fakeSquadLoader) LoadPlayerAssistsModel(_ context.Context, _, _, _ string) (*domain.PlayerAssistsModel, error) {
	return nil, nil
}

func (f *fakeSquadLoader) LoadPopulationalAssistsCoef(_ context.Context, _, _ string) (float64, float64, bool, error) {
	return 0, 0, false, nil
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
		[]string{"friend1"}, temporal.Period1Y, nil, nil, nil, nil)
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
		[]string{"f1", "f2", "f3"}, temporal.PeriodAll, nil, nil, nil, nil)
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
		[]string{"friend1"}, temporal.PeriodAll, nil, nil, nil, nil)
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
		[]string{"friend1"}, temporal.PeriodAll, nil, nil, nil, nil)
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
		[]string{"friend1"}, temporal.PeriodAll, nil, nil, nil, nil)
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
		nil, temporal.PeriodAll, nil, nil, nil, nil); err == nil {
		t.Error("empty mainGT should error")
	}
	if _, err := svc.GetSquadPage(context.Background(), "halo_infinite", "main",
		[]string{"f1", "f2", "f3", "f4"}, temporal.PeriodAll, nil, nil, nil, nil); err == nil {
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
		[]string{"f1", "f2"}, temporal.PeriodAll, nil, nil, nil, nil); err != nil {
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
		[]string{"f1"}, temporal.PeriodAll, nil, nil, nil, nil)
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
		[]string{"f1"}, temporal.PeriodAll, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("GetSquadPage: %v", err)
	}
	if resp.SharedMatches[0].MatchID != "m_a" {
		t.Errorf("equal times: alphabetical fallback expected, got first=%s",
			resp.SharedMatches[0].MatchID)
	}
}

// rowWithStats construit une row avec stats (Kills/Deaths/Assists/Outcome/etc.)
// pour tester le Header.
func rowWithStats(matchID string, ts time.Time, outcome canonical.Outcome,
	kills, deaths, assists, timePlayed int, accuracy float64,
	performanceScore float64,
) canonical.PlayerMatchRow {
	k, d, a, tp := kills, deaths, assists, timePlayed
	acc := accuracy
	perf := performanceScore
	return canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			MatchID:      matchID,
			StartedAtUTC: ts,
			Outcome:      outcome,
		},
		Self: canonical.MatchParticipant{
			Outcome:    outcome,
			Kills:      &k,
			Deaths:     &d,
			Assists:    &a,
			TimePlayed: &tp,
			Accuracy:   &acc,
		},
		Enrichment: canonical.PlayerMatchEnrichment{
			PerformanceScore: &perf,
		},
	}
}

func TestSquadServiceV2_GetSquadPage_HeaderSoloKPIs(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main": {
				rowWithStats("m1", t0, canonical.OutcomeWin, 10, 5, 2, 600, 50, 75),
				rowWithStats("m2", t0.Add(time.Hour), canonical.OutcomeLoss, 5, 8, 1, 400, 45, 40),
			},
			"f1": {
				rowWithStats("m1", t0, canonical.OutcomeWin, 8, 4, 3, 600, 48, 70),
				rowWithStats("m2", t0.Add(time.Hour), canonical.OutcomeLoss, 6, 5, 2, 400, 52, 55),
			},
		},
	}
	svc := NewSquadServiceV2(loader)
	resp, err := svc.GetSquadPage(context.Background(), "halo_infinite", "main",
		[]string{"f1"}, temporal.PeriodAll, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("GetSquadPage: %v", err)
	}
	if resp.Header == nil {
		t.Fatal("Header should be filled when shared matches exist")
	}
	if resp.Header.SoloKPIs == nil {
		t.Fatal("SoloKPIs should be filled")
	}
	// Main player : 2 matchs, 10+5=15 kills total, 1000s play
	if resp.Header.SoloKPIs.MatchesCount != 2 {
		t.Errorf("SoloKPIs MatchesCount want 2, got %d", resp.Header.SoloKPIs.MatchesCount)
	}
	if resp.Header.SoloKPIs.Outcomes.Wins != 1 || resp.Header.SoloKPIs.Outcomes.Losses != 1 {
		t.Errorf("SoloKPIs outcomes: %+v", resp.Header.SoloKPIs.Outcomes)
	}
}

func TestSquadServiceV2_GetSquadPage_HeaderPlayerCardsAndScore(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main": {rowWithStats("m1", t0, canonical.OutcomeWin, 10, 5, 2, 600, 50, 80)},
			"f1":   {rowWithStats("m1", t0, canonical.OutcomeWin, 8, 4, 3, 600, 48, 70)},
			"f2":   {rowWithStats("m1", t0, canonical.OutcomeWin, 6, 6, 1, 600, 45, 60)},
		},
	}
	svc := NewSquadServiceV2(loader)
	resp, err := svc.GetSquadPage(context.Background(), "halo_infinite", "main",
		[]string{"f1", "f2"}, temporal.PeriodAll, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("GetSquadPage: %v", err)
	}
	if len(resp.Header.PlayerCards) != 3 {
		t.Fatalf("want 3 player cards, got %d", len(resp.Header.PlayerCards))
	}
	// Cards triees par gamertag asc -> f1, f2, main (ordre alphabetique)
	if resp.Header.PlayerCards[0].Gamertag != "f1" {
		t.Errorf("first card gamertag: want f1, got %s", resp.Header.PlayerCards[0].Gamertag)
	}
	// Score equipe : moyenne 80+70+60=70, +5 winrate (1.0 > 0.6), peut +5 minKD
	// (KD respectifs : 10/5=2.0, 8/4=2.0, 6/6=1.0 -> minKD=1.0 NON > 1.0, pas de bonus).
	if resp.Header.SquadScore == nil {
		t.Fatal("SquadScore should be filled")
	}
	if resp.Header.SquadScore.BaseAvg != 70 {
		t.Errorf("BaseAvg want 70, got %v", resp.Header.SquadScore.BaseAvg)
	}
	if resp.Header.SquadScore.BonusWinRate != 5 {
		t.Errorf("BonusWinRate want 5 (winrate=1.0), got %d", resp.Header.SquadScore.BonusWinRate)
	}
	// Comparison ▲▼ : main (80) > avg=70 -> above
	for _, c := range resp.Header.PlayerCards {
		if c.Gamertag == "main" && c.Comparison != "above" {
			t.Errorf("main (80 vs 70) should be 'above', got %s", c.Comparison)
		}
		if c.Gamertag == "f2" && c.Comparison != "below" {
			t.Errorf("f2 (60 vs 70) should be 'below', got %s", c.Comparison)
		}
	}
}

func TestSquadServiceV2_GetSquadPage_HeaderNilWhenNoShared(t *testing.T) {
	t.Parallel()
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main": {rowWithStats("m1", time.Now(), canonical.OutcomeWin, 5, 2, 0, 300, 50, 60)},
			"f1":   {rowWithStats("m2", time.Now(), canonical.OutcomeWin, 5, 2, 0, 300, 50, 60)},
		},
	}
	svc := NewSquadServiceV2(loader)
	resp, err := svc.GetSquadPage(context.Background(), "halo_infinite", "main",
		[]string{"f1"}, temporal.PeriodAll, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("GetSquadPage: %v", err)
	}
	// Pas de match commun -> SoloKPIs nil : sur la page Escouade, SoloKPIs
	// reflete le scope escouade (matchs partages). Sans intersection, le
	// briefing reste vide pour rester aligne avec le scope demande.
	if resp.Header.SoloKPIs != nil {
		t.Errorf("no shared matches -> SoloKPIs should be nil, got %+v", resp.Header.SoloKPIs)
	}
	if len(resp.Header.PlayerCards) != 0 {
		t.Errorf("no shared matches -> no player cards, got %d", len(resp.Header.PlayerCards))
	}
	if resp.Header.SquadScore != nil {
		t.Error("no shared matches -> no SquadScore")
	}
}

// ---------------------------------------------------------------------------
// Tests filterRowsByCascade + playlistLabelForFilter
// ---------------------------------------------------------------------------

// rowWithPlaylist construit une row dont Summary.Playlist est peuplé avec les
// labels EN et FR (miroir de assetReference() dans player_matches_repo.go).
func rowWithPlaylist(matchID, playlistEN, playlistFR string) canonical.PlayerMatchRow {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	r := row(matchID, t0, canonical.OutcomeWin)
	r.Summary.Playlist = &canonical.AssetReference{
		Kind:         "playlist",
		ID:           "pl-" + matchID,
		DefaultLabel: playlistEN,
		Labels:       map[string]string{"en": playlistEN, "fr": playlistFR},
	}
	return r
}

// rowWithExperience construit une row avec les flags IsRanked / IsPvE positionnés.
func rowWithExperience(matchID string, isRanked, isPvE bool) canonical.PlayerMatchRow {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	r := row(matchID, t0, canonical.OutcomeWin)
	r.Summary.IsRanked = &isRanked
	r.Summary.IsPvE = &isPvE
	return r
}

func TestFilterRowsByCascade_PlaylistFRLabel(t *testing.T) {
	t.Parallel()
	// filtersResolve retourne COALESCE(playlist_name_fr, playlist_name) = label FR.
	// filterRowsByCascade doit utiliser Labels["fr"] pour que la comparaison aboutisse.
	rows := []canonical.PlayerMatchRow{
		rowWithPlaylist("m1", "Quick Play", "Partie rapide"),
		rowWithPlaylist("m2", "Ranked Arena", "Arène classée"),
		rowWithPlaylist("m3", "Quick Play", "Partie rapide"),
	}

	got := filterRowsByCascade(rows, nil, []string{"Partie rapide"}, nil, nil)
	if len(got) != 2 {
		t.Fatalf("want 2 rows for 'Partie rapide', got %d", len(got))
	}
	for _, r := range got {
		if r.Summary.MatchID == "m2" {
			t.Error("m2 (Arène classée) should have been filtered out")
		}
	}
}

func TestFilterRowsByCascade_PlaylistENFallback(t *testing.T) {
	t.Parallel()
	// Quand playlist_name_fr est absent, COALESCE retourne l'anglais. Le label FR
	// stocké dans Labels["fr"] = label anglais = ce que filtersResolve envoie.
	rows := []canonical.PlayerMatchRow{
		rowWithPlaylist("m1", "Social Slayer", "Social Slayer"), // pas de traduction FR
		rowWithPlaylist("m2", "Big Team Battle", "Big Team Battle"),
	}

	got := filterRowsByCascade(rows, nil, []string{"Social Slayer"}, nil, nil)
	if len(got) != 1 || got[0].Summary.MatchID != "m1" {
		t.Errorf("want only m1, got %v", matchIDs(got))
	}
}

func TestFilterRowsByCascade_NilPlaylistExcluded(t *testing.T) {
	t.Parallel()
	// Une row sans Playlist est exclue quand un filtre playlist est actif.
	rows := []canonical.PlayerMatchRow{
		rowWithPlaylist("m1", "Quick Play", "Partie rapide"),
		row("m2", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), canonical.OutcomeWin), // Playlist nil
	}

	got := filterRowsByCascade(rows, nil, []string{"Partie rapide"}, nil, nil)
	if len(got) != 1 || got[0].Summary.MatchID != "m1" {
		t.Errorf("want only m1, got %v", matchIDs(got))
	}
}

func TestFilterRowsByCascade_ExperienceFilter(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		rowWithExperience("pvp-nc", false, false), // PVP non classé
		rowWithExperience("pvp-c", true, false),   // PVP classé
		rowWithExperience("pve", false, true),     // PVE
	}

	got := filterRowsByCascade(rows, []string{"PVP non classé"}, nil, nil, nil)
	if len(got) != 1 || got[0].Summary.MatchID != "pvp-nc" {
		t.Errorf("want only pvp-nc, got %v", matchIDs(got))
	}

	got2 := filterRowsByCascade(rows, []string{"PVE", "PVP classé"}, nil, nil, nil)
	if len(got2) != 2 {
		t.Errorf("want 2 rows (PVE+ranked), got %d", len(got2))
	}
}

func TestFilterRowsByCascade_CombinedFilters(t *testing.T) {
	t.Parallel()
	// Seule une row satisfaisant à la fois experience_type ET playlist passe.
	rows := []canonical.PlayerMatchRow{
		func() canonical.PlayerMatchRow {
			r := rowWithPlaylist("m1", "Ranked Arena", "Arène classée")
			ranked := true
			r.Summary.IsRanked = &ranked
			return r
		}(),
		func() canonical.PlayerMatchRow {
			r := rowWithPlaylist("m2", "Quick Play", "Partie rapide")
			isRanked := false
			r.Summary.IsRanked = &isRanked
			return r
		}(),
		func() canonical.PlayerMatchRow {
			r := rowWithPlaylist("m3", "Ranked Arena", "Arène classée")
			isRanked := false
			r.Summary.IsRanked = &isRanked
			return r
		}(),
	}

	got := filterRowsByCascade(rows, []string{"PVP classé"}, []string{"Arène classée"}, nil, nil)
	if len(got) != 1 || got[0].Summary.MatchID != "m1" {
		t.Errorf("only m1 is ranked + Arène classée, got %v", matchIDs(got))
	}
}

func TestFilterRowsByCascade_EmptyFilters(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{
		rowWithPlaylist("m1", "Quick Play", "Partie rapide"),
		rowWithPlaylist("m2", "Ranked Arena", "Arène classée"),
	}

	got := filterRowsByCascade(rows, nil, nil, nil, nil)
	if len(got) != 2 {
		t.Errorf("no filter = all rows pass, got %d", len(got))
	}
}

func TestFilterRowsByCascade_MapFilter(t *testing.T) {
	t.Parallel()
	// Filtre par label FR de carte (COALESCE(map_name_fr, map_name)).
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	makeRowWithMap := func(id, mapEN, mapFR string) canonical.PlayerMatchRow {
		r := row(id, t0, canonical.OutcomeWin)
		r.Summary.Map = &canonical.AssetReference{
			Kind:         "map",
			ID:           "map-" + id,
			DefaultLabel: mapEN,
			Labels:       map[string]string{"en": mapEN, "fr": mapFR},
		}
		return r
	}
	rows := []canonical.PlayerMatchRow{
		makeRowWithMap("m1", "Recharge", "Décharge"),
		makeRowWithMap("m2", "Bazaar", "Bazar"),
		makeRowWithMap("m3", "Recharge", "Décharge"),
	}

	got := filterRowsByCascade(rows, nil, nil, []string{"Décharge"}, nil)
	if len(got) != 2 {
		t.Fatalf("want 2 rows for 'Décharge', got %d", len(got))
	}
	for _, r := range got {
		if r.Summary.MatchID == "m2" {
			t.Error("m2 (Bazar) should have been filtered out")
		}
	}
}

// matchIDs extrait les MatchID d'une slice pour affichage dans les erreurs.
func matchIDs(rows []canonical.PlayerMatchRow) []string {
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.Summary.MatchID
	}
	return ids
}

// ---------------------------------------------------------------------------
// Tests filterRowsByCascade — filtre modes (PairMode)
// ---------------------------------------------------------------------------

// rowWithMode construit une row dont Summary.PairMode est peuplé avec les
// labels EN et FR, miroir de assetReference() dans player_matches_repo.go.
func rowWithMode(matchID, modeEN, modeFR string) canonical.PlayerMatchRow {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	r := row(matchID, t0, canonical.OutcomeWin)
	r.Summary.PairMode = &canonical.AssetReference{
		Kind:         "pair_mode",
		ID:           "mode-" + matchID,
		DefaultLabel: modeEN,
		Labels:       map[string]string{"en": modeEN, "fr": modeFR},
	}
	return r
}

func TestFilterRowsByCascade_ModeFRLabel(t *testing.T) {
	t.Parallel()
	// filtersResolve retourne COALESCE(pair_name_fr, pair_name) = label FR.
	// filterRowsByCascade doit utiliser Labels["fr"] pour que la comparaison aboutisse.
	rows := []canonical.PlayerMatchRow{
		rowWithMode("m1", "Slayer", "Slayer"),
		rowWithMode("m2", "Capture the Flag", "Capture the Flag"),
		rowWithMode("m3", "Slayer", "Slayer"),
	}

	got := filterRowsByCascade(rows, nil, nil, nil, []string{"Slayer"})
	if len(got) != 2 {
		t.Fatalf("want 2 rows for mode 'Slayer', got %d", len(got))
	}
	for _, r := range got {
		if r.Summary.MatchID == "m2" {
			t.Error("m2 (Capture the Flag) should have been filtered out")
		}
	}
}

func TestFilterRowsByCascade_ModeENFallback(t *testing.T) {
	t.Parallel()
	// Quand pair_name_fr est absent, COALESCE retourne l'anglais. Le label FR
	// stocké dans Labels["fr"] = label anglais = ce que filtersResolve envoie.
	rows := []canonical.PlayerMatchRow{
		rowWithMode("m1", "Extraction", "Extraction"),
		rowWithMode("m2", "Oddball", "Oddball"),
	}

	got := filterRowsByCascade(rows, nil, nil, nil, []string{"Extraction"})
	if len(got) != 1 || got[0].Summary.MatchID != "m1" {
		t.Errorf("want only m1, got %v", matchIDs(got))
	}
}

func TestFilterRowsByCascade_NilModeExcluded(t *testing.T) {
	t.Parallel()
	// Une row sans PairMode NI game_variant est exclue quand un filtre modes est actif.
	rows := []canonical.PlayerMatchRow{
		rowWithMode("m1", "Slayer", "Slayer"),
		row("m2", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), canonical.OutcomeWin), // PairMode + GameVariant nil
	}

	got := filterRowsByCascade(rows, nil, nil, nil, []string{"Slayer"})
	if len(got) != 1 || got[0].Summary.MatchID != "m1" {
		t.Errorf("want only m1 (m2 has nil PairMode/GameVariant), got %v", matchIDs(got))
	}
}

// rowWithGameVariant construit une row SANS PairMode dont Summary.GameVariant
// porte les labels EN/FR — modèle Halo 5 (pas de pair, mode dérivé du game_variant).
func rowWithGameVariant(matchID, variantEN, variantFR string) canonical.PlayerMatchRow {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	r := row(matchID, t0, canonical.OutcomeWin)
	r.Summary.GameVariant = &canonical.AssetReference{
		Kind:         "game_variant",
		ID:           "gv-" + matchID,
		DefaultLabel: variantEN,
		Labels:       map[string]string{"en": variantEN, "fr": variantFR},
	}
	return r
}

// TestFilterRowsByCascade_ModeGameVariantFallback : titre sans pair (Halo 5) — le
// filtre modes matche sur le game_variant NORMALISÉ, parité avec le chokepoint
// filters_service.modeUI (Value d'option = NormalizeModeLabel(game_variant_fr)).
func TestFilterRowsByCascade_ModeGameVariantFallback(t *testing.T) {
	t.Parallel()
	// Variant FR porteur d'un préfixe technique → normalisé en "Assassin".
	rows := []canonical.PlayerMatchRow{
		rowWithGameVariant("m1", "HaloMultiplayer:Slayer", "HaloMultiplayer:Assassin"),
		rowWithGameVariant("m2", "HaloMultiplayer:CTF", "HaloMultiplayer:Capture du drapeau"),
	}

	got := filterRowsByCascade(rows, nil, nil, nil, []string{"Assassin"})
	if len(got) != 1 || got[0].Summary.MatchID != "m1" {
		t.Errorf("want only m1 (game_variant → Assassin), got %v", matchIDs(got))
	}
}

// TestFilterRowsByCascade_ModePairPrimesOverGameVariant : PairMode présent prime
// sur game_variant (iso-comportement Infinite : le variant n'est jamais consulté).
func TestFilterRowsByCascade_ModePairPrimesOverGameVariant(t *testing.T) {
	t.Parallel()
	r := rowWithMode("m1", "Slayer", "Slayer")
	r.Summary.GameVariant = &canonical.AssetReference{
		Kind:         "game_variant",
		ID:           "gv-m1",
		DefaultLabel: "Assassin",
		Labels:       map[string]string{"en": "Assassin", "fr": "Assassin"},
	}
	rows := []canonical.PlayerMatchRow{r}

	if got := filterRowsByCascade(rows, nil, nil, nil, []string{"Slayer"}); len(got) != 1 {
		t.Errorf("pair 'Slayer' doit matcher, got %v", matchIDs(got))
	}
	if got := filterRowsByCascade(rows, nil, nil, nil, []string{"Assassin"}); len(got) != 0 {
		t.Errorf("game_variant ignoré quand pair présent, got %v", matchIDs(got))
	}
}

// =============================================================================
// SessionBriefing — KPIsByXUID + TeamAvgKPIs (drill-down + trends ▲/▼)
// =============================================================================

// rowWithStatsXUID est rowWithStats avec un xuid renseigne sur Self.Identity.
// Necessaire pour les tests SessionBriefing qui dependent de gtToXUID resolu.
func rowWithStatsXUID(xuid, matchID string, ts time.Time, outcome canonical.Outcome,
	kills, deaths, assists, timePlayed int, accuracy float64, perfScore float64,
) canonical.PlayerMatchRow {
	row := rowWithStats(matchID, ts, outcome, kills, deaths, assists, timePlayed, accuracy, perfScore)
	row.Self.Identity.XUID = xuid
	return row
}

func TestSquadServiceV2_GetSquadPage_HeaderKPIsByXUIDAndTeamAvg(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main": {rowWithStatsXUID("xuid-main", "m1", t0, canonical.OutcomeWin, 10, 5, 2, 600, 50, 80)},
			"f1":   {rowWithStatsXUID("xuid-f1", "m1", t0, canonical.OutcomeWin, 8, 4, 3, 600, 48, 70)},
			"f2":   {rowWithStatsXUID("xuid-f2", "m1", t0, canonical.OutcomeWin, 6, 6, 1, 600, 45, 60)},
		},
	}
	svc := NewSquadServiceV2(loader)
	resp, err := svc.GetSquadPage(context.Background(), "halo_infinite", "main",
		[]string{"f1", "f2"}, temporal.PeriodAll, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("GetSquadPage: %v", err)
	}
	// 3 entrees dans KPIsByXUID
	if len(resp.Header.KPIsByXUID) != 3 {
		t.Fatalf("KPIsByXUID: want 3 entries, got %d", len(resp.Header.KPIsByXUID))
	}
	for _, xuid := range []string{"xuid-main", "xuid-f1", "xuid-f2"} {
		if resp.Header.KPIsByXUID[xuid] == nil {
			t.Errorf("KPIsByXUID[%q] should be non-nil", xuid)
		}
	}
	// TeamAvgKPIs : moyenne kills_per_game = (10 + 8 + 6) / 3 = 8.0
	if resp.Header.TeamAvgKPIs == nil {
		t.Fatal("TeamAvgKPIs should be filled")
	}
	if math.Abs(resp.Header.TeamAvgKPIs.KillsPerGame-8.0) > 1e-9 {
		t.Errorf("TeamAvgKPIs.KillsPerGame: want 8.0, got %v", resp.Header.TeamAvgKPIs.KillsPerGame)
	}
	// Chaque PlayerScoreCard a son XUID
	for _, card := range resp.Header.PlayerCards {
		if card.XUID == "" {
			t.Errorf("PlayerScoreCard for %q: XUID should be set", card.Gamertag)
		}
	}
}

func TestSquadServiceV2_GetSquadPage_KPIsByXUIDEmptyWhenNoXUIDs(t *testing.T) {
	t.Parallel()
	// Rows sans XUID -> gtToXUID vide -> KPIsByXUID nil/empty et TeamAvgKPIs nil.
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main": {rowWithStats("m1", t0, canonical.OutcomeWin, 10, 5, 2, 600, 50, 80)},
			"f1":   {rowWithStats("m1", t0, canonical.OutcomeWin, 8, 4, 3, 600, 48, 70)},
		},
	}
	svc := NewSquadServiceV2(loader)
	resp, err := svc.GetSquadPage(context.Background(), "halo_infinite", "main",
		[]string{"f1"}, temporal.PeriodAll, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("GetSquadPage: %v", err)
	}
	if len(resp.Header.KPIsByXUID) != 0 {
		t.Errorf("no XUIDs -> KPIsByXUID should be empty, got %d entries", len(resp.Header.KPIsByXUID))
	}
	if resp.Header.TeamAvgKPIs != nil {
		t.Errorf("no KPIsByXUID -> TeamAvgKPIs should be nil, got %+v", resp.Header.TeamAvgKPIs)
	}
	// PlayerCards sans XUID -> XUID vide (comportement defini, pas de panic)
	for _, card := range resp.Header.PlayerCards {
		if card.XUID != "" {
			t.Errorf("PlayerScoreCard for %q: XUID should be empty when no xuid in rows, got %q", card.Gamertag, card.XUID)
		}
	}
}
