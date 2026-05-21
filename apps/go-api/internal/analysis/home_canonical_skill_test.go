// Package analysis — home_canonical_skill_test.go : tests unitaires pour
// buildCanonicalSkillBadge + edge cases de InferHomeSkillHistoryFromCanonical
// (audit #10 coverage extension).
package analysis

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
)

// ─── buildCanonicalSkillBadge ──────────────────────────────────────────────

func TestBuildCanonicalSkillBadge_OnyxIgnoresSubTier(t *testing.T) {
	t.Parallel()
	// Onyx n'a pas de sub-tier ; même avec subTier != nil l'URL doit pointer
	// vers le badge Onyx fixe.
	sub := 3
	label, url := buildCanonicalSkillBadge("Onyx", "onyx", &sub)
	if label == nil || *label != "Onyx" {
		t.Errorf("label = %v, want Onyx", label)
	}
	if url == nil || *url != csrRankImageOnyxURL {
		t.Errorf("url = %v, want %q", url, csrRankImageOnyxURL)
	}
}

func TestBuildCanonicalSkillBadge_OnyxCaseInsensitive(t *testing.T) {
	t.Parallel()
	// La comparaison Onyx doit être case-insensitive (EqualFold) côté tierEN.
	label, url := buildCanonicalSkillBadge("ONYX", "ONYX", nil)
	if label == nil || *label != "ONYX" {
		t.Errorf("label = %v, want ONYX", label)
	}
	if url == nil || *url != csrRankImageOnyxURL {
		t.Errorf("url = %v, want Onyx URL", url)
	}
}

func TestBuildCanonicalSkillBadge_GoldWithSubTier(t *testing.T) {
	t.Parallel()
	sub := 3
	label, url := buildCanonicalSkillBadge("Or", "gold", &sub)
	if label == nil || *label != "Or 3" {
		t.Errorf("label = %v, want Or 3", label)
	}
	wantURL := "/static/ranks/halo_infinite/120px-HINF-CSR_Gold3.png"
	if url == nil || *url != wantURL {
		t.Errorf("url = %v, want %q", url, wantURL)
	}
}

func TestBuildCanonicalSkillBadge_DiamondLocalizedDisplay(t *testing.T) {
	t.Parallel()
	// Le tier code EN est utilisé pour l'URL (Diamond), display pour le label (Diamant).
	sub := 1
	label, url := buildCanonicalSkillBadge("diamant", "diamond", &sub)
	if label == nil || *label != "Diamant 1" {
		t.Errorf("label = %v, want Diamant 1", label)
	}
	if url == nil || *url != "/static/ranks/halo_infinite/120px-HINF-CSR_Diamond1.png" {
		t.Errorf("url = %v", url)
	}
}

func TestBuildCanonicalSkillBadge_EmptyTierCodeReturnsNil(t *testing.T) {
	t.Parallel()
	// Sans tier code EN → impossible de construire l'URL → (nil, nil).
	sub := 3
	label, url := buildCanonicalSkillBadge("Or", "", &sub)
	if label != nil || url != nil {
		t.Errorf("(label, url) = (%v, %v), want (nil, nil) when tierEN empty", label, url)
	}
}

func TestBuildCanonicalSkillBadge_WhitespaceTierCodeReturnsNil(t *testing.T) {
	t.Parallel()
	sub := 3
	label, url := buildCanonicalSkillBadge("Or", "   ", &sub)
	if label != nil || url != nil {
		t.Errorf("(label, url) = (%v, %v), want (nil, nil)", label, url)
	}
}

func TestBuildCanonicalSkillBadge_NilSubTierNonOnyx(t *testing.T) {
	t.Parallel()
	// Non-Onyx sans subTier → st = 0 → en-dehors de [1..6] → (nil, nil).
	label, url := buildCanonicalSkillBadge("Or", "gold", nil)
	if label != nil || url != nil {
		t.Errorf("(label, url) = (%v, %v), want (nil, nil)", label, url)
	}
}

func TestBuildCanonicalSkillBadge_SubTierOutOfRange(t *testing.T) {
	t.Parallel()
	// SubTier hors [1..6] doit retourner (nil, nil).
	for _, st := range []int{0, -1, 7, 100} {
		stCopy := st
		label, url := buildCanonicalSkillBadge("Or", "gold", &stCopy)
		if label != nil || url != nil {
			t.Errorf("subTier=%d: (label, url) = (%v, %v), want (nil, nil)", st, label, url)
		}
	}
}

func TestBuildCanonicalSkillBadge_EmptyDisplayUsesTierENCap(t *testing.T) {
	t.Parallel()
	// Si display vide → fallback sur tierEN capitalisé.
	sub := 5
	label, url := buildCanonicalSkillBadge("", "platinum", &sub)
	if label == nil || *label != "Platinum 5" {
		t.Errorf("label = %v, want Platinum 5", label)
	}
	if url == nil || *url != "/static/ranks/halo_infinite/120px-HINF-CSR_Platinum5.png" {
		t.Errorf("url = %v", url)
	}
}

