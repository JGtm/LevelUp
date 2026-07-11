package coach

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/progression/milestones"
	"levelup/go-api/internal/progression/records"
	"levelup/go-api/internal/progression/streaks"
)

// generator_test.go — unit tests du générateur d'alertes coach.

// ─── Helpers ────────────────────────────────────────────────────────────────

func fixedDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 12, 0, 0, 0, time.UTC)
}

func findAlertByType(alerts []Alert, t AlertType) (Alert, bool) {
	for _, a := range alerts {
		if a.Type == t {
			return a, true
		}
	}
	return Alert{}, false
}

func findAlertByDedup(alerts []Alert, t AlertType, dedup string) (Alert, bool) {
	for _, a := range alerts {
		if a.Type == t && a.DedupKey == dedup {
			return a, true
		}
	}
	return Alert{}, false
}

// ─── Streak alerts ──────────────────────────────────────────────────────────

func TestGenerate_StreakMilestone_FiresAtThreshold(t *testing.T) {
	g := NewGenerator()
	now := fixedDate(2026, 5, 18)

	// Streak qui vient d'atteindre 8 jours (palier 8 défini dans
	// streakMilestoneThresholds).
	incAt := now
	alerts := g.Generate(context.Background(), GenerateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: now,
		StreakResults: []streaks.EvaluationResult{{
			Transition: streaks.TransitionIncremented,
			Streak: streaks.Streak{
				ID: "s1", UserID: "u1", TitleSlug: "halo_infinite",
				Type: streaks.StreakTypeDailyPlay, StartedAt: now.AddDate(0, 0, -7),
				CurrentLength: 8, BestLength: 8, LastIncrementAt: &incAt,
				Status: streaks.StreakStatusActive,
			},
		}},
	})

	a, ok := findAlertByType(alerts, AlertTypeStreakMilestone)
	if !ok {
		t.Fatal("expected AlertTypeStreakMilestone")
	}
	if a.DedupKey != "daily_play|8" {
		t.Errorf("DedupKey = %q, want daily_play|8", a.DedupKey)
	}
	if a.Params["length"] != 8 {
		t.Errorf("length = %v, want 8", a.Params["length"])
	}
	if a.Params["multiplier"] != 1.25 {
		t.Errorf("multiplier = %v, want 1.25", a.Params["multiplier"])
	}
}

func TestGenerate_StreakMilestone_SkippedAtNonThreshold(t *testing.T) {
	g := NewGenerator()
	now := fixedDate(2026, 5, 18)
	incAt := now
	alerts := g.Generate(context.Background(), GenerateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: now,
		StreakResults: []streaks.EvaluationResult{{
			Transition: streaks.TransitionIncremented,
			Streak: streaks.Streak{
				CurrentLength:   5, // pas un palier (4, 8, 15, 30)
				LastIncrementAt: &incAt,
				Type:            streaks.StreakTypeDailyPlay,
			},
		}},
	})
	if _, ok := findAlertByType(alerts, AlertTypeStreakMilestone); ok {
		t.Errorf("no alert expected at length=5")
	}
}

func TestGenerate_StreakBroken_NoAlert(t *testing.T) {
	g := NewGenerator()
	now := fixedDate(2026, 5, 18)
	alerts := g.Generate(context.Background(), GenerateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: now,
		StreakResults: []streaks.EvaluationResult{{
			Transition: streaks.TransitionBroken,
			Streak: streaks.Streak{
				CurrentLength: 7, Type: streaks.StreakTypeDailyPlay,
			},
		}},
	})
	if _, ok := findAlertByType(alerts, AlertTypeStreakMilestone); ok {
		t.Errorf("broken transition should produce NO alert (positive feedback only)")
	}
}

// ─── Record alerts ──────────────────────────────────────────────────────────

func TestGenerate_RecordBroken(t *testing.T) {
	g := NewGenerator()
	now := fixedDate(2026, 5, 18)
	prev := 80.0
	alerts := g.Generate(context.Background(), GenerateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: now,
		RecordResults: []records.DetectionResult{{
			Metric: records.MetricPerformanceScore, Period: records.RecordPeriodAllTime,
			Value: 92, PreviousValue: &prev, MatchID: "m_42", NewPB: true,
		}},
	})
	a, ok := findAlertByDedup(alerts, AlertTypeRecordBroken, "performance_score|all_time")
	if !ok {
		t.Fatal("expected RecordBroken alert")
	}
	if a.Params["value"] != 92.0 {
		t.Errorf("value = %v, want 92", a.Params["value"])
	}
	if a.Params["previous_value"] != 80.0 {
		t.Errorf("previous_value = %v, want 80", a.Params["previous_value"])
	}
	// Vérifier que la catégorie est bien personal_record (existante).
	if a.Type.NotificationCategory() != notifications.CategoryPersonalRecord {
		t.Errorf("Category = %s, want personal_record", a.Type.NotificationCategory())
	}
}

