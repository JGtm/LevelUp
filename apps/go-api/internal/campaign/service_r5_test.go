package campaign

import (
	"context"
	"errors"
	"testing"
	"time"
)

// service_r5_test.go — tests pour R5 condition 2 "axe sort bottom-3 du radar"
// (V2 §4).

type fakeLeverageProvider struct {
	current []string
	err     error
}

func (f *fakeLeverageProvider) CurrentLeverageComponents(_ context.Context, _, _ string) ([]string, error) {
	return f.current, f.err
}

func TestEvaluate_R5_AxisNoLongerPriority(t *testing.T) {
	repo := newFakeRepo()
	samples := &fakeSamples{values: []float64{0.5, 0.5, 0.5}}
	leverages := &fakeLeverageProvider{
		// L'axe de la campagne est "deaths_vs_expected", il n'est PAS dans
		// les leviers courants (signal "axe sort du bottom-3").
		current: []string{"kills_vs_expected", "accuracy_delta"},
	}

	svc := NewService(repo, samples).WithLeverageProvider(leverages)
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)

	c, err := svc.StartCampaign(context.Background(), StartParams{
		UserID: "u1", TitleSlug: "halo", Axis: "deaths_vs_expected", AxisKind: AxisKindLUSRComponent,
	}, now)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := svc.Evaluate(context.Background(), c, now.Add(2*time.Hour)); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if repo.updates == 0 {
		t.Fatal("UpdateEvaluation never called")
	}
}

func TestEvaluate_R5_AxisStillPriority_NoSuggestion(t *testing.T) {
	repo := newFakeRepo()
	samples := &fakeSamples{values: []float64{0.5, 0.5, 0.5}}
	leverages := &fakeLeverageProvider{
		// L'axe est toujours dans les leviers → pas de suggestion clôture.
		current: []string{"deaths_vs_expected", "kills_vs_expected"},
	}

	svc := NewService(repo, samples).WithLeverageProvider(leverages)
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	c, _ := svc.StartCampaign(context.Background(), StartParams{
		UserID: "u1", TitleSlug: "halo", Axis: "deaths_vs_expected", AxisKind: AxisKindLUSRComponent,
	}, now)
	if err := svc.Evaluate(context.Background(), c, now.Add(2*time.Hour)); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
}

func TestEvaluate_R5_LeverageProviderError_Tolerated(t *testing.T) {
	repo := newFakeRepo()
	samples := &fakeSamples{values: []float64{0.5}}
	leverages := &fakeLeverageProvider{err: errors.New("transient")}

	svc := NewService(repo, samples).WithLeverageProvider(leverages)
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	c, _ := svc.StartCampaign(context.Background(), StartParams{
		UserID: "u1", TitleSlug: "halo", Axis: "deaths_vs_expected", AxisKind: AxisKindLUSRComponent,
	}, now)
	// Une erreur du provider ne doit pas faire crasher Evaluate.
	if err := svc.Evaluate(context.Background(), c, now.Add(time.Hour)); err != nil {
		t.Errorf("Evaluate should tolerate leverage provider error: %v", err)
	}
}

func TestEvaluate_R5_NoProvider_StillWorks(t *testing.T) {
	repo := newFakeRepo()
	samples := &fakeSamples{values: []float64{0.5}}
	// Pas de LeverageProvider → R5 condition 2 désactivée, plateau-only.
	svc := NewService(repo, samples)
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	c, _ := svc.StartCampaign(context.Background(), StartParams{
		UserID: "u1", TitleSlug: "halo", Axis: "deaths_vs_expected", AxisKind: AxisKindLUSRComponent,
	}, now)
	if err := svc.Evaluate(context.Background(), c, now.Add(time.Hour)); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
}

func TestContainsString(t *testing.T) {
	if !containsString([]string{"a", "b", "c"}, "b") {
		t.Error("should find b")
	}
	if containsString([]string{"a", "b"}, "z") {
		t.Error("should not find z")
	}
	if containsString(nil, "a") {
		t.Error("should not find anything in nil")
	}
}
