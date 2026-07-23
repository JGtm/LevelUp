// Package service — tests des helpers de binning Mock 11.
package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// helper : construit un summary minimaliste.
func mkSummary(id, label string, started time.Time, scores ...float64) domain.EngagementMatchSummary {
	s := domain.EngagementMatchSummary{
		MatchID:     id,
		Label:       label,
		StartedAt:   started,
		PaceJoueur:  10,
		PaceTeam:    8,
		PaceAttendu: 9,
		PaceLobby:   7,
		MatchCount:  1,
	}
	if len(scores) > 0 {
		v := scores[0]
		s.EngagementScore = &v
	}
	return s
}

func TestAggregateEngagementBySession_GroupePuisMoyenne(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	label := "Session du 01/05"
	rows := []legacymatch.StatsMatchRow{
		{MatchID: "m1", StartTime: t0, SessionLabel: &label},
		{MatchID: "m2", StartTime: t0.Add(30 * time.Minute), SessionLabel: &label},
		{MatchID: "m3", StartTime: t0.Add(2 * time.Hour)}, // pas de session_label
	}
	// summaries : m1 score=50, m2 score=70, m3 sans score
	s1 := mkSummary("m1", "M1", t0, 50)
	s1.PaceJoueur = 10
	s2 := mkSummary("m2", "M2", t0.Add(30*time.Minute), 70)
	s2.PaceJoueur = 20
	s3 := mkSummary("m3", "M3", t0.Add(2*time.Hour))
	s3.PaceJoueur = 30

	got := aggregateEngagementBySession(
		[]domain.EngagementMatchSummary{s1, s2, s3}, rows,
	)
	if len(got) != 2 {
		t.Fatalf("expected 2 buckets (1 session + 1 singleton), got %d", len(got))
	}
	// Premier bucket = session avec m1+m2 (le plus ancien start)
	if got[0].Label != label {
		t.Errorf("expected first bucket label %q, got %q", label, got[0].Label)
	}
	if got[0].MatchCount != 2 {
		t.Errorf("expected match_count=2 for session bucket, got %d", got[0].MatchCount)
	}
	// Moyenne de 10 et 20 = 15
	if got[0].PaceJoueur != 15 {
		t.Errorf("expected pace_joueur=15, got %f", got[0].PaceJoueur)
	}
	// Score moyen 50 et 70 = 60
	if got[0].EngagementScore == nil || *got[0].EngagementScore != 60 {
		t.Errorf("expected engagement_score=60, got %v", got[0].EngagementScore)
	}
	// Second bucket = m3 singleton, sans score
	if got[1].MatchCount != 1 {
		t.Errorf("expected match_count=1 for singleton bucket, got %d", got[1].MatchCount)
	}
	if got[1].EngagementScore != nil {
		t.Errorf("expected nil engagement_score for singleton without score, got %v", got[1].EngagementScore)
	}
}

func TestRollupEngagementByPeriod_GroupementHebdoEtMensuel(t *testing.T) {
	t.Parallel()
	// Trois matchs sur deux semaines distinctes en mai 2026.
	// W18 : 30/04 (jeudi, ISO week 18)
	// W19 : 04/05 et 06/05
	m1 := mkSummary("m1", "M1", time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC), 40)
	m1.PaceJoueur = 10
	m2 := mkSummary("m2", "M2", time.Date(2026, 5, 4, 14, 0, 0, 0, time.UTC), 60)
	m2.PaceJoueur = 20
	m3 := mkSummary("m3", "M3", time.Date(2026, 5, 6, 18, 0, 0, 0, time.UTC), 80)
	m3.PaceJoueur = 30
	summaries := []domain.EngagementMatchSummary{m1, m2, m3}

	weeks := rollupEngagementByPeriod(summaries, "week")
	if len(weeks) != 2 {
		t.Fatalf("expected 2 weekly buckets, got %d", len(weeks))
	}
	if weeks[0].MatchCount != 1 || weeks[1].MatchCount != 2 {
		t.Errorf("expected counts [1, 2], got [%d, %d]", weeks[0].MatchCount, weeks[1].MatchCount)
	}
	// Moyenne semaine 2 : 20 et 30 = 25
	if weeks[1].PaceJoueur != 25 {
		t.Errorf("expected week2 pace_joueur=25, got %f", weeks[1].PaceJoueur)
	}
	// Score moyen semaine 2 : 60 et 80 = 70
	if weeks[1].EngagementScore == nil || *weeks[1].EngagementScore != 70 {
		t.Errorf("expected week2 engagement_score=70, got %v", weeks[1].EngagementScore)
	}

	// Tout au même mois (mai 2026, m1 est sur avril donc 2 buckets)
	months := rollupEngagementByPeriod(summaries, "month")
	if len(months) != 2 {
		t.Fatalf("expected 2 monthly buckets, got %d", len(months))
	}
	if months[1].Label != "2026-05" || months[1].MatchCount != 2 {
		t.Errorf("expected mai bucket=2026-05 with 2 matchs, got %q with %d", months[1].Label, months[1].MatchCount)
	}
}