func TestGenerate_RecordNearMiss(t *testing.T) {
	g := NewGenerator()
	now := fixedDate(2026, 5, 18)
	prev := 100.0
	alerts := g.Generate(context.Background(), GenerateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: now,
		RecordResults: []records.DetectionResult{{
			Metric: records.MetricKDA, Period: records.RecordPeriod30d,
			Value: 96, PreviousValue: &prev, NearMiss: true,
		}},
	})
	a, ok := findAlertByDedup(alerts, AlertTypeRecordNearMiss, "kda|30d")
	if !ok {
		t.Fatal("expected RecordNearMiss alert")
	}
	if a.Params["target"] != 100.0 {
		t.Errorf("target = %v, want 100", a.Params["target"])
	}
}

// ─── B5/B12 : collapse période la plus large + seed silencieux ────────────────

func countAlertsByType(alerts []Alert, t AlertType) int {
	n := 0
	for _, a := range alerts {
		if a.Type == t {
			n++
		}
	}
	return n
}

// DP3 : un record battu sur 30d+90d+all_time → 1 seule alerte (all_time).
func TestGenerate_RecordBroken_CollapseWidestPeriod(t *testing.T) {
	g := NewGenerator()
	now := fixedDate(2026, 5, 18)
	prev := 80.0
	res := []records.DetectionResult{
		{Metric: records.MetricKDA, Period: records.RecordPeriod30d, Value: 92, PreviousValue: &prev, NewPB: true},
		{Metric: records.MetricKDA, Period: records.RecordPeriod90d, Value: 92, PreviousValue: &prev, NewPB: true},
		{Metric: records.MetricKDA, Period: records.RecordPeriodAllTime, Value: 92, PreviousValue: &prev, NewPB: true},
	}
	alerts := g.Generate(context.Background(), GenerateInput{UserID: "u1", Now: now, RecordResults: res})
	if got := countAlertsByType(alerts, AlertTypeRecordBroken); got != 1 {
		t.Fatalf("attendu 1 alerte RecordBroken (all_time), got %d", got)
	}
	if _, ok := findAlertByDedup(alerts, AlertTypeRecordBroken, "kda|all_time"); !ok {
		t.Error("la période conservée doit être all_time")
	}
}

// Record battu sur 30d seul → 1 alerte (30d).
func TestGenerate_RecordBroken_SinglePeriodKept(t *testing.T) {
	g := NewGenerator()
	now := fixedDate(2026, 5, 18)
	prev := 80.0
	res := []records.DetectionResult{
		{Metric: records.MetricKDA, Period: records.RecordPeriod30d, Value: 92, PreviousValue: &prev, NewPB: true},
	}
	alerts := g.Generate(context.Background(), GenerateInput{UserID: "u1", Now: now, RecordResults: res})
	if _, ok := findAlertByDedup(alerts, AlertTypeRecordBroken, "kda|30d"); !ok {
		t.Error("attendu 1 alerte RecordBroken 30d")
	}
}

// Near-miss 90d + all_time → 1 alerte (all_time).
func TestGenerate_RecordNearMiss_CollapseWidest(t *testing.T) {
	g := NewGenerator()
	now := fixedDate(2026, 5, 18)
	prev := 100.0
	res := []records.DetectionResult{
		{Metric: records.MetricKDA, Period: records.RecordPeriod90d, Value: 97, PreviousValue: &prev, NearMiss: true},
		{Metric: records.MetricKDA, Period: records.RecordPeriodAllTime, Value: 97, PreviousValue: &prev, NearMiss: true},
	}
	alerts := g.Generate(context.Background(), GenerateInput{UserID: "u1", Now: now, RecordResults: res})
	if got := countAlertsByType(alerts, AlertTypeRecordNearMiss); got != 1 {
		t.Fatalf("attendu 1 alerte near-miss (all_time), got %d", got)
	}
	if _, ok := findAlertByDedup(alerts, AlertTypeRecordNearMiss, "kda|all_time"); !ok {
		t.Error("la période conservée doit être all_time")
	}
}

