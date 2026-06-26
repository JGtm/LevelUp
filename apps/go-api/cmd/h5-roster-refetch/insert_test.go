package main

import "testing"

// TestMakeResolver_LocalFirstThenUniversalThenSkip vérifie l'ordre de résolution :
// (1) kill-feed local, (2) fallback universel, (3) skip ("").
func TestMakeResolver_LocalFirstThenUniversalThenSkip(t *testing.T) {
	idMap := map[string]string{"LocalGuy": "xLocal", "Empty": ""}
	universalCalls := map[string]int{}
	universal := func(gt string) string {
		universalCalls[gt]++
		if gt == "UniversalGuy" {
			return "xUniversal"
		}
		return ""
	}
	resolve := makeResolver(idMap, universal)

	// (1) kill-feed local prime — le fallback n'est PAS consulté.
	if got := resolve("LocalGuy"); got != "xLocal" {
		t.Errorf("LocalGuy = %q, want xLocal (kill-feed local)", got)
	}
	if universalCalls["LocalGuy"] != 0 {
		t.Errorf("universal consulté pour LocalGuy (%d fois) — le local doit primer", universalCalls["LocalGuy"])
	}

	// (2) absent du local → fallback universel.
	if got := resolve("UniversalGuy"); got != "xUniversal" {
		t.Errorf("UniversalGuy = %q, want xUniversal (fallback universel)", got)
	}

	// (3) ni local ni universel → "" (skip).
	if got := resolve("Ghost"); got != "" {
		t.Errorf("Ghost = %q, want \"\" (skip)", got)
	}

	// gamertag vide → "" sans consulter le fallback.
	if got := resolve(""); got != "" {
		t.Errorf("vide = %q, want \"\"", got)
	}
	if universalCalls[""] != 0 {
		t.Errorf("universal consulté pour gamertag vide — court-circuit attendu")
	}

	// Entrée locale présente mais xuid vide → traitée comme absente → fallback puis skip.
	if got := resolve("Empty"); got != "" {
		t.Errorf("Empty (xuid local vide) = %q, want \"\" (pas de xuid fabriqué)", got)
	}
}

// TestMakeResolver_NilUniversalSkips : sans fallback (--resolver OFF), un gamertag absent
// du kill-feed local est directement sauté.
func TestMakeResolver_NilUniversalSkips(t *testing.T) {
	resolve := makeResolver(map[string]string{"A": "xA"}, nil)
	if got := resolve("A"); got != "xA" {
		t.Errorf("A = %q, want xA", got)
	}
	if got := resolve("B"); got != "" {
		t.Errorf("B = %q, want \"\" (pas de fallback, skip)", got)
	}
}
