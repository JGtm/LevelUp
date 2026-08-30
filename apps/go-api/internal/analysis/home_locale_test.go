// Package analysis — home_locale_test.go : tests unitaires pour les helpers
// locale/outcome/dominance/UUID utilisés par la projection home (audit #10
// coverage extension).
package analysis

import (
	"testing"

	"levelup/go-api/internal/legacymatch"
)

// ─── labelFR ─────────────────────────────────────────────────────────────

func TestLabelFR_PrefersFR(t *testing.T) {
	t.Parallel()
	if got := labelFR("Bonjour", "Hello"); got != "Bonjour" {
		t.Errorf("labelFR(Bonjour, Hello) = %q, want Bonjour", got)
	}
}

func TestLabelFR_FallbackEN(t *testing.T) {
	t.Parallel()
	if got := labelFR("", "Hello"); got != "Hello" {
		t.Errorf("labelFR(empty, Hello) = %q, want Hello", got)
	}
}

func TestLabelFR_BothEmpty(t *testing.T) {
	t.Parallel()
	if got := labelFR("", ""); got != "" {
		t.Errorf("labelFR(empty, empty) = %q, want empty", got)
	}
}

// ─── normalizeHomeLocale ──────────────────────────────────────────────────

func TestNormalizeHomeLocale_EN(t *testing.T) {
	t.Parallel()
	cases := []string{"en", "EN", "en-US", "en-GB", " en "}
	for _, in := range cases {
		if got := normalizeHomeLocale(in); got != "en" {
			t.Errorf("normalizeHomeLocale(%q) = %q, want en", in, got)
		}
	}
}

func TestNormalizeHomeLocale_FR(t *testing.T) {
	t.Parallel()
	// Toute valeur non-en (FR, vide, inconnu) doit retourner "fr" par défaut.
	cases := []string{"fr", "FR", "fr-FR", "", "  ", "unknown", "de-DE"}
	for _, in := range cases {
		if got := normalizeHomeLocale(in); got != "fr" {
			t.Errorf("normalizeHomeLocale(%q) = %q, want fr", in, got)
		}
	}
}

// ─── labelForLocale ───────────────────────────────────────────────────────

func TestLabelForLocale_EN_PrefersEN(t *testing.T) {
	t.Parallel()
	if got := labelForLocale("en", "Bonjour", "Hello"); got != "Hello" {
		t.Errorf("labelForLocale(en) = %q, want Hello", got)
	}
}

func TestLabelForLocale_EN_FallbackFR(t *testing.T) {
	t.Parallel()
	if got := labelForLocale("en", "Bonjour", ""); got != "Bonjour" {
		t.Errorf("labelForLocale(en, FR-only) = %q, want Bonjour", got)
	}
}

func TestLabelForLocale_FR(t *testing.T) {
	t.Parallel()
	if got := labelForLocale("fr", "Bonjour", "Hello"); got != "Bonjour" {
		t.Errorf("labelForLocale(fr) = %q, want Bonjour", got)
	}
}

func TestLabelForLocale_FR_FallbackEN(t *testing.T) {
	t.Parallel()
	if got := labelForLocale("fr", "", "Hello"); got != "Hello" {
		t.Errorf("labelForLocale(fr, EN-only) = %q, want Hello", got)
	}
}

// ─── outcomeLabelForLocale ────────────────────────────────────────────────

func TestOutcomeLabelForLocale_EN_Win(t *testing.T) {
	t.Parallel()
	if got := outcomeLabelForLocale(homeOutcomeWin, "en"); got != "Victory" {
		t.Errorf("outcomeLabelForLocale(WIN, en) = %q, want Victory", got)
	}
}

func TestOutcomeLabelForLocale_EN_Loss(t *testing.T) {
	t.Parallel()
	if got := outcomeLabelForLocale(homeOutcomeLoss, "en"); got != "Defeat" {
		t.Errorf("outcomeLabelForLocale(LOSS, en) = %q, want Defeat", got)
	}
}

func TestOutcomeLabelForLocale_EN_Unknown(t *testing.T) {
	t.Parallel()
	if got := outcomeLabelForLocale(99, "en"); got != "Match" {
		t.Errorf("outcomeLabelForLocale(99, en) = %q, want Match", got)
	}
}

func TestOutcomeLabelForLocale_FR_Win(t *testing.T) {
	t.Parallel()
	got := outcomeLabelForLocale(homeOutcomeWin, "fr")
	if got == "" || got == "Match" {
		t.Errorf("outcomeLabelForLocale(WIN, fr) = %q, want non-empty FR label", got)
	}
}