// Métriques DIFFÉRENTES → les deux passent (collapse est per-métrique).
func TestGenerate_RecordDifferentMetrics_BothPass(t *testing.T) {
	g := NewGenerator()
	now := fixedDate(2026, 5, 18)
	prevA, prevB := 80.0, 100.0
	res := []records.DetectionResult{
		{Metric: records.MetricKDA, Period: records.RecordPeriod30d, Value: 92, PreviousValue: &prevA, NewPB: true},
		{Metric: records.MetricPerformanceScore, Period: records.RecordPeriodAllTime, Value: 97, PreviousValue: &prevB, NearMiss: true},
	}
	alerts := g.Generate(context.Background(), GenerateInput{UserID: "u1", Now: now, RecordResults: res})
	if _, ok := findAlertByDedup(alerts, AlertTypeRecordBroken, "kda|30d"); !ok {
		t.Error("attendu RecordBroken kda|30d")
	}
	if _, ok := findAlertByDedup(alerts, AlertTypeRecordNearMiss, "performance_score|all_time"); !ok {
		t.Error("attendu RecordNearMiss performance_score|all_time")
	}
}

// DP12 : premier record (PreviousValue == nil) → seed silencieux, 0 alerte.
func TestGenerate_RecordBroken_SeedSilencieux(t *testing.T) {
	g := NewGenerator()
	now := fixedDate(2026, 5, 18)
	res := []records.DetectionResult{
		{Metric: records.MetricKDA, Period: records.RecordPeriod30d, Value: 92, PreviousValue: nil, NewPB: true},
		{Metric: records.MetricKDA, Period: records.RecordPeriod90d, Value: 92, PreviousValue: nil, NewPB: true},
		{Metric: records.MetricKDA, Period: records.RecordPeriodAllTime, Value: 92, PreviousValue: nil, NewPB: true},
	}
	alerts := g.Generate(context.Background(), GenerateInput{UserID: "u1", Now: now, RecordResults: res})
	if got := countAlertsByType(alerts, AlertTypeRecordBroken); got != 0 {
		t.Errorf("premier record sans référence → 0 alerte (seed), got %d", got)
	}
}

// ─── Milestone alerts ───────────────────────────────────────────────────────

func TestGenerate_MilestoneUnlocked(t *testing.T) {
	g := NewGenerator()
	now := fixedDate(2026, 5, 18)
	alerts := g.Generate(context.Background(), GenerateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: now,
		MilestoneResults: []milestones.DetectionResult{{
			Milestone: milestones.CatalogEntry{
				ID: "halo.matches.100", Metric: "matches_played", Threshold: 100,
				TitleEN: "Centurion", TitleFR: "Centurion",
			},
			Earned: true,
		}},
	})
	a, ok := findAlertByDedup(alerts, AlertTypeMilestoneUnlocked, "halo.matches.100")
	if !ok {
		t.Fatal("expected MilestoneUnlocked alert")
	}
	if a.Params["title_fr"] != "Centurion" {
		t.Errorf("title_fr missing")
	}
}

func TestGenerate_MilestoneNearMiss(t *testing.T) {
	g := NewGenerator()
	now := fixedDate(2026, 5, 18)
	alerts := g.Generate(context.Background(), GenerateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: now,
		MilestoneResults: []milestones.DetectionResult{{
			Milestone: milestones.CatalogEntry{ID: "halo.kills.1000", Metric: "kills", Threshold: 1000, TitleEN: "Killer", TitleFR: "Tueur"},
			NearMiss:  true,
			Progress:  0.92,
		}},
	})
	a, ok := findAlertByDedup(alerts, AlertTypeMilestoneNearMiss, "halo.kills.1000")
	if !ok {
		t.Fatal("expected MilestoneNearMiss alert")
	}
	if a.Params["progress"].(float64) < 0.91 {
		t.Errorf("progress = %v, want >= 0.91", a.Params["progress"])
	}
}

// ─── LUSR tier approach ─────────────────────────────────────────────────────