func TestBuildCanonicalSkillBadge_WhitespaceDisplayUsesFallback(t *testing.T) {
	t.Parallel()
	// Display avec whitespace seulement → trim → fallback sur tierEN capitalisé.
	sub := 2
	label, url := buildCanonicalSkillBadge("   ", "bronze", &sub)
	if label == nil || *label != "Bronze 2" {
		t.Errorf("label = %v, want Bronze 2", label)
	}
	if url == nil || *url != "/static/ranks/halo_infinite/120px-HINF-CSR_Bronze2.png" {
		t.Errorf("url = %v", url)
	}
}

func TestBuildCanonicalSkillBadge_SubTier6Allowed(t *testing.T) {
	t.Parallel()
	// La borne supérieure inclusive est 6.
	sub := 6
	label, url := buildCanonicalSkillBadge("Argent", "silver", &sub)
	if label == nil || *label != "Argent 6" {
		t.Errorf("label = %v, want Argent 6", label)
	}
	if url == nil || *url != "/static/ranks/halo_infinite/120px-HINF-CSR_Silver6.png" {
		t.Errorf("url = %v", url)
	}
}

// ─── InferHomeSkillHistoryFromCanonical : edge cases ──────────────────────

func TestInferHomeSkillHistoryFromCanonical_EmptyRows(t *testing.T) {
	t.Parallel()
	r, u := InferHomeSkillHistoryFromCanonical(nil)
	if r || u {
		t.Errorf("empty input → (false, false), got (%v, %v)", r, u)
	}
}

func TestInferHomeSkillHistoryFromCanonical_OnlyPvE(t *testing.T) {
	t.Parallel()
	// Tous PvE → exclus → (false, false).
	pve := true
	rows := []canonical.PlayerMatchRow{
		{Summary: canonical.MatchSummary{IsPvE: &pve}},
		{Summary: canonical.MatchSummary{IsPvE: &pve}},
	}
	r, u := InferHomeSkillHistoryFromCanonical(rows)
	if r || u {
		t.Errorf("PvE only → (false, false), got (%v, %v)", r, u)
	}
}

func TestInferHomeSkillHistoryFromCanonical_OnlyRanked(t *testing.T) {
	t.Parallel()
	ranked := true
	rows := []canonical.PlayerMatchRow{
		{Summary: canonical.MatchSummary{IsRanked: &ranked}},
	}
	r, u := InferHomeSkillHistoryFromCanonical(rows)
	if !r || u {
		t.Errorf("ranked only → (true, false), got (%v, %v)", r, u)
	}
}

func TestInferHomeSkillHistoryFromCanonical_OnlyUnranked(t *testing.T) {
	t.Parallel()
	ranked := false
	rows := []canonical.PlayerMatchRow{
		{Summary: canonical.MatchSummary{IsRanked: &ranked}},
	}
	r, u := InferHomeSkillHistoryFromCanonical(rows)
	if r || !u {
		t.Errorf("unranked only → (false, true), got (%v, %v)", r, u)
	}
}

func TestInferHomeSkillHistoryFromCanonical_NilIsRankedDefaultsUnranked(t *testing.T) {
	t.Parallel()
	// IsRanked == nil → considéré comme unranked (par défaut Go bool == false).
	rows := []canonical.PlayerMatchRow{
		{Summary: canonical.MatchSummary{IsRanked: nil}},
	}
	r, u := InferHomeSkillHistoryFromCanonical(rows)
	if r || !u {
		t.Errorf("nil IsRanked → (false, true), got (%v, %v)", r, u)
	}
}

func TestInferHomeSkillHistoryFromCanonical_EarlyExitWhenBothFound(t *testing.T) {
	t.Parallel()
	// Dès qu'on a ranked + unranked, l'algorithme break out.
	ranked := true
	unranked := false
	rows := []canonical.PlayerMatchRow{
		{Summary: canonical.MatchSummary{IsRanked: &ranked}},
		{Summary: canonical.MatchSummary{IsRanked: &unranked}},
		{Summary: canonical.MatchSummary{IsRanked: &ranked}}, // pas atteint, mais valide
	}
	r, u := InferHomeSkillHistoryFromCanonical(rows)
	if !r || !u {
		t.Errorf("mixed → (true, true), got (%v, %v)", r, u)
	}
}

func TestInferHomeSkillHistoryFromCanonical_PvEFalseExplicit(t *testing.T) {
	t.Parallel()
	// IsPvE = false (explicite) ne doit pas exclure.
	pve := false
	ranked := true
	rows := []canonical.PlayerMatchRow{
		{Summary: canonical.MatchSummary{IsPvE: &pve, IsRanked: &ranked}},
	}
	r, u := InferHomeSkillHistoryFromCanonical(rows)
	if !r || u {
		t.Errorf("PvE=false explicit → (true, false), got (%v, %v)", r, u)
	}
}

func TestInferHomeSkillHistoryFromCanonical_NilPvEDoesNotExclude(t *testing.T) {
	t.Parallel()
	// IsPvE = nil → row pas exclue (defensive: pas de présomption PvE).
	ranked := true
	rows := []canonical.PlayerMatchRow{
		{Summary: canonical.MatchSummary{IsPvE: nil, IsRanked: &ranked}},
	}
	r, _ := InferHomeSkillHistoryFromCanonical(rows)
	if !r {
		t.Errorf("nil IsPvE shouldn't exclude row")
	}
}
