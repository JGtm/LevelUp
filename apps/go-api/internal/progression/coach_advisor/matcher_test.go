package coach_advisor_test

import (
	"testing"

	"levelup/go-api/internal/prestige"
	"levelup/go-api/internal/progression/coach_advisor"
)

func tpl(id, metric string, lusr, axes []string) prestige.Template {
	return prestige.Template{
		ID:             id,
		Metric:         metric,
		LUSRComponents: lusr,
		RadarAxes:      axes,
	}
}

func TestMatchTemplateToSignal_PrefersLUSRComponentMatch(t *testing.T) {
	templates := []prestige.Template{
		tpl("a", "accuracy", []string{"accuracy_delta"}, []string{"combat"}),
		tpl("b", "kda", []string{"kills_vs_expected"}, []string{"combat"}),
		tpl("c", "win_rate", nil, []string{"score"}),
	}
	sig := coach_advisor.Signal{
		Kind:          coach_advisor.SignalLOWESSPositive,
		Metric:        "accuracy",
		LUSRComponent: "accuracy_delta",
		RadarAxis:     "combat",
		Strength:      0.7,
	}

	got := coach_advisor.MatchTemplateToSignal(sig, templates, coach_advisor.DefaultMatcherWeights())

	if len(got) != 3 {
		t.Fatalf("expected 3 scores, got %d", len(got))
	}
	if got[0].Template.ID != "a" {
		t.Errorf("expected top match 'a', got %q (score %f)", got[0].Template.ID, got[0].Score)
	}
	// Template a: LUSR+axis+metric = 0.5+0.3+0.2 = 1.0
	if got[0].Score < 0.99 {
		t.Errorf("expected ~1.0 for full match, got %f", got[0].Score)
	}
}

func TestMatchTemplateToSignal_AxisOnlyWhenLUSREmpty(t *testing.T) {
	templates := []prestige.Template{
		tpl("a", "accuracy", nil, []string{"combat"}),
		tpl("b", "win_rate", nil, []string{"score"}),
	}
	sig := coach_advisor.Signal{
		Kind:      coach_advisor.SignalCombatPatternFragile,
		RadarAxis: "combat",
	}

	got := coach_advisor.MatchTemplateToSignal(sig, templates, coach_advisor.DefaultMatcherWeights())

	if got[0].Template.ID != "a" {
		t.Errorf("expected 'a' to win on axis alone, got %q", got[0].Template.ID)
	}
	if got[0].Score != 0.3 {
		t.Errorf("expected 0.3 (axis weight), got %f", got[0].Score)
	}
}

func TestMatchTemplateToSignal_NoMatchesReturnsZeroScores(t *testing.T) {
	templates := []prestige.Template{
		tpl("a", "headshots", []string{"headshots_per_kill"}, []string{"impact"}),
	}
	sig := coach_advisor.Signal{
		Kind:          coach_advisor.SignalRecordApproach,
		Metric:        "win_rate",
		LUSRComponent: "win_rate_delta",
		RadarAxis:     "score",
	}

	got := coach_advisor.MatchTemplateToSignal(sig, templates, coach_advisor.DefaultMatcherWeights())

	if len(got) != 1 {
		t.Fatalf("expected 1 score, got %d", len(got))
	}
	if got[0].Score != 0 {
		t.Errorf("expected score 0 (no overlap), got %f", got[0].Score)
	}
}

func TestMatchTemplateToSignal_StableOrderOnTie(t *testing.T) {
	templates := []prestige.Template{
		tpl("z", "accuracy", []string{"accuracy_delta"}, nil),
		tpl("a", "accuracy", []string{"accuracy_delta"}, nil),
	}
	sig := coach_advisor.Signal{
		LUSRComponent: "accuracy_delta",
	}

	got := coach_advisor.MatchTemplateToSignal(sig, templates, coach_advisor.DefaultMatcherWeights())

	if got[0].Template.ID != "a" || got[1].Template.ID != "z" {
		t.Errorf("expected alphabetical tie-break ('a' before 'z'), got %q then %q",
			got[0].Template.ID, got[1].Template.ID)
	}
}

func TestMatchTemplateToSignal_EmptyTemplatesEmptyResult(t *testing.T) {
	sig := coach_advisor.Signal{LUSRComponent: "accuracy_delta"}
	got := coach_advisor.MatchTemplateToSignal(sig, nil, coach_advisor.DefaultMatcherWeights())
	if len(got) != 0 {
		t.Errorf("expected empty result, got %d entries", len(got))
	}
}

func TestFilterByMinScore_KeepsAboveThreshold(t *testing.T) {
	scores := []coach_advisor.MatchScore{
		{Template: tpl("a", "", nil, nil), Score: 0.8},
		{Template: tpl("b", "", nil, nil), Score: 0.4},
		{Template: tpl("c", "", nil, nil), Score: 0.39},
		{Template: tpl("d", "", nil, nil), Score: 0.0},
	}
	got := coach_advisor.FilterByMinScore(scores, 0.4)

	if len(got) != 2 {
		t.Fatalf("expected 2 entries >= 0.4, got %d", len(got))
	}
	if got[0].Template.ID != "a" || got[1].Template.ID != "b" {
		t.Errorf("expected order preserved (a, b), got %q, %q", got[0].Template.ID, got[1].Template.ID)
	}
}

func TestDefaultMatcherWeights_SumsToOne(t *testing.T) {
	w := coach_advisor.DefaultMatcherWeights()
	sum := w.LUSRComponentWeight + w.RadarAxisWeight + w.MetricMatchWeight
	if sum < 0.999 || sum > 1.001 {
		t.Errorf("default weights should sum to 1.0, got %f", sum)
	}
}