func TestAggregateEngagementBySession_DurationSummedPerBin(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	label := "Session du 01/05"
	rows := []legacymatch.StatsMatchRow{
		{MatchID: "m1", StartTime: t0, SessionLabel: &label},
		{MatchID: "m2", StartTime: t0.Add(30 * time.Minute), SessionLabel: &label},
		{MatchID: "m3", StartTime: t0.Add(2 * time.Hour)}, // singleton
	}
	s1 := mkSummary("m1", "M1", t0, 50)
	s1.DurationSeconds = 600
	s2 := mkSummary("m2", "M2", t0.Add(30*time.Minute), 70)
	s2.DurationSeconds = 720
	s3 := mkSummary("m3", "M3", t0.Add(2*time.Hour))
	s3.DurationSeconds = 480

	got := aggregateEngagementBySession([]domain.EngagementMatchSummary{s1, s2, s3}, rows)
	if len(got) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(got))
	}
	// Bucket session = somme des durées de m1+m2.
	if got[0].DurationSeconds != 1320 {
		t.Errorf("expected session bucket duration 1320, got %d", got[0].DurationSeconds)
	}
	// Bucket singleton = sa propre durée.
	if got[1].DurationSeconds != 480 {
		t.Errorf("expected singleton bucket duration 480, got %d", got[1].DurationSeconds)
	}
}

func TestRollupEngagementByPeriod_DurationSummedPerBin(t *testing.T) {
	t.Parallel()
	// W19 : deux matchs la même semaine ISO → durées sommées.
	m1 := mkSummary("m1", "M1", time.Date(2026, 5, 4, 14, 0, 0, 0, time.UTC), 60)
	m1.DurationSeconds = 600
	m2 := mkSummary("m2", "M2", time.Date(2026, 5, 6, 18, 0, 0, 0, time.UTC), 80)
	m2.DurationSeconds = 900
	weeks := rollupEngagementByPeriod([]domain.EngagementMatchSummary{m1, m2}, "week")
	if len(weeks) != 1 {
		t.Fatalf("expected 1 weekly bucket, got %d", len(weeks))
	}
	if weeks[0].DurationSeconds != 1500 {
		t.Errorf("expected summed duration 1500, got %d", weeks[0].DurationSeconds)
	}
}

func TestPeriodKey_WeekISO(t *testing.T) {
	t.Parallel()
	// Lundi 4 mai 2026 = ISO 2026-W19 ; le bucket start doit être ce lundi.
	monday := time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)
	wed := time.Date(2026, 5, 6, 18, 0, 0, 0, time.UTC) // même semaine
	k1, _, s1 := periodKey(monday, "week")
	k2, _, s2 := periodKey(wed, "week")
	if k1 != k2 {
		t.Errorf("expected same week key for monday and wednesday of same ISO week, got %q vs %q", k1, k2)
	}
	if !s1.Equal(s2) {
		t.Errorf("expected same week start, got %v vs %v", s1, s2)
	}
	if s1.Weekday() != time.Monday {
		t.Errorf("expected week start = monday, got %s", s1.Weekday())
	}
}
