// Package halo_infinite — citations_custom_helpers_test.go : tests des helpers
// awards (sumAwardsExact, sumAwardsWithPrefix, sumAwardsContaining) et des
// fonctions custom citations restantes (Wraith/Mongoose/Warthog Destroyer,
// FlagEmDown, Wins Firefight/Slayer/Strongholds, Vandalism) — audit #4 round 2.
package halo_infinite

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// ─── sumAwardsExact ───────────────────────────────────────────────────────

func TestSumAwardsExact_Empty(t *testing.T) {
	t.Parallel()
	if got := sumAwardsExact(nil, "a", "b"); got != 0 {
		t.Errorf("sumAwardsExact(nil) = %d, want 0", got)
	}
}

func TestSumAwardsExact_NoMatches(t *testing.T) {
	t.Parallel()
	awards := map[string]int{"foo": 5}
	if got := sumAwardsExact(awards, "bar"); got != 0 {
		t.Errorf("sumAwardsExact(no match) = %d, want 0", got)
	}
}

func TestSumAwardsExact_SingleMatch(t *testing.T) {
	t.Parallel()
	awards := map[string]int{"hijacked_warthog": 3, "destroyed_wraith": 2}
	if got := sumAwardsExact(awards, "destroyed_wraith"); got != 2 {
		t.Errorf("sumAwardsExact(single) = %d, want 2", got)
	}
}

func TestSumAwardsExact_MultipleNames(t *testing.T) {
	t.Parallel()
	awards := map[string]int{
		"destroyed_wraith": 2,
		"Wraith Destroyed": 1,
		"other":            99,
	}
	got := sumAwardsExact(awards, "destroyed_wraith", "Wraith Destroyed")
	if got != 3 {
		t.Errorf("sumAwardsExact = %d, want 3 (2+1)", got)
	}
}

// ─── sumAwardsWithPrefix ──────────────────────────────────────────────────

func TestSumAwardsWithPrefix_Empty(t *testing.T) {
	t.Parallel()
	if got := sumAwardsWithPrefix(nil, "hijacked_"); got != 0 {
		t.Errorf("sumAwardsWithPrefix(nil) = %d, want 0", got)
	}
}

func TestSumAwardsWithPrefix_Matches(t *testing.T) {
	t.Parallel()
	awards := map[string]int{
		"hijacked_warthog": 2,
		"hijacked_wraith":  3,
		"destroyed_ghost":  1, // pas matché
	}
	if got := sumAwardsWithPrefix(awards, "hijacked_"); got != 5 {
		t.Errorf("sumAwardsWithPrefix = %d, want 5 (2+3)", got)
	}
}

func TestSumAwardsWithPrefix_EmptyPrefix(t *testing.T) {
	t.Parallel()
	// Empty prefix matches all (strings.HasPrefix returns true for empty).
	awards := map[string]int{"a": 1, "b": 2}
	if got := sumAwardsWithPrefix(awards, ""); got != 3 {
		t.Errorf("empty prefix should match all: got %d, want 3", got)
	}
}

func TestSumAwardsWithPrefix_NoMatches(t *testing.T) {
	t.Parallel()
	awards := map[string]int{"foo_bar": 5}
	if got := sumAwardsWithPrefix(awards, "xxx_"); got != 0 {
		t.Errorf("no match: got %d, want 0", got)
	}
}

// ─── sumAwardsContaining ──────────────────────────────────────────────────

func TestSumAwardsContaining_Empty(t *testing.T) {
	t.Parallel()
	if got := sumAwardsContaining(nil, "hijack"); got != 0 {
		t.Errorf("sumAwardsContaining(nil) = %d, want 0", got)
	}
}

func TestSumAwardsContaining_FindsMiddle(t *testing.T) {
	t.Parallel()
	awards := map[string]int{
		"hijacked_warthog": 2, // contient "hijack"
		"skyjack_pilot":    1, // contient "jack" pas "hijack"
		"normal_kill":      5,
	}
	if got := sumAwardsContaining(awards, "hijack"); got != 2 {
		t.Errorf("got %d, want 2", got)
	}
}

func TestSumAwardsContaining_EmptySubstr(t *testing.T) {
	t.Parallel()
	// Empty substr matches all (strings.Contains returns true for empty).
	awards := map[string]int{"a": 1, "b": 2}
	if got := sumAwardsContaining(awards, ""); got != 3 {
		t.Errorf("empty substr should match all: got %d, want 3", got)
	}
}

// ─── computeWinsFirefight ─────────────────────────────────────────────────

func TestComputeWinsFirefight_PVEFlag(t *testing.T) {
	t.Parallel()
	ctx := domain.CitationContext{
		Outcome:     2,
		IsFirefight: true,
		Playlist:    "any",
		Awards:      map[string]int{},
		Stats:       map[string]float64{},
		Medals:      map[int64]int{},
	}
	if got := computeWinsFirefight(ctx); got != 1 {
		t.Errorf("PvE flag win: got %d, want 1", got)
	}
}

