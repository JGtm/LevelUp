// Package analysis — home_canonical_helpers_test.go : tests unitaires pour les
// helpers transverses de la projection canonique (derefIntZero,
// canonicalOutcomeToInt, cleanAssetLabel, assetLabels, modeLabels,
// dominantNameFromRows) — audit #4 round 2.
package analysis

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// ─── derefIntZero ─────────────────────────────────────────────────────────

func TestDerefIntZero_Nil(t *testing.T) {
	t.Parallel()
	if got := derefIntZero(nil); got != 0 {
		t.Errorf("derefIntZero(nil) = %d, want 0", got)
	}
}

func TestDerefIntZero_Zero(t *testing.T) {
	t.Parallel()
	v := 0
	if got := derefIntZero(&v); got != 0 {
		t.Errorf("derefIntZero(&0) = %d, want 0", got)
	}
}

func TestDerefIntZero_Positive(t *testing.T) {
	t.Parallel()
	v := 42
	if got := derefIntZero(&v); got != 42 {
		t.Errorf("derefIntZero(&42) = %d, want 42", got)
	}
}

func TestDerefIntZero_Negative(t *testing.T) {
	t.Parallel()
	// Pas de clamping : la fonction est un simple deref, pas de logique métier.
	v := -10
	if got := derefIntZero(&v); got != -10 {
		t.Errorf("derefIntZero(&-10) = %d, want -10", got)
	}
}

// ─── canonicalOutcomeToInt ────────────────────────────────────────────────

func TestCanonicalOutcomeToInt_AllKnown(t *testing.T) {
	t.Parallel()
	cases := map[canonical.Outcome]int{
		canonical.OutcomeWin:  domain.OutcomeWin,
		canonical.OutcomeLoss: domain.OutcomeLoss,
		canonical.OutcomeTie:  domain.OutcomeDraw,
		canonical.OutcomeDNF:  domain.OutcomeDNF,
	}
	for outcome, want := range cases {
		if got := canonicalOutcomeToInt(outcome); got != want {
			t.Errorf("canonicalOutcomeToInt(%q) = %d, want %d", outcome, got, want)
		}
	}
}

func TestCanonicalOutcomeToInt_Empty(t *testing.T) {
	t.Parallel()
	// Outcome vide → 0 (pas un domain.Outcome valide).
	if got := canonicalOutcomeToInt(canonical.Outcome("")); got != 0 {
		t.Errorf("canonicalOutcomeToInt(empty) = %d, want 0", got)
	}
}

func TestCanonicalOutcomeToInt_Unknown(t *testing.T) {
	t.Parallel()
	if got := canonicalOutcomeToInt(canonical.Outcome("nope")); got != 0 {
		t.Errorf("canonicalOutcomeToInt(nope) = %d, want 0", got)
	}
}

// ─── cleanAssetLabel ──────────────────────────────────────────────────────

func TestCleanAssetLabel_NormalString(t *testing.T) {
	t.Parallel()
	if got := cleanAssetLabel("Bazaar"); got != "Bazaar" {
		t.Errorf("cleanAssetLabel(Bazaar) = %q, want Bazaar", got)
	}
}

func TestCleanAssetLabel_TrimsWhitespace(t *testing.T) {
	t.Parallel()
	if got := cleanAssetLabel("  Bazaar  "); got != "Bazaar" {
		t.Errorf("cleanAssetLabel(   Bazaar   ) = %q, want Bazaar", got)
	}
}

func TestCleanAssetLabel_Empty(t *testing.T) {
	t.Parallel()
	if got := cleanAssetLabel(""); got != "" {
		t.Errorf("cleanAssetLabel(empty) = %q, want empty", got)
	}
}

func TestCleanAssetLabel_WhitespaceOnly(t *testing.T) {
	t.Parallel()
	if got := cleanAssetLabel("   "); got != "" {
		t.Errorf("cleanAssetLabel(whitespace) = %q, want empty", got)
	}
}

func TestCleanAssetLabel_RejectsUUID(t *testing.T) {
	t.Parallel()
	uuid := "3e1e4cec-4f2c-44c6-b8d2-96b85c66c702"
	if got := cleanAssetLabel(uuid); got != "" {
		t.Errorf("cleanAssetLabel(UUID) = %q, want empty", got)
	}
}

func TestCleanAssetLabel_RejectsUUIDUpper(t *testing.T) {
	t.Parallel()
	// Case-insensitive match du regex UUID.
	uuid := "3E1E4CEC-4F2C-44C6-B8D2-96B85C66C702"
	if got := cleanAssetLabel(uuid); got != "" {
		t.Errorf("cleanAssetLabel(UUID upper) = %q, want empty", got)
	}
}

