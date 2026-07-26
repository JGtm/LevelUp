package prestige

import (
	"testing"
	"time"
)

// ─────────── Evaluator threshold ───────────

func TestEvaluateThreshold_TargetReached(t *testing.T) {
	tuning := DefaultTuning()
	c := Challenge{
		Status:      StatusActive,
		Metric:      "FieldKDA",
		Target:      1.5,
		WindowType:  WindowSession,
		WindowValue: "1",
		EvalType:    EvalThreshold,
	}
	matches := []MatchSample{
		{MetricValue: 1.6}, {MetricValue: 1.7}, {MetricValue: 1.5},
		{MetricValue: 1.4}, {MetricValue: 1.6}, // 5+ matchs requis pour win_rate, pas KDA
	}
	res := EvaluateThreshold(tuning, c, matches, time.Now())
	if !res.StatusChanged {
		t.Fatalf("expected status change")
	}
	if res.NewStatus != StatusCompleted {
		t.Errorf("got %s want completed", res.NewStatus)
	}
	if res.Reason != EvalReasonTargetReached {
		t.Errorf("got reason %s", res.Reason)
	}
}

func TestEvaluateThreshold_BelowTarget(t *testing.T) {
	tuning := DefaultTuning()
	c := Challenge{
		Status:      StatusActive,
		Metric:      "FieldKDA",
		Target:      2.0,
		WindowType:  WindowSession,
		WindowValue: "1",
	}
	matches := []MatchSample{
		{MetricValue: 1.0}, {MetricValue: 1.2},
	}
	res := EvaluateThreshold(tuning, c, matches, time.Now())
	if res.StatusChanged {
		t.Error("expected no status change")
	}
	if res.Reason != EvalReasonProgress {
		t.Errorf("got reason %s want progress", res.Reason)
	}
}

func TestEvaluateThreshold_WinRateInsufficient(t *testing.T) {
	tuning := DefaultTuning()
	c := Challenge{
		Status:      StatusActive,
		Metric:      "FieldWinRate",
		Target:      60.0,
		WindowType:  WindowSession,
		WindowValue: "1",
	}
	// Seulement 3 matchs (min session = 5)
	matches := []MatchSample{
		{IsWin: true}, {IsWin: true}, {IsWin: true},
	}
	res := EvaluateThreshold(tuning, c, matches, time.Now())
	if res.StatusChanged {
		t.Error("should not change status with insufficient matches")
	}
	if res.Reason != EvalReasonInsufficient {
		t.Errorf("got %s want insufficient", res.Reason)
	}
}

func TestEvaluateThreshold_WinRateMet(t *testing.T) {
	tuning := DefaultTuning()
	c := Challenge{
		Status:      StatusActive,
		Metric:      "FieldWinRate",
		Target:      60.0,
		WindowType:  WindowSession,
		WindowValue: "1",
	}
	matches := []MatchSample{
		{IsWin: true}, {IsWin: true}, {IsWin: true},
		{IsWin: true}, {IsWin: false}, // 4/5 = 80%
	}
	res := EvaluateThreshold(tuning, c, matches, time.Now())
	if !res.StatusChanged {
		t.Error("expected status change at 80% win rate")
	}
	if res.NewStatus != StatusCompleted {
		t.Errorf("got %s want completed", res.NewStatus)
	}
}

func TestEvaluateThreshold_DeadlinePassedMissed(t *testing.T) {
	tuning := DefaultTuning()
	deadline := "2026-01-01"
	c := Challenge{
		Status:      StatusActive,
		Metric:      "FieldKDA",
		Target:      2.0,
		WindowType:  WindowDeadline,
		WindowValue: deadline,
	}
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) // après deadline
	matches := []MatchSample{{MetricValue: 1.0}}
	res := EvaluateThreshold(tuning, c, matches, now)
	if res.NewStatus != StatusExpired {
		t.Errorf("got %s want expired", res.NewStatus)
	}
	if res.Reason != EvalReasonDeadlinePassed {
		t.Errorf("got %s want deadline_passed", res.Reason)
	}
}

// ─────────── Evaluator cumulative ───────────

