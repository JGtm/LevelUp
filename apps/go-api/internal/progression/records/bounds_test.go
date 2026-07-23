package records

import (
	"context"
	"math"
	"testing"
)

// bounds_test.go — tests A4 : bornes de vraisemblance (write-side) + catalogue
// des métriques connues (read-side).

func TestIsPlausibleValue(t *testing.T) {
	cases := []struct {
		metric TrackedMetric
		value  float64
		want   bool
		desc   string
	}{
		{MetricAccuracy, 0.55, true, "accuracy 0.55 (0..1) plausible"},
		{MetricAccuracy, 1.0, true, "accuracy 1.0 borne haute incluse"},
		{MetricAccuracy, 73.33, false, "accuracy 73.33 (7333 %) aberrante — incident prod"},
		{MetricAccuracy, -0.1, false, "accuracy négative aberrante"},
		{MetricKDA, 4.2, true, "kda 4.2 plausible"},
		{MetricKDA, -20, true, "kda -20 borne basse incluse"},
		{MetricKDA, 107, false, "best_kda 107 aberrant — incident prod"},
		{MetricKDA, -25, false, "kda -25 sous la borne"},
		{MetricKPM, 1.5, true, "kpm 1.5 plausible"},
		{MetricKPM, 21, false, "kpm 21 au-dessus de la borne"},
		{MetricPerformanceScore, 92, true, "perf 92 plausible"},
		{MetricPerformanceScore, 250, false, "perf 250 hors 0..100"},
		{MetricPSPM, 500, true, "pspm 500 plausible"},
		{MetricPSPM, math.NaN(), false, "NaN toujours rejeté"},
		{MetricPSPM, math.Inf(1), false, "Inf toujours rejeté"},
		{TrackedMetric("mystery"), 999999, true, "métrique sans borne connue = permissif"},
	}
	for _, c := range cases {
		if got := IsPlausibleValue(c.metric, c.value); got != c.want {
			t.Errorf("%s: IsPlausibleValue(%s, %v) = %v, want %v", c.desc, c.metric, c.value, got, c.want)
		}
	}
}

func TestIsKnownMetric(t *testing.T) {
	for _, m := range DefaultTrackedMetrics() {
		if !IsKnownMetric(string(m)) {
			t.Errorf("métrique suivie %q devrait être connue", m)
		}
	}
	for _, bad := range []string{"best_kda", "", "kills_total", "accuracy_pct"} {
		if IsKnownMetric(bad) {
			t.Errorf("métrique %q ne devrait PAS être au catalogue", bad)
		}
	}
}

// TestDetect_OutOfBoundsValueNotPersisted vérifie qu'une valeur aberrante
// (accuracy > 1) ne crée PAS de PB, ni dans le repo ni dans l'historique.
func TestDetect_OutOfBoundsValueNotPersisted(t *testing.T) {
	pb := newFakePBRepo()
	hist := newFakeHistoryRepo()
	d := newDetector(pb, hist)
	now := fixedDate(2026, 5, 18)

	// 12 matchs d'accuracy corrompue (73.33 = 7333 %, hors [0,1]).
	matches := make([]MatchInput, 12)
	for i := 0; i < 12; i++ {
		matches[i] = MatchInput{
			MatchID:  "m" + string(rune('a'+i)),
			PlayedAt: now.AddDate(0, 0, -i),
			Metrics:  map[TrackedMetric]float64{MetricAccuracy: 73.33},
		}
	}

	results, err := d.Detect(context.Background(), DetectInput{
		XUID: "x1", UserID: "x1", TitleSlug: "halo_infinite",
		Now: now, Matches: matches,
		Metrics: []TrackedMetric{MetricAccuracy},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	for _, r := range results {
		if r.NewPB {
			t.Errorf("valeur hors bornes ne doit pas créer de PB: %+v", r)
		}
	}
	if len(pb.pbs) != 0 {
		t.Errorf("PBRepo doit rester vide, %d entrées", len(pb.pbs))
	}
	if len(hist.entries) != 0 {
		t.Errorf("historique doit rester vide, %d entrées", len(hist.entries))
	}
}

// TestDetect_PlausibleValuePersists garantit que la borne ne bloque pas les
// valeurs légitimes (accuracy 0..1).
func TestDetect_PlausibleValuePersists(t *testing.T) {
	pb := newFakePBRepo()
	hist := newFakeHistoryRepo()
	d := newDetector(pb, hist)
	now := fixedDate(2026, 5, 18)

	matches := make([]MatchInput, 12)
	for i := 0; i < 12; i++ {
		matches[i] = MatchInput{
			MatchID:  "m" + string(rune('a'+i)),
			PlayedAt: now.AddDate(0, 0, -i),
			Metrics:  map[TrackedMetric]float64{MetricAccuracy: 0.40 + float64(i)*0.01},
		}
	}
	results, err := d.Detect(context.Background(), DetectInput{
		XUID: "x1", UserID: "x1", TitleSlug: "halo_infinite",
		Now: now, Matches: matches,
		Metrics: []TrackedMetric{MetricAccuracy},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	anyNewPB := false
	for _, r := range results {
		if r.NewPB {
			anyNewPB = true
		}
	}
	if !anyNewPB {
		t.Error("une valeur plausible devrait créer au moins un PB")
	}
}