// ─── assetLabels ──────────────────────────────────────────────────────────

func TestAssetLabels_Nil(t *testing.T) {
	t.Parallel()
	en, fr := assetLabels(nil)
	if en != "" || fr != "" {
		t.Errorf("assetLabels(nil) = (%q, %q), want (empty, empty)", en, fr)
	}
}

func TestAssetLabels_OnlyDefaultLabel(t *testing.T) {
	t.Parallel()
	ref := &canonical.AssetReference{DefaultLabel: "Bazaar"}
	en, fr := assetLabels(ref)
	if en != "Bazaar" || fr != "" {
		t.Errorf("assetLabels(default only) = (%q, %q), want (Bazaar, empty)", en, fr)
	}
}

func TestAssetLabels_LabelsOverrideDefault(t *testing.T) {
	t.Parallel()
	ref := &canonical.AssetReference{
		DefaultLabel: "FallbackEN",
		Labels:       map[string]string{"en": "Bazaar", "fr": "Bazaar FR"},
	}
	en, fr := assetLabels(ref)
	if en != "Bazaar" {
		t.Errorf("EN: got %q, want Bazaar (Labels[en] should override DefaultLabel)", en)
	}
	if fr != "Bazaar FR" {
		t.Errorf("FR: got %q, want Bazaar FR", fr)
	}
}

func TestAssetLabels_EmptyLabelsKeepDefault(t *testing.T) {
	t.Parallel()
	// Labels["en"] empty → keep DefaultLabel via cleanAssetLabel("") returning "".
	ref := &canonical.AssetReference{
		DefaultLabel: "FallbackEN",
		Labels:       map[string]string{"en": "  "},
	}
	en, _ := assetLabels(ref)
	if en != "FallbackEN" {
		t.Errorf("EN with whitespace label: got %q, want FallbackEN (fallback)", en)
	}
}

func TestAssetLabels_UUIDLabelsRejected(t *testing.T) {
	t.Parallel()
	uuid := "3e1e4cec-4f2c-44c6-b8d2-96b85c66c702"
	ref := &canonical.AssetReference{
		DefaultLabel: "Bazaar",
		Labels:       map[string]string{"en": uuid, "fr": uuid},
	}
	en, fr := assetLabels(ref)
	if en != "Bazaar" {
		t.Errorf("UUID en should be rejected, fallback to DefaultLabel; got en=%q", en)
	}
	if fr != "" {
		t.Errorf("UUID fr should be empty; got fr=%q", fr)
	}
}

func TestAssetLabels_NoFRLabel(t *testing.T) {
	t.Parallel()
	ref := &canonical.AssetReference{
		DefaultLabel: "Bazaar",
		Labels:       map[string]string{"en": "Bazaar"},
	}
	_, fr := assetLabels(ref)
	if fr != "" {
		t.Errorf("missing FR: got fr=%q, want empty", fr)
	}
}

// ─── modeLabels ───────────────────────────────────────────────────────────

func TestModeLabels_PrefersPairMode(t *testing.T) {
	t.Parallel()
	// PairMode défini → utilisé en priorité, GameVariant ignoré.
	r := canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			PairMode:    &canonical.AssetReference{DefaultLabel: "Arena:Slayer"},
			GameVariant: &canonical.AssetReference{DefaultLabel: "Slayer Variant"},
		},
	}
	en, _ := modeLabels(r)
	if en != "Arena:Slayer" {
		t.Errorf("modeLabels: got %q, want Arena:Slayer (PairMode preferred)", en)
	}
}

func TestModeLabels_FallbackGameVariant(t *testing.T) {
	t.Parallel()
	// PairMode nil → fallback GameVariant.
	r := canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			GameVariant: &canonical.AssetReference{DefaultLabel: "Slayer Variant"},
		},
	}
	en, _ := modeLabels(r)
	if en != "Slayer Variant" {
		t.Errorf("modeLabels fallback: got %q, want Slayer Variant", en)
	}
}

func TestModeLabels_PairModeEmptyFallbackGameVariant(t *testing.T) {
	t.Parallel()
	// PairMode présent mais labels tous vides → fallback GameVariant.
	r := canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			PairMode:    &canonical.AssetReference{DefaultLabel: ""},
			GameVariant: &canonical.AssetReference{DefaultLabel: "Slayer Variant"},
		},
	}
	en, _ := modeLabels(r)
	if en != "Slayer Variant" {
		t.Errorf("modeLabels empty PairMode: got %q, want Slayer Variant (fallback)", en)
	}
}

func TestModeLabels_BothNil(t *testing.T) {
	t.Parallel()
	r := canonical.PlayerMatchRow{}
	en, fr := modeLabels(r)
	if en != "" || fr != "" {
		t.Errorf("modeLabels(empty): got (%q, %q), want (empty, empty)", en, fr)
	}
}