func TestGenerate_LUSRTierApproach_WithinDelta(t *testing.T) {
	g := NewGenerator()
	now := fixedDate(2026, 5, 18)
	alerts := g.Generate(context.Background(), GenerateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: now,
		LUSR: LUSRSnapshot{Mu: 1495, NextTierName: "diamond_iv", NextTierMu: 1500},
	})
	a, ok := findAlertByDedup(alerts, AlertTypeLUSRTierApproach, "diamond_iv")
	if !ok {
		t.Fatal("expected LUSRTierApproach alert (gap=5)")
	}
	if a.Params["gap"].(float64) != 5 {
		t.Errorf("gap = %v, want 5", a.Params["gap"])
	}
}

func TestGenerate_LUSRTierApproach_TooFar(t *testing.T) {
	g := NewGenerator()
	now := fixedDate(2026, 5, 18)
	alerts := g.Generate(context.Background(), GenerateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: now,
		LUSR: LUSRSnapshot{Mu: 1480, NextTierName: "diamond_iv", NextTierMu: 1500}, // gap=20 > 10
	})
	if _, ok := findAlertByType(alerts, AlertTypeLUSRTierApproach); ok {
		t.Errorf("no alert expected at gap=20")
	}
}

func TestGenerate_LUSRTierApproach_MaxTier(t *testing.T) {
	g := NewGenerator()
	now := fixedDate(2026, 5, 18)
	alerts := g.Generate(context.Background(), GenerateInput{
		LUSR: LUSRSnapshot{Mu: 1900, NextTierName: "", NextTierMu: 0}, // tier max
		Now:  now,
	})
	if _, ok := findAlertByType(alerts, AlertTypeLUSRTierApproach); ok {
		t.Errorf("no alert at max tier")
	}
}

// ─── LOWESS trends ──────────────────────────────────────────────────────────

func TestGenerate_LOWESSPositive(t *testing.T) {
	g := NewGenerator()
	now := fixedDate(2026, 5, 18)
	alerts := g.Generate(context.Background(), GenerateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: now,
		LOWESSTrends: []LOWESSTrend{
			{Component: "accuracy", Slope: 0.003, Window: 14},
			{Component: "kills_vs_expected", Slope: -0.001, Window: 14}, // négatif → ignoré
			{Component: "kda", Slope: 0.01, Window: 7},                  // fenêtre trop courte → ignoré
		},
	})
	if _, ok := findAlertByDedup(alerts, AlertTypeLOWESSPositive, "accuracy"); !ok {
		t.Error("expected LOWESSPositive for accuracy")
	}
	if _, ok := findAlertByDedup(alerts, AlertTypeLOWESSPositive, "kills_vs_expected"); ok {
		t.Error("kills_vs_expected has negative slope, no alert")
	}
	if _, ok := findAlertByDedup(alerts, AlertTypeLOWESSPositive, "kda"); ok {
		t.Error("kda window < 14, no alert")
	}
}

func TestGenerate_LOWESSSoftNegative(t *testing.T) {
	g := NewGenerator()
	now := fixedDate(2026, 5, 18)
	alerts := g.Generate(context.Background(), GenerateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: now,
		LOWESSTrends: []LOWESSTrend{
			{Component: "accuracy", Slope: -0.15, Window: 14},          // baisse soutenue → alerte
			{Component: "kills_vs_expected", Slope: -0.05, Window: 14}, // au-dessus du seuil (-0.10) → ignoré
			{Component: "kda", Slope: -0.20, Window: 7},                // fenêtre trop courte → ignoré
			{Component: "win_factor", Slope: 0.10, Window: 14},         // positif → pas soft-neg
		},
	})
	if _, ok := findAlertByDedup(alerts, AlertTypeLOWESSSoftNegative, "accuracy|soft_neg"); !ok {
		t.Error("expected LOWESSSoftNegative for accuracy (slope -0.15 < -0.10)")
	}
	if _, ok := findAlertByDedup(alerts, AlertTypeLOWESSSoftNegative, "kills_vs_expected|soft_neg"); ok {
		t.Error("kills_vs_expected slope -0.05 above threshold, no soft-neg alert")
	}
	if _, ok := findAlertByDedup(alerts, AlertTypeLOWESSSoftNegative, "kda|soft_neg"); ok {
		t.Error("kda window < 14, no alert")
	}
	if _, ok := findAlertByDedup(alerts, AlertTypeLOWESSSoftNegative, "win_factor|soft_neg"); ok {
		t.Error("win_factor positive, no soft-neg alert")
	}
}

