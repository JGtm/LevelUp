package coach_advisor_test

import (
	"testing"

	"levelup/go-api/internal/prestige"
	"levelup/go-api/internal/progression/coach_advisor"
)

func mkSig(kind coach_advisor.SignalKind, axis string, strength float64) coach_advisor.Signal {
	return coach_advisor.Signal{
		Kind:      kind,
		RadarAxis: axis,
		Strength:  strength,
	}
}

func TestTryCompose_NoArcWhenFewerThanMinSignals(t *testing.T) {
	signals := []coach_advisor.Signal{
		mkSig(coach_advisor.SignalLOWESSPositive, "combat", 0.8),
	}
	_, ok := coach_advisor.TryCompose(signals, coach_advisor.DefaultArcComposerConfig())
	if ok {
		t.Error("expected no arc with only 1 signal (< MinSignalsForArc=2)")
	}
}

func TestTryCompose_NoArcWhenAxesScattered(t *testing.T) {
	// Chaque signal sur un axis différent → aucun axis avec >=2
	signals := []coach_advisor.Signal{
		mkSig(coach_advisor.SignalLOWESSPositive, "combat", 0.8),
		mkSig(coach_advisor.SignalRecordApproach, "survival", 0.7),
		mkSig(coach_advisor.SignalMilestoneApproach, "score", 0.9),
	}
	_, ok := coach_advisor.TryCompose(signals, coach_advisor.DefaultArcComposerConfig())
	if ok {
		t.Error("expected no arc with scattered axes")
	}
}

func TestTryCompose_NoArcWhenAllSignalsWeak(t *testing.T) {
	// 3 signaux sur combat mais aucun strength >= 0.6
	signals := []coach_advisor.Signal{
		mkSig(coach_advisor.SignalLOWESSPositive, "combat", 0.55),
		mkSig(coach_advisor.SignalRecordApproach, "combat", 0.51),
		mkSig(coach_advisor.SignalMilestoneApproach, "combat", 0.50),
	}
	_, ok := coach_advisor.TryCompose(signals, coach_advisor.DefaultArcComposerConfig())
	if ok {
		t.Error("expected no arc when all signals < synthesis_min_strength (RequireOneStrongSignal=true)")
	}
}

func TestTryCompose_HappyPath_ThreeSignalsSameAxis(t *testing.T) {
	signals := []coach_advisor.Signal{
		mkSig(coach_advisor.SignalLOWESSPositive, "combat", 0.85),
		mkSig(coach_advisor.SignalCombatPatternActive, "combat", 0.7),
		mkSig(coach_advisor.SignalRecordApproach, "combat", 0.55),
	}
	spec, ok := coach_advisor.TryCompose(signals, coach_advisor.DefaultArcComposerConfig())
	if !ok {
		t.Fatal("expected arc composed")
	}
	if spec.RadarAxis != "combat" {
		t.Errorf("RadarAxis: got %q, want combat", spec.RadarAxis)
	}
	if len(spec.Steps) != 3 {
		t.Errorf("Steps: got %d, want 3", len(spec.Steps))
	}
	// Vérif tri par strength desc
	if spec.Steps[0].Signal.Strength != 0.85 {
		t.Errorf("step 1 strength: got %f, want 0.85 (strongest)", spec.Steps[0].Signal.Strength)
	}
	if spec.Steps[2].Signal.Strength != 0.55 {
		t.Errorf("step 3 strength: got %f, want 0.55 (weakest)", spec.Steps[2].Signal.Strength)
	}
	// Vérif progression de tier Normal → Heroic → Legendary
	if spec.Steps[0].SuggestedTier != prestige.TierNormal {
		t.Errorf("step 1 tier: got %q, want Normal", spec.Steps[0].SuggestedTier)
	}
	if spec.Steps[1].SuggestedTier != prestige.TierHeroic {
		t.Errorf("step 2 tier: got %q, want Heroic", spec.Steps[1].SuggestedTier)
	}
	if spec.Steps[2].SuggestedTier != prestige.TierLegendary {
		t.Errorf("step 3 tier: got %q, want Legendary", spec.Steps[2].SuggestedTier)
	}
	// Vérif positions 1-indexed
	for i, s := range spec.Steps {
		if s.Position != i+1 {
			t.Errorf("step %d position: got %d, want %d", i, s.Position, i+1)
		}
	}
	// Vérif labels combat
	if spec.TitleEN != "Combat Mastery" || spec.TitleFR != "Domination en combat" {
		t.Errorf("Title combat: got %q / %q", spec.TitleEN, spec.TitleFR)
	}
	if spec.DescriptionEN == "" || spec.DescriptionFR == "" {
		t.Error("Description should be non-empty")
	}
	if spec.AverageStrength < 0.69 || spec.AverageStrength > 0.71 {
		t.Errorf("AverageStrength: got %f, want ~0.7", spec.AverageStrength)
	}
}

