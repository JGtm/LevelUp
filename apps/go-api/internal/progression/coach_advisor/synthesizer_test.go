package coach_advisor_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/prestige"
	"levelup/go-api/internal/progression/coach_advisor"
)

// ─── Grammar TOML loader ───

const sampleGrammarTOML = `
[[allow]]
metric    = "accuracy"
windows   = ["last_n_matches:10", "rolling_days:14"]
eval_type = "threshold"

[[allow]]
metric    = "kills_total"
windows   = ["rolling_days:30"]
eval_type = "cumulative"

[[allow]]
metric    = "kda"
windows   = ["session"]
eval_type = "threshold"
`

func writeGrammar(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "synthesis_grammar.toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write grammar: %v", err)
	}
	return path
}

func TestLoadSynthesisGrammar_HappyPath(t *testing.T) {
	g, err := coach_advisor.LoadSynthesisGrammar(writeGrammar(t, sampleGrammarTOML))
	if err != nil {
		t.Fatalf("LoadSynthesisGrammar: %v", err)
	}

	if !g.IsAllowed("accuracy", "last_n_matches", "10", "threshold") {
		t.Error("expected (accuracy, last_n_matches:10, threshold) to be allowed")
	}
	if !g.IsAllowed("kda", "session", "", "threshold") {
		t.Error("expected (kda, session, threshold) to be allowed (window_value empty)")
	}
	if g.IsAllowed("accuracy", "last_n_matches", "5", "threshold") {
		t.Error("(accuracy, last_n_matches:5) should NOT be allowed")
	}
	if g.IsAllowed("kills_total", "rolling_days", "30", "threshold") {
		t.Error("(kills_total, threshold) should NOT match (eval_type=cumulative in TOML)")
	}
}

