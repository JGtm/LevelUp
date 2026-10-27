// Package analysis — home_canonical_helpers_test.go : tests unitaires pour les
// helpers transverses de la projection canonique (derefIntZero,
// canonicalOutcomeToInt, cleanAssetLabel, assetLabels, modeLabels,
// dominantNameFromRows) — audit #4 round 2.
package analysis

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/i18n"
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

// enFr rend les libellés anglais et français d un Text (les tests historiques raisonnent en paires).
func enFr(t i18n.Text) (en, fr string) { return t[i18n.LocaleEN], t[i18n.LocaleFR] }

// ─── assetText ──────────────────────────────────────────────────────────

func TestAssetText_Nil(t *testing.T) {
	t.Parallel()
	en, fr := enFr(assetText(nil))
	if en != "" || fr != "" {
		t.Errorf("assetText(nil) = (%q, %q), want (empty, empty)", en, fr)
	}
}

func TestAssetText_OnlyDefaultLabel(t *testing.T) {
	t.Parallel()
	ref := &canonical.AssetReference{DefaultLabel: "Bazaar"}
	en, fr := enFr(assetText(ref))
	if en != "Bazaar" || fr != "" {
		t.Errorf("assetText(default only) = (%q, %q), want (Bazaar, empty)", en, fr)
	}
}

func TestAssetText_LabelsOverrideDefault(t *testing.T) {
	t.Parallel()
	ref := &canonical.AssetReference{
		DefaultLabel: "FallbackEN",
		Labels:       map[string]string{"en": "Bazaar", "fr": "Bazaar FR"},
	}
	en, fr := enFr(assetText(ref))
	if en != "Bazaar" {
		t.Errorf("EN: got %q, want Bazaar (Labels[en] should override DefaultLabel)", en)
	}
	if fr != "Bazaar FR" {
		t.Errorf("FR: got %q, want Bazaar FR", fr)
	}
}

func TestAssetText_EmptyLabelsKeepDefault(t *testing.T) {
	t.Parallel()
	// Labels["en"] empty → keep DefaultLabel via cleanAssetLabel("") returning "".
	ref := &canonical.AssetReference{
		DefaultLabel: "FallbackEN",
		Labels:       map[string]string{"en": "  "},
	}
	en, _ := enFr(assetText(ref))
	if en != "FallbackEN" {
		t.Errorf("EN with whitespace label: got %q, want FallbackEN (fallback)", en)
	}
}

func TestAssetText_UUIDLabelsRejected(t *testing.T) {
	t.Parallel()
	uuid := "3e1e4cec-4f2c-44c6-b8d2-96b85c66c702"
	ref := &canonical.AssetReference{
		DefaultLabel: "Bazaar",
		Labels:       map[string]string{"en": uuid, "fr": uuid},
	}
	en, fr := enFr(assetText(ref))
	if en != "Bazaar" {
		t.Errorf("UUID en should be rejected, fallback to DefaultLabel; got en=%q", en)
	}
	if fr != "" {
		t.Errorf("UUID fr should be empty; got fr=%q", fr)
	}
}

func TestAssetText_NoFRLabel(t *testing.T) {
	t.Parallel()
	ref := &canonical.AssetReference{
		DefaultLabel: "Bazaar",
		Labels:       map[string]string{"en": "Bazaar"},
	}
	_, fr := enFr(assetText(ref))
	if fr != "" {
		t.Errorf("missing FR: got fr=%q, want empty", fr)
	}
}

// ─── modeText ───────────────────────────────────────────────────────────

func TestModeText_LitLeModeFourniParLaSource(t *testing.T) {
	t.Parallel()
	// Le mode est PairMode ; GameVariant (type de jeu exposé à part) n'y change rien.
	r := canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			PairMode:    &canonical.AssetReference{DefaultLabel: "Arena:Slayer"},
			GameVariant: &canonical.AssetReference{DefaultLabel: "Slayer Variant"},
		},
	}
	en, _ := enFr(modeText(r))
	if en != "Arena:Slayer" {
		t.Errorf("modeText: got %q, want Arena:Slayer", en)
	}
}

func TestModeText_SansModeNeRepliePasSurGameVariant(t *testing.T) {
	t.Parallel()
	// PairMode nil, ou sans libellé : pas de mode, même si GameVariant est nommé.
	for name, pair := range map[string]*canonical.AssetReference{
		"nil":         nil,
		"sans nom":    {DefaultLabel: ""},
		"UUID brut":   {DefaultLabel: "3f2a9c1e-7b4d-4c8e-9a1f-0d5e6b7c8d9e"},
		"identifiant": {ID: "gv-1", Kind: "game_variant"},
	} {
		r := canonical.PlayerMatchRow{
			Summary: canonical.MatchSummary{
				PairMode:    pair,
				GameVariant: &canonical.AssetReference{DefaultLabel: "Slayer Variant"},
			},
		}
		if en, fr := enFr(modeText(r)); en != "" || fr != "" {
			t.Errorf("%s : modeText = (%q, %q), want vide (aucun repli sur GameVariant)", name, en, fr)
		}
	}
}