func TestModeLabels_PairModeFRFR(t *testing.T) {
	t.Parallel()
	// PairMode avec label FR → renvoie FR aussi.
	r := canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			PairMode: &canonical.AssetReference{
				DefaultLabel: "Arena:Slayer",
				Labels:       map[string]string{"en": "Arena:Slayer", "fr": "Arène:Massacre"},
			},
		},
	}
	en, fr := modeLabels(r)
	if en != "Arena:Slayer" || fr != "Arène:Massacre" {
		t.Errorf("modeLabels: got (%q, %q), want (Arena:Slayer, Arène:Massacre)", en, fr)
	}
}

// ─── dominantNameFromRows ────────────────────────────────────────────────

func TestDominantNameFromRows_Empty(t *testing.T) {
	t.Parallel()
	got := dominantNameFromRows(nil, "fr", func(r canonical.PlayerMatchRow) (string, string) {
		return "", ""
	})
	if got != nil {
		t.Errorf("dominantNameFromRows(nil) = %v, want nil", got)
	}
}

func TestDominantNameFromRows_SingleEntry(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{{}}
	got := dominantNameFromRows(rows, "fr", func(r canonical.PlayerMatchRow) (string, string) {
		return "Slayer", "Massacre"
	})
	if got == nil || *got != "Massacre" {
		t.Errorf("dominantNameFromRows(fr): got %v, want Massacre", got)
	}
}

func TestDominantNameFromRows_LocaleEN(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{{}}
	got := dominantNameFromRows(rows, "en", func(r canonical.PlayerMatchRow) (string, string) {
		return "Slayer", "Massacre"
	})
	if got == nil || *got != "Slayer" {
		t.Errorf("dominantNameFromRows(en): got %v, want Slayer", got)
	}
}

func TestDominantNameFromRows_MajorityWins(t *testing.T) {
	t.Parallel()
	rows := make([]canonical.PlayerMatchRow, 5)
	i := 0
	got := dominantNameFromRows(rows, "fr", func(r canonical.PlayerMatchRow) (string, string) {
		i++
		// 3x Slayer, 2x CTF.
		if i <= 3 {
			return "Slayer", "Massacre"
		}
		return "CTF", "Drapeau"
	})
	if got == nil || *got != "Massacre" {
		t.Errorf("dominantNameFromRows majority: got %v, want Massacre (3x vs 2x)", got)
	}
}

func TestDominantNameFromRows_TieAlphabetical(t *testing.T) {
	t.Parallel()
	rows := make([]canonical.PlayerMatchRow, 2)
	i := 0
	got := dominantNameFromRows(rows, "fr", func(r canonical.PlayerMatchRow) (string, string) {
		i++
		// 1x Slayer (key="Slayer"), 1x CTF (key="CTF") — égalité, CTF < Slayer
		// → CTF gagne.
		if i == 1 {
			return "Slayer", "Massacre"
		}
		return "CTF", "Drapeau"
	})
	if got == nil || *got != "Drapeau" {
		t.Errorf("dominantNameFromRows tie: got %v, want Drapeau (alphabetical key tie-break)", got)
	}
}

func TestDominantNameFromRows_SkipsEmpty(t *testing.T) {
	t.Parallel()
	rows := make([]canonical.PlayerMatchRow, 3)
	i := 0
	got := dominantNameFromRows(rows, "fr", func(r canonical.PlayerMatchRow) (string, string) {
		i++
		if i == 1 {
			return "", "" // ignoré
		}
		return "Slayer", "Massacre"
	})
	if got == nil || *got != "Massacre" {
		t.Errorf("dominantNameFromRows skipping empty: got %v, want Massacre", got)
	}
}

func TestDominantNameFromRows_AllEmpty(t *testing.T) {
	t.Parallel()
	rows := make([]canonical.PlayerMatchRow, 3)
	got := dominantNameFromRows(rows, "fr", func(r canonical.PlayerMatchRow) (string, string) {
		return "", ""
	})
	if got != nil {
		t.Errorf("dominantNameFromRows all empty: got %v, want nil", got)
	}
}

func TestDominantNameFromRows_KeyFromFROnly(t *testing.T) {
	t.Parallel()
	// EN vide → key = FR ; reste cohérent.
	rows := make([]canonical.PlayerMatchRow, 1)
	got := dominantNameFromRows(rows, "fr", func(r canonical.PlayerMatchRow) (string, string) {
		return "", "Massacre"
	})
	if got == nil || *got != "Massacre" {
		t.Errorf("dominantNameFromRows fr only: got %v, want Massacre", got)
	}
}
