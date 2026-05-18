package service

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

func TestSessionPageService_GetPage_DefaultLatestSession(t *testing.T) {
	now := time.Date(2026, 4, 21, 20, 0, 0, 0, time.UTC)
	repo := &mockSessionPageStatsRepo{
		matches: []legacymatch.StatsMatchRow{
			makeSessionPageMatch("m1", now.Add(-6*time.Hour), "2026-04-21 14h", false, "Quick Play", "Slayer", 10, 8, 2, 74.2, 54.1),
			makeSessionPageMatch("m2", now.Add(-5*time.Hour), "2026-04-21 14h", false, "Quick Play", "Slayer", 12, 7, 3, 76.2, 58.1),
			makeSessionPageMatch("m3", now.Add(-2*time.Hour), "2026-04-21 18h", false, "BTB Social", "CTF", 14, 9, 4, 68.5, 61.0),
			makeSessionPageMatch("m4", now.Add(-90*time.Minute), "2026-04-21 18h", false, "BTB Social", "CTF", 16, 10, 5, 71.5, 64.0),
			makeSessionPageMatch("m5", now.Add(-30*time.Minute), "2026-04-21 19h30", true, "Ranked Arena", "Oddball", 11, 6, 4, 62.1, 67.0),
			makeSessionPageMatch("m6", now.Add(-10*time.Minute), "2026-04-21 19h30", true, "Ranked Arena", "Oddball", 13, 5, 6, 64.8, 70.0),
		},
	}
	svc := NewSessionPageService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(repo.matches, repo.err), "halo_infinite", "Test")

	resp, err := svc.GetPage(context.Background(), domain.SessionPageRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CurrentSession == nil || resp.CurrentSession.SessionLabel != "2026-04-21 19h30" {
		t.Fatalf("unexpected current session: %#v", resp.CurrentSession)
	}
	if len(resp.AvailableSessions) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(resp.AvailableSessions))
	}
	if resp.SuggestedCompare == nil || resp.SuggestedCompare.SessionLabel != "2026-04-21 18h" {
		t.Fatalf("unexpected suggestion: %#v", resp.SuggestedCompare)
	}
	if resp.CurrentSession.DominantCategory == nil || *resp.CurrentSession.DominantCategory != "Ranked" {
		t.Fatalf("unexpected category: %#v", resp.CurrentSession.DominantCategory)
	}
	if len(resp.Matches) != 2 {
		t.Fatalf("expected 2 detailed matches, got %d", len(resp.Matches))
	}
	if resp.CompareEnabled {
		t.Fatal("compare should be disabled by default")
	}
}