func TestOutcomeLabelForLocale_FR_Unknown(t *testing.T) {
	t.Parallel()
	if got := outcomeLabelForLocale(99, "fr"); got != "Match" {
		t.Errorf("outcomeLabelForLocale(99, fr) = %q, want Match", got)
	}
}

// ─── outcomeLabel & outcomeTone (round-out) ───────────────────────────────

func TestOutcomeLabel_AllKnown(t *testing.T) {
	t.Parallel()
	for _, code := range []int{homeOutcomeWin, homeOutcomeLoss, homeOutcomeTie, homeOutcomeDNF} {
		if got := outcomeLabel(code); got == "" {
			t.Errorf("outcomeLabel(%d) returned empty", code)
		}
	}
}

func TestOutcomeTone_AllKnown(t *testing.T) {
	t.Parallel()
	want := map[int]string{
		homeOutcomeWin:  OutcomeToneWin,
		homeOutcomeLoss: OutcomeToneLoss,
		homeOutcomeTie:  OutcomeToneTie,
		homeOutcomeDNF:  OutcomeToneDNF,
	}
	for code, w := range want {
		if got := outcomeTone(code); got != w {
			t.Errorf("outcomeTone(%d) = %q, want %q", code, got, w)
		}
	}
}

// ─── buildHomeScoreLabel ──────────────────────────────────────────────────

func TestBuildHomeScoreLabel_TeamZero(t *testing.T) {
	t.Parallel()
	m := legacymatch.HomeMatchRow{TeamID: 0, Team0Score: 50, Team1Score: 30}
	got := buildHomeScoreLabel(m)
	if got == nil || *got != "50 - 30" {
		t.Errorf("buildHomeScoreLabel(team0) = %v, want 50 - 30", got)
	}
}

func TestBuildHomeScoreLabel_TeamOne_Swaps(t *testing.T) {
	t.Parallel()
	// TeamID=1 doit inverser l'ordre des scores (perspective du joueur).
	m := legacymatch.HomeMatchRow{TeamID: 1, Team0Score: 50, Team1Score: 30}
	got := buildHomeScoreLabel(m)
	if got == nil || *got != "30 - 50" {
		t.Errorf("buildHomeScoreLabel(team1) = %v, want 30 - 50", got)
	}
}

func TestBuildHomeScoreLabel_NegativeReturnsNil(t *testing.T) {
	t.Parallel()
	// Un score négatif signale "score indisponible" → nil.
	m := legacymatch.HomeMatchRow{TeamID: 0, Team0Score: -1, Team1Score: 30}
	if got := buildHomeScoreLabel(m); got != nil {
		t.Errorf("buildHomeScoreLabel(negative) = %v, want nil", got)
	}
}

func TestBuildHomeScoreLabel_ZeroZero(t *testing.T) {
	t.Parallel()
	// 0-0 reste un score valide (égalité, match court).
	m := legacymatch.HomeMatchRow{TeamID: 0, Team0Score: 0, Team1Score: 0}
	got := buildHomeScoreLabel(m)
	if got == nil || *got != "0 - 0" {
		t.Errorf("buildHomeScoreLabel(0-0) = %v, want 0 - 0", got)
	}
}

// ─── buildHomeNarrativeBadges ─────────────────────────────────────────────

func TestBuildHomeNarrativeBadges_AllFlags(t *testing.T) {
	t.Parallel()
	cases := map[int]string{
		homeDominanceDomination:       "dominant",
		homeDominanceHumiliation:      "humiliation",
		homeDominanceRemontada:        "remontada",
		homeDominanceDebacle:          "debacle",
		homeDominanceCounterRemontada: "contre_remontada",
	}
	for flag, want := range cases {
		got := buildHomeNarrativeBadges(flag)
		if len(got) != 1 || got[0] != want {
			t.Errorf("buildHomeNarrativeBadges(%d) = %v, want [%q]", flag, got, want)
		}
	}
}

func TestBuildHomeNarrativeBadges_NoneReturnsNil(t *testing.T) {
	t.Parallel()
	// 0 (DominanceNone) et toute valeur inconnue retournent nil.
	for _, flag := range []int{0, 99, -1} {
		if got := buildHomeNarrativeBadges(flag); got != nil {
			t.Errorf("buildHomeNarrativeBadges(%d) = %v, want nil", flag, got)
		}
	}
}

// ─── normalizeHomeModeLabel (alias) ──────────────────────────────────────