func TestComputeWinsFirefight_PlaylistKeyword(t *testing.T) {
	t.Parallel()
	ctx := domain.CitationContext{
		Outcome:  2,
		Playlist: "firefight: kotr",
		Awards:   map[string]int{},
		Stats:    map[string]float64{},
		Medals:   map[int64]int{},
	}
	if got := computeWinsFirefight(ctx); got != 1 {
		t.Errorf("playlist keyword: got %d, want 1", got)
	}
}

func TestComputeWinsFirefight_NotWin(t *testing.T) {
	t.Parallel()
	ctx := domain.CitationContext{
		Outcome:     3, // loss
		IsFirefight: true,
		Awards:      map[string]int{},
		Stats:       map[string]float64{},
		Medals:      map[int64]int{},
	}
	if got := computeWinsFirefight(ctx); got != 0 {
		t.Errorf("loss: got %d, want 0", got)
	}
}

func TestComputeWinsFirefight_NoFirefightContext(t *testing.T) {
	t.Parallel()
	ctx := domain.CitationContext{
		Outcome:  2,
		Playlist: "team slayer",
		Awards:   map[string]int{},
		Stats:    map[string]float64{},
		Medals:   map[int64]int{},
	}
	if got := computeWinsFirefight(ctx); got != 0 {
		t.Errorf("no firefight context: got %d, want 0", got)
	}
}

// ─── computeWinsSlayer ────────────────────────────────────────────────────

func TestComputeWinsSlayer_PlaylistSlayer(t *testing.T) {
	t.Parallel()
	ctx := domain.CitationContext{
		Outcome:  2,
		Playlist: "team slayer",
		Awards:   map[string]int{},
		Stats:    map[string]float64{},
		Medals:   map[int64]int{},
	}
	if got := computeWinsSlayer(ctx); got != 1 {
		t.Errorf("slayer playlist: got %d, want 1", got)
	}
}

func TestComputeWinsSlayer_GameVariantAssassin(t *testing.T) {
	t.Parallel()
	ctx := domain.CitationContext{
		Outcome:     2,
		GameVariant: "assassin variant",
		Awards:      map[string]int{},
		Stats:       map[string]float64{},
		Medals:      map[int64]int{},
	}
	if got := computeWinsSlayer(ctx); got != 1 {
		t.Errorf("assassin variant: got %d, want 1", got)
	}
}

func TestComputeWinsSlayer_NotSlayer(t *testing.T) {
	t.Parallel()
	ctx := domain.CitationContext{
		Outcome:  2,
		Playlist: "btb ctf",
		Awards:   map[string]int{},
		Stats:    map[string]float64{},
		Medals:   map[int64]int{},
	}
	if got := computeWinsSlayer(ctx); got != 0 {
		t.Errorf("non-slayer: got %d, want 0", got)
	}
}

// ─── computeWinsStrongholds ───────────────────────────────────────────────

func TestComputeWinsStrongholds_PlaylistStronghold(t *testing.T) {
	t.Parallel()
	ctx := domain.CitationContext{
		Outcome:  2,
		Playlist: "ranked strongholds",
		Awards:   map[string]int{},
		Stats:    map[string]float64{},
		Medals:   map[int64]int{},
	}
	if got := computeWinsStrongholds(ctx); got != 1 {
		t.Errorf("strongholds: got %d, want 1", got)
	}
}

func TestComputeWinsStrongholds_PlaylistBases(t *testing.T) {
	t.Parallel()
	// "bases" est l'équivalent FR.
	ctx := domain.CitationContext{
		Outcome:  2,
		Playlist: "bases",
		Awards:   map[string]int{},
		Stats:    map[string]float64{},
		Medals:   map[int64]int{},
	}
	if got := computeWinsStrongholds(ctx); got != 1 {
		t.Errorf("bases (FR): got %d, want 1", got)
	}
}

func TestComputeWinsStrongholds_NotWin(t *testing.T) {
	t.Parallel()
	ctx := domain.CitationContext{
		Outcome:  3,
		Playlist: "strongholds",
		Awards:   map[string]int{},
		Stats:    map[string]float64{},
		Medals:   map[int64]int{},
	}
	if got := computeWinsStrongholds(ctx); got != 0 {
		t.Errorf("loss: got %d, want 0", got)
	}
}

// ─── computeFlagEmDown ────────────────────────────────────────────────────

func TestComputeFlagEmDown_EnglishAwards(t *testing.T) {
	t.Parallel()
	ctx := domain.CitationContext{
		Awards: map[string]int{
			"Flag Carrier Kill":   2,
			"Flag Carrier Killed": 1,
		},
		Stats:  map[string]float64{},
		Medals: map[int64]int{},
	}
	if got := computeFlagEmDown(ctx); got != 3 {
		t.Errorf("english awards: got %d, want 3", got)
	}
}

