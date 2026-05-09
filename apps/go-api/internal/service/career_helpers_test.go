package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

const heroPlaylist = "Hero"

// ---------------------------------------------------------------------------
// formatRankLabel
// ---------------------------------------------------------------------------

func TestFormatRankLabel_WithLabel(t *testing.T) {
	lbl := heroPlaylist
	rank := &domain.CareerRankData{RankLabel: &lbl}
	got := formatRankLabel(rank)
	if got != heroPlaylist {
		t.Errorf("expected Hero, got %s", got)
	}
}

func TestFormatRankLabel_NameAndTier(t *testing.T) {
	name := "Diamond"
	tier := "III"
	rank := &domain.CareerRankData{RankName: &name, RankTier: &tier}
	got := formatRankLabel(rank)
	if got != "Diamond - III" {
		t.Errorf("expected Diamond - III, got %s", got)
	}
}

func TestFormatRankLabel_NameOnly(t *testing.T) {
	name := "Onyx"
	rank := &domain.CareerRankData{RankName: &name}
	got := formatRankLabel(rank)
	if got != "Onyx" {
		t.Errorf("expected Onyx, got %s", got)
	}
}

func TestFormatRankLabel_Fallback(t *testing.T) {
	rank := &domain.CareerRankData{RankNumber: 42}
	got := formatRankLabel(rank)
	if got != "Rang 42" {
		t.Errorf("expected 'Rang 42', got %s", got)
	}
}

// ---------------------------------------------------------------------------
// computeProgressPct
// ---------------------------------------------------------------------------

func TestComputeProgressPct_MaxRank_Returns100(t *testing.T) {
	got := computeProgressPct(500, 1000, true)
	if got != 100.0 {
		t.Errorf("expected 100, got %f", got)
	}
}

func TestComputeProgressPct_ZeroXPForNext(t *testing.T) {
	got := computeProgressPct(500, 0, false)
	if got != 0.0 {
		t.Errorf("expected 0, got %f", got)
	}
}

func TestComputeProgressPct_Normal_75pct(t *testing.T) {
	got := computeProgressPct(750, 1000, false)
	if got != 75.0 {
		t.Errorf("expected 75, got %f", got)
	}
}

func TestComputeProgressPct_Over100(t *testing.T) {
	got := computeProgressPct(1500, 1000, false)
	if got != 100.0 {
		t.Errorf("expected capped at 100, got %f", got)
	}
}

// ---------------------------------------------------------------------------
// summaryXPTotal
// ---------------------------------------------------------------------------

func TestSummaryXPTotal_Nil(t *testing.T) {
	if summaryXPTotal(nil) != 0 {
		t.Error("expected 0 for nil rank")
	}
}

func TestSummaryXPTotal_NilXP(t *testing.T) {
	rank := &domain.CareerRankData{}
	if summaryXPTotal(rank) != 0 {
		t.Error("expected 0 for nil XPTotal")
	}
}

func TestSummaryXPTotal_WithValue(t *testing.T) {
	xp := 50000
	rank := &domain.CareerRankData{XPTotal: &xp}
	if summaryXPTotal(rank) != 50000 {
		t.Errorf("expected 50000, got %d", summaryXPTotal(rank))
	}
}

// ---------------------------------------------------------------------------
// buildHeroProgress
// ---------------------------------------------------------------------------

func TestBuildHeroProgress_Zero(t *testing.T) {
	hp := buildHeroProgress(0)
	if hp.Percentage != 0 {
		t.Errorf("expected 0%%, got %f", hp.Percentage)
	}
	if hp.XPRemaining != xpHeroTotal {
		t.Errorf("expected XPRemaining=%d", xpHeroTotal)
	}
}

func TestBuildHeroProgress_Complete(t *testing.T) {
	hp := buildHeroProgress(xpHeroTotal + 1000)
	if hp.Percentage != 100.0 {
		t.Errorf("expected 100%%, got %f", hp.Percentage)
	}
	if hp.XPRemaining != 0 {
		t.Errorf("expected 0 remaining, got %d", hp.XPRemaining)
	}
}

// ---------------------------------------------------------------------------
// computeFallbackXPPerDay
// ---------------------------------------------------------------------------

func TestComputeFallbackXPPerDay_ZeroDays(t *testing.T) {
	got := computeFallbackXPPerDay(1000, time.Now())
	if got != 0.0 {
		t.Errorf("expected 0 for same-day, got %f", got)
	}
}

func TestComputeFallbackXPPerDay_ZeroXP(t *testing.T) {
	got := computeFallbackXPPerDay(0, time.Now().Add(-30*24*time.Hour))
	if got != 0.0 {
		t.Errorf("expected 0 for zero XP, got %f", got)
	}
}

func TestComputeFallbackXPPerDay_Normal(t *testing.T) {
	got := computeFallbackXPPerDay(3000, time.Now().Add(-30*24*time.Hour))
	if got <= 0 {
		t.Errorf("expected positive rate, got %f", got)
	}
	if got > 200 {
		t.Errorf("expected reasonable rate, got %f", got)
	}
}

// ---------------------------------------------------------------------------
// buildLUSRSummary
// ---------------------------------------------------------------------------

func TestBuildLUSRSummary_Empty(t *testing.T) {
	s := buildLUSRSummary(nil)
	if s.CurrentRating != nil {
		t.Error("expected nil rating for empty history")
	}
}

