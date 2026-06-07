// Package analysis — home_canonical_skill_test.go : tests unitaires pour
// buildCanonicalSkillBadge + edge cases de InferHomeSkillHistoryFromCanonical
// (audit #10 coverage extension).
package analysis

import (
	"math"
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
	if label == nil || *label != "Or III" {
		t.Errorf("label = %q, want Or III", derefStr(label))
	}
	wantURL := "/static/ranks/halo_infinite/120px-HINF-CSR_Gold3.png"
	if url == nil || *url != wantURL {
		t.Errorf("url = %q, want %q", derefStr(url), wantURL)
	}
}

func TestBuildCanonicalSkillBadge_DiamondLocalizedDisplay(t *testing.T) {
	t.Parallel()
	// Le tier code EN est utilisé pour l'URL (Diamond), display pour le label (Diamant).
	sub := 1
	label, url := buildCanonicalSkillBadge("diamant", "diamond", &sub)
	if label == nil || *label != "Diamant I" {
		t.Errorf("label = %q, want Diamant I", derefStr(label))
	}
	if url == nil || *url != "/static/ranks/halo_infinite/120px-HINF-CSR_Diamond1.png" {
		t.Errorf("url = %q", derefStr(url))
	}
}

// derefStr helper : retourne *p ou "<nil>" — assainit les error messages
// pour *string (le helper deref du package gère *float64 dans squad_breakdown.go).
func derefStr(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
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
	if label == nil || *label != "Platinum V" {
		t.Errorf("label = %q, want Platinum V", derefStr(label))
	}
	if url == nil || *url != "/static/ranks/halo_infinite/120px-HINF-CSR_Platinum5.png" {
		t.Errorf("url = %q", derefStr(url))
	}
}

func TestBuildCanonicalSkillBadge_WhitespaceDisplayUsesFallback(t *testing.T) {
	t.Parallel()
	// Display avec whitespace seulement → trim → fallback sur tierEN capitalisé.
	sub := 2
	label, url := buildCanonicalSkillBadge("   ", "bronze", &sub)
	if label == nil || *label != "Bronze II" {
		t.Errorf("label = %q, want Bronze II", derefStr(label))
	}
	if url == nil || *url != "/static/ranks/halo_infinite/120px-HINF-CSR_Bronze2.png" {
		t.Errorf("url = %q", derefStr(url))
	}
}

func TestBuildCanonicalSkillBadge_SubTier6Allowed(t *testing.T) {
	t.Parallel()
	// La borne supérieure inclusive est 6.
	sub := 6
	label, url := buildCanonicalSkillBadge("Argent", "silver", &sub)
	if label == nil || *label != "Argent VI" {
		t.Errorf("label = %q, want Argent VI", derefStr(label))
	}
	if url == nil || *url != "/static/ranks/halo_infinite/120px-HINF-CSR_Silver6.png" {
		t.Errorf("url = %q", derefStr(url))
	}
}

// ─── BuildCSRTierLabelFromEN ───────────────────────────────────────────────

func TestBuildCSRTierLabelFromEN(t *testing.T) {
	t.Parallel()
	sub3 := 3
	cases := []struct {
		name        string
		tierEN      string
		subTier     *int
		frPreferred bool
		want        string // "" → attend nil
	}{
		{"diamond FR + sous-palier", "Diamond", &sub3, true, "Diamant III"},
		{"casse normalisée (lowercase)", "diamond", &sub3, true, "Diamant III"},
		{"casse normalisée (UPPER)", "DIAMOND", &sub3, true, "Diamant III"},
		{"onyx sans sous-palier", "Onyx", &sub3, true, "Onyx"},
		{"EN quand frPreferred=false", "Diamond", &sub3, false, "Diamond III"},
		{"tier inconnu → laissé EN", "Mythic", &sub3, true, "Mythic III"},
		{"vide → nil", "", &sub3, true, ""},
		{"whitespace → nil", "   ", &sub3, true, ""},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := BuildCSRTierLabelFromEN(c.tierEN, c.subTier, c.frPreferred)
			if c.want == "" {
				if got != nil {
					t.Errorf("%s: got %q, want nil", c.name, derefStr(got))
				}
				return
			}
			if got == nil || *got != c.want {
				t.Errorf("%s: got %q, want %q", c.name, derefStr(got), c.want)
			}
		})
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

// ─── SkillTierBand ─────────────────────────────────────────────────────────

func TestSkillTierBand(t *testing.T) {
	t.Parallel()
	// Remplissage ORDINAL via le sous-palier (n/6), indépendant de l'échelle de
	// valeur (CSR ≠ LUSR ; sous-paliers LUSR sur l'échelle μ interne). Onyx = plein.
	cases := []struct {
		name         string
		tierEN       string
		subTier      int
		wantOK       bool
		wantProgress float64
	}{
		{"Gold IV → 4/6", "Gold", 4, true, 66.6667},
		{"Diamant IV → 4/6 (échelle indifférente)", "Diamond", 4, true, 66.6667},
		{"entrée de palier (I) → 1/6", "Bronze", 1, true, 16.6667},
		{"sommet de palier (VI) → 100%", "Gold", 6, true, 100},
		{"Onyx → barre pleine (sommet)", "Onyx", 0, true, 100},
		{"Onyx insensible au sous-palier", "onyx", 3, true, 100},
		{"sans rang (sub_tier 0) → pas de bande", "Diamond", 0, false, 0},
		{"placement (tier vide) → pas de bande", "", 0, false, 0},
		{"sous-palier hors borne → pas de bande", "Gold", 7, false, 0},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			pct, ok := SkillTierBand(c.tierEN, c.subTier)
			if ok != c.wantOK {
				t.Fatalf("ok = %v, want %v", ok, c.wantOK)
			}
			if !c.wantOK {
				return
			}
			if math.Abs(pct-c.wantProgress) > 0.01 {
				t.Errorf("progressPct = %v, want %v", pct, c.wantProgress)
			}
		})
	}
}

func TestNextSubTierLabel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		tierEN      string
		subTier     int
		frPreferred bool
		want        string // "" → attend nil
	}{
		{"Or I → Or II (FR)", "Gold", 1, true, "Or II"},
		{"Or V → Or VI", "Gold", 5, true, "Or VI"},
		{"Or VI → Platine I (palier suivant)", "Gold", 6, true, "Platine I"},
		{"Diamant VI → Onyx (sommet)", "Diamond", 6, true, "Onyx"},
		{"Gold I → Gold II (EN)", "Gold", 1, false, "Gold II"},
		{"casse normalisée", "diamond", 6, true, "Onyx"},
		{"Onyx → nil (déjà au sommet)", "Onyx", 0, true, ""},
		{"sous-palier 0 (placement) → nil", "Gold", 0, true, ""},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := NextSubTierLabel(c.tierEN, c.subTier, c.frPreferred)
			if c.want == "" {
				if got != nil {
					t.Errorf("got %q, want nil", derefStr(got))
				}
				return
			}
			if got == nil || *got != c.want {
				t.Errorf("got %q, want %q", derefStr(got), c.want)
			}
		})
	}
}
