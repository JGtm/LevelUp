package patterns

import (
	"testing"
)

// TestSelectLevers_NoPatterns vérifie qu'aucun levier n'est créé sans patterns.
func TestSelectLevers_NoPatterns(t *testing.T) {
	levers := selectLevers(nil, nil, nil, DefaultPatternConfig())
	if len(levers) != 0 {
		t.Errorf("levers = %d, want 0", len(levers))
	}
}

// TestSelectLevers_WeaknessPatternsCreateLever vérifie la création d'un levier.
func TestSelectLevers_WeaknessPatternsCreateLever(t *testing.T) {
	ctx := []ContextualPattern{
		{
			Type:    ContextByMode,
			Key:     "CTF",
			Signal:  SignalWeakness,
			WinRate: 0.30,
			Delta:   -0.25,
		},
	}
	levers := selectLevers(ctx, nil, nil, DefaultPatternConfig())
	if len(levers) != 1 {
		t.Fatalf("levers = %d, want 1", len(levers))
	}
	if levers[0].Axis != "mode_selection" {
		t.Errorf("axis = %q, want mode_selection", levers[0].Axis)
	}
	if levers[0].Rank != 1 {
		t.Errorf("rank = %d, want 1", levers[0].Rank)
	}
}

// TestSelectLevers_TiltHighCreatesSessionManagementLever vérifie la création
// d'un levier session_management depuis un pattern tilt.
func TestSelectLevers_TiltHighCreatesSessionManagementLever(t *testing.T) {
	beh := []BehavioralPattern{
		{
			Type:     BehaviorTilt,
			Severity: SeverityHigh,
		},
	}
	levers := selectLevers(nil, beh, nil, DefaultPatternConfig())
	if len(levers) == 0 {
		t.Fatal("aucun levier créé depuis tilt High")
	}
	found := false
	for _, l := range levers {
		if l.Axis == "session_management" {
			found = true
			break
		}
	}
	if !found {
		t.Error("levier session_management absent")
	}
}

// TestSelectLevers_SortedByImpactMax5 vérifie le tri et la limite à 5.
func TestSelectLevers_SortedByImpactMax5(t *testing.T) {
	// 6 patterns de faiblesse avec des deltas différents
	ctx := make([]ContextualPattern, 6)
	for i := range ctx {
		ctx[i] = ContextualPattern{
			Type:    ContextByMode,
			Key:     "Mode" + string(rune('A'+i)),
			Signal:  SignalWeakness,
			WinRate: 0.20,
			Delta:   -0.15 - float64(i)*0.05, // impacts différents
		}
	}
	levers := selectLevers(ctx, nil, nil, DefaultPatternConfig())
	if len(levers) > 5 {
		t.Errorf("levers = %d, want max 5", len(levers))
	}
	// Vérifier le tri par impact décroissant
	for i := 1; i < len(levers); i++ {
		if levers[i].Impact > levers[i-1].Impact {
			t.Errorf("levers[%d].Impact=%f > levers[%d].Impact=%f : pas trié", i, levers[i].Impact, i-1, levers[i-1].Impact)
		}
	}
	// Vérifier que le Rank est bien 1-based
	for i, l := range levers {
		if l.Rank != i+1 {
			t.Errorf("levers[%d].Rank = %d, want %d", i, l.Rank, i+1)
		}
	}
}