// ─── Comeback welcome ───────────────────────────────────────────────────────

func TestGenerate_ComebackWelcome_AfterLongPause(t *testing.T) {
	g := NewGenerator()
	now := fixedDate(2026, 5, 18)
	lastMatch := now.AddDate(0, 0, -7)
	alerts := g.Generate(context.Background(), GenerateInput{
		UserID: "u1", TitleSlug: "halo_infinite", Now: now,
		LastMatchAt: &lastMatch, HasNewActivity: true,
	})
	a, ok := findAlertByType(alerts, AlertTypeComebackWelcome)
	if !ok {
		t.Fatal("expected ComebackWelcome alert")
	}
	if a.Params["days_away"] != 7 {
		t.Errorf("days_away = %v, want 7", a.Params["days_away"])
	}
}

func TestGenerate_ComebackWelcome_NoNewActivity(t *testing.T) {
	g := NewGenerator()
	now := fixedDate(2026, 5, 18)
	lastMatch := now.AddDate(0, 0, -10)
	alerts := g.Generate(context.Background(), GenerateInput{
		LastMatchAt: &lastMatch, HasNewActivity: false, // pas de retour
		Now: now,
	})
	if _, ok := findAlertByType(alerts, AlertTypeComebackWelcome); ok {
		t.Errorf("no alert without new activity")
	}
}

func TestGenerate_ComebackWelcome_PauseTooShort(t *testing.T) {
	g := NewGenerator()
	now := fixedDate(2026, 5, 18)
	lastMatch := now.AddDate(0, 0, -3) // < 5j seuil
	alerts := g.Generate(context.Background(), GenerateInput{
		LastMatchAt: &lastMatch, HasNewActivity: true, Now: now,
	})
	if _, ok := findAlertByType(alerts, AlertTypeComebackWelcome); ok {
		t.Errorf("no alert when pause < ComebackPauseThreshold")
	}
}

// ─── Dedup filter ───────────────────────────────────────────────────────────

// constWindow retourne un résolveur de fenêtre constant (tests legacy à fenêtre
// unique, avant la résolution par catégorie DP13).
func constWindow(d time.Duration) func(notifications.Category) time.Duration {
	return func(notifications.Category) time.Duration { return d }
}

func TestFilterRecent_SkipsAlreadyEmitted(t *testing.T) {
	now := fixedDate(2026, 5, 18)
	alerts := []Alert{
		{Type: AlertTypeRecordNearMiss, DedupKey: "kda|30d"},
		{Type: AlertTypeStreakMilestone, DedupKey: "daily_play|8"},
	}
	// Notif récente déjà émise pour kda|30d.
	paramsKDA, _ := json.Marshal(map[string]any{"dedup_key": "kda|30d"})
	recent := []notifications.Notification{
		{
			Category:  notifications.CategoryRecordNearMiss,
			Params:    paramsKDA,
			CreatedAt: now.Add(-2 * time.Hour),
		},
	}
	out := FilterRecent(alerts, recent, now, constWindow(24*time.Hour))
	if len(out) != 1 {
		t.Fatalf("expected 1 alert remaining, got %d", len(out))
	}
	if out[0].Type != AlertTypeStreakMilestone {
		t.Errorf("remaining alert should be StreakMilestone, got %s", out[0].Type)
	}
}

func TestFilterRecent_OutsideWindow_DoesNotFilter(t *testing.T) {
	now := fixedDate(2026, 5, 18)
	alerts := []Alert{{Type: AlertTypeRecordNearMiss, DedupKey: "kda|30d"}}
	paramsKDA, _ := json.Marshal(map[string]any{"dedup_key": "kda|30d"})
	recent := []notifications.Notification{
		{
			Category:  notifications.CategoryRecordNearMiss,
			Params:    paramsKDA,
			CreatedAt: now.Add(-48 * time.Hour), // > 24h fenêtre
		},
	}
	out := FilterRecent(alerts, recent, now, constWindow(24*time.Hour))
	if len(out) != 1 {
		t.Errorf("expected alert to pass through (outside dedup window)")
	}
}

