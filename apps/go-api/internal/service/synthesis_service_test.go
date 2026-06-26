// Package service â€” synthesis_service_test.go : tests pour SynthesisService.
// Sprint 55 D9.
package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
)

var errSynthTest = errors.New("synthesis repo error")

// --- mock SynthesisRepository ---

type mockSynthesisRepo struct {
	synthRows   []legacymatch.SynthesisMatchRow
	synthErr    error
	heatmapRows []domain.SynthesisHeatmapRow
	heatmapErr  error
}

func (m *mockSynthesisRepo) LoadSynthesisMatches(_ context.Context, _ string) ([]legacymatch.SynthesisMatchRow, error) {
	return m.synthRows, m.synthErr
}

func (m *mockSynthesisRepo) LoadSynthesisHeatmap(_ context.Context, _ string) ([]domain.SynthesisHeatmapRow, error) {
	return m.heatmapRows, m.heatmapErr
}

func (m *mockSynthesisRepo) EnrichCanonicalAssetTranslations(_ context.Context, _ []canonical.PlayerMatchRow) error {
	return nil
}

// --- mock PlayerMatchesRepository pour tests P4.3 finale ---
//
// Convertit []SynthesisMatchRow en []canonical.PlayerMatchRow pour exercer
// le path canonical (le seul path en service aprÃ¨s P4.3 finale).
type mockSynthesisPlayerMatches struct {
	rows []legacymatch.SynthesisMatchRow
	err  error
}

func (m *mockSynthesisPlayerMatches) LoadPlayerMatches(_ context.Context, _ string, _ string, _ port.PlayerMatchFilters) ([]canonical.PlayerMatchRow, error) {
	if m.err != nil {
		return nil, m.err
	}
	out := make([]canonical.PlayerMatchRow, len(m.rows))
	for i, r := range m.rows {
		k, d := r.Kills, r.Deaths
		var outcome canonical.Outcome
		switch r.Outcome {
		case domain.OutcomeWin:
			outcome = canonical.OutcomeWin
		case domain.OutcomeLoss:
			outcome = canonical.OutcomeLoss
		case domain.OutcomeDraw:
			outcome = canonical.OutcomeTie
		case domain.OutcomeDNF:
			outcome = canonical.OutcomeDNF
		}
		out[i] = canonical.PlayerMatchRow{
			Summary: canonical.MatchSummary{
				MatchID:      r.MatchID,
				StartedAtUTC: r.StartTime,
				Outcome:      outcome,
				Map:          &canonical.AssetReference{DefaultLabel: "TestMap"},
				PairMode:     &canonical.AssetReference{DefaultLabel: "Slayer"},
			},
			Self: canonical.MatchParticipant{
				Kills: &k, Deaths: &d, KDA: r.KDA, Outcome: outcome,
				Accuracy: r.Accuracy, TimePlayed: r.TimePlayedSecs,
			},
			Enrichment: canonical.PlayerMatchEnrichment{
				IsWithFriends:    r.IsWithFriends,
				PerformanceScore: r.PerformanceScore,
				SessionLabel:     r.SessionLabel,
			},
		}
	}
	return out, nil
}

func (m *mockSynthesisPlayerMatches) InvalidatePlayer(_, _ string) {}

// withSynthMock attache le mock canonical au service pour exercer le path
// canonical (seul path actif aprÃ¨s P4.3 finale).
func withSynthMock(svc *SynthesisService, rows []legacymatch.SynthesisMatchRow, err error) *SynthesisService {
	pm := &mockSynthesisPlayerMatches{rows: rows, err: err}
	return svc.WithPlayerMatchesRepo(pm, "halo_infinite", "TestPlayer")
}

// --- helpers ---

func makeSynthRows(n int) []legacymatch.SynthesisMatchRow {
	rows := make([]legacymatch.SynthesisMatchRow, n)
	for i := range rows {
		kills := 10 - i%10
		deaths := 3 + i%5
		kda := float64(kills) / float64(deaths+1)
		rows[i] = legacymatch.SynthesisMatchRow{
			MatchID:   fmt.Sprintf("match-%d", i),
			StartTime: time.Now().UTC().Add(-time.Duration(i) * time.Hour),
			Outcome:   2 + i%2, // alternance WIN/LOSS
			Kills:     kills,
			Deaths:    deaths,
			KDA:       &kda,
		}
	}
	return rows
}

