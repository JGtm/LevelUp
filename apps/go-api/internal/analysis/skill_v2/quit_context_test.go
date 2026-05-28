package skill_v2

import "testing"

func TestInferQuitContext_Leading(t *testing.T) {
	// Équipe 0 mène 3-1 au moment du quit (à 5000ms).
	frags := []TeamFrag{
		{TimeMs: 1000, TeamID: 0},
		{TimeMs: 2000, TeamID: 0},
		{TimeMs: 3000, TeamID: 1},
		{TimeMs: 4000, TeamID: 0},
		{TimeMs: 9000, TeamID: 1}, // après le quit → ignoré
	}
	if got := InferQuitContext(frags, 5000, 0); got != QuitWhileLeading {
		t.Errorf("got %v, want QuitWhileLeading (équipe menait 3-1)", got)
	}
}

func TestInferQuitContext_Trailing(t *testing.T) {
	// Équipe 0 perd 1-3 au moment du quit.
	frags := []TeamFrag{
		{TimeMs: 1000, TeamID: 1},
		{TimeMs: 2000, TeamID: 1},
		{TimeMs: 3000, TeamID: 0},
		{TimeMs: 4000, TeamID: 1},
	}
	if got := InferQuitContext(frags, 5000, 0); got != QuitWhileTrailing {
		t.Errorf("got %v, want QuitWhileTrailing (équipe perdait 1-3)", got)
	}
}

func TestInferQuitContext_Tied(t *testing.T) {
	frags := []TeamFrag{
		{TimeMs: 1000, TeamID: 0},
		{TimeMs: 2000, TeamID: 1},
	}
	if got := InferQuitContext(frags, 5000, 0); got != QuitWhileTied {
		t.Errorf("got %v, want QuitWhileTied (1-1)", got)
	}
}

func TestInferQuitContext_OnlyCountsBeforeQuit(t *testing.T) {
	// L'équipe 0 mène 2-0 à 3000ms (le quit), mais l'équipe 1 marque 5 fois APRÈS.
	frags := []TeamFrag{
		{TimeMs: 1000, TeamID: 0},
		{TimeMs: 2000, TeamID: 0},
		{TimeMs: 4000, TeamID: 1},
		{TimeMs: 5000, TeamID: 1},
		{TimeMs: 6000, TeamID: 1},
		{TimeMs: 7000, TeamID: 1},
		{TimeMs: 8000, TeamID: 1},
	}
	// Au moment du quit (3000ms) l'équipe 0 menait → Leading, même si elle perd à la fin.
	if got := InferQuitContext(frags, 3000, 0); got != QuitWhileLeading {
		t.Errorf("got %v, want QuitWhileLeading (menait au quit malgré la défaite finale)", got)
	}
}

func TestInferQuitContext_NoFragsYet(t *testing.T) {
	if got := InferQuitContext(nil, 1000, 0); got != QuitWhileTied {
		t.Errorf("got %v, want QuitWhileTied (0-0 en début de match)", got)
	}
}

func TestInferQuitContext_PerspectiveTeam1(t *testing.T) {
	// Mêmes frags, vus depuis l'équipe 1 : elle perd 1-3 → Trailing.
	frags := []TeamFrag{
		{TimeMs: 1000, TeamID: 0},
		{TimeMs: 2000, TeamID: 0},
		{TimeMs: 3000, TeamID: 1},
		{TimeMs: 4000, TeamID: 0},
	}
	if got := InferQuitContext(frags, 5000, 1); got != QuitWhileTrailing {
		t.Errorf("got %v, want QuitWhileTrailing (équipe 1 perd 1-3)", got)
	}
}