func TestFilterRecent_DifferentDedupKey_DoesNotFilter(t *testing.T) {
	now := fixedDate(2026, 5, 18)
	alerts := []Alert{{Type: AlertTypeRecordNearMiss, DedupKey: "kda|all_time"}}
	paramsKDA30, _ := json.Marshal(map[string]any{"dedup_key": "kda|30d"})
	recent := []notifications.Notification{
		{Category: notifications.CategoryRecordNearMiss, Params: paramsKDA30, CreatedAt: now},
	}
	out := FilterRecent(alerts, recent, now, constWindow(24*time.Hour))
	if len(out) != 1 {
		t.Errorf("different dedup_key should not filter")
	}
}

// ─── B15/DP13 : dédup par catégorie (nudges d'état = 30 jours) ────────────────

func recentNotif(cat notifications.Category, dedupKey string, at time.Time) notifications.Notification {
	p, _ := json.Marshal(map[string]any{"dedup_key": dedupKey})
	return notifications.Notification{Category: cat, Params: p, CreatedAt: at}
}

// Un nudge d'état (lusr_tier_approach) émis il y a 5 jours reste filtré : la
// fenêtre est 30 jours (DedupWindowFor), pas 24 h.
func TestFilterRecent_StateNudge_30DayWindow(t *testing.T) {
	now := fixedDate(2026, 5, 18)
	alerts := []Alert{{Type: AlertTypeLUSRTierApproach, DedupKey: "Diamond III"}}

	// Il y a 5 jours → dans la fenêtre 30 j → filtré.
	recent5d := []notifications.Notification{
		recentNotif(notifications.CategoryLUSRTierApproach, "Diamond III", now.AddDate(0, 0, -5)),
	}
	if out := FilterRecent(alerts, recent5d, now, DedupWindowFor); len(out) != 0 {
		t.Errorf("nudge d'état émis il y a 5 j → doit être filtré (fenêtre 30 j), got %d", len(out))
	}

	// Il y a 35 jours → hors fenêtre 30 j → passe.
	recent35d := []notifications.Notification{
		recentNotif(notifications.CategoryLUSRTierApproach, "Diamond III", now.AddDate(0, 0, -35)),
	}
	if out := FilterRecent(alerts, recent35d, now, DedupWindowFor); len(out) != 1 {
		t.Errorf("nudge d'état émis il y a 35 j → doit passer, got %d", len(out))
	}
}

// Une catégorie ÉVÉNEMENT (personal_record) reste à 24 h : émise il y a 5 jours,
// elle ne filtre plus une nouvelle occurrence.
func TestFilterRecent_EventCategory_24hWindow(t *testing.T) {
	now := fixedDate(2026, 5, 18)
	alerts := []Alert{{Type: AlertTypeRecordBroken, DedupKey: "kda|all_time"}}
	recent := []notifications.Notification{
		recentNotif(notifications.CategoryPersonalRecord, "kda|all_time", now.AddDate(0, 0, -5)),
	}
	if out := FilterRecent(alerts, recent, now, DedupWindowFor); len(out) != 1 {
		t.Errorf("événement émis il y a 5 j (fenêtre 24 h) → doit passer, got %d", len(out))
	}
}

func TestDedupWindowFor_Resolution(t *testing.T) {
	if DedupWindowFor(notifications.CategoryLUSRTierApproach) != StateNudgeDedupWindow {
		t.Error("lusr_tier_approach doit être un nudge d'état (30 j)")
	}
	if DedupWindowFor(notifications.CategoryPersonalRecord) != DedupWindow {
		t.Error("personal_record doit rester à 24 h")
	}
	if DedupWindowFor(notifications.CategoryStreakMilestone) != DedupWindow {
		t.Error("streak_milestone doit rester à 24 h")
	}
}

func TestAnnotateDedupKey_AddsToParams(t *testing.T) {
	a := Alert{
		Type:     AlertTypeMilestoneUnlocked,
		DedupKey: "halo.matches.100",
		Params:   map[string]any{"title_fr": "Centurion"},
	}
	AnnotateDedupKey(&a)
	if a.Params["dedup_key"] != "halo.matches.100" {
		t.Errorf("dedup_key not annotated: %v", a.Params)
	}
}

// ─── Mapping AlertType → Category ──────────────────────────────────────────

func TestAlertType_NotificationCategory_AllMapped(t *testing.T) {
	for _, at := range AllAlertTypes() {
		cat := at.NotificationCategory()
		if cat == "" {
			t.Errorf("AlertType %s has no category mapping", at)
		}
	}
}

