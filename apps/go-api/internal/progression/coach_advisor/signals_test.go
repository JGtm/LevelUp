package coach_advisor_test

import (
	"testing"

	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/progression/coach"
	"levelup/go-api/internal/progression/coach_advisor"
)

func mkAlert(t coach.AlertType, params map[string]any) coach.Alert {
	return coach.Alert{Type: t, Severity: notifications.SeverityInfo, Params: params}
}

func TestSignalsFromAlerts_IgnoresNonActionableTypes(t *testing.T) {
	in := []coach.Alert{
		mkAlert(coach.AlertTypeRecordBroken, map[string]any{}),
		mkAlert(coach.AlertTypeMilestoneUnlocked, map[string]any{}),
		mkAlert(coach.AlertTypeStreakMilestone, map[string]any{}),
		mkAlert(coach.AlertPatternStrength, map[string]any{}),
		mkAlert(coach.AlertPatternWeakness, map[string]any{}),
	}
	got := coach_advisor.SignalsFromAlerts(in)
	if len(got) != 0 {
		t.Errorf("expected 0 signals (all ignored), got %d", len(got))
	}
}

func TestSignalsFromAlerts_LOWESSPositive_StrengthScalesWithSlope(t *testing.T) {
	in := []coach.Alert{
		mkAlert(coach.AlertTypeLOWESSPositive, map[string]any{
			"component": "accuracy_delta",
			"slope":     0.25,
			"window":    14,
		}),
	}
	got := coach_advisor.SignalsFromAlerts(in)
	if len(got) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(got))
	}
	s := got[0]
	if s.Kind != coach_advisor.SignalLOWESSPositive {
		t.Errorf("Kind: got %q, want lowess_positive", s.Kind)
	}
	if s.LUSRComponent != "accuracy_delta" {
		t.Errorf("LUSRComponent: got %q", s.LUSRComponent)
	}
	if s.RadarAxis != "combat" {
		t.Errorf("RadarAxis: got %q, want combat (via mapping)", s.RadarAxis)
	}
	// slope=0.25, saturation à 0.5 → strength = 0.5
	if s.Strength < 0.49 || s.Strength > 0.51 {
		t.Errorf("Strength: got %f, want ~0.5", s.Strength)
	}
}

func TestSignalsFromAlerts_LOWESS_SaturatesAtOne(t *testing.T) {
	in := []coach.Alert{
		mkAlert(coach.AlertTypeLOWESSPositive, map[string]any{
			"component": "kills_vs_expected",
			"slope":     2.0,
		}),
	}
	got := coach_advisor.SignalsFromAlerts(in)
	if got[0].Strength != 1.0 {
		t.Errorf("Strength: got %f, want 1.0 (saturated)", got[0].Strength)
	}
}

func TestSignalsFromAlerts_CombatFragile_StrengthFromDRGap(t *testing.T) {
	// medianDR = 0.5, drP80 = 1.59 → gap = 1 - 0.5/1.59 ≈ 0.685
	in := []coach.Alert{
		mkAlert(coach.AlertTypeCombatPatternFragile, map[string]any{
			"median_dr": 0.5,
		}),
	}
	got := coach_advisor.SignalsFromAlerts(in)
	if len(got) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(got))
	}
	s := got[0]
	if s.Kind != coach_advisor.SignalCombatPatternFragile {
		t.Errorf("Kind: got %q", s.Kind)
	}
	if s.RadarAxis != "survival" {
		t.Errorf("RadarAxis: got %q, want survival", s.RadarAxis)
	}
	if s.Strength < 0.6 || s.Strength > 0.75 {
		t.Errorf("Strength: got %f, want ~0.685", s.Strength)
	}
}

func TestSignalsFromAlerts_RecordNearMiss_StrengthFromGap(t *testing.T) {
	// value=95, target=100 → gap_ratio = 0.05 → strength = 0.95
	in := []coach.Alert{
		mkAlert(coach.AlertTypeRecordNearMiss, map[string]any{
			"metric": "accuracy",
			"value":  95.0,
			"target": 100.0,
			"period": "30d",
		}),
	}
	got := coach_advisor.SignalsFromAlerts(in)
	if len(got) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(got))
	}
	s := got[0]
	if s.Kind != coach_advisor.SignalRecordApproach {
		t.Errorf("Kind: got %q", s.Kind)
	}
	if s.Metric != "accuracy" || s.RadarAxis != "combat" {
		t.Errorf("Metric/Axis: %q/%q", s.Metric, s.RadarAxis)
	}
	if s.Strength < 0.94 || s.Strength > 0.96 {
		t.Errorf("Strength: got %f, want ~0.95", s.Strength)
	}
}