func TestEvaluateCumulative_TargetReached(t *testing.T) {
	tuning := DefaultTuning()
	c := Challenge{
		Status:   StatusActive,
		Metric:   "medal:Killtacular",
		Target:   5.0,
		EvalType: EvalCumulative,
	}
	// Total agrégé par l'appelant (2 + 3) — EvaluateCumulative reçoit la somme,
	// plus une liste d'événements dont aucune source de production n'existait.
	res := EvaluateCumulative(tuning, c, 5.0, time.Now())
	if !res.StatusChanged {
		t.Error("expected status change")
	}
	if res.NewStatus != StatusCompleted {
		t.Errorf("got %s want completed", res.NewStatus)
	}
	if res.NewValue != 5.0 {
		t.Errorf("got value %.2f want 5", res.NewValue)
	}
}

func TestEvaluateCumulative_DeadlinePassedMissed(t *testing.T) {
	tuning := DefaultTuning()
	c := Challenge{
		Status:      StatusActive,
		Target:      10.0,
		WindowType:  WindowDeadline,
		WindowValue: "2026-01-01",
	}
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	res := EvaluateCumulative(tuning, c, 3.0, now)
	if res.NewStatus != StatusExpired {
		t.Errorf("got %s want expired", res.NewStatus)
	}
}

// TestEvaluateCumulative_SumsNeverAverages — RÉGRESSION 2026-07-26. Les deux
// chemins d'évaluation retombaient sur EvaluateThreshold : un cumulatif était
// donc MOYENNÉ. Sur « 220 tirs à la tête » réalisés en 176 sur 40 matchs, la
// jauge affichait 4,4 (la moyenne par match) au lieu de 176 — 2 % de progression
// affichée sur un objectif à 80 %. Ce test verrouille la sémantique : la valeur
// rendue est le TOTAL, tel quel, jamais divisé par un nombre de matchs.
func TestEvaluateCumulative_SumsNeverAverages(t *testing.T) {
	tuning := DefaultTuning()
	c := Challenge{
		Status:      StatusActive,
		Metric:      "headshot_kills",
		Target:      220,
		EvalType:    EvalCumulative,
		WindowType:  WindowLastNMatches,
		WindowValue: "20",
	}
	res := EvaluateCumulative(tuning, c, 176.0, time.Now())
	if res.NewValue != 176.0 {
		t.Errorf("NewValue = %.2f, want 176 (le TOTAL, jamais une moyenne)", res.NewValue)
	}
	if res.StatusChanged || res.NewStatus != StatusActive {
		t.Errorf("176 < 220 → le défi reste actif, got status=%s changed=%v",
			res.NewStatus, res.StatusChanged)
	}
	if res.Reason != EvalReasonProgress {
		t.Errorf("Reason = %s, want progress", res.Reason)
	}
}

// TestEvalSince_NeverBeforeCreatedAt — invariant anti-complétion-rétroactive :
// la borne basse d'un défi n'est jamais antérieure à sa création, et une fenêtre
// rolling_days ne peut que la RESSERRER, jamais l'élargir.
func TestEvalSince_NeverBeforeCreatedAt(t *testing.T) {
	created := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name       string
		windowType WindowType
		value      string
		want       time.Time
	}{
		{"last_n_matches borné par created_at", WindowLastNMatches, "20", created},
		{"rolling 30j plus ancien que created_at → created_at gagne", WindowRollingDays, "30", created},
		{"rolling 7j plus récent que created_at → la fenêtre resserre", WindowRollingDays, "7",
			now.AddDate(0, 0, -7)},
		{"rolling avec valeur illisible → created_at", WindowRollingDays, "abc", created},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := evalSince(created, tc.windowType, tc.value, now); !got.Equal(tc.want) {
				t.Errorf("evalSince = %v, want %v", got, tc.want)
			}
		})
	}
}

// ─────────── PP amounts ───────────