// TestAlertType_LOWESSSoftNegative_NeutralCategory verrouille le cadrage : le
// soft-négatif NE DOIT PAS réutiliser threshold_crossed (notif positive « Palier
// franchi ») mais sa catégorie neutre dédiée. Cf. Coach V3 Phase A.
func TestAlertType_LOWESSSoftNegative_NeutralCategory(t *testing.T) {
	got := AlertTypeLOWESSSoftNegative.NotificationCategory()
	if got != notifications.CategoryTrendConsolidate {
		t.Errorf("LOWESSSoftNegative category = %s, want trend_consolidate", got)
	}
	if got == notifications.CategoryThresholdCrossed {
		t.Error("soft-négatif ne doit pas réutiliser threshold_crossed (cadrage positif)")
	}
}

// ─── buildCombatPatternAlerts ───────────────────────────────────────────────

func TestBuildCombatPatternAlerts_NilMedians_NoAlerts(t *testing.T) {
	alerts := buildCombatPatternAlerts(GenerateInput{CombatMedians: nil})
	if len(alerts) != 0 {
		t.Errorf("nil CombatMedians should produce 0 alerts, got %d", len(alerts))
	}
}

func TestBuildCombatPatternAlerts_Actif_LowOC_HighResidual(t *testing.T) {
	cm := &CombatMedians{
		MedianOC:    combatOCP80Threshold * 0.50, // < 70% P80 → fragile OC
		MedianDR:    combatDRP80Threshold,        // bon DR → pas de fragile
		AvgResidual: 10.0,                        // > +5 → actif
		HasResidual: true,
	}
	alerts := buildCombatPatternAlerts(GenerateInput{CombatMedians: cm})
	if _, ok := findAlertByType(alerts, AlertTypeCombatPatternActif); !ok {
		t.Error("expected AlertTypeCombatPatternActif for low OC + high residual")
	}
	if _, ok := findAlertByType(alerts, AlertTypeCombatPatternFragile); ok {
		t.Error("unexpected AlertTypeCombatPatternFragile with good DR")
	}
}

func TestBuildCombatPatternAlerts_Discret_LowResidual(t *testing.T) {
	cm := &CombatMedians{
		MedianOC:    combatOCP80Threshold,
		MedianDR:    combatDRP80Threshold,
		AvgResidual: -10.0, // < -5 → discret
		HasResidual: true,
	}
	alerts := buildCombatPatternAlerts(GenerateInput{CombatMedians: cm})
	if _, ok := findAlertByType(alerts, AlertTypeCombatPatternDiscret); !ok {
		t.Error("expected AlertTypeCombatPatternDiscret for avg_residual < -5")
	}
}

func TestBuildCombatPatternAlerts_Fragile_LowDR(t *testing.T) {
	cm := &CombatMedians{
		MedianOC:    combatOCP80Threshold,
		MedianDR:    combatDRP80Threshold * 0.50, // < 70% P80 → fragile DR
		AvgResidual: 0,
		HasResidual: false,
	}
	alerts := buildCombatPatternAlerts(GenerateInput{CombatMedians: cm})
	if _, ok := findAlertByType(alerts, AlertTypeCombatPatternFragile); !ok {
		t.Error("expected AlertTypeCombatPatternFragile for low DR")
	}
}

func TestBuildCombatPatternAlerts_NoAlert_AllGood(t *testing.T) {
	cm := &CombatMedians{
		MedianOC:    combatOCP80Threshold, // bon OC
		MedianDR:    combatDRP80Threshold, // bon DR
		AvgResidual: 0.0,                  // modéré (>= -5, pas > +5)
		HasResidual: true,
	}
	alerts := buildCombatPatternAlerts(GenerateInput{CombatMedians: cm})
	if len(alerts) != 0 {
		t.Errorf("no alerts expected when OC/DR/residual are all good, got %d", len(alerts))
	}
}

func TestBuildCombatPatternAlerts_ActifRequiresHasResidual(t *testing.T) {
	cm := &CombatMedians{
		MedianOC:    combatOCP80Threshold * 0.50,
		MedianDR:    combatDRP80Threshold,
		AvgResidual: 10.0,
		HasResidual: false, // pas assez de données résidu → pas d'alerte actif
	}
	alerts := buildCombatPatternAlerts(GenerateInput{CombatMedians: cm})
	if _, ok := findAlertByType(alerts, AlertTypeCombatPatternActif); ok {
		t.Error("AlertTypeCombatPatternActif should not fire without HasResidual")
	}
}