func TestSessionPageService_GetPage_EnableCompareUsesSuggestion(t *testing.T) {
	repo := &mockSessionPageStatsRepo{matches: makeSessionPageDataset()}
	svc := NewSessionPageService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(repo.matches, repo.err), "halo_infinite", "Test")

	resp, err := svc.GetPage(context.Background(), domain.SessionPageRequest{EnableCompare: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.CompareEnabled {
		t.Fatal("expected compare to be enabled")
	}
	if resp.CompareSession == nil || resp.CompareSession.SessionLabel != "2026-04-21 18h" {
		t.Fatalf("unexpected compare session: %#v", resp.CompareSession)
	}
	if len(resp.CompareMetrics) == 0 {
		t.Fatal("expected compare metrics")
	}
	assertSessionMetricPresent(t, resp.CompareMetrics, "score")
	assertSessionMetricPresent(t, resp.CompareMetrics, "kills_per_match")
}

func TestSessionPageService_GetPage_ManualCompareLabelWins(t *testing.T) {
	repo := &mockSessionPageStatsRepo{matches: makeSessionPageDataset()}
	svc := NewSessionPageService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(repo.matches, repo.err), "halo_infinite", "Test")
	manual := "2026-04-21 14h"

	resp, err := svc.GetPage(context.Background(), domain.SessionPageRequest{
		EnableCompare:       true,
		CompareSessionLabel: &manual,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CompareSession == nil || resp.CompareSession.SessionLabel != manual {
		t.Fatalf("manual compare label not applied: %#v", resp.CompareSession)
	}
}

func TestSessionPageService_GetPage_AppliesPeriodFilter(t *testing.T) {
	repo := &mockSessionPageStatsRepo{matches: makeSessionPageDataset()}
	svc := NewSessionPageService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(repo.matches, repo.err), "halo_infinite", "Test")
	start := time.Date(2026, 4, 21, 17, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 21, 21, 0, 0, 0, time.UTC)

	resp, err := svc.GetPage(context.Background(), domain.SessionPageRequest{
		Filters: domain.FilterContextInput{
			FilterMode: "period",
			Period: domain.PeriodInput{
				StartDate: &start,
				EndDate:   &end,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.AvailableSessions) != 2 {
		t.Fatalf("expected 2 filtered sessions, got %d (%v)", len(resp.AvailableSessions), resp.AvailableSessions)
	}
	if resp.CurrentSession == nil || resp.CurrentSession.SessionLabel != "2026-04-21 19h30" {
		t.Fatalf("unexpected filtered current session: %#v", resp.CurrentSession)
	}
}

func TestSessionPageService_GetPage_NoSessionsAfterFiltering(t *testing.T) {
	repo := &mockSessionPageStatsRepo{matches: makeSessionPageDataset()}
	svc := NewSessionPageService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(repo.matches, repo.err), "halo_infinite", "Test")
	start := time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 22, 1, 0, 0, 0, time.UTC)

	resp, err := svc.GetPage(context.Background(), domain.SessionPageRequest{
		Filters: domain.FilterContextInput{
			FilterMode: "period",
			Period: domain.PeriodInput{
				StartDate: &start,
				EndDate:   &end,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.AvailableSessions) != 0 {
		t.Fatalf("expected no sessions, got %v", resp.AvailableSessions)
	}
	if len(resp.Matches) != 0 {
		t.Fatalf("expected no matches, got %d", len(resp.Matches))
	}
	if len(resp.CompareMetrics) != 0 {
		t.Fatalf("expected no compare metrics, got %d", len(resp.CompareMetrics))
	}
}

func TestSessionPageService_GetPage_UnknownCurrentSessionReturnsEmptyState(t *testing.T) {
	repo := &mockSessionPageStatsRepo{matches: makeSessionPageDataset()}
	svc := NewSessionPageService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(repo.matches, repo.err), "halo_infinite", "Test")
	unknown := "2026-04-21 23h"

	resp, err := svc.GetPage(context.Background(), domain.SessionPageRequest{SessionLabel: &unknown})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CurrentSession != nil {
		t.Fatalf("expected no current session, got %#v", resp.CurrentSession)
	}
	if len(resp.AvailableSessions) != 3 {
		t.Fatalf("expected available sessions to stay listed, got %v", resp.AvailableSessions)
	}
	if len(resp.Matches) != 0 {
		t.Fatalf("expected no detailed matches, got %d", len(resp.Matches))
	}
}

func TestSessionPageService_GetPage_MissingManualCompareDisablesComparison(t *testing.T) {
	repo := &mockSessionPageStatsRepo{matches: makeSessionPageDataset()}
	svc := NewSessionPageService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(repo.matches, repo.err), "halo_infinite", "Test")
	missing := "2026-04-20 22h"

	resp, err := svc.GetPage(context.Background(), domain.SessionPageRequest{
		EnableCompare:       true,
		CompareSessionLabel: &missing,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CompareEnabled {
		t.Fatal("expected compare to be disabled when compare session is missing")
	}
	if resp.CompareSession != nil {
		t.Fatalf("expected no compare session, got %#v", resp.CompareSession)
	}
	if len(resp.CompareMetrics) != 0 {
		t.Fatalf("expected no compare metrics, got %d", len(resp.CompareMetrics))
	}
}

func TestSessionPageService_GetPage_SingleSessionHasNoSuggestion(t *testing.T) {
	repo := &mockSessionPageStatsRepo{matches: makeSessionPageDataset()[:2]}
	svc := NewSessionPageService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(repo.matches, repo.err), "halo_infinite", "Test")

	resp, err := svc.GetPage(context.Background(), domain.SessionPageRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SuggestedCompare != nil {
		t.Fatalf("expected no suggestion, got %#v", resp.SuggestedCompare)
	}
	if len(resp.AvailableSessions) != 1 {
		t.Fatalf("expected one session, got %v", resp.AvailableSessions)
	}
}

func TestBuildSessionCompareSuggestion_CategoryRankedReason(t *testing.T) {
	labels := []string{"2026-04-21 18h", "2026-04-21 19h30"}
	rows := []legacymatch.StatsMatchRow{
		makeSessionPageMatch("m1", time.Date(2026, 4, 21, 18, 0, 0, 0, time.UTC), "2026-04-21 18h", true, "Ranked Arena", "Oddball", 10, 8, 3, 60.0, 55.0),
		makeSessionPageMatch("m2", time.Date(2026, 4, 21, 18, 20, 0, 0, time.UTC), "2026-04-21 18h", true, "Ranked Arena", "Slayer", 12, 9, 4, 62.0, 57.0),
		makeSessionPageMatch("m3", time.Date(2026, 4, 21, 19, 30, 0, 0, time.UTC), "2026-04-21 19h30", true, "Ranked Arena", "Oddball", 13, 7, 5, 64.0, 63.0),
		makeSessionPageMatch("m4", time.Date(2026, 4, 21, 19, 45, 0, 0, time.UTC), "2026-04-21 19h30", true, "Ranked Arena", "Slayer", 15, 6, 6, 67.0, 66.0),
		makeSessionPageMatch("m5", time.Date(2026, 4, 21, 19, 55, 0, 0, time.UTC), "2026-04-21 19h30", true, "Ranked Arena", "CTF", 11, 5, 4, 61.0, 62.0),
	}

	suggestion, candidateCount := buildSessionCompareSuggestion(labels, "2026-04-21 19h30", rows)
	if candidateCount != 1 {
		t.Fatalf("expected one candidate, got %d", candidateCount)
	}
	if suggestion == nil {
		t.Fatal("expected a suggestion")
	}
	if suggestion.Strategy != "category-ranked-close-volume" {
		t.Fatalf("unexpected strategy: %s", suggestion.Strategy)
	}
	if suggestion.Reason != "même catégorie ranked · même statut classé · écart de 1 match(s)" {
		t.Fatalf("unexpected reason: %s", suggestion.Reason)
	}
}

type mockSessionPageStatsRepo struct {
	matches []legacymatch.StatsMatchRow
	err     error
}

func (m *mockSessionPageStatsRepo) LoadStatsMatches(_ context.Context) ([]legacymatch.StatsMatchRow, error) {
	return m.matches, m.err
}

func (m *mockSessionPageStatsRepo) LoadLUSRHistory(_ context.Context) ([]domain.LUSRMatchRating, error) {
	return nil, nil
}

func (m *mockSessionPageStatsRepo) LoadMatchParticipants(_ context.Context) ([]domain.ParticipantRow, error) {
	return nil, nil
}

func makeSessionPageDataset() []legacymatch.StatsMatchRow {
	now := time.Date(2026, 4, 21, 20, 0, 0, 0, time.UTC)
	return []legacymatch.StatsMatchRow{
		makeSessionPageMatch("m1", now.Add(-6*time.Hour), "2026-04-21 14h", false, "Quick Play", "Slayer", 10, 8, 2, 74.2, 54.1),
		makeSessionPageMatch("m2", now.Add(-5*time.Hour), "2026-04-21 14h", false, "Quick Play", "Slayer", 12, 7, 3, 76.2, 58.1),
		makeSessionPageMatch("m3", now.Add(-2*time.Hour), "2026-04-21 18h", false, "BTB Social", "CTF", 14, 9, 4, 68.5, 61.0),
		makeSessionPageMatch("m4", now.Add(-90*time.Minute), "2026-04-21 18h", false, "BTB Social", "CTF", 16, 10, 5, 71.5, 64.0),
		makeSessionPageMatch("m5", now.Add(-30*time.Minute), "2026-04-21 19h30", true, "Ranked Arena", "Oddball", 11, 6, 4, 62.1, 67.0),
		makeSessionPageMatch("m6", now.Add(-10*time.Minute), "2026-04-21 19h30", true, "Ranked Arena", "Oddball", 13, 5, 6, 64.8, 70.0),
	}
}

func makeSessionPageMatch(
	matchID string,
	start time.Time,
	sessionLabel string,
	isRanked bool,
	playlistName string,
	pairName string,
	kills int,
	deaths int,
	assists int,
	accuracy float64,
	perf float64,
) legacymatch.StatsMatchRow {
	label := sessionLabel
	win := analysis.OutcomeWin
	return legacymatch.StatsMatchRow{
		MatchID:           matchID,
		StartTime:         start,
		Outcome:           &win,
		Kills:             kills,
		Deaths:            deaths,
		Assists:           assists,
		Accuracy:          sessionFloat64Ptr(accuracy),
		PerfScoreComputed: sessionFloat64Ptr(perf),
		IsRanked:          isRanked,
		PlaylistName:      playlistName,
		PairName:          pairName,
		SessionLabel:      &label,
	}
}

func assertSessionMetricPresent(t *testing.T, metrics []domain.SessionCompareMetricRow, key string) {
	t.Helper()
	for _, row := range metrics {
		if row.Key == key {
			return
		}
	}
	t.Fatalf("metric %q not found in %#v", key, metrics)
}

func sessionFloat64Ptr(value float64) *float64 {
	return &value
}

// Tests P3 — drawer compare side-by-side : nouveaux champs CompareMatches,
// PreviousSessionLabel, NextSessionLabel sur SessionPageResponse.

func TestSessionPageService_GetPage_ExposesPreviousAndNextLabels(t *testing.T) {
	// Dataset 3 sessions : labels[0]=14h (oldest), labels[1]=18h, labels[2]=19h30 (newest).
	// Session courante = 18h (milieu) → prev=14h, next=19h30.
	repo := &mockSessionPageStatsRepo{matches: makeSessionPageDataset()}
	svc := NewSessionPageService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(repo.matches, repo.err), "halo_infinite", "Test")
	current := "2026-04-21 18h"

	resp, err := svc.GetPage(context.Background(), domain.SessionPageRequest{SessionLabel: &current})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.PreviousSessionLabel == nil || *resp.PreviousSessionLabel != "2026-04-21 14h" {
		t.Fatalf("expected previous=14h, got %v", resp.PreviousSessionLabel)
	}
	if resp.NextSessionLabel == nil || *resp.NextSessionLabel != "2026-04-21 19h30" {
		t.Fatalf("expected next=19h30, got %v", resp.NextSessionLabel)
	}
}

func TestSessionPageService_GetPage_PreviousNextNilAtBoundaries(t *testing.T) {
	repo := &mockSessionPageStatsRepo{matches: makeSessionPageDataset()}
	svc := NewSessionPageService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(repo.matches, repo.err), "halo_infinite", "Test")

	// Plus ancienne session : prev=nil, next=18h.
	oldest := "2026-04-21 14h"
	resp, err := svc.GetPage(context.Background(), domain.SessionPageRequest{SessionLabel: &oldest})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.PreviousSessionLabel != nil {
		t.Fatalf("expected previous=nil at oldest session, got %v", resp.PreviousSessionLabel)
	}
	if resp.NextSessionLabel == nil || *resp.NextSessionLabel != "2026-04-21 18h" {
		t.Fatalf("expected next=18h at oldest session, got %v", resp.NextSessionLabel)
	}

	// Plus récente session : prev=18h, next=nil.
	newest := "2026-04-21 19h30"
	resp, err = svc.GetPage(context.Background(), domain.SessionPageRequest{SessionLabel: &newest})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.PreviousSessionLabel == nil || *resp.PreviousSessionLabel != "2026-04-21 18h" {
		t.Fatalf("expected previous=18h at newest session, got %v", resp.PreviousSessionLabel)
	}
	if resp.NextSessionLabel != nil {
		t.Fatalf("expected next=nil at newest session, got %v", resp.NextSessionLabel)
	}
}

func TestSessionPageService_GetPage_CompareMatchesPopulatedWhenEnabled(t *testing.T) {
	repo := &mockSessionPageStatsRepo{matches: makeSessionPageDataset()}
	svc := NewSessionPageService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(repo.matches, repo.err), "halo_infinite", "Test")
	manual := "2026-04-21 14h" // 2 matchs dans le dataset

	resp, err := svc.GetPage(context.Background(), domain.SessionPageRequest{
		EnableCompare:       true,
		CompareSessionLabel: &manual,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.CompareEnabled {
		t.Fatal("expected compare enabled")
	}
	if len(resp.CompareMatches) != 2 {
		t.Fatalf("expected 2 compare matches (session 14h), got %d", len(resp.CompareMatches))
	}
	// Vérifie que les rows portent bien le label de la session comparée.
	for _, row := range resp.CompareMatches {
		if row.SessionLabel == nil || *row.SessionLabel != manual {
			t.Fatalf("compare match has unexpected session label: %#v", row.SessionLabel)
		}
	}
}

func TestSessionPageService_GetPage_CompareMatchesEmptyWhenDisabled(t *testing.T) {
	repo := &mockSessionPageStatsRepo{matches: makeSessionPageDataset()}
	svc := NewSessionPageService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(repo.matches, repo.err), "halo_infinite", "Test")

	resp, err := svc.GetPage(context.Background(), domain.SessionPageRequest{}) // EnableCompare=false
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.CompareMatches) != 0 {
		t.Fatalf("expected empty compare matches when compare disabled, got %d", len(resp.CompareMatches))
	}
}

func TestNeighboringSessionLabels(t *testing.T) {
	labels := []string{"A", "B", "C", "D"}

	t.Run("middle session", func(t *testing.T) {
		prev, next := neighboringSessionLabels(labels, "B")
		if prev == nil || *prev != "A" {
			t.Fatalf("expected prev=A, got %v", prev)
		}
		if next == nil || *next != "C" {
			t.Fatalf("expected next=C, got %v", next)
		}
	})

	t.Run("first session", func(t *testing.T) {
		prev, next := neighboringSessionLabels(labels, "A")
		if prev != nil {
			t.Fatalf("expected prev=nil, got %v", prev)
		}
		if next == nil || *next != "B" {
			t.Fatalf("expected next=B, got %v", next)
		}
	})

	t.Run("last session", func(t *testing.T) {
		prev, next := neighboringSessionLabels(labels, "D")
		if prev == nil || *prev != "C" {
			t.Fatalf("expected prev=C, got %v", prev)
		}
		if next != nil {
			t.Fatalf("expected next=nil, got %v", next)
		}
	})

	t.Run("single session", func(t *testing.T) {
		prev, next := neighboringSessionLabels([]string{"only"}, "only")
		if prev != nil || next != nil {
			t.Fatalf("expected (nil,nil), got (%v,%v)", prev, next)
		}
	})

	t.Run("unknown session", func(t *testing.T) {
		prev, next := neighboringSessionLabels(labels, "Z")
		if prev != nil || next != nil {
			t.Fatalf("expected (nil,nil) for unknown, got (%v,%v)", prev, next)
		}
	})
}