func TestNormalizeHomeModeLabel_DelegatesToNormalize(t *testing.T) {
	t.Parallel()
	// Alias trivial : doit produire le même résultat que NormalizeModeLabel.
	in := "Arena:Slayer on Bazaar"
	if normalizeHomeModeLabel(in) != NormalizeModeLabel(in) {
		t.Errorf("normalizeHomeModeLabel doit déléguer à NormalizeModeLabel")
	}
}

func TestNormalizeHomeModeLabel_PassesMapLabels(t *testing.T) {
	t.Parallel()
	// L'alias doit transmettre les mapLabels variadic à NormalizeModeLabel.
	got := normalizeHomeModeLabel("Arena:Slayer on Bazaar", "Bazaar")
	if got != "Slayer" {
		t.Errorf("normalizeHomeModeLabel(Arena:Slayer on Bazaar, Bazaar) = %q, want Slayer", got)
	}
}

// ─── copyOptionalString ───────────────────────────────────────────────────

func TestCopyOptionalString_NilReturnsNil(t *testing.T) {
	t.Parallel()
	if got := copyOptionalString(nil); got != nil {
		t.Errorf("copyOptionalString(nil) = %v, want nil", got)
	}
}

func TestCopyOptionalString_TrimsWhitespace(t *testing.T) {
	t.Parallel()
	in := "  hello  "
	got := copyOptionalString(&in)
	if got == nil || *got != "hello" {
		t.Errorf("copyOptionalString(   hello   ) = %v, want hello", got)
	}
}

func TestCopyOptionalString_EmptyAfterTrimReturnsNil(t *testing.T) {
	t.Parallel()
	in := "   "
	if got := copyOptionalString(&in); got != nil {
		t.Errorf("copyOptionalString(whitespace) = %v, want nil", got)
	}
}

func TestCopyOptionalString_NewAddress(t *testing.T) {
	t.Parallel()
	// Le résultat ne doit PAS pointer vers la même adresse que l'entrée
	// (un trim crée une nouvelle string). Empêche les bugs alias-mutation.
	in := "value"
	got := copyOptionalString(&in)
	if got == nil {
		t.Fatal("copyOptionalString returned nil")
	}
	if got == &in {
		t.Errorf("copyOptionalString should return a new pointer, got same address")
	}
}

// ─── optionalStringValue ──────────────────────────────────────────────────

func TestOptionalStringValue_Nil(t *testing.T) {
	t.Parallel()
	if got := optionalStringValue(nil); got != "" {
		t.Errorf("optionalStringValue(nil) = %q, want empty", got)
	}
}

func TestOptionalStringValue_NonNil(t *testing.T) {
	t.Parallel()
	v := "value"
	if got := optionalStringValue(&v); got != "value" {
		t.Errorf("optionalStringValue(value) = %q, want value", got)
	}
}

func TestOptionalStringValue_EmptyPointer(t *testing.T) {
	t.Parallel()
	// Pointeur vers chaîne vide → retourne "" (pas nil-safe différenciation).
	v := ""
	if got := optionalStringValue(&v); got != "" {
		t.Errorf("optionalStringValue(empty ptr) = %q, want empty", got)
	}
}

// ─── homeUUIDRe (regex UUID v4) ───────────────────────────────────────────

func TestHomeUUIDRe_ValidUUID(t *testing.T) {
	t.Parallel()
	cases := []string{
		"12345678-abcd-1234-9876-1234567890ab",
		"12345678-ABCD-1234-9876-1234567890AB", // case-insensitive
		"00000000-0000-0000-0000-000000000000",
		"ffffffff-ffff-ffff-ffff-ffffffffffff",
	}
	for _, in := range cases {
		if !homeUUIDRe.MatchString(in) {
			t.Errorf("homeUUIDRe should match %q", in)
		}
	}
}

func TestHomeUUIDRe_InvalidUUID(t *testing.T) {
	t.Parallel()
	cases := []string{
		"",                                       // vide
		"not-a-uuid",                             // forme incorrecte
		"12345678-abcd-1234-9876",                // trop court
		"12345678-abcd-1234-9876-1234567890abcd", // trop long
		"GGGGGGGG-abcd-1234-9876-1234567890ab",   // hors hex
		"Bazaar",                                 // nom de map ordinaire
	}
	for _, in := range cases {
		if homeUUIDRe.MatchString(in) {
			t.Errorf("homeUUIDRe should NOT match %q", in)
		}
	}
}