func TestPPForCompletion_TierMatrix(t *testing.T) {
	tuning := DefaultTuning()
	cases := []struct {
		tier     Tier
		isSquad  bool
		dataTier DataTier
		want     int
	}{
		{TierNormal, false, DataFull, 50},
		{TierHeroic, false, DataFull, 75},
		{TierLegendary, false, DataFull, 125},
		{TierMythic, false, DataFull, 200},

		// Squad +20%
		{TierNormal, true, DataFull, 60},
		{TierMythic, true, DataFull, 240},

		// Estimated /2
		{TierHeroic, false, DataEstimated, 38}, // 75 * 0.5 = 37.5 → round 38
		{TierMythic, false, DataEstimated, 100},

		// Tracking = 0
		{TierMythic, false, DataTracking, 0},
		{TierMythic, true, DataTracking, 0},
	}
	for _, tc := range cases {
		got := PPForCompletion(tuning, tc.tier, tc.isSquad, tc.dataTier)
		if got != tc.want {
			t.Errorf("tier=%s squad=%v dt=%s: got %d want %d",
				tc.tier, tc.isSquad, tc.dataTier, got, tc.want)
		}
	}
}

func TestPPForArcCompletion(t *testing.T) {
	tuning := DefaultTuning() // ArcCompletionBonusRatio = 0.5
	cases := []struct {
		objectivesPP int
		want         int
	}{
		{0, 0},     // pas d'objectif créditant → pas de bonus
		{-10, 0},   // garde-fou
		{300, 150}, // 3 défis Héroïques (3×75) → +50 % = 150
		{75, 38},   // 1 seul défi Héroïque → 37.5 arrondi à 38
		{550, 275}, // arc mixte (75+125+150 + ...) → +50 %
	}
	for _, tc := range cases {
		if got := PPForArcCompletion(tuning, tc.objectivesPP); got != tc.want {
			t.Errorf("objectivesPP=%d: got %d want %d", tc.objectivesPP, got, tc.want)
		}
	}
}

func TestPPForMatch(t *testing.T) {
	tuning := DefaultTuning()
	if got := PPForMatch(tuning, false); got != 10 {
		t.Errorf("loss got %d want 10", got)
	}
	if got := PPForMatch(tuning, true); got != 25 {
		t.Errorf("win got %d want 25", got)
	}
}

func TestPPForStreak(t *testing.T) {
	tuning := DefaultTuning()
	if got := PPForStreak(tuning); got != 30 {
		t.Errorf("got %d want 30", got)
	}
}

func TestPPForMedal_Bounds(t *testing.T) {
	tuning := DefaultTuning()
	if got := PPForMedal(tuning, 0); got != 5 {
		t.Errorf("min got %d want 5", got)
	}
	if got := PPForMedal(tuning, 1); got != 20 {
		t.Errorf("max got %d want 20", got)
	}
	// Test clamp
	if got := PPForMedal(tuning, -0.5); got != 5 {
		t.Errorf("negative clamp got %d want 5", got)
	}
	if got := PPForMedal(tuning, 1.5); got != 20 {
		t.Errorf("over-1 clamp got %d want 20", got)
	}
}

// ─────────── Tuning ───────────

func TestDefaultTuning_Validate(t *testing.T) {
	if err := DefaultTuning().Validate(); err != nil {
		t.Errorf("DefaultTuning should be valid, got %v", err)
	}
}

func TestTuning_LoadFromFile_Fallback(t *testing.T) {
	// Fichier inexistant → fallback DefaultTuning + warn (pas d'erreur retournée)
	got := LoadTuning("/nonexistent/path/tuning.toml")
	if err := got.Validate(); err != nil {
		t.Errorf("fallback should be valid, got %v", err)
	}
}

func TestTuning_CooldownDuration(t *testing.T) {
	tuning := DefaultTuning()
	if d := tuning.CooldownDuration(StatusExpired); d != 12*time.Hour {
		t.Errorf("expired got %v want 12h", d)
	}
	if d := tuning.CooldownDuration(StatusAbandoned); d != 24*time.Hour {
		t.Errorf("abandoned got %v want 24h", d)
	}
	if d := tuning.CooldownDuration(StatusActive); d != 0 {
		t.Errorf("active should have no cooldown, got %v", d)
	}
}