func TestComputeFlagEmDown_RunnerStopped(t *testing.T) {
	t.Parallel()
	// Note : la clé FR "Porteur arrêté" est encodée en mojibake dans la source
	// historique (citations_custom.go) — on évite ce cas et on teste juste
	// l'EN snake_case + Flag Carrier Kill.
	ctx := domain.CitationContext{
		Awards: map[string]int{
			"runner_stopped":    4,
			"Flag Carrier Kill": 2,
		},
		Stats:  map[string]float64{},
		Medals: map[int64]int{},
	}
	if got := computeFlagEmDown(ctx); got != 6 {
		t.Errorf("FR awards: got %d, want 6", got)
	}
}

func TestComputeFlagEmDown_None(t *testing.T) {
	t.Parallel()
	ctx := domain.CitationContext{
		Awards: map[string]int{"unrelated": 5},
		Stats:  map[string]float64{},
		Medals: map[int64]int{},
	}
	if got := computeFlagEmDown(ctx); got != 0 {
		t.Errorf("no flag awards: got %d, want 0", got)
	}
}

// ─── computeWraithDestroyer ───────────────────────────────────────────────

func TestComputeWraithDestroyer_Sum(t *testing.T) {
	t.Parallel()
	ctx := domain.CitationContext{
		Awards: map[string]int{
			"destroyed_wraith": 2,
			"Wraith Destroyed": 1,
			"Wraith destroyed": 1,
		},
		Stats:  map[string]float64{},
		Medals: map[int64]int{},
	}
	if got := computeWraithDestroyer(ctx); got != 4 {
		t.Errorf("got %d, want 4", got)
	}
}

func TestComputeWraithDestroyer_None(t *testing.T) {
	t.Parallel()
	ctx := domain.CitationContext{
		Awards: map[string]int{"hijacked_warthog": 5},
		Stats:  map[string]float64{},
		Medals: map[int64]int{},
	}
	if got := computeWraithDestroyer(ctx); got != 0 {
		t.Errorf("no wraith: got %d, want 0", got)
	}
}

// ─── computeMongooseDestroyer ─────────────────────────────────────────────

func TestComputeMongooseDestroyer_Sum(t *testing.T) {
	t.Parallel()
	ctx := domain.CitationContext{
		Awards: map[string]int{
			"destroyed_mongoose": 3,
			"Mongoose Destroyed": 1,
		},
		Stats:  map[string]float64{},
		Medals: map[int64]int{},
	}
	if got := computeMongooseDestroyer(ctx); got != 4 {
		t.Errorf("got %d, want 4", got)
	}
}

func TestComputeMongooseDestroyer_None(t *testing.T) {
	t.Parallel()
	ctx := domain.CitationContext{
		Awards: map[string]int{"random": 5},
		Stats:  map[string]float64{},
		Medals: map[int64]int{},
	}
	if got := computeMongooseDestroyer(ctx); got != 0 {
		t.Errorf("no mongoose: got %d, want 0", got)
	}
}

// ─── computeWarthogDestroyer ──────────────────────────────────────────────

func TestComputeWarthogDestroyer_Sum(t *testing.T) {
	t.Parallel()
	// Inclut Rocket Warthog (variante).
	ctx := domain.CitationContext{
		Awards: map[string]int{
			"destroyed_warthog":        2,
			"destroyed_rocket_warthog": 1,
			"Warthog Destroyed":        1,
			"Rocket Warthog Destroyed": 1,
		},
		Stats:  map[string]float64{},
		Medals: map[int64]int{},
	}
	if got := computeWarthogDestroyer(ctx); got != 5 {
		t.Errorf("got %d, want 5", got)
	}
}

// ─── DispatchCustom ───────────────────────────────────────────────────────

func TestDispatchCustom_UnknownReturnsZero(t *testing.T) {
	t.Parallel()
	ctx := domain.CitationContext{
		Stats:  map[string]float64{},
		Awards: map[string]int{},
		Medals: map[int64]int{},
	}
	if got := DispatchCustom("nonexistent_function", ctx); got != 0 {
		t.Errorf("unknown function: got %d, want 0", got)
	}
}

func TestDispatchCustom_RoutesToCorrectFunction(t *testing.T) {
	t.Parallel()
	ctx := domain.CitationContext{
		Stats:    map[string]float64{"kda": 10.0},
		Playlist: "arena slayer",
		Awards:   map[string]int{},
		Medals:   map[int64]int{},
	}
	// compute_bulldozer doit déclencher (KDA > 8 en Slayer hors BTB/FF).
	if got := DispatchCustom("compute_bulldozer", ctx); got != 1 {
		t.Errorf("compute_bulldozer: got %d, want 1", got)
	}
}

func TestDispatchCustom_RoutesWraithDestroyer(t *testing.T) {
	t.Parallel()
	ctx := domain.CitationContext{
		Stats: map[string]float64{},
		Awards: map[string]int{
			"destroyed_wraith": 3,
		},
		Medals: map[int64]int{},
	}
	if got := DispatchCustom("compute_wraith_destroyer", ctx); got != 3 {
		t.Errorf("compute_wraith_destroyer: got %d, want 3", got)
	}
}