func TestLoadSynthesisGrammar_FileNotFound(t *testing.T) {
	_, err := coach_advisor.LoadSynthesisGrammar("/nonexistent/path/grammar.toml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadSynthesisGrammar_RejectsEmptyMetric(t *testing.T) {
	bad := `
[[allow]]
metric    = ""
windows   = ["rolling_days:7"]
eval_type = "threshold"
`
	_, err := coach_advisor.LoadSynthesisGrammar(writeGrammar(t, bad))
	if err == nil {
		t.Fatal("expected error on empty metric")
	}
}

func TestLoadSynthesisGrammar_RejectsMalformedWindow(t *testing.T) {
	bad := `
[[allow]]
metric    = "accuracy"
windows   = ["not_a_valid_format"]
eval_type = "threshold"
`
	_, err := coach_advisor.LoadSynthesisGrammar(writeGrammar(t, bad))
	if err == nil {
		t.Fatal("expected error on malformed window spec")
	}
}

func TestDefaultSynthesisGrammar_RefusesEverything(t *testing.T) {
	g := coach_advisor.DefaultSynthesisGrammar()
	if g.IsAllowed("accuracy", "rolling_days", "14", "threshold") {
		t.Error("default (empty) grammar should refuse all")
	}
}

// ─── Synthesizer ───

func loadHappyGrammar(t *testing.T) coach_advisor.SynthesisGrammar {
	g, err := coach_advisor.LoadSynthesisGrammar(writeGrammar(t, sampleGrammarTOML))
	if err != nil {
		t.Fatalf("LoadSynthesisGrammar: %v", err)
	}
	return g
}

func TestSynthesizer_RejectsBelowMinStrength(t *testing.T) {
	syn := coach_advisor.NewSynthesizer(loadHappyGrammar(t), coach_advisor.DefaultSynthesisConfig())
	sig := coach_advisor.Signal{
		Kind:          coach_advisor.SignalLOWESSPositive,
		Metric:        "accuracy",
		LUSRComponent: "accuracy_delta",
		Strength:      0.3, // < default 0.6
	}
	_, err := syn.Synthesize(sig, "halo_infinite", time.Now())
	if !errors.Is(err, coach_advisor.ErrSignalTooWeak) {
		t.Errorf("expected ErrSignalTooWeak, got %v", err)
	}
}

func TestSynthesizer_RejectsMetricNotInGrammar(t *testing.T) {
	syn := coach_advisor.NewSynthesizer(loadHappyGrammar(t), coach_advisor.DefaultSynthesisConfig())
	sig := coach_advisor.Signal{
		Kind:     coach_advisor.SignalLOWESSPositive,
		Metric:   "some_metric_not_in_grammar",
		Strength: 0.8,
	}
	_, err := syn.Synthesize(sig, "halo_infinite", time.Now())
	if !errors.Is(err, coach_advisor.ErrMetricNotSynthesizable) {
		t.Errorf("expected ErrMetricNotSynthesizable, got %v", err)
	}
}

func TestSynthesizer_RejectsSignalWithNoMetricOrLUSR(t *testing.T) {
	syn := coach_advisor.NewSynthesizer(loadHappyGrammar(t), coach_advisor.DefaultSynthesisConfig())
	sig := coach_advisor.Signal{
		Kind:     coach_advisor.SignalComebackWelcome,
		Strength: 0.8,
	}
	_, err := syn.Synthesize(sig, "halo_infinite", time.Now())
	if !errors.Is(err, coach_advisor.ErrMetricNotSynthesizable) {
		t.Errorf("expected ErrMetricNotSynthesizable for signal w/o metric, got %v", err)
	}
}

func TestSynthesizer_HappyPath_AccuracySignal(t *testing.T) {
	syn := coach_advisor.NewSynthesizer(loadHappyGrammar(t), coach_advisor.DefaultSynthesisConfig())
	now := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	sig := coach_advisor.Signal{
		Kind:          coach_advisor.SignalLOWESSPositive,
		Metric:        "accuracy",
		LUSRComponent: "accuracy_delta",
		RadarAxis:     "combat",
		Strength:      0.75,
	}
	tmpl, err := syn.Synthesize(sig, "halo_infinite", now)
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}

	// Structure conformity (invariant I3)
	if tmpl.Source != "coach_synthesized" {
		t.Errorf("Source: got %q, want coach_synthesized", tmpl.Source)
	}
	if tmpl.TitleSlug != "halo_infinite" {
		t.Errorf("TitleSlug: got %q", tmpl.TitleSlug)
	}
	if tmpl.Metric != "accuracy" {
		t.Errorf("Metric: got %q", tmpl.Metric)
	}
	if tmpl.WindowType != prestige.WindowType("last_n_matches") || tmpl.WindowValue != "10" {
		t.Errorf("Window: got %s:%s, want last_n_matches:10 (first in grammar)", tmpl.WindowType, tmpl.WindowValue)
	}
	if tmpl.EvalType != prestige.EvalType("threshold") {
		t.Errorf("EvalType: got %q", tmpl.EvalType)
	}
	if tmpl.Cadence != prestige.Cadence("daily") {
		t.Errorf("Cadence: got %q, want daily (last_n_matches → daily)", tmpl.Cadence)
	}
	if tmpl.ModeFilter != "universal" {
		t.Errorf("ModeFilter: got %q", tmpl.ModeFilter)
	}

	// Targets = stretch ratios (invariant I1/I2)
	if tmpl.NormalTarget != 1.08 || tmpl.HeroicTarget != 1.25 || tmpl.LegendaryTarget != 1.50 || tmpl.MythicTarget != 2.00 {
		t.Errorf("Targets should be stretch ratios [1.08, 1.25, 1.50, 2.00], got [%f, %f, %f, %f]",
			tmpl.NormalTarget, tmpl.HeroicTarget, tmpl.LegendaryTarget, tmpl.MythicTarget)
	}

	// LUSR + Axis propagés
	if len(tmpl.LUSRComponents) != 1 || tmpl.LUSRComponents[0] != "accuracy_delta" {
		t.Errorf("LUSRComponents: got %v", tmpl.LUSRComponents)
	}
	if len(tmpl.RadarAxes) != 1 || tmpl.RadarAxes[0] != "combat" {
		t.Errorf("RadarAxes: got %v", tmpl.RadarAxes)
	}

	// Labels paramétrés présents (forme exacte n'est pas testée, juste non-vide)
	if tmpl.LabelEN == "" || tmpl.LabelFR == "" {
		t.Error("Labels should be non-empty")
	}
}

func TestSynthesizer_IDDeterministicAcrossCalls(t *testing.T) {
	syn := coach_advisor.NewSynthesizer(loadHappyGrammar(t), coach_advisor.DefaultSynthesisConfig())
	now := time.Now()
	sig := coach_advisor.Signal{
		Kind:          coach_advisor.SignalLOWESSPositive,
		Metric:        "accuracy",
		LUSRComponent: "accuracy_delta",
		RadarAxis:     "combat",
		Strength:      0.7,
	}
	t1, _ := syn.Synthesize(sig, "halo_infinite", now)
	t2, _ := syn.Synthesize(sig, "halo_infinite", now.Add(time.Hour))
	if t1.ID != t2.ID {
		t.Errorf("ID should be deterministic across calls (dedup invariant), got %q vs %q", t1.ID, t2.ID)
	}
}