func TestTryCompose_CappedAtMaxArcSteps(t *testing.T) {
	signals := []coach_advisor.Signal{
		mkSig(coach_advisor.SignalLOWESSPositive, "combat", 0.95),
		mkSig(coach_advisor.SignalCombatPatternActive, "combat", 0.85),
		mkSig(coach_advisor.SignalRecordApproach, "combat", 0.75),
		mkSig(coach_advisor.SignalMilestoneApproach, "combat", 0.65),
		mkSig(coach_advisor.SignalCombatPatternFragile, "combat", 0.55),
	}
	spec, ok := coach_advisor.TryCompose(signals, coach_advisor.DefaultArcComposerConfig())
	if !ok {
		t.Fatal("expected arc")
	}
	if len(spec.Steps) != 4 {
		t.Errorf("Steps should be capped to 4, got %d", len(spec.Steps))
	}
	// Le top 4 strength : 0.95, 0.85, 0.75, 0.65 — pas 0.55
	for _, step := range spec.Steps {
		if step.Signal.Strength == 0.55 {
			t.Error("0.55 signal should have been dropped (cap to MaxArcSteps=4, sorted by strength)")
		}
	}
	if spec.Steps[3].SuggestedTier != prestige.TierMythic {
		t.Errorf("step 4 tier: got %q, want Mythic", spec.Steps[3].SuggestedTier)
	}
}

func TestTryCompose_PicksAxisWithMostSignals(t *testing.T) {
	// 2 sur combat, 3 sur survival → survival gagne (et un signal fort présent)
	signals := []coach_advisor.Signal{
		mkSig(coach_advisor.SignalLOWESSPositive, "combat", 0.8),
		mkSig(coach_advisor.SignalCombatPatternActive, "combat", 0.7),
		mkSig(coach_advisor.SignalRecordApproach, "survival", 0.85),
		mkSig(coach_advisor.SignalMilestoneApproach, "survival", 0.65),
		mkSig(coach_advisor.SignalCombatPatternFragile, "survival", 0.7),
	}
	spec, ok := coach_advisor.TryCompose(signals, coach_advisor.DefaultArcComposerConfig())
	if !ok {
		t.Fatal("expected arc")
	}
	if spec.RadarAxis != "survival" {
		t.Errorf("RadarAxis: got %q, want survival (most signals)", spec.RadarAxis)
	}
	if spec.TitleFR != "Reprise solide" {
		t.Errorf("Title FR: got %q, want 'Reprise solide'", spec.TitleFR)
	}
}

func TestTryCompose_TieBreakAlphabetical(t *testing.T) {
	// 2 sur combat, 2 sur survival → tie. Tie-break alphabétique : "combat" < "survival" → combat gagne
	signals := []coach_advisor.Signal{
		mkSig(coach_advisor.SignalLOWESSPositive, "combat", 0.8),
		mkSig(coach_advisor.SignalCombatPatternActive, "combat", 0.7),
		mkSig(coach_advisor.SignalRecordApproach, "survival", 0.85),
		mkSig(coach_advisor.SignalMilestoneApproach, "survival", 0.65),
	}
	spec, ok := coach_advisor.TryCompose(signals, coach_advisor.DefaultArcComposerConfig())
	if !ok {
		t.Fatal("expected arc")
	}
	if spec.RadarAxis != "combat" {
		t.Errorf("Tie-break should pick alphabetical first ('combat' < 'survival'), got %q", spec.RadarAxis)
	}
}

func TestTryCompose_IgnoresSignalsWithoutAxis(t *testing.T) {
	signals := []coach_advisor.Signal{
		mkSig(coach_advisor.SignalLOWESSPositive, "combat", 0.8),
		mkSig(coach_advisor.SignalComebackWelcome, "", 0.7), // pas d'axis
		mkSig(coach_advisor.SignalCombatPatternActive, "combat", 0.7),
	}
	spec, ok := coach_advisor.TryCompose(signals, coach_advisor.DefaultArcComposerConfig())
	if !ok {
		t.Fatal("expected arc on the 2 combat signals (comeback ignored)")
	}
	if len(spec.Steps) != 2 {
		t.Errorf("Steps: got %d, want 2 (signal w/o axis ignored)", len(spec.Steps))
	}
}

func TestTryCompose_AllAxesLabelsKnown(t *testing.T) {
	// Vérif que tous les axes prévus ADR 0021 §"Cohérence narrative" produisent un titre.
	for _, axis := range []string{"combat", "survival", "support", "objective", "score", "impact"} {
		signals := []coach_advisor.Signal{
			mkSig(coach_advisor.SignalLOWESSPositive, axis, 0.7),
			mkSig(coach_advisor.SignalRecordApproach, axis, 0.7),
		}
		spec, ok := coach_advisor.TryCompose(signals, coach_advisor.DefaultArcComposerConfig())
		if !ok {
			t.Errorf("axis %q: expected arc composed", axis)
			continue
		}
		if spec.TitleEN == "Coach Arc" || spec.TitleFR == "Arc du coach" {
			t.Errorf("axis %q: fallback title used, expected mapped title (got %q / %q)", axis, spec.TitleEN, spec.TitleFR)
		}
	}
}

func TestTryCompose_UnknownAxis_FallbackTitle(t *testing.T) {
	signals := []coach_advisor.Signal{
		mkSig(coach_advisor.SignalLOWESSPositive, "future_title_axis", 0.8),
		mkSig(coach_advisor.SignalRecordApproach, "future_title_axis", 0.7),
	}
	spec, ok := coach_advisor.TryCompose(signals, coach_advisor.DefaultArcComposerConfig())
	if !ok {
		t.Fatal("expected arc even on unknown axis")
	}
	if spec.TitleEN != "Coach Arc" || spec.TitleFR != "Arc du coach" {
		t.Errorf("unknown axis: expected fallback titles, got %q / %q", spec.TitleEN, spec.TitleFR)
	}
}
