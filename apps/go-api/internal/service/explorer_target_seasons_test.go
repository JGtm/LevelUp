package service

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

func tm(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func TestBuildMatchesPerSeason_Bucketing(t *testing.T) {
	endS1 := tm("2022-05-03T17:00:00Z")
	endS2 := tm("2022-11-08T17:00:00Z")
	seasons := []SeasonCatalogEntry{
		{ID: "season1", Label: "Heroes of Reach", Start: tm("2021-12-08T17:00:00Z"), End: &endS1, Extra: map[string]string{"short_label": "S1"}},
		{ID: "season2", Label: "Lone Wolves", Start: tm("2022-05-03T17:00:00Z"), End: &endS2, Extra: map[string]string{"short_label": "S2"}},
		{ID: "season3", Label: "Open", Start: tm("2022-11-08T17:00:00Z"), End: nil}, // saison ouverte, pas de short_label
	}
	starts := []time.Time{
		tm("2022-01-10T10:00:00Z"), // S1
		tm("2022-02-10T10:00:00Z"), // S1
		tm("2022-06-10T10:00:00Z"), // S2
		tm("2023-01-10T10:00:00Z"), // S3 (ouverte)
		tm("2020-01-01T10:00:00Z"), // avant toute saison → ignoré
	}

	got := buildMatchesPerSeason(starts, seasons)
	if len(got) != 3 {
		t.Fatalf("attendu 3 saisons avec matchs, got %d (%+v)", len(got), got)
	}
	// Ordre préservé (DisplayOrder via l'ordre du slice).
	if got[0].SeasonID != "season1" || got[0].Matches != 2 || got[0].SeasonName != "S1" {
		t.Errorf("S1 attendu (2 matchs, label S1), got %+v", got[0])
	}
	if got[1].Matches != 1 || got[1].SeasonName != "S2" {
		t.Errorf("S2 attendu 1 match, got %+v", got[1])
	}
	// season3 sans short_label → fallback Label.
	if got[2].SeasonID != "season3" || got[2].Matches != 1 || got[2].SeasonName != "Open" {
		t.Errorf("S3 attendu 1 match label fallback 'Open', got %+v", got[2])
	}
}

func TestBuildMatchesPerSeason_Empty(t *testing.T) {
	seasons := []SeasonCatalogEntry{{ID: "s1", Start: tm("2021-01-01T00:00:00Z")}}
	if got := buildMatchesPerSeason(nil, seasons); got != nil {
		t.Errorf("starts vide → nil, got %v", got)
	}
	if got := buildMatchesPerSeason([]time.Time{tm("2021-06-01T00:00:00Z")}, nil); got != nil {
		t.Errorf("seasons vide → nil, got %v", got)
	}
}

func TestExtractSeasonNumber(t *testing.T) {
	cases := map[string]int{
		"Seasons/Season7.json":          7,
		"Seasons/Season10.json":         10,
		"Seasons/Season6-2.json":        6, // premier entier
		"Csr/Seasons/CsrSeason9-1.json": 9,
		"S13":                           13,
		"season1":                       1,
		"Season-Winter-Break-22.json":   22,
		"":                              -1,
		"nope":                          -1,
	}
	for in, want := range cases {
		if got := extractSeasonNumber(in); got != want {
			t.Errorf("extractSeasonNumber(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestGroupMatchmadeSeasonsByNumber(t *testing.T) {
	in := []string{
		"Seasons/Season6.json",
		"Seasons/Season6-2.json", // opération intra-S6 → même numéro
		"Seasons/Season7.json",
		"Csr/Seasons/CsrSeason6-1.json", // chemin CSR → ignoré
		"garbage",                       // ignoré
	}
	got := groupMatchmadeSeasonsByNumber(in)
	if len(got[6]) != 2 {
		t.Errorf("S6 attendu 2 chemins (Season6 + Season6-2), got %v", got[6])
	}
	if len(got[7]) != 1 {
		t.Errorf("S7 attendu 1 chemin, got %v", got[7])
	}
	if _, ok := got[9]; ok {
		t.Errorf("un chemin CSR ne doit pas créer d'entrée matchmade (n°9): %v", got)
	}
}

// mockSeasonSR : compte de matchs canné par (chemin CMS, filtre ranked).
type mockSeasonSR struct {
	ranked   map[string]int
	unranked map[string]int
	calls    int32
}

func (m *mockSeasonSR) FetchSeasonServiceRecord(_ context.Context, _, seasonID string, isRanked *bool) (int, error) {
	atomic.AddInt32(&m.calls, 1)
	if isRanked == nil {
		return m.ranked[seasonID] + m.unranked[seasonID], nil
	}
	if *isRanked {
		return m.ranked[seasonID], nil
	}
	return m.unranked[seasonID], nil
}

func breakdownTestSeasons() []SeasonCatalogEntry {
	mk := func(id, short, csr string) SeasonCatalogEntry {
		return SeasonCatalogEntry{ID: id, Label: id, Extra: map[string]string{"short_label": short, "csr_season_id": csr}}
	}
	return []SeasonCatalogEntry{
		mk("season6", "S6", "CsrSeason6-1"),
		mk("season7", "S7", "CsrSeason7-1"),
		mk("season8", "S8", "CsrSeason8-1"), // non joué
	}
}

func TestComputeSeasonBreakdown_Live(t *testing.T) {
	sr := &mockSeasonSR{
		ranked:   map[string]int{"Seasons/Season6.json": 10, "Seasons/Season6-2.json": 5, "Seasons/Season7.json": 20},
		unranked: map[string]int{"Seasons/Season6.json": 3, "Seasons/Season6-2.json": 2, "Seasons/Season7.json": 8},
	}
	csrCalls := int32(0)
	csr := SeasonCSRPeakFunc(func(_ context.Context, _, _ string, engaged []string) (*SeasonCSRPeak, error) {
		if len(engaged) == 0 {
			return nil, nil // contrat : pas d'appel utile sans playlist engagée
		}
		atomic.AddInt32(&csrCalls, 1)
		badge := "/static/ranks/halo_infinite/120px-HINF-CSR_Onyx.png"
		return &SeasonCSRPeak{Tier: "Onyx", SubTier: 0, BadgeURL: &badge}, nil
	})

	svc := NewExplorerService(&mockExplorerRepo{}, "me").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{
			Seasons: breakdownTestSeasons(), SeasonSR: sr, SeasonCSR: csr, TitleSlug: "halo_infinite",
		})

	played := []string{"Seasons/Season6.json", "Seasons/Season6-2.json", "Seasons/Season7.json", "Csr/Seasons/CsrSeason6-1.json"}
	engaged := []string{"ranked-arena-asset"}
	out, status := svc.computeSeasonBreakdown(ctxAuth(true, "me"), "target-xuid", "Target", true, played, engaged)
	if status != domain.ExplorerLiveOK {
		t.Errorf("status attendu ok (breakdown live), got %q", status)
	}

	if len(out) != 3 {
		t.Fatalf("attendu 3 saisons (catalogue complet), got %d", len(out))
	}
	// S6 : total sommé sur les deux chemins (opération Season6-2 incluse) :
	// (10+3) + (5+2) = 20.
	if out[0].Matches != 20 {
		t.Errorf("S6 attendu total=20, got %+v", out[0])
	}
	if out[0].CSRTier != "Onyx" || out[0].CSRBadgeImageURL == nil {
		t.Errorf("S6 badge CSR attendu, got tier=%q badge=%v", out[0].CSRTier, out[0].CSRBadgeImageURL)
	}
	// S7 : un seul chemin → 20+8 = 28.
	if out[1].Matches != 28 {
		t.Errorf("S7 attendu total=28, got %+v", out[1])
	}
	// S8 : non jouée → barre vide, aucun appel CSR.
	if out[2].Matches != 0 || out[2].CSRTier != "" {
		t.Errorf("S8 attendu vide, got %+v", out[2])
	}
	if csrCalls != 2 {
		t.Errorf("CSR attendu 2 appels (S6+S7 jouées), got %d", csrCalls)
	}
}

func TestComputeSeasonBreakdown_FallbackNoAuth(t *testing.T) {
	sr := &mockSeasonSR{}
	svc := NewExplorerService(&mockExplorerRepo{}, "me").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{
			Seasons: breakdownTestSeasons(), SeasonSR: sr, TitleSlug: "halo_infinite",
		})

	// hasAuth=false → fallback bucketing local : aucun appel live par saison.
	_, status := svc.computeSeasonBreakdown(context.Background(), "target-xuid", "Target", false,
		[]string{"Seasons/Season7.json"}, []string{"ranked-arena-asset"})
	if atomic.LoadInt32(&sr.calls) != 0 {
		t.Errorf("aucun appel SeasonSR attendu sans auth, got %d", sr.calls)
	}
	if status != domain.ExplorerLiveNoAuth {
		t.Errorf("status attendu no_auth, got %q", status)
	}
}

// TestComputeSeasonBreakdown_NoEngagedPlaylists_SkipsCSR : un joueur sans playlist
// ranked engagée (engagedPlaylistIDs vide → ex. joueur social) ne déclenche AUCUN
// appel CSR (optim), mais les totaux de matchs par saison restent calculés.
func TestComputeSeasonBreakdown_NoEngagedPlaylists_SkipsCSR(t *testing.T) {
	sr := &mockSeasonSR{
		ranked:   map[string]int{"Seasons/Season7.json": 0},
		unranked: map[string]int{"Seasons/Season7.json": 12},
	}
	csrCalls := int32(0)
	csr := SeasonCSRPeakFunc(func(_ context.Context, _, _ string, _ []string) (*SeasonCSRPeak, error) {
		atomic.AddInt32(&csrCalls, 1)
		return nil, nil
	})
	svc := NewExplorerService(&mockExplorerRepo{}, "me").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{
			Seasons: breakdownTestSeasons(), SeasonSR: sr, SeasonCSR: csr, TitleSlug: "halo_infinite",
		})

	out, status := svc.computeSeasonBreakdown(ctxAuth(true, "me"), "target-xuid", "Target", true,
		[]string{"Seasons/Season7.json"}, nil /* aucune playlist engagée */)
	if status != domain.ExplorerLiveOK {
		t.Errorf("status attendu ok (breakdown live), got %q", status)
	}

	if out[1].Matches != 12 {
		t.Errorf("S7 total attendu 12, got %+v", out[1])
	}
	if csrCalls != 0 {
		t.Errorf("aucun appel CSR attendu sans playlist engagée, got %d", csrCalls)
	}
}