// --- filterSynthesisByPeriod ---

func TestFilterSynthesisByPeriod_All(t *testing.T) {
	rows := makeSynthRows(10)
	got, applied, ignored := filterSynthesisByPeriod(rows, "all", domain.FilterContextInput{})
	if len(got) != 10 {
		t.Fatalf("want 10 rows, got %d", len(got))
	}
	if len(applied) != 0 {
		t.Errorf("want 0 applied, got %v", applied)
	}
	if len(ignored) != 0 {
		t.Errorf("want 0 ignored, got %v", ignored)
	}
}

func TestFilterSynthesisByPeriod_1w(t *testing.T) {
	rows := []legacymatch.SynthesisMatchRow{
		{MatchID: "recent", StartTime: time.Now().UTC().Add(-24 * time.Hour), Outcome: 2, Kills: 5},
		{MatchID: "old", StartTime: time.Now().UTC().Add(-30 * 24 * time.Hour), Outcome: 3, Kills: 3},
	}
	got, applied, _ := filterSynthesisByPeriod(rows, "1w", domain.FilterContextInput{})
	if len(got) != 1 {
		t.Fatalf("want 1 row in 1w period, got %d", len(got))
	}
	if got[0].MatchID != "recent" {
		t.Errorf("want recent match, got %s", got[0].MatchID)
	}
	if len(applied) == 0 {
		t.Error("want applied filter for 1w")
	}
}

func TestFilterSynthesisByPeriod_Unknown(t *testing.T) {
	rows := makeSynthRows(5)
	got, _, _ := filterSynthesisByPeriod(rows, "unknown_period", domain.FilterContextInput{})
	if len(got) != 5 {
		t.Fatalf("unknown period should return all rows, got %d", len(got))
	}
}

// --- buildHighlightsPreview ---

func TestBuildHighlightsPreview_Empty(t *testing.T) {
	h := buildHighlightsPreview(nil)
	if len(h.TopByKills) != 0 || len(h.TopByKDA) != 0 || len(h.WorstByDeaths) != 0 {
		t.Error("empty rows should return empty highlights")
	}
}

func TestBuildHighlightsPreview_TopByKills(t *testing.T) {
	rows := []legacymatch.SynthesisMatchRow{
		{MatchID: "a", Kills: 5},
		{MatchID: "b", Kills: 12},
		{MatchID: "c", Kills: 8},
	}
	h := buildHighlightsPreview(rows)
	if len(h.TopByKills) == 0 {
		t.Fatal("want top by kills, got empty")
	}
	if h.TopByKills[0].Kills != 12 {
		t.Errorf("top kill should be 12, got %d", h.TopByKills[0].Kills)
	}
}

func TestBuildHighlightsPreview_WorstByDeaths(t *testing.T) {
	rows := []legacymatch.SynthesisMatchRow{
		{MatchID: "a", Deaths: 2},
		{MatchID: "b", Deaths: 15},
		{MatchID: "c", Deaths: 7},
	}
	h := buildHighlightsPreview(rows)
	if len(h.WorstByDeaths) == 0 {
		t.Fatal("want worst by deaths, got empty")
	}
	if h.WorstByDeaths[0].Deaths != 15 {
		t.Errorf("worst deaths should be 15, got %d", h.WorstByDeaths[0].Deaths)
	}
}

func TestBuildHighlightsPreview_LimitTopN(t *testing.T) {
	rows := makeSynthRows(20)
	h := buildHighlightsPreview(rows)
	if len(h.TopByKills) > highlightTopN {
		t.Errorf("top by kills capped at %d, got %d", highlightTopN, len(h.TopByKills))
	}
	if len(h.WorstByDeaths) > highlightTopN {
		t.Errorf("worst by deaths capped at %d, got %d", highlightTopN, len(h.WorstByDeaths))
	}
}

// --- buildBreakdowns ---

func TestBuildBreakdowns_Empty(t *testing.T) {
	b := buildBreakdowns(nil)
	if len(b.TopMaps) != 0 || len(b.TopModes) != 0 {
		t.Error("empty heatmap should return empty breakdowns")
	}
}

