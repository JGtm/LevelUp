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
	out := FilterRecent(alerts, recent, now, 24*time.Hour)
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
	out := FilterRecent(alerts, recent, now, 24*time.Hour)
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
	out := FilterRecent(alerts, recent, now, 24*time.Hour)
	if len(out) != 1 {
		t.Errorf("different dedup_key should not filter")
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
