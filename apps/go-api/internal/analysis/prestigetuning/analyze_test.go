package prestigetuning

import (
	"testing"
	"time"
)

var fixedNow = time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)

func testGrammar() GrammarView {
	return NewGrammarView(map[string][]string{
		"accuracy": {"last_n_matches:10", "rolling_days:7"},
		"kda":      {"session", "last_n_matches:10"},
		"win_rate": {"rolling_days:14"},
	})
}

// nominal : métrique avec échantillon suffisant et complétion sous le seuil →
// recommandation d'ajustement.
func TestAnalyze_RecommendAdjust(t *testing.T) {
	counts := []MetricWindowCount{
		{Source: "coach", Metric: "accuracy", WindowType: "last_n_matches", WindowValue: "10",
			Created: 40, Completed: 8},
		{Source: "coach", Metric: "accuracy", WindowType: "rolling_days", WindowValue: "7",
			Created: 20, Completed: 4},
	}
	rep := Analyze(counts, nil, testGrammar(), DefaultThresholds(), fixedNow)

	m := findMetric(t, rep.Metrics, "accuracy")
	if m.Status != StatusRecommendAdjust {
		t.Fatalf("status = %q, want %q", m.Status, StatusRecommendAdjust)
	}
	if m.Sample != 60 {
		t.Errorf("sample = %d, want 60", m.Sample)
	}
	if got := m.CompletionRate; got < 0.19 || got > 0.21 {
		t.Errorf("completion = %.3f, want ~0.20", got)
	}
	if len(m.Windows) != 2 {
		t.Errorf("windows = %d, want 2", len(m.Windows))
	}
}

// échantillon sous le minimum → données insuffisantes, aucune recommandation.
func TestAnalyze_InsufficientSample(t *testing.T) {
	counts := []MetricWindowCount{
		{Source: "coach", Metric: "accuracy", WindowType: "last_n_matches", WindowValue: "10",
			Created: 10, Completed: 1},
	}
	rep := Analyze(counts, nil, testGrammar(), DefaultThresholds(), fixedNow)
	m := findMetric(t, rep.Metrics, "accuracy")
	if m.Status != StatusInsufficientData {
		t.Fatalf("status = %q, want %q", m.Status, StatusInsufficientData)
	}
}

// complétion saine au-dessus du seuil → healthy, pas de recommandation.
func TestAnalyze_Healthy(t *testing.T) {
	counts := []MetricWindowCount{
		{Source: "coach", Metric: "accuracy", WindowType: "last_n_matches", WindowValue: "10",
			Created: 100, Completed: 60},
	}
	rep := Analyze(counts, nil, testGrammar(), DefaultThresholds(), fixedNow)
	m := findMetric(t, rep.Metrics, "accuracy")
	if m.Status != StatusHealthy {
		t.Fatalf("status = %q, want %q", m.Status, StatusHealthy)
	}
}

// métrique télémétrie absente de la grammaire → orpheline.
func TestAnalyze_OrphanMetric(t *testing.T) {
	counts := []MetricWindowCount{
		{Source: "coach", Metric: "shots_fired", WindowType: "rolling_days", WindowValue: "7",
			Created: 80, Completed: 5},
	}
	rep := Analyze(counts, nil, testGrammar(), DefaultThresholds(), fixedNow)
	if len(rep.Orphans) != 1 {
		t.Fatalf("orphans = %d, want 1", len(rep.Orphans))
	}
	if rep.Orphans[0].Metric != "shots_fired" || rep.Orphans[0].InGrammar {
		t.Errorf("orphan mal classé : %+v", rep.Orphans[0])
	}
	// une orpheline ne doit JAMAIS produire de recommandation d'ajustement,
	// même sous le seuil de complétion (non actionnable sur le TOML).
	if rep.Orphans[0].Status == StatusRecommendAdjust {
		t.Errorf("orpheline ne doit pas être 'recommend_adjust'")
	}
}

// métrique de grammaire sans aucune télémétrie → listée en données insuffisantes.
func TestAnalyze_GrammarMetricNoData(t *testing.T) {
	rep := Analyze(nil, nil, testGrammar(), DefaultThresholds(), fixedNow)
	if len(rep.Metrics) != 3 {
		t.Fatalf("metrics = %d, want 3 (toutes les métriques de grammaire)", len(rep.Metrics))
	}
	for _, m := range rep.Metrics {
		if m.Status != StatusInsufficientData || m.Sample != 0 {
			t.Errorf("%s : status=%q sample=%d, want insufficient/0", m.Metric, m.Status, m.Sample)
		}
	}
}

// la source non analysée n'influence pas les recommandations mais alimente le
// contexte total d'événements.
func TestAnalyze_SourceFilter(t *testing.T) {
	counts := []MetricWindowCount{
		{Source: "user", Metric: "accuracy", WindowType: "last_n_matches", WindowValue: "10",
			Created: 200, Completed: 2}, // user, ignoré pour la reco
		{Source: "coach", Metric: "accuracy", WindowType: "last_n_matches", WindowValue: "10",
			Created: 60, Completed: 40}, // coach, sain
	}
	rep := Analyze(counts, nil, testGrammar(), DefaultThresholds(), fixedNow)
	m := findMetric(t, rep.Metrics, "accuracy")
	if m.Status != StatusHealthy {
		t.Errorf("status = %q, want healthy (user ignoré)", m.Status)
	}
	if m.Sample != 60 {
		t.Errorf("sample = %d, want 60 (coach seulement)", m.Sample)
	}
	if rep.TotalEvents != 302 {
		t.Errorf("total events = %d, want 302 (user 202 + coach 100)", rep.TotalEvents)
	}
}

// seuils personnalisés respectés.
func TestAnalyze_CustomThresholds(t *testing.T) {
	counts := []MetricWindowCount{
		{Source: "coach", Metric: "kda", WindowType: "session", WindowValue: "",
			Created: 30, Completed: 12}, // 40%
	}
	thr := Thresholds{MinCompletionRate: 0.50, MinSample: 25, Source: "coach"}
	rep := Analyze(counts, nil, testGrammar(), thr, fixedNow)
	m := findMetric(t, rep.Metrics, "kda")
	if m.Status != StatusRecommendAdjust {
		t.Fatalf("status = %q, want recommend_adjust (40%% < 50%%, n=30 >= 25)", m.Status)
	}
	// fenêtre "session" (window_value vide) doit être reconnue dans la grammaire.
	if len(m.Windows) != 1 || !m.Windows[0].InGrammar {
		t.Errorf("fenêtre session non reconnue : %+v", m.Windows)
	}
}

func findMetric(t *testing.T, metrics []MetricRecommendation, name string) MetricRecommendation {
	t.Helper()
	for _, m := range metrics {
		if m.Metric == name {
			return m
		}
	}
	t.Fatalf("métrique %q absente du rapport", name)
	return MetricRecommendation{}
}