func TestBuildBreakdowns_Aggregates(t *testing.T) {
	rows := []domain.SynthesisHeatmapRow{
		{MapName: "Aquarius", ModeName: "Slayer", MatchCount: 5, Wins: 3},
		{MapName: "Aquarius", ModeName: "Oddball", MatchCount: 2, Wins: 1},
		{MapName: "Bazaar", ModeName: "Slayer", MatchCount: 3, Wins: 2},
	}
	b := buildBreakdowns(rows)

	// Aquarius doit avoir 7 matchs (5+2)
	var aquarius *domain.SynthesisMapEntry
	for i := range b.TopMaps {
		if b.TopMaps[i].MapName == "Aquarius" {
			aquarius = &b.TopMaps[i]
			break
		}
	}
	if aquarius == nil {
		t.Fatal("Aquarius not found in breakdowns")
	}
	if aquarius.MatchCount != 7 {
		t.Errorf("Aquarius match count should be 7, got %d", aquarius.MatchCount)
	}
	if aquarius.Wins != 4 {
		t.Errorf("Aquarius wins should be 4, got %d", aquarius.Wins)
	}
}

// --- GetSynthesisPage ---

func TestGetSynthesisPage_Success(t *testing.T) {
	kda := 2.0
	repo := &mockSynthesisRepo{
		synthRows: []legacymatch.SynthesisMatchRow{
			{MatchID: "m1", StartTime: time.Now().UTC(), Outcome: 2, Kills: 10, Deaths: 3, KDA: &kda},
			{MatchID: "m2", StartTime: time.Now().UTC().Add(-time.Hour), Outcome: 3, Kills: 4, Deaths: 8, KDA: &kda},
		},
		heatmapRows: []domain.SynthesisHeatmapRow{
			{MapName: "Aquarius", ModeName: "Slayer", MatchCount: 2, Wins: 1},
		},
	}

	svc := withSynthMock(NewSynthesisService(repo), repo.synthRows, repo.synthErr)
	resp, err := svc.GetSynthesisPage(context.Background(), "xuid-test", domain.SynthesisRequest{Period: "all"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Scope.MatchCount != 2 {
		t.Errorf("scope match count = %d, want 2", resp.Scope.MatchCount)
	}
	if resp.Overview.TotalMatches != 2 {
		t.Errorf("overview total matches = %d, want 2", resp.Overview.TotalMatches)
	}
	if len(resp.Breakdowns.TopMaps) == 0 {
		t.Error("expected non-empty map breakdown")
	}
}

func TestGetSynthesisPage_DefaultPeriodAll(t *testing.T) {
	repo := &mockSynthesisRepo{synthRows: []legacymatch.SynthesisMatchRow{}}
	svc := withSynthMock(NewSynthesisService(repo), repo.synthRows, repo.synthErr)

	resp, err := svc.GetSynthesisPage(context.Background(), "xuid", domain.SynthesisRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Scope.Period != "all" {
		t.Errorf("default period should be 'all', got %q", resp.Scope.Period)
	}
}

func TestGetSynthesisPage_RepoError(t *testing.T) {
	repo := &mockSynthesisRepo{synthErr: errSynthTest}
	svc := withSynthMock(NewSynthesisService(repo), repo.synthRows, repo.synthErr)

	_, err := svc.GetSynthesisPage(context.Background(), "xuid", domain.SynthesisRequest{})
	if err == nil {
		t.Error("expected error from repo, got nil")
	}
}

// --- D9 : scope rÃ©ellement appliquÃ© ---

// TestGetSynthesisPage_ScopeApplied_Period vÃ©rifie que GetSynthesisPage filtre
// les matchs selon la pÃ©riode demandÃ©e et que scope.MatchCount le reflÃ¨te.
func TestGetSynthesisPage_ScopeApplied_Period(t *testing.T) {
	repo := &mockSynthesisRepo{
		synthRows: []legacymatch.SynthesisMatchRow{
			{MatchID: "recent", StartTime: time.Now().UTC().Add(-24 * time.Hour), Outcome: 2, Kills: 5},
			{MatchID: "old", StartTime: time.Now().UTC().Add(-30 * 24 * time.Hour), Outcome: 3, Kills: 3},
		},
	}
	svc := withSynthMock(NewSynthesisService(repo), repo.synthRows, repo.synthErr)

	resp, err := svc.GetSynthesisPage(context.Background(), "xuid", domain.SynthesisRequest{Period: "1w"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Scope.MatchCount != 1 {
		t.Errorf("scope.MatchCount = %d, want 1 (only recent match in 1w)", resp.Scope.MatchCount)
	}
	if resp.Scope.Period != "1w" {
		t.Errorf("scope.Period = %q, want %q", resp.Scope.Period, "1w")
	}
	if len(resp.Scope.FiltersApplied) == 0 {
		t.Error("scope.FiltersApplied should be non-empty when period filter is active")
	}
}

// TestGetSynthesisPage_Overview_MatchesScope vÃ©rifie que overview.TotalMatches
// correspond exactement au nombre de matchs dans le scope (aprÃ¨s filtrage).
func TestGetSynthesisPage_Overview_MatchesScope(t *testing.T) {
	kda := 1.5
	repo := &mockSynthesisRepo{
		synthRows: []legacymatch.SynthesisMatchRow{
			{MatchID: "m1", StartTime: time.Now().UTC(), Outcome: 2, Kills: 6, Deaths: 3, KDA: &kda},
			{MatchID: "m2", StartTime: time.Now().UTC().Add(-time.Hour), Outcome: 3, Kills: 2, Deaths: 5, KDA: &kda},
			{MatchID: "m3", StartTime: time.Now().UTC().Add(-2 * time.Hour), Outcome: 2, Kills: 8, Deaths: 2, KDA: &kda},
		},
	}
	svc := withSynthMock(NewSynthesisService(repo), repo.synthRows, repo.synthErr)

	resp, err := svc.GetSynthesisPage(context.Background(), "xuid", domain.SynthesisRequest{Period: "all"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Overview.TotalMatches != resp.Scope.MatchCount {
		t.Errorf("overview.TotalMatches (%d) != scope.MatchCount (%d)",
			resp.Overview.TotalMatches, resp.Scope.MatchCount)
	}
	if resp.Overview.TotalMatches != 3 {
		t.Errorf("overview.TotalMatches = %d, want 3", resp.Overview.TotalMatches)
	}
}

// TestGetSynthesisPage_Highlights_WithinScope vÃ©rifie que les MatchIDs dans
// highlights.TopByKills appartiennent aux matchs du scope filtrÃ©.
func TestGetSynthesisPage_Highlights_WithinScope(t *testing.T) {
	kda := 2.0
	repo := &mockSynthesisRepo{
		synthRows: []legacymatch.SynthesisMatchRow{
			{MatchID: "scoped-1", StartTime: time.Now().UTC().Add(-2 * time.Hour), Outcome: 2, Kills: 10, Deaths: 1, KDA: &kda},
			{MatchID: "scoped-2", StartTime: time.Now().UTC().Add(-3 * time.Hour), Outcome: 2, Kills: 7, Deaths: 2, KDA: &kda},
		},
	}
	svc := withSynthMock(NewSynthesisService(repo), repo.synthRows, repo.synthErr)

	resp, err := svc.GetSynthesisPage(context.Background(), "xuid", domain.SynthesisRequest{Period: "all"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	scopedIDs := make(map[string]bool)
	for _, row := range repo.synthRows {
		scopedIDs[row.MatchID] = true
	}
	for _, h := range resp.HighlightsPreview.TopByKills {
		if !scopedIDs[h.MatchID] {
			t.Errorf("highlight match_id %q not in scoped rows", h.MatchID)
		}
	}
	if len(resp.HighlightsPreview.TopByKills) == 0 {
		t.Error("expected non-empty TopByKills with 2 scoped matches")
	}
}

// --- computeSynthesisBestRefs : isolation des "Top X" cliquables (2026-05-27) ---

func makeCanonicalBestRow(matchID string, kills, deaths, dmg, spree int, kda, perf, acc float64) canonical.PlayerMatchRow { //nolint:revive // test helper with explicit per-metric params

	k, d, dm, sp := kills, deaths, dmg, spree
	row := canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{MatchID: matchID, Outcome: canonical.OutcomeWin},
		Self: canonical.MatchParticipant{
			Kills: &k, Deaths: &d, DamageDealt: &dm, MaxKillingSpree: &sp,
			Outcome: canonical.OutcomeWin,
		},
	}
	if kda > 0 {
		row.Self.KDA = &kda
	}
	if acc > 0 {
		row.Self.Accuracy = &acc
	}
	if perf > 0 {
		row.Enrichment.PerformanceScore = &perf
	}
	return row
}

func TestComputeSynthesisBestRefs_PicksWinningMatchPerMetric(t *testing.T) {
	rows := []canonical.PlayerMatchRow{
		// kills max
		makeCanonicalBestRow("m-kills", 25, 5, 1200, 4, 3.0, 1500, 0.50),
		// kda max
		makeCanonicalBestRow("m-kda", 18, 2, 1000, 5, 9.0, 1300, 0.45),
		// perf max
		makeCanonicalBestRow("m-perf", 12, 6, 1100, 3, 2.0, 2100, 0.40),
		// accuracy max
		makeCanonicalBestRow("m-acc", 10, 4, 900, 3, 2.5, 1200, 0.85),
		// damage max
		makeCanonicalBestRow("m-dmg", 20, 7, 2200, 6, 2.8, 1400, 0.55),
		// killing spree max
		makeCanonicalBestRow("m-spree", 22, 8, 1800, 12, 2.5, 1600, 0.50),
	}

	refs := computeSynthesisBestRefs(rows, true)

	cases := []struct {
		name    string
		got     *domain.BestMatchRef
		wantID  string
		wantVal float64
	}{
		{"kills", refs.kills, "m-kills", 25},
		{"kda", refs.kda, "m-kda", 9.0},
		{"perf", refs.perf, "m-perf", 2100},
		{"accuracy", refs.accuracy, "m-acc", 0.85},
		{"damage", refs.damage, "m-dmg", 2200},
		{"killing_spree", refs.killingSpree, "m-spree", 12},
	}
	for _, c := range cases {
		if c.got == nil {
			t.Errorf("%s: ref is nil", c.name)
			continue
		}
		if c.got.MatchID != c.wantID {
			t.Errorf("%s: match_id = %s, want %s", c.name, c.got.MatchID, c.wantID)
		}
		if c.got.Value != c.wantVal {
			t.Errorf("%s: value = %v, want %v", c.name, c.got.Value, c.wantVal)
		}
	}
}

func TestComputeSynthesisBestRefs_TieKeepsFirstSeen(t *testing.T) {
	rows := []canonical.PlayerMatchRow{
		makeCanonicalBestRow("first", 10, 5, 0, 0, 2.5, 0, 0),
		makeCanonicalBestRow("second", 10, 5, 0, 0, 2.5, 0, 0), // égalité parfaite
	}
	refs := computeSynthesisBestRefs(rows, true)
	if refs.kills == nil || refs.kills.MatchID != "first" {
		t.Errorf("kills tie: want first, got %+v", refs.kills)
	}
	if refs.kda == nil || refs.kda.MatchID != "first" {
		t.Errorf("kda tie: want first, got %+v", refs.kda)
	}
}

func TestComputeSynthesisBestRefs_NilFieldsYieldNilRef(t *testing.T) {
	rows := []canonical.PlayerMatchRow{
		// row sans accuracy ni perf ni KDA
		{
			Summary: canonical.MatchSummary{MatchID: "m1"},
			Self: canonical.MatchParticipant{
				Kills: intPtr(5), Deaths: intPtr(2),
			},
		},
	}
	refs := computeSynthesisBestRefs(rows, true)
	if refs.accuracy != nil {
		t.Errorf("accuracy: want nil ref when Self.Accuracy nil, got %+v", refs.accuracy)
	}
	if refs.perf != nil {
		t.Errorf("perf: want nil ref when PerformanceScore nil, got %+v", refs.perf)
	}
	if refs.kda != nil {
		t.Errorf("kda: want nil ref when Self.KDA nil, got %+v", refs.kda)
	}
	if refs.kills == nil || refs.kills.MatchID != "m1" {
		t.Errorf("kills: want ref to m1, got %+v", refs.kills)
	}
}

func TestComputeSynthesisBestRefs_HeadshotsAndPersonalScore(t *testing.T) {
	// Test isolé pour les 2 métriques ajoutées (2026-05-27 post-RA) afin de
	// garder makeCanonicalBestRow à 8 paramètres (sinon dette PLR0913).
	hs1, ps1 := 8, 1200
	hs2, ps2 := 14, 900 // m2 max headshots
	hs3, ps3 := 5, 3500 // m3 max personal score
	rows := []canonical.PlayerMatchRow{
		{
			Summary: canonical.MatchSummary{MatchID: "m1"},
			Self:    canonical.MatchParticipant{HeadshotKills: &hs1, PersonalScore: &ps1},
		},
		{
			Summary: canonical.MatchSummary{MatchID: "m2"},
			Self:    canonical.MatchParticipant{HeadshotKills: &hs2, PersonalScore: &ps2},
		},
		{
			Summary: canonical.MatchSummary{MatchID: "m3"},
			Self:    canonical.MatchParticipant{HeadshotKills: &hs3, PersonalScore: &ps3},
		},
	}
	refs := computeSynthesisBestRefs(rows, true)
	if refs.headshots == nil || refs.headshots.MatchID != "m2" || refs.headshots.Value != 14 {
		t.Errorf("headshots: want m2/14, got %+v", refs.headshots)
	}
	if refs.personalScore == nil || refs.personalScore.MatchID != "m3" || refs.personalScore.Value != 3500 {
		t.Errorf("personal_score: want m3/3500, got %+v", refs.personalScore)
	}
}

func TestComputeSynthesisBestRefs_ExcludesDNFAndPvE(t *testing.T) {
	// Les « Meilleures stats » excluent les matchs non terminés (DNF) et le
	// PvE/Firefight : le record doit pointer sur le meilleur match PvP TERMINÉ,
	// même si un DNF ou un Firefight a une valeur brute plus élevée.
	dnf := makeCanonicalBestRow("m-dnf", 40, 1, 5000, 20, 10, 3000, 0.9)
	dnf.Summary.Outcome = canonical.OutcomeDNF
	dnf.Self.Outcome = canonical.OutcomeDNF

	pve := makeCanonicalBestRow("m-pve", 60, 1, 8000, 30, 15, 4000, 0.95)
	isPvE := true
	pve.Summary.IsPvE = &isPvE

	clean := makeCanonicalBestRow("m-clean", 25, 5, 1200, 4, 3.0, 1500, 0.50)

	refs := computeSynthesisBestRefs([]canonical.PlayerMatchRow{dnf, pve, clean}, true)

	if refs.kills == nil || refs.kills.MatchID != "m-clean" || refs.kills.Value != 25 {
		t.Errorf("kills: want m-clean/25 (DNF+PvE exclus), got %+v", refs.kills)
	}
	if refs.damage == nil || refs.damage.MatchID != "m-clean" {
		t.Errorf("damage: want m-clean (DNF+PvE exclus), got %+v", refs.damage)
	}
	if refs.killingSpree == nil || refs.killingSpree.MatchID != "m-clean" {
		t.Errorf("killing_spree: want m-clean (DNF+PvE exclus), got %+v", refs.killingSpree)
	}
}

func TestComputeSynthesisBestRefs_AllZeroSkipsRef(t *testing.T) {
	// Si toutes les valeurs sont 0 (joueur n'a jamais tué/infligé de dégâts),
	// la carte FE ne doit pas s'afficher -> ref nil.
	rows := []canonical.PlayerMatchRow{
		makeCanonicalBestRow("m1", 0, 0, 0, 0, 0, 0, 0),
		makeCanonicalBestRow("m2", 0, 0, 0, 0, 0, 0, 0),
	}
	refs := computeSynthesisBestRefs(rows, true)
	if refs.kills != nil {
		t.Errorf("kills: want nil when all zero, got %+v", refs.kills)
	}
	if refs.damage != nil {
		t.Errorf("damage: want nil when all zero, got %+v", refs.damage)
	}
}
