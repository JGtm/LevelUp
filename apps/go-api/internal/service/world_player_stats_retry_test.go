package service

import (
	"context"
	"errors"
	"testing"
	"time"

	syncpkg "levelup/go-api/internal/sync"
)

func TestIsTransientErr(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"429", &syncpkg.HTTPError{StatusCode: 429}, true},
		{"503", &syncpkg.HTTPError{StatusCode: 503}, true},
		{"500", &syncpkg.HTTPError{StatusCode: 500}, true},
		{"404 non transitoire", &syncpkg.HTTPError{StatusCode: 404}, false},
		{"401 non transitoire", &syncpkg.HTTPError{StatusCode: 401}, false},
		{"ctx canceled", context.Canceled, false},
		{"ctx deadline", context.DeadlineExceeded, false},
		{"erreur réseau générique", errors.New("connreset"), true},
		{"429 wrappé", errors.New("x: " + (&syncpkg.HTTPError{StatusCode: 429}).Error()), true}, // texte seul → traité réseau (transitoire) : acceptable
	}
	for _, c := range cases {
		if got := isTransientErr(c.err); got != c.want {
			t.Errorf("%s: isTransientErr = %v, want %v", c.name, got, c.want)
		}
	}
}

// flakySource échoue les `failN` premiers appels (par méthode) avec un 429 puis réussit.
type flakySource struct {
	histFails, statsFails int
}

func (f *flakySource) GetMatchHistory(_ context.Context, _, _ string, _, _ int) ([]syncpkg.MatchHistoryEntry, error) {
	if f.histFails > 0 {
		f.histFails--
		return nil, &syncpkg.HTTPError{StatusCode: 429}
	}
	return []syncpkg.MatchHistoryEntry{{MatchID: "m1"}}, nil
}

func (f *flakySource) GetMatchStats(_ context.Context, _ string) (map[string]any, error) {
	if f.statsFails > 0 {
		f.statsFails--
		return nil, &syncpkg.HTTPError{StatusCode: 429}
	}
	return buildMatch("42", "Csr/Seasons/CsrSeason13-2.json", tArena, 2, 5, 5, 5), nil
}

// TestWithRetry_RecoversAfter429 vérifie qu'un 429 transitoire est retenté et que
// le joueur est finalement agrégé (pas de perte). Délais réduits pour le test.
func TestWithRetry_RecoversAfter429(t *testing.T) {
	old := retryDelays
	retryDelays = []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}
	defer func() { retryDelays = old }()

	src := &flakySource{histFails: 2, statsFails: 2} // 2 échecs chacun avant succès
	agg := NewWorldStatsAggregator(src, &fakeResolver{m: map[string]string{"Neo": "42"}},
		WorldStatsAggregatorConfig{
			TargetSeasons: map[string]bool{"csrseason13-2": true},
			MaxPages:      1,
		})

	stats, err := agg.AggregatePlayer(context.Background(), "Neo")
	if err != nil {
		t.Fatalf("AggregatePlayer: 429 transitoire aurait dû être retenté, got err=%v", err)
	}
	if len(stats) != 1 || stats[0].Kills != 5 {
		t.Fatalf("stats = %+v, want 1 bucket 5 kills (récupéré après retries)", stats)
	}
}

// TestWithRetry_GivesUpAfterMaxAttempts vérifie qu'au-delà des retries, l'erreur remonte.
func TestWithRetry_GivesUpAfterMaxAttempts(t *testing.T) {
	old := retryDelays
	retryDelays = []time.Duration{time.Millisecond, time.Millisecond}
	defer func() { retryDelays = old }()

	src := &flakySource{histFails: 99} // toujours 429 sur l'historique
	agg := NewWorldStatsAggregator(src, &fakeResolver{m: map[string]string{"Neo": "42"}},
		WorldStatsAggregatorConfig{TargetSeasons: map[string]bool{"csrseason13-2": true}, MaxPages: 1})

	if _, err := agg.AggregatePlayer(context.Background(), "Neo"); err == nil {
		t.Fatal("attendu une erreur après épuisement des retries")
	}
}
