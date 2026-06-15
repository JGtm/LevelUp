package mappings

import (
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/canonical"
)

const haloOutcomesTOML = `
[meta]
title_slug = "halo_infinite"
schema_version = 1
[outcomes.win]
labels = { en = "Win", fr = "Victoire" }
color_token = "outcome.positive"
raw_code = 2
[outcomes.loss]
labels = { en = "Loss", fr = "Défaite" }
color_token = "outcome.negative"
raw_code = 3
[outcomes.tie]
labels = { en = "Tie", fr = "Égalité" }
color_token = "outcome.neutral"
raw_code = 1
[outcomes.dnf]
labels = { en = "DNF", fr = "Non terminé" }
color_token = "outcome.neutral"
raw_code = 4
`

// TestOutcomeRawCode_RoundTripAndSQL — MT-06 : Canonical/RawCode round-trip +
// SQLIsWinExpr exact (parité Halo "outcome = 2").
func TestOutcomeRawCode_RoundTripAndSQL(t *testing.T) {
	set, err := LoadOutcomesFromBytes("halo.toml", []byte(haloOutcomesTOML))
	if err != nil {
		t.Fatal(err)
	}

	cases := map[int]canonical.Outcome{
		2: canonical.OutcomeWin, 3: canonical.OutcomeLoss,
		1: canonical.OutcomeTie, 4: canonical.OutcomeDNF,
	}
	for code, want := range cases {
		if got, ok := set.Canonical(code); !ok || got != want {
			t.Errorf("Canonical(%d) = %q,%v ; want %q", code, got, ok, want)
		}
	}
	for _, o := range canonical.AllOutcomes() {
		code, ok := set.RawCode(o)
		if !ok {
			t.Errorf("RawCode(%q) absent", o)
			continue
		}
		if back, ok := set.Canonical(code); !ok || back != o {
			t.Errorf("round-trip %q : RawCode=%d Canonical=%q", o, code, back)
		}
	}
	if expr, ok := set.SQLIsWinExpr("outcome"); !ok || expr != "outcome = 2" {
		t.Errorf("SQLIsWinExpr = %q,%v ; want \"outcome = 2\"", expr, ok)
	}
}

// TestOutcomeRawCode_SyntheticRoutesDifferently — ORACLE (b) : des codes
// inversés routent VRAIMENT différemment (preuve non cosmétique du seam).
func TestOutcomeRawCode_SyntheticRoutesDifferently(t *testing.T) {
	const synthTOML = `
[meta]
title_slug = "synthetic_title_b"
schema_version = 1
[outcomes.win]
labels = { en = "W", fr = "V" }
color_token = "outcome.positive"
raw_code = 7
[outcomes.loss]
labels = { en = "L", fr = "D" }
color_token = "outcome.negative"
raw_code = 9
`
	set, err := LoadOutcomesFromBytes("synth.toml", []byte(synthTOML))
	if err != nil {
		t.Fatal(err)
	}
	if expr, ok := set.SQLIsWinExpr("outcome"); !ok || expr != "outcome = 7" {
		t.Errorf("synthetic SQLIsWinExpr = %q,%v ; want \"outcome = 7\"", expr, ok)
	}
	if o, ok := set.Canonical(7); !ok || o != canonical.OutcomeWin {
		t.Errorf("synthetic Canonical(7) = %q,%v ; want win", o, ok)
	}
	if _, ok := set.Canonical(2); ok {
		t.Error("synthetic : le code Halo 2 ne doit PAS être mappé")
	}
}

// TestOutcomeRawCode_DuplicateRejected — codes dupliqués rejetés au load.
func TestOutcomeRawCode_DuplicateRejected(t *testing.T) {
	const dupTOML = `
[meta]
title_slug = "x"
schema_version = 1
[outcomes.win]
labels = { en = "W", fr = "V" }
color_token = "outcome.positive"
raw_code = 5
[outcomes.loss]
labels = { en = "L", fr = "D" }
color_token = "outcome.negative"
raw_code = 5
`
	if _, err := LoadOutcomesFromBytes("dup.toml", []byte(dupTOML)); err == nil {
		t.Error("raw_code dupliqué (5) doit être rejeté")
	}
}

// TestOutcomeRawCode_NilAndAbsentDegrade — nil set + titre sans raw_code
// dégradent proprement (false, pas de panic).
func TestOutcomeRawCode_NilAndAbsentDegrade(t *testing.T) {
	var nilSet *OutcomeMappingSet
	if _, ok := nilSet.SQLIsWinExpr("outcome"); ok {
		t.Error("nil set : SQLIsWinExpr doit retourner false")
	}
	if _, ok := nilSet.Canonical(2); ok {
		t.Error("nil set : Canonical doit retourner false")
	}

	const noCode = `
[meta]
title_slug = "x"
schema_version = 1
[outcomes.win]
labels = { en = "W", fr = "V" }
color_token = "outcome.positive"
`
	set, err := LoadOutcomesFromBytes("nocode.toml", []byte(noCode))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := set.SQLIsWinExpr("outcome"); ok {
		t.Error("set sans raw_code : SQLIsWinExpr doit dégrader (false)")
	}
}

// TestOutcomeRawCode_RealHaloParity — ORACLE (a) : le VRAI outcomes.toml de
// halo_infinite produit l'expression et les codes Halo attendus.
func TestOutcomeRawCode_RealHaloParity(t *testing.T) {
	root := repoRootForTest(t)
	set, err := LoadOutcomesFromFile(
		filepath.Join(root, "config", "titles", "halo_infinite", "mappings", "outcomes.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if expr, ok := set.SQLIsWinExpr("outcome"); !ok || expr != "outcome = 2" {
		t.Errorf("halo réel SQLIsWinExpr = %q,%v ; want \"outcome = 2\"", expr, ok)
	}
	if o, ok := set.Canonical(2); !ok || o != canonical.OutcomeWin {
		t.Errorf("halo réel Canonical(2) = %q ; want win", o)
	}
}
