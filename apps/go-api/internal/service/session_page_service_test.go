package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
)

// fakeSessionHighlightLoader implémente highlightEventsLoader pour tester le
// profil d'intensité de la session sans DuckDB (renvoie des events fixes).
type fakeSessionHighlightLoader struct {
	events []canonical.HighlightEvent
	err    error
}

func (f *fakeSessionHighlightLoader) Load(_ context.Context, _ port.HighlightEventFilters) ([]canonical.HighlightEvent, error) {
	return f.events, f.err
}

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
	// Période resserrée sur 19h40→20h00 : seul m6 (19h50) de la session 19h30 est
	// retenu pour la VUE (m5 à 19h30 est exclu).
	start := time.Date(2026, 4, 21, 19, 40, 0, 0, time.UTC)
	end := time.Date(2026, 4, 21, 20, 0, 0, 0, time.UTC)

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
	// La période s'applique bien à la VUE : la session courante est 19h30 mais ne
	// retient qu'un seul match (m6), m5 étant hors période.
	if resp.CurrentSession == nil || resp.CurrentSession.SessionLabel != "2026-04-21 19h30" {
		t.Fatalf("unexpected filtered current session: %#v", resp.CurrentSession)
	}
	if len(resp.Matches) != 1 {
		t.Fatalf("expected 1 match in view after period filter, got %d", len(resp.Matches))
	}
	// Mais le VIVIER de comparaison reste élargi : les 3 sessions (période ignorée),
	// sinon le bouton Comparer disparaîtrait dès qu'on resserre la période.
	if len(resp.AvailableSessions) != 3 {
		t.Fatalf("expected 3 sessions in compare pool (period ignored), got %d (%v)", len(resp.AvailableSessions), resp.AvailableSessions)
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

// ADR 0029 Couche B : une session demandée explicitement mais introuvable
// renvoie désormais session_not_found (404) au lieu d'une page vide 200
// trompeuse (ancien comportement de fallback silencieux).
func TestSessionPageService_GetPage_UnknownRequestedSessionReturnsNotFound(t *testing.T) {
	repo := &mockSessionPageStatsRepo{matches: makeSessionPageDataset()}
	svc := NewSessionPageService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(repo.matches, repo.err), "halo_infinite", "Test")
	unknown := "2026-04-21 23h"

	_, err := svc.GetPage(context.Background(), domain.SessionPageRequest{SessionLabel: &unknown})
	if err == nil {
		t.Fatal("attendu session_not_found, obtenu nil")
	}
	var apiErr *domain.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "session_not_found" {
		t.Fatalf("attendu APIError session_not_found, obtenu %v", err)
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
	if suggestion.Reason != "même composition (solo) · même catégorie ranked · même statut classé · écart de 1 match(s)" {
		t.Fatalf("unexpected reason: %s", suggestion.Reason)
	}
}

func TestKeepMultiMatchSessionLabels(t *testing.T) {
	mk := func(id, label string) legacymatch.StatsMatchRow {
		l := label
		return legacymatch.StatsMatchRow{MatchID: id, SessionLabel: &l}
	}
	rows := []legacymatch.StatsMatchRow{
		mk("m1", "S-multi"), mk("m2", "S-multi"), // 2 matchs → gardé
		mk("m3", "S-single"), // 1 match → exclu
		{MatchID: "m4"},      // label nil → ignoré
	}
	got := keepMultiMatchSessionLabels([]string{"S-multi", "S-single"}, rows)
	if len(got) != 1 || got[0] != "S-multi" {
		t.Fatalf("expected [S-multi], got %v", got)
	}
}

// TestSessionPageService_GetPage_DropsSingleMatchSessionFromList : une session d'un seul
// match n'apparaît pas dans available_sessions ni dans la navigation, mais reste
// accessible en deep-link (req.SessionLabel).
func TestSessionPageService_GetPage_DropsSingleMatchSessionFromList(t *testing.T) {
	now := time.Date(2026, 4, 21, 20, 0, 0, 0, time.UTC)
	// 2 sessions de 2 matchs + 1 session "orpheline" d'un seul match (la plus récente).
	dataset := append(makeSessionPageDataset(),
		makeSessionPageMatch("m7", now.Add(-2*time.Minute), "2026-04-21 19h58", true, "Ranked Arena", "Oddball", 9, 9, 1, 60, 60),
	)
	repo := &mockSessionPageStatsRepo{matches: dataset}
	svc := NewSessionPageService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(repo.matches, repo.err), "halo_infinite", "Test")

	resp, err := svc.GetPage(context.Background(), domain.SessionPageRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, lbl := range resp.AvailableSessions {
		if lbl == "2026-04-21 19h58" {
			t.Fatalf("single-match session should not be listed, got %v", resp.AvailableSessions)
		}
	}
	// Le landing par défaut saute la session d'un match → 19h30 (la plus récente ≥2 matchs).
	if resp.CurrentSession == nil || resp.CurrentSession.SessionLabel != "2026-04-21 19h30" {
		t.Fatalf("expected landing on 19h30, got %#v", resp.CurrentSession)
	}

	// Deep-link vers la session d'un seul match : toujours résoluble.
	single := "2026-04-21 19h58"
	resp, err = svc.GetPage(context.Background(), domain.SessionPageRequest{SessionLabel: &single})
	if err != nil {
		t.Fatalf("unexpected error (deep-link): %v", err)
	}
	if resp.CurrentSession == nil || resp.CurrentSession.SessionLabel != single {
		t.Fatalf("deep-link to single-match session should resolve, got %#v", resp.CurrentSession)
	}
	if len(resp.Matches) != 1 {
		t.Fatalf("expected 1 match for single-match session, got %d", len(resp.Matches))
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

// TestSessionPageService_AttachSessionIntensity : le profil d'intensité (frags par
// phase) est calculé pour la session courante et la comparée quand le repo highlight
// events est câblé ; il reste nil (best-effort) sans repo. Le dénominateur retombe sur
// le max-time des events (durées timelines vides ici), donc les phases sont non nulles.
func TestSessionPageService_AttachSessionIntensity(t *testing.T) {
	const xuid = "player1"
	tt := time.Date(2026, 4, 21, 20, 0, 0, 0, time.UTC)
	currentMatches := []legacymatch.StatsMatchRow{{MatchID: "m1", StartTime: tt, MapName: "Aquarius"}}
	compareMatches := []legacymatch.StatsMatchRow{{MatchID: "c1", StartTime: tt, MapName: "Bazaar"}}
	events := []canonical.HighlightEvent{
		{MatchID: "m1", EventType: string(canonical.EventKill), TimeMS: 100, XUID: xuid},
		{MatchID: "m1", EventType: string(canonical.EventKill), TimeMS: 900, XUID: xuid},
		{MatchID: "c1", EventType: string(canonical.EventKill), TimeMS: 200, XUID: xuid},
	}

	// Sans repo highlight events → dégradation gracieuse (IntensityRows nil).
	noRepo := NewSessionPageService(nil)
	respNil := &domain.SessionPageResponse{}
	noRepo.attachSessionIntensity(context.Background(), respNil, nil, currentMatches, compareMatches)
	if respNil.IntensityRows != nil || respNil.CompareIntensityRows != nil {
		t.Fatalf("sans repo highlight events, IntensityRows doit rester nil, got %#v / %#v",
			respNil.IntensityRows, respNil.CompareIntensityRows)
	}

	// Avec repo câblé → profil courant + comparé renseignés.
	svc := NewSessionPageService(nil).
		WithHighlightEventsRepo(&fakeSessionHighlightLoader{events: events}, xuid)
	resp := &domain.SessionPageResponse{}
	svc.attachSessionIntensity(context.Background(), resp, nil, currentMatches, compareMatches)
	if len(resp.IntensityRows) != 1 || resp.IntensityRows[0].MatchID != "m1" {
		t.Fatalf("IntensityRows courant inattendu: %#v", resp.IntensityRows)
	}
	if len(resp.CompareIntensityRows) != 1 || resp.CompareIntensityRows[0].MatchID != "c1" {
		t.Fatalf("CompareIntensityRows inattendu: %#v", resp.CompareIntensityRows)
	}
	// Au moins une phase non nulle (frags répartis sur la durée = max-time proxy).
	var sum float64
	for _, p := range resp.IntensityRows[0].Phases {
		sum += p
	}
	if sum <= 0 {
		t.Fatalf("phases du profil courant toutes nulles: %#v", resp.IntensityRows[0].Phases)
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

// TestSessionPageService_ComparePoolIgnoresPeriodNarrowing : quand le filtre de
// période resserre la VUE sur une seule session, le VIVIER de comparaison reste
// élargi (toutes les sessions de la catégorie) → le bouton Comparer reste pertinent.
// Régression du bug "le bouton Comparer disparaît quand on filtre sur une session".
func TestSessionPageService_ComparePoolIgnoresPeriodNarrowing(t *testing.T) {
	now := time.Date(2026, 4, 21, 20, 0, 0, 0, time.UTC)
	repo := &mockSessionPageStatsRepo{
		matches: []legacymatch.StatsMatchRow{
			makeSessionPageMatch("m1", now.Add(-6*time.Hour), "2026-04-21 14h", false, "Quick Play", "Slayer", 10, 8, 2, 74, 54),
			makeSessionPageMatch("m2", now.Add(-5*time.Hour), "2026-04-21 14h", false, "Quick Play", "Slayer", 12, 7, 3, 76, 58),
			makeSessionPageMatch("m3", now.Add(-2*time.Hour), "2026-04-21 18h", false, "Quick Play", "Slayer", 14, 9, 4, 68, 61),
			makeSessionPageMatch("m4", now.Add(-90*time.Minute), "2026-04-21 18h", false, "Quick Play", "Slayer", 16, 10, 5, 71, 64),
			makeSessionPageMatch("m5", now.Add(-30*time.Minute), "2026-04-21 19h30", false, "Quick Play", "Slayer", 11, 6, 4, 62, 67),
			makeSessionPageMatch("m6", now.Add(-10*time.Minute), "2026-04-21 19h30", false, "Quick Play", "Slayer", 13, 5, 6, 64, 70),
		},
	}
	svc := NewSessionPageService(repo).WithPlayerMatchesRepo(newStatsMockFromRows(repo.matches, repo.err), "halo_infinite", "Test")

	// Période resserrée sur la dernière session uniquement (19h00 → 20h00).
	start := now.Add(-time.Hour)
	resp, err := svc.GetPage(context.Background(), domain.SessionPageRequest{
		Filters: domain.FilterContextInput{
			FilterMode: "period",
			Period:     domain.PeriodInput{StartDate: &start, EndDate: &now},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// La VUE ne ressort qu'une session (effet du filtre période).
	if resp.CurrentSession == nil || resp.CurrentSession.SessionLabel != "2026-04-21 19h30" {
		t.Fatalf("current session = %#v, want 19h30", resp.CurrentSession)
	}
	// Le VIVIER de comparaison reste élargi : les 3 sessions (période ignorée) →
	// available_sessions.length >= 2 côté front → bouton Comparer visible.
	if len(resp.AvailableSessions) != 3 {
		t.Fatalf("available_sessions (vivier compare) = %d, want 3 (période ignorée)", len(resp.AvailableSessions))
	}
	// Une suggestion existe (sessions antérieures dans le vivier élargi).
	if resp.SuggestedCompare == nil {
		t.Fatal("expected a compare suggestion from the broadened pool")
	}
}

// TestComputeSessionPlacements_CSR vérifie le placement X/Y CSR (réutilise
// applyMatchPlacements via la conversion) : remaining=2, seuil défaut 5 → 3/5.
func TestComputeSessionPlacements_CSR(t *testing.T) {
	svc := &SessionPageService{} // csrThreshold nil → seuil défaut 5
	start := time.Date(2026, 4, 21, 20, 0, 0, 0, time.UTC)
	rem := 2
	rows := []legacymatch.StatsMatchRow{
		{MatchID: "m1", StartTime: start, SkillRatingType: "csr", SkillMeasurementRemaining: &rem},
		{MatchID: "m2", StartTime: start, SkillRatingType: "csr"}, // établi (pas de remaining) → pas de placement
	}
	pl := svc.computeSessionPlacements(context.Background(), rows)
	if p, ok := pl["m1"]; !ok || p.done != 3 || p.total != 5 {
		t.Fatalf("m1 placement = %+v (ok=%v), want {done:3 total:5}", pl["m1"], ok)
	}
	if _, ok := pl["m2"]; ok {
		t.Fatal("m2 ne doit pas être en placement (pas de remaining)")
	}

	// applyPlacementsToRows renseigne les rows correspondantes.
	detail := []domain.SessionDetailMatchRow{{MatchID: "m1"}, {MatchID: "m2"}}
	applyPlacementsToRows(detail, pl)
	if detail[0].PlacementDone == nil || *detail[0].PlacementDone != 3 || detail[0].PlacementTotal == nil || *detail[0].PlacementTotal != 5 {
		t.Fatalf("m1 row placement non appliqué: done=%v total=%v", detail[0].PlacementDone, detail[0].PlacementTotal)
	}
	if detail[1].PlacementDone != nil {
		t.Fatal("m2 row ne doit pas avoir de placement")
	}
}

// TestBuildSessionDetailRows_SkillTierLabel vérifie que la colonne "Rang" reçoit le
// LIBELLÉ du palier ("Or III" / "Gold III"), comme l'Explorer, et non la valeur brute.
func TestBuildSessionDetailRows_SkillTierLabel(t *testing.T) {
	start := time.Date(2026, 4, 21, 20, 0, 0, 0, time.UTC)
	label := "S"
	win := analysis.OutcomeWin
	gold, goldFR, sub := "gold", "Or", 3
	deref := func(rs []domain.SessionDetailMatchRow) string {
		if len(rs) == 1 && rs[0].SkillTierLabel != nil {
			return *rs[0].SkillTierLabel
		}
		return "<nil>"
	}
	ranked := []legacymatch.StatsMatchRow{{
		MatchID: "m1", StartTime: start, Outcome: &win, Kills: 1, Deaths: 1, SessionLabel: &label,
		SkillRatingType: "csr", SkillTierCode: &gold, SkillTierCodeFR: &goldFR, SkillSubTier: &sub,
	}}
	if got := deref(buildSessionDetailRows(ranked, nil, "fr", nil)); got != "Or III" {
		t.Fatalf("FR SkillTierLabel = %q, want %q", got, "Or III")
	}
	if got := deref(buildSessionDetailRows(ranked, nil, "en", nil)); got != "Gold III" {
		t.Fatalf("EN SkillTierLabel = %q, want %q", got, "Gold III")
	}
	// Non rankée (pas de tier) → nil (le front affiche "-").
	noTier := []legacymatch.StatsMatchRow{{MatchID: "m2", StartTime: start, Outcome: &win, SessionLabel: &label}}
	if got := deref(buildSessionDetailRows(noTier, nil, "fr", nil)); got != "<nil>" {
		t.Fatalf("no-tier SkillTierLabel = %q, want <nil>", got)
	}
}

// TestBuildSessionDetailRows_ModeUILocale verrouille la résolution locale-aware du
// libellé de mode (mode_ui) consommé par le tableau ET le graphe "Modes joués".
// Même contrat que Home/Explorer : FR si une traduction FR existe (PairNameFR,
// alimenté en amont par l'enrichissement canonical mode_name_tr), sinon sous-mode
// normalisé EN. Le 3e cas documente le repli quand mode_name_tr ne couvre pas le mode.
func TestBuildSessionDetailRows_ModeUILocale(t *testing.T) {
	start := time.Date(2026, 4, 21, 20, 0, 0, 0, time.UTC)
	label := "2026-04-21 20h"
	win := analysis.OutcomeWin
	modeUIOf := func(rows []domain.SessionDetailMatchRow) string {
		if len(rows) != 1 {
			t.Fatalf("expected 1 row, got %d", len(rows))
		}
		return rows[0].ModeUI
	}

	withFR := []legacymatch.StatsMatchRow{{
		MatchID: "m1", StartTime: start, Outcome: &win, Kills: 10, Deaths: 8,
		PairName: "Arena:Team Slayer on Live Fire", PairNameFR: "Slayer en équipe", SessionLabel: &label,
	}}
	if got := modeUIOf(buildSessionDetailRows(withFR, nil, "fr", nil)); got != "Slayer en équipe" {
		t.Fatalf("FR ModeUI = %q, want %q", got, "Slayer en équipe")
	}
	if got := modeUIOf(buildSessionDetailRows(withFR, nil, "en", nil)); got != "Team Slayer" {
		t.Fatalf("EN ModeUI = %q, want %q (trad FR ignorée en EN)", got, "Team Slayer")
	}

	// PairNameFR vide + aucun GameVariant FR (mode_name_tr sans entrée) → repli EN.
	noFR := []legacymatch.StatsMatchRow{{
		MatchID: "m2", StartTime: start, Outcome: &win, Kills: 5, Deaths: 5,
		PairName: "Arena:Team Slayer on Live Fire", PairNameFR: "", SessionLabel: &label,
	}}
	if got := modeUIOf(buildSessionDetailRows(noFR, nil, "fr", nil)); got != "Team Slayer" {
		t.Fatalf("FR sans variant : ModeUI = %q, want %q", got, "Team Slayer")
	}

	// PairNameFR vide MAIS GameVariant FR localisé ("Assassin en équipe : Arène",
	// format réel asset_translations[game_variant]) → repli sur le variant FR.
	// asset_translations[pair] n'étant pas localisé, c'est le SEUL chemin FR pour
	// les modes absents de mode_name_tr (aligné Home).
	variantFR := []legacymatch.StatsMatchRow{{
		MatchID: "m3", StartTime: start, Outcome: &win, Kills: 7, Deaths: 4,
		PairName: "Arena:Team Slayer on Catalyst - Forge", PairNameFR: "",
		GameVariantName: "Team Slayer:Arena", GameVariantNameFR: "Assassin en équipe : Arène",
		SessionLabel: &label,
	}}
	if got := modeUIOf(buildSessionDetailRows(variantFR, nil, "fr", nil)); got != "Assassin en équipe" {
		t.Fatalf("FR repli GameVariant : ModeUI = %q, want %q", got, "Assassin en équipe")
	}
	// En EN, le repli FR ne s'applique jamais → sous-mode normalisé EN du pair.
	if got := modeUIOf(buildSessionDetailRows(variantFR, nil, "en", nil)); got != "Team Slayer" {
		t.Fatalf("EN avec variant FR : ModeUI = %q, want %q", got, "Team Slayer")
	}
}