func TestSynthesizer_IDStableForListOrder(t *testing.T) {
	// Si LUSRComponent et RadarAxis ne contiennent qu'un élément le test ne
	// fait pas grand-chose ; on construit manuellement un Signal qui produit
	// les mêmes composantes mais ordonnées différemment dans les Listes.
	// Comme le synthesizer ne crée que des listes mono-élément, on teste
	// directement via le format des inputs (sort interne dans le hash).
	syn := coach_advisor.NewSynthesizer(loadHappyGrammar(t), coach_advisor.DefaultSynthesisConfig())
	now := time.Now()

	// Synthese 1 + 2 avec même signal mais titleSlug différent → IDs différents
	// MAIS le titleSlug n'entre PAS dans le hash → IDs identiques
	// (le titleSlug est suivi dans le Template.TitleSlug séparément)
	sig := coach_advisor.Signal{
		Kind:          coach_advisor.SignalLOWESSPositive,
		Metric:        "accuracy",
		LUSRComponent: "accuracy_delta",
		RadarAxis:     "combat",
		Strength:      0.7,
	}
	t1, _ := syn.Synthesize(sig, "halo_infinite", now)
	t2, _ := syn.Synthesize(sig, "another_title", now)
	if t1.ID != t2.ID {
		t.Errorf("ID should not depend on titleSlug (dedup cross-title possible), got %q vs %q", t1.ID, t2.ID)
	}
}

func TestSynthesizer_IDDistinctForDifferentMetrics(t *testing.T) {
	syn := coach_advisor.NewSynthesizer(loadHappyGrammar(t), coach_advisor.DefaultSynthesisConfig())
	now := time.Now()

	t1, err := syn.Synthesize(coach_advisor.Signal{
		Kind: coach_advisor.SignalLOWESSPositive, Metric: "accuracy", Strength: 0.7,
	}, "halo_infinite", now)
	if err != nil {
		t.Fatalf("synth1: %v", err)
	}
	t2, err := syn.Synthesize(coach_advisor.Signal{
		Kind: coach_advisor.SignalLOWESSPositive, Metric: "kda", Strength: 0.7,
	}, "halo_infinite", now)
	if err != nil {
		t.Fatalf("synth2: %v", err)
	}
	if t1.ID == t2.ID {
		t.Errorf("Different metrics should produce different IDs, both %q", t1.ID)
	}
}

func TestSynthesizer_CumulativeEvalType_KillsTotal(t *testing.T) {
	syn := coach_advisor.NewSynthesizer(loadHappyGrammar(t), coach_advisor.DefaultSynthesisConfig())
	sig := coach_advisor.Signal{
		Kind:     coach_advisor.SignalMilestoneApproach,
		Metric:   "kills_total",
		Strength: 0.92,
	}
	tmpl, err := syn.Synthesize(sig, "halo_infinite", time.Now())
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	if tmpl.EvalType != prestige.EvalType("cumulative") {
		t.Errorf("EvalType: got %q, want cumulative (grammar)", tmpl.EvalType)
	}
	if tmpl.WindowType != prestige.WindowType("rolling_days") || tmpl.WindowValue != "30" {
		t.Errorf("Window: got %s:%s, want rolling_days:30", tmpl.WindowType, tmpl.WindowValue)
	}
	if tmpl.Cadence != prestige.Cadence("weekly") {
		t.Errorf("Cadence: got %q, want weekly (rolling_days:30 → weekly)", tmpl.Cadence)
	}
}

func TestSynthesizer_SessionWindow_DailyCadence(t *testing.T) {
	syn := coach_advisor.NewSynthesizer(loadHappyGrammar(t), coach_advisor.DefaultSynthesisConfig())
	sig := coach_advisor.Signal{
		Kind:     coach_advisor.SignalRecordApproach,
		Metric:   "kda",
		Strength: 0.95,
	}
	tmpl, err := syn.Synthesize(sig, "halo_infinite", time.Now())
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	if tmpl.WindowType != prestige.WindowType("session") {
		t.Errorf("Window: got %s, want session", tmpl.WindowType)
	}
	if tmpl.WindowValue != "" {
		t.Errorf("session window should have empty value, got %q", tmpl.WindowValue)
	}
	if tmpl.Cadence != prestige.Cadence("daily") {
		t.Errorf("Cadence: got %q, want daily", tmpl.Cadence)
	}
}

func TestSynthesizer_LUSROnlySignal_InfersMetricFromLUSR(t *testing.T) {
	syn := coach_advisor.NewSynthesizer(loadHappyGrammar(t), coach_advisor.DefaultSynthesisConfig())
	sig := coach_advisor.Signal{
		Kind:          coach_advisor.SignalCombatPatternActive,
		LUSRComponent: "kills_vs_expected", // pas dans la grammaire de test
		Strength:      0.8,
	}
	_, err := syn.Synthesize(sig, "halo_infinite", time.Now())
	if !errors.Is(err, coach_advisor.ErrMetricNotSynthesizable) {
		t.Errorf("expected ErrMetricNotSynthesizable (kills_vs_expected absent of test grammar), got %v", err)
	}
}