func TestBuildLUSRSummary_SinglePoint(t *testing.T) {
	tier := "Gold"
	pg := sessionCategoryRanked
	now := time.Now()
	history := []domain.LUSRCheckpointDTO{
		{RatingValue: 1500.0, TierLabel: &tier, PlaylistGroup: &pg, RecordedAt: &now},
	}
	s := buildLUSRSummary(history)
	if s.CurrentRating == nil || *s.CurrentRating != 1500.0 {
		t.Error("expected 1500 rating")
	}
	if s.TrendLabel != nil {
		t.Error("expected nil trend for single point")
	}
}

func TestBuildLUSRSummary_TwoPoints_Up(t *testing.T) {
	now := time.Now()
	history := []domain.LUSRCheckpointDTO{
		{RatingValue: 1400.0, RecordedAt: &now},
		{RatingValue: 1500.0, RecordedAt: &now},
	}
	s := buildLUSRSummary(history)
	if s.TrendLabel == nil || *s.TrendLabel != "+100" {
		t.Errorf("expected +100 trend, got %v", s.TrendLabel)
	}
}

func TestBuildLUSRSummary_TwoPoints_Down(t *testing.T) {
	now := time.Now()
	history := []domain.LUSRCheckpointDTO{
		{RatingValue: 1500.0, RecordedAt: &now},
		{RatingValue: 1400.0, RecordedAt: &now},
	}
	s := buildLUSRSummary(history)
	if s.TrendLabel == nil || *s.TrendLabel != "-100" {
		t.Errorf("expected -100 trend, got %v", s.TrendLabel)
	}
}

func TestBuildLUSRSummary_BestRating(t *testing.T) {
	now := time.Now()
	history := []domain.LUSRCheckpointDTO{
		{RatingValue: 1200.0, RecordedAt: &now},
		{RatingValue: 1600.0, RecordedAt: &now},
		{RatingValue: 1400.0, RecordedAt: &now},
	}
	s := buildLUSRSummary(history)
	if s.CurrentRating == nil || *s.CurrentRating != 1600.0 {
		t.Errorf("expected best=1600, got %v", s.CurrentRating)
	}
}

// ---------------------------------------------------------------------------
// splitTopRows / reverseTopMatches
// ---------------------------------------------------------------------------

func TestSplitTopRows_Even(t *testing.T) {
	rows := make([]domain.TopMatchRawRow, 6)
	for i := range rows {
		if i < 3 {
			rows[i].Outcome = 2 // WIN
		} else {
			rows[i].Outcome = 3 // LOSS
		}
	}
	best, worst := splitTopRows(rows)
	if len(best) != 3 || len(worst) != 3 {
		t.Errorf("expected 3/3, got %d/%d", len(best), len(worst))
	}
}

func TestSplitTopRows_ByOutcome(t *testing.T) {
	// 10 WIN + 20 LOSS → split par outcome
	rows := make([]domain.TopMatchRawRow, 30)
	for i := range rows {
		if i < 10 {
			rows[i].Outcome = 2 // WIN
		} else {
			rows[i].Outcome = 3 // LOSS
		}
	}
	best, worst := splitTopRows(rows)
	if len(best) != 10 {
		t.Errorf("expected best=10, got %d", len(best))
	}
	if len(worst) != 20 {
		t.Errorf("expected worst=20, got %d", len(worst))
	}
}

func TestReverseTopMatches(t *testing.T) {
	rows := []domain.TopMatchRawRow{
		{PerformanceScore: 1.0},
		{PerformanceScore: 2.0},
		{PerformanceScore: 3.0},
	}
	reverseTopMatches(rows)
	if rows[0].PerformanceScore != 3.0 || rows[2].PerformanceScore != 1.0 {
		t.Error("reverse failed")
	}
}

// ---------------------------------------------------------------------------
// buildCareerSummary
// ---------------------------------------------------------------------------

func TestBuildCareerSummary_Nil(t *testing.T) {
	s := buildCareerSummary(nil)
	if s.RankLabel != "" {
		t.Error("expected empty label for nil rank")
	}
}

func TestBuildCareerSummary_WithData(t *testing.T) {
	lbl := heroPlaylist
	xp := 100000
	nextXP := 5000
	name := heroPlaylist
	tier := "I"
	rank := &domain.CareerRankData{
		RankNumber:    272,
		RankLabel:     &lbl,
		RankName:      &name,
		RankTier:      &tier,
		CurrentXP:     2500,
		XPForNextRank: &nextXP,
		XPTotal:       &xp,
		RecordedAt:    time.Now(),
	}
	s := buildCareerSummary(rank)
	if s.RankLabel != heroPlaylist {
		t.Errorf("expected Hero, got %s", s.RankLabel)
	}
	if s.XPTotal != 100000 {
		t.Errorf("expected 100000, got %d", s.XPTotal)
	}
}

// ---------------------------------------------------------------------------
// buildProjections
// ---------------------------------------------------------------------------

func TestBuildProjections_TooFewPoints(t *testing.T) {
	p := buildProjections(nil, 5000)
	if p.XPPerDayActive != 0 {
		t.Error("expected 0 for empty history")
	}
}

func TestBuildProjections_Normal(t *testing.T) {
	now := time.Now()
	history := []domain.XPHistoryPoint{
		{RecordedAt: now.Add(-10 * 24 * time.Hour), XPTotal: 1000},
		{RecordedAt: now, XPTotal: 2000},
	}
	p := buildProjections(history, 2000)
	if p.XPPerDayActive <= 0 {
		t.Errorf("expected positive XPPerDayActive, got %f", p.XPPerDayActive)
	}
	if p.XPPerDayFallback <= 0 {
		t.Errorf("expected positive XPPerDayFallback, got %f", p.XPPerDayFallback)
	}
}
