package notify

// labels_test.go — PMT-11 : oracle DOUBLE du seam NotifyLabels.
//   (a) parité Halo byte-identique (libellés actuels inchangés, FR+EN) ;
//   (b) routage du titre synthétique (outcomes.toml divergent → « Triomphe ») ;
//   + dégradation failsafe (src nil / Outcomes() nil / clé absente → Halo).

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/games/mappings"
)

// fakeOutcomeSrc satisfait OutcomeSource avec un set injecté (titre fictif).
type fakeOutcomeSrc struct{ set *mappings.OutcomeMappingSet }

func (f fakeOutcomeSrc) Outcomes() *mappings.OutcomeMappingSet { return f.set }

// syntheticOutcomes charge un set divergent (copie du corpus synthetic_title_b).
func syntheticOutcomes(t *testing.T) *mappings.OutcomeMappingSet {
	t.Helper()
	toml := []byte(`
[meta]
title_slug     = "synthetic_title_b"
schema_version = 1

[outcomes.win]
labels = { en = "Victory", fr = "Triomphe" }
color_token = "outcome.positive"

[outcomes.loss]
labels = { en = "Defeat", fr = "Défaite" }
color_token = "outcome.negative"

[outcomes.tie]
labels = { en = "Draw", fr = "Match nul" }
color_token = "outcome.neutral"

[outcomes.dnf]
labels = { en = "Forfeit", fr = "Forfait" }
color_token = "outcome.neutral"
`)
	set, err := mappings.LoadOutcomesFromBytes("synth_outcomes.toml", toml)
	if err != nil {
		t.Fatalf("LoadOutcomesFromBytes: %v", err)
	}
	return set
}

func renderOutcome(t *testing.T, outcome int, lang string, labels NotifyLabels) string {
	t.Helper()
	lm := &LastMatchInfo{
		MapName: "M", PlaylistName: "P", VariantName: "V",
		Outcome: outcome, StartTime: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	return strings.Join(lastMatchLines(lm, lang, labels), "\n")
}

// TestNotifyLabels_HaloParity (oracle a) : les libellés Halo par défaut sont
// inchangés (FR+EN) pour les 4 outcomes — aucune régression de contenu.
func TestNotifyLabels_HaloParity(t *testing.T) {
	cases := []struct {
		outcome int
		fr, en  string
	}{
		{2, "Victoire", "Win"},
		{3, "Défaite", "Loss"},
		{1, "Égalité", "Draw"},
		{4, "Abandon", "Quit"},
	}
	for _, c := range cases {
		if got := HaloLabels().Outcome(outcomeCanonicalKey[c.outcome], "fr"); got != c.fr {
			t.Errorf("outcome %d FR = %q, want %q (parité Halo)", c.outcome, got, c.fr)
		}
		if got := HaloLabels().Outcome(outcomeCanonicalKey[c.outcome], "en"); got != c.en {
			t.Errorf("outcome %d EN = %q, want %q (parité Halo)", c.outcome, got, c.en)
		}
	}
}

// TestNotifyLabels_BuildSyncEmbedDefaultEqualsHalo (oracle a) : la valeur produite
// par BuildSyncEmbed est strictement égale à BuildSyncEmbedWithLabels(HaloLabels()).
func TestNotifyLabels_BuildSyncEmbedDefaultEqualsHalo(t *testing.T) {
	players := []PlayerSyncResult{{
		Gamertag: "GT", MatchesSynced: 1,
		LastMatch: &LastMatchInfo{MapName: "Aquarius", PlaylistName: "Ranked", VariantName: "Slayer",
			IsRanked: true, Outcome: 2, Kills: 15, Deaths: 8, Assists: 4,
			StartTime: time.Date(2026, 1, 1, 14, 30, 0, 0, time.UTC)},
	}}
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	end := start.Add(time.Minute)
	for _, lang := range []string{"fr", "en"} {
		a := BuildSyncEmbed("sync_delta", start, end, players, true, lang)
		b := BuildSyncEmbedWithLabels("sync_delta", start, end, players, true, lang, HaloLabels())
		if !reflect.DeepEqual(a, b) {
			t.Errorf("[%s] BuildSyncEmbed != BuildSyncEmbedWithLabels(HaloLabels()) — parité Halo cassée", lang)
		}
	}
}

// TestNotifyLabels_SyntheticRouted (oracle b) : un titre au manifeste divergent
// rend ses propres outcomes via le seam — « Triomphe » et pas « Victoire ».
func TestNotifyLabels_SyntheticRouted(t *testing.T) {
	labels := LabelsFor(fakeOutcomeSrc{set: syntheticOutcomes(t)}, "")

	winFR := renderOutcome(t, 2, "fr", labels)
	if !strings.Contains(winFR, "Triomphe") {
		t.Errorf("win FR via titre B = %q, attendu « Triomphe »", winFR)
	}
	if strings.Contains(winFR, "Victoire") {
		t.Errorf("win FR via titre B contient « Victoire » (Halo) — le seam ne route pas")
	}
	// Pont int→clé canonique : Quit Discord (4) → dnf → « Forfait ».
	if dnfFR := renderOutcome(t, 4, "fr", labels); !strings.Contains(dnfFR, "Forfait") {
		t.Errorf("dnf FR via titre B = %q, attendu « Forfait » (pont 4→dnf)", dnfFR)
	}
	if tieFR := renderOutcome(t, 1, "fr", labels); !strings.Contains(tieFR, "Match nul") {
		t.Errorf("tie FR via titre B = %q, attendu « Match nul »", tieFR)
	}
}

// TestNotifyLabels_FailsafeDegradation : src nil / Outcomes() nil / clé absente
// dégradent proprement vers les libellés Halo, sans panic.
func TestNotifyLabels_FailsafeDegradation(t *testing.T) {
	if got := LabelsFor(nil, "").Outcome("win", "fr"); got != "Victoire" {
		t.Errorf("LabelsFor(nil).Outcome(win,fr) = %q, want Victoire (Halo)", got)
	}
	if got := LabelsFor(fakeOutcomeSrc{set: nil}, "").Outcome("win", "fr"); got != "Victoire" {
		t.Errorf("Outcomes()==nil → %q, want Victoire (fallback Halo)", got)
	}

	// Set partiel (seulement win) : la clé présente route, la clé absente dégrade.
	partial, err := mappings.LoadOutcomesFromBytes("partial.toml", []byte(`
[meta]
title_slug     = "x"
schema_version = 1
[outcomes.win]
labels = { en = "Won", fr = "Gagné" }
color_token = "outcome.positive"
`))
	if err != nil {
		t.Fatalf("load partial: %v", err)
	}
	labels := LabelsFor(fakeOutcomeSrc{set: partial}, "")
	if got := labels.Outcome("win", "fr"); got != "Gagné" {
		t.Errorf("win présent → %q, want Gagné", got)
	}
	if got := labels.Outcome("dnf", "fr"); got != "Abandon" {
		t.Errorf("dnf absent du titre → %q, want Abandon (fallback Halo)", got)
	}
}
