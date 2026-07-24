// Package halo_infinite — citations_custom_test.go : tests des fonctions custom
// citations Halo-specific (déplacées depuis analysis/citations_test.go en P5.4).
package halo_infinite

import (
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// =============================================================================
// Tests fonctions custom
// =============================================================================

func TestComputeBulldozer_OK(t *testing.T) {
	ctx := domain.CitationContext{
		Stats:    map[string]float64{"kda": 9.0},
		Playlist: "arena slayer",
		Awards:   map[string]int{},
		Medals:   map[int64]int{},
	}
	if got := computeBulldozer(ctx); got != 1 {
		t.Errorf("attendu 1, got %d", got)
	}
}

func TestComputeBulldozer_LowKDA(t *testing.T) {
	ctx := domain.CitationContext{
		Stats:    map[string]float64{"kda": 7.9},
		Playlist: "slayer",
		Awards:   map[string]int{},
		Medals:   map[int64]int{},
	}
	if got := computeBulldozer(ctx); got != 0 {
		t.Errorf("KDA trop faible — attendu 0, got %d", got)
	}
}

func TestComputeBulldozer_BTBExcluded(t *testing.T) {
	ctx := domain.CitationContext{
		Stats:    map[string]float64{"kda": 10.0},
		Playlist: "btb slayer",
		Awards:   map[string]int{},
		Medals:   map[int64]int{},
	}
	if got := computeBulldozer(ctx); got != 0 {
		t.Errorf("BTB doit être exclu — attendu 0, got %d", got)
	}
}

func TestComputeWinsCTF_Win(t *testing.T) {
	ctx := domain.CitationContext{
		Outcome:  2,
		Playlist: "ctf ranked",
		Awards:   map[string]int{},
		Medals:   map[int64]int{},
		Stats:    map[string]float64{},
	}
	if got := computeWinsCTF(ctx); got != 1 {
		t.Errorf("attendu 1, got %d", got)
	}
}

func TestComputeWinsCTF_Loss(t *testing.T) {
	ctx := domain.CitationContext{
		Outcome:  3,
		Playlist: "ctf",
		Awards:   map[string]int{},
		Medals:   map[int64]int{},
		Stats:    map[string]float64{},
	}
	if got := computeWinsCTF(ctx); got != 0 {
		t.Errorf("défaite — attendu 0, got %d", got)
	}
}

func TestComputeAnnexionForcee_EventsWalk(t *testing.T) {
	ctx := domain.CitationContext{
		PlayerXUID: "xuid1",
		Events: []domain.CitationEventRow{
			{EventType: "mode", XUID: "", TimeMS: 1},
			{EventType: "mode", XUID: "", TimeMS: 2},
			{EventType: "mode", XUID: "", TimeMS: 3},       // streak=3 → +1
			{EventType: "death", XUID: "xuid1", TimeMS: 4}, // reset
			{EventType: "mode", XUID: "", TimeMS: 5},
			{EventType: "mode", XUID: "", TimeMS: 6},
			{EventType: "mode", XUID: "", TimeMS: 7}, // streak=3 → +1
		},
		Awards: map[string]int{},
		Medals: map[int64]int{},
		Stats:  map[string]float64{},
	}
	if got := computeAnnexionForcee(ctx); got != 2 {
		t.Errorf("attendu 2, got %d", got)
	}
}

func TestComputeAnnexionForcee_FallbackAwards(t *testing.T) {
	ctx := domain.CitationContext{
		PlayerXUID: "xuid1",
		Events:     nil, // pas d'events → fallback
		Awards:     map[string]int{"zone_captured": 9},
		Medals:     map[int64]int{},
		Stats:      map[string]float64{},
	}
	if got := computeAnnexionForcee(ctx); got != 3 {
		t.Errorf("attendu 3 (9/3), got %d", got)
	}
}

func TestComputeHijack_PrefixAndContains(t *testing.T) {
	ctx := domain.CitationContext{
		Awards: map[string]int{
			"hijacked_mongoose": 2,
			"skyjack_pilot":     1,
		},
		Medals: map[int64]int{},
		Stats:  map[string]float64{},
	}
	// hijacked_ → 2, skyjack → 1
	if got := computeHijack(ctx); got < 3 {
		t.Errorf("attendu ≥3, got %d", got)
	}
}

func TestComputeVandalism_DestroyedPrefix(t *testing.T) {
	ctx := domain.CitationContext{
		Awards: map[string]int{
			"destroyed_ghost":  3,
			"destroyed_wraith": 1,
		},
		Medals: map[int64]int{},
		Stats:  map[string]float64{},
	}
	if got := computeVandalism(ctx); got == 0 {
		t.Errorf("attendu > 0, got 0")
	}
}

// =============================================================================
// Câblage citation → moteur (I7) : player_vs_everything = victoires Firefight
// =============================================================================

// TestPlayerVsEverything_FirefightWinsThroughEngine vérifie le câblage complet de
// la décision I7 : la citation player_vs_everything (MappingType custom →
// compute_wins_firefight, enregistrée via l'init() de ce package) traverse
// ComputeFullMatchCitations et produit +1 par match Firefight GAGNÉ, 0 pour une
// défaite Firefight et 0 pour une victoire PvP. Le comportement de la fonction
// elle-même est couvert séparément (citations_custom_helpers_test.go) ; ce test
// fige la liaison mapping ↔ dispatcher ↔ cap de palier.
func TestPlayerVsEverything_FirefightWinsThroughEngine(t *testing.T) {
	fn := "compute_wins_firefight"
	tt := "5,10,15,25,50"
	mappings := []domain.CitationFullMapping{{
		NameNorm:       "player_vs_everything",
		MappingType:    domain.CitationMappingTypeCustom,
		CustomFunction: &fn,
		TierTargets:    &tt,
	}}

	ctxFor := func(outcome int, firefight bool, playlist string) domain.CitationContext {
		return domain.CitationContext{
			Outcome:     outcome,
			IsFirefight: firefight,
			Playlist:    playlist,
			Stats:       map[string]float64{},
			Awards:      map[string]int{},
			Medals:      map[int64]int{},
		}
	}
	deltaFor := func(deltas []domain.CitationMatchDelta) int {
		for _, d := range deltas {
			if d.NameNorm == "player_vs_everything" {
				return d.Value
			}
		}
		return 0
	}

	cases := []struct {
		name      string
		outcome   int
		firefight bool
		playlist  string
		want      int
	}{
		{"firefight gagné", domain.OutcomeWin, true, "firefight: kotr", 1},
		{"firefight perdu", domain.OutcomeLoss, true, "firefight: kotr", 0},
		{"pvp gagné", domain.OutcomeWin, false, "team slayer", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			deltas := analysis.ComputeFullMatchCitations(
				analysis.CitationProgressInput{Ctx: ctxFor(c.outcome, c.firefight, c.playlist)},
				mappings,
			)
			if got := deltaFor(deltas); got != c.want {
				t.Errorf("%s: player_vs_everything delta = %d, want %d", c.name, got, c.want)
			}
		})
	}
}