func TestSignalsFromAlerts_MilestoneNearMiss_StrengthFromProgress(t *testing.T) {
	// progress=185, threshold=200 → ratio=0.925
	in := []coach.Alert{
		mkAlert(coach.AlertTypeMilestoneNearMiss, map[string]any{
			"metric":    "kills_total",
			"threshold": 200.0,
			"progress":  185.0,
		}),
	}
	got := coach_advisor.SignalsFromAlerts(in)
	if got[0].Kind != coach_advisor.SignalMilestoneApproach {
		t.Errorf("Kind: got %q", got[0].Kind)
	}
	if got[0].RadarAxis != "combat" {
		t.Errorf("RadarAxis (kills_total → combat): got %q", got[0].RadarAxis)
	}
	if got[0].Strength < 0.92 || got[0].Strength > 0.93 {
		t.Errorf("Strength: got %f, want ~0.925", got[0].Strength)
	}
}

func TestSignalsFromAlerts_LUSRTierApproach_CloserGapMeansStronger(t *testing.T) {
	// gap=2 (μ à 2 pts du tier) → strength = 1 - 2/10 = 0.8
	in := []coach.Alert{
		mkAlert(coach.AlertTypeLUSRTierApproach, map[string]any{
			"gap":            2.0,
			"next_tier_name": "diamond_iv",
		}),
	}
	got := coach_advisor.SignalsFromAlerts(in)
	if got[0].Strength < 0.79 || got[0].Strength > 0.81 {
		t.Errorf("Strength: got %f, want ~0.8", got[0].Strength)
	}
}

func TestSignalsFromAlerts_Comeback_FixedStrength(t *testing.T) {
	in := []coach.Alert{
		mkAlert(coach.AlertTypeComebackWelcome, map[string]any{
			"days_away": 7,
		}),
	}
	got := coach_advisor.SignalsFromAlerts(in)
	if got[0].Strength != 0.6 {
		t.Errorf("Comeback strength: got %f, want 0.6", got[0].Strength)
	}
}

func TestSignalsFromAlerts_CombatDiscret_AbsResidual(t *testing.T) {
	in := []coach.Alert{
		mkAlert(coach.AlertTypeCombatPatternDiscret, map[string]any{
			"avg_residual": -10.0,
		}),
	}
	got := coach_advisor.SignalsFromAlerts(in)
	if got[0].Kind != coach_advisor.SignalCombatPatternDiscreet {
		t.Errorf("Kind: got %q", got[0].Kind)
	}
	if got[0].RadarAxis != "support" {
		t.Errorf("RadarAxis: got %q, want support", got[0].RadarAxis)
	}
	if got[0].Strength < 0.65 || got[0].Strength > 0.70 {
		t.Errorf("Strength: got %f, want ~0.667 (|−10|/15)", got[0].Strength)
	}
}

func TestSignalsFromAlerts_CombatActif_PositiveResidual(t *testing.T) {
	in := []coach.Alert{
		mkAlert(coach.AlertTypeCombatPatternActif, map[string]any{
			"median_oc":    0.4,
			"avg_residual": 8.0,
		}),
	}
	got := coach_advisor.SignalsFromAlerts(in)
	if got[0].Kind != coach_advisor.SignalCombatPatternActive {
		t.Errorf("Kind: got %q", got[0].Kind)
	}
	if got[0].LUSRComponent != "kills_vs_expected" {
		t.Errorf("LUSRComponent: got %q", got[0].LUSRComponent)
	}
}

func TestSignalsFromAlerts_MultipleAlerts_OrderPreserved(t *testing.T) {
	in := []coach.Alert{
		mkAlert(coach.AlertTypeLOWESSPositive, map[string]any{"component": "accuracy_delta", "slope": 0.3}),
		mkAlert(coach.AlertTypeRecordBroken, map[string]any{}), // ignoré
		mkAlert(coach.AlertTypeComebackWelcome, map[string]any{}),
	}
	got := coach_advisor.SignalsFromAlerts(in)
	if len(got) != 2 {
		t.Fatalf("got %d, want 2 (record_broken ignored)", len(got))
	}
	if got[0].Kind != coach_advisor.SignalLOWESSPositive {
		t.Errorf("order: got[0] = %q", got[0].Kind)
	}
	if got[1].Kind != coach_advisor.SignalComebackWelcome {
		t.Errorf("order: got[1] = %q", got[1].Kind)
	}
}

func TestSignalsFromAlerts_EmptyInput_EmptyOutput(t *testing.T) {
	got := coach_advisor.SignalsFromAlerts(nil)
	if len(got) != 0 {
		t.Errorf("expected 0 signals, got %d", len(got))
	}
}

func TestSignalsFromAlerts_UnknownMetric_AxisEmpty(t *testing.T) {
	in := []coach.Alert{
		mkAlert(coach.AlertTypeRecordNearMiss, map[string]any{
			"metric": "some_metric_not_mapped",
			"value":  10.0,
			"target": 11.0,
		}),
	}
	got := coach_advisor.SignalsFromAlerts(in)
	if got[0].RadarAxis != "" {
		t.Errorf("unknown metric: RadarAxis should be empty, got %q", got[0].RadarAxis)
	}
	// Signal still emitted — matcher will rely on metric alone
}