func TestModeText_BothNil(t *testing.T) {
	t.Parallel()
	r := canonical.PlayerMatchRow{}
	en, fr := enFr(modeText(r))
	if en != "" || fr != "" {
		t.Errorf("modeText(empty): got (%q, %q), want (empty, empty)", en, fr)
	}
}

func TestModeText_PairModeFRFR(t *testing.T) {
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
	en, fr := enFr(modeText(r))
	if en != "Arena:Slayer" || fr != "Arène:Massacre" {
		t.Errorf("modeLabels: got (%q, %q), want (Arena:Slayer, Arène:Massacre)", en, fr)
	}
}

// ─── dominantNameFromRows ────────────────────────────────────────────────

func TestDominantNameFromRows_Empty(t *testing.T) {
	t.Parallel()
	got := dominantNameFromRows(nil, "fr", func(r canonical.PlayerMatchRow) i18n.Text {
		return nil
	})
	if got != nil {
		t.Errorf("dominantNameFromRows(nil) = %v, want nil", got)
	}
}

func TestDominantNameFromRows_SingleEntry(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{{}}
	got := dominantNameFromRows(rows, "fr", func(r canonical.PlayerMatchRow) i18n.Text {
		return i18n.FromColumns("Slayer", "Massacre")
	})
	if got == nil || *got != "Massacre" {
		t.Errorf("dominantNameFromRows(fr): got %v, want Massacre", got)
	}
}

func TestDominantNameFromRows_LocaleEN(t *testing.T) {
	t.Parallel()
	rows := []canonical.PlayerMatchRow{{}}
	got := dominantNameFromRows(rows, "en", func(r canonical.PlayerMatchRow) i18n.Text {
		return i18n.FromColumns("Slayer", "Massacre")
	})
	if got == nil || *got != "Slayer" {
		t.Errorf("dominantNameFromRows(en): got %v, want Slayer", got)
	}
}

func TestDominantNameFromRows_MajorityWins(t *testing.T) {
	t.Parallel()
	rows := make([]canonical.PlayerMatchRow, 5)
	i := 0
	got := dominantNameFromRows(rows, "fr", func(r canonical.PlayerMatchRow) i18n.Text {
		i++
		// 3x Slayer, 2x CTF.
		if i <= 3 {
			return i18n.FromColumns("Slayer", "Massacre")
		}
		return i18n.FromColumns("CTF", "Drapeau")
	})
	if got == nil || *got != "Massacre" {
		t.Errorf("dominantNameFromRows majority: got %v, want Massacre (3x vs 2x)", got)
	}
}

func TestDominantNameFromRows_TieAlphabetical(t *testing.T) {
	t.Parallel()
	rows := make([]canonical.PlayerMatchRow, 2)
	i := 0
	got := dominantNameFromRows(rows, "fr", func(r canonical.PlayerMatchRow) i18n.Text {
		i++
		// 1x Slayer (key="Slayer"), 1x CTF (key="CTF") — égalité, CTF < Slayer
		// → CTF gagne.
		if i == 1 {
			return i18n.FromColumns("Slayer", "Massacre")
		}
		return i18n.FromColumns("CTF", "Drapeau")
	})
	if got == nil || *got != "Drapeau" {
		t.Errorf("dominantNameFromRows tie: got %v, want Drapeau (alphabetical key tie-break)", got)
	}
}

func TestDominantNameFromRows_SkipsEmpty(t *testing.T) {
	t.Parallel()
	rows := make([]canonical.PlayerMatchRow, 3)
	i := 0
	got := dominantNameFromRows(rows, "fr", func(r canonical.PlayerMatchRow) i18n.Text {
		i++
		if i == 1 {
			return nil // ignoré
		}
		return i18n.FromColumns("Slayer", "Massacre")
	})
	if got == nil || *got != "Massacre" {
		t.Errorf("dominantNameFromRows skipping empty: got %v, want Massacre", got)
	}
}

func TestDominantNameFromRows_AllEmpty(t *testing.T) {
	t.Parallel()
	rows := make([]canonical.PlayerMatchRow, 3)
	got := dominantNameFromRows(rows, "fr", func(r canonical.PlayerMatchRow) i18n.Text {
		return nil
	})
	if got != nil {
		t.Errorf("dominantNameFromRows all empty: got %v, want nil", got)
	}
}

func TestDominantNameFromRows_KeyFromFROnly(t *testing.T) {
	t.Parallel()
	// EN vide → key = FR ; reste cohérent.
	rows := make([]canonical.PlayerMatchRow, 1)
	got := dominantNameFromRows(rows, "fr", func(r canonical.PlayerMatchRow) i18n.Text {
		return i18n.FromColumns("", "Massacre")
	})
	if got == nil || *got != "Massacre" {
		t.Errorf("dominantNameFromRows fr only: got %v, want Massacre", got)
	}
}
