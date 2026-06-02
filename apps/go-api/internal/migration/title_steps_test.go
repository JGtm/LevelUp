package migration

import "testing"

// TestStepsForTarget_NilProvider_IsLegacy : sans provider, stepsForTarget ==
// ForTarget (comportement legacy strictement préservé — la bascille B est no-op).
func TestStepsForTarget_NilProvider_IsLegacy(t *testing.T) {
	titleStepsProvider = nil // garantit l'état par défaut
	got := stepsForTarget(TargetPlayer)
	want := ForTarget(TargetPlayer)
	if len(got) != len(want) {
		t.Fatalf("stepsForTarget(nil provider) = %d steps, want %d (ForTarget)", len(got), len(want))
	}
	for i := range want {
		if got[i].Name != want[i].Name {
			t.Errorf("rang %d: %q != %q", i, got[i].Name, want[i].Name)
		}
	}
}

// TestCombineSteps_OverrideByName : extra remplace son homonyme dans base, et
// ajoute les nouveaux. (L'ordre final est ré-imposé par canonicalOrder ailleurs.)
func TestCombineSteps_OverrideByName(t *testing.T) {
	base := []Migration{
		{Name: "a", TargetDB: TargetPlayer},
		{Name: "b", TargetDB: TargetPlayer},
		{Name: "c", TargetDB: TargetPlayer},
	}
	extra := []Migration{
		{Name: "b", TargetDB: TargetPlayer, Description: "title-owned"},
		{Name: "d", TargetDB: TargetPlayer},
	}
	out := combineSteps(base, extra)
	names := map[string]string{}
	for _, m := range out {
		if _, dup := names[m.Name]; dup {
			t.Fatalf("doublon de Name dans le résultat: %q", m.Name)
		}
		names[m.Name] = m.Description
	}
	if len(out) != 4 {
		t.Fatalf("combineSteps = %d steps, want 4 (a,b,c,d)", len(out))
	}
	if names["b"] != "title-owned" {
		t.Errorf("b devrait être la version title-owned (override), got desc %q", names["b"])
	}
	for _, n := range []string{"a", "b", "c", "d"} {
		if _, ok := names[n]; !ok {
			t.Errorf("step %q manquant du résultat", n)
		}
	}
}

// TestCombineSteps_EmptyExtra_ReturnsBase : extra vide ⇒ base inchangée.
func TestCombineSteps_EmptyExtra_ReturnsBase(t *testing.T) {
	base := []Migration{{Name: "a"}, {Name: "b"}}
	if out := combineSteps(base, nil); len(out) != 2 {
		t.Fatalf("combineSteps(base, nil) = %d, want 2", len(out))
	}
}
