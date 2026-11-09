// Package analysis — home_locale_test.go : tests unitaires pour les helpers
// locale/outcome/dominance/UUID utilisés par la projection home (audit #10
// coverage extension).
package analysis

import (
	"testing"

	"levelup/go-api/internal/i18n"

	"levelup/go-api/internal/legacymatch"
)

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

// ─── Text.PickOptional (remplace labelForLocale) ──────────────────────────

// Les libellés de l'accueil se choisissent par Text.PickOptional : la langue, puis la langue de
// repli, jamais le français pour l'anglais. Écart avec l'ancien labelForLocale (langue, repli,
// défaut) : sous l'anglais, un libellé anglais absent ne retombe plus sur le français.
func TestHomePick(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		locale string
		text   i18n.Text
		want   string
	}{
		{"en prefere en", "en", i18n.FromColumns("Hello", "Bonjour"), "Hello"},
		{"en sans anglais: pas de francais", "en", i18n.FromColumns("", "Bonjour"), ""},
		{"fr prefere fr", "fr", i18n.FromColumns("Hello", "Bonjour"), "Bonjour"},
		{"fr sans francais: repli anglais", "fr", i18n.FromColumns("Hello", ""), "Hello"},
		{"langue inconnue: langue par defaut", "unknown", i18n.FromColumns("Hello", "Bonjour"), "Bonjour"},
	}
	for _, tc := range cases {
		if got := tc.text.PickOptional(normalizeHomeLocale(tc.locale)); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

// outcomeLabelForLocale et outcomeLabel (analysis) ont été supprimés le 2026-09-07 (D5,
// lot Q4) : la clé canonique d'issue vient du titre (mappings.Canonical), jamais d'une map
// Go FR/EN. Le texte du composite Title, quand il existe, est injecté par le service depuis
// l'adapter sémantique (RecentMatchesOptions.OutcomeText) — testé côté service.

// ─── outcomeTone (round-out) ───────────────────────────────────────────────

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
		homeDominanceSabordage:        "sabordage",
		homeDominanceAbnegation:       "abnegation",
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
