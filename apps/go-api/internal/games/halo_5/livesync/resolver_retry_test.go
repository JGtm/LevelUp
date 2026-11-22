package livesync

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/observability"
)

// noSleep remplace l'attente et l'horloge de la résolution par un compteur de durée : aucune suite ne dort.
func noSleep(t *testing.T) *time.Duration {
	t.Helper()
	var slept time.Duration
	var clock time.Time
	oldSleep, oldNow := resolveSleep, resolveNow
	resolveSleep = func(_ context.Context, d time.Duration) error { slept += d; clock = clock.Add(d); return nil }
	resolveNow = func() time.Time { return clock }
	t.Cleanup(func() { resolveSleep, resolveNow = oldSleep, oldNow })
	return &slept
}

// throttledResolver rend un 429 les `throttles` premières fois, puis résout.
type throttledResolver struct {
	throttles int
	calls     int
}

func (f *throttledResolver) ResolveXUID(_ context.Context, gt string) (string, error) {
	f.calls++
	if f.calls <= f.throttles {
		return "", fmt.Errorf("chain resolver throttled (429): %w", errors.New("peoplehub HTTP 429: slow down"))
	}
	return "x-" + gt, nil
}

// Un 429 deux fois puis la réponse : le participant est résolu, après un recul croissant.
func TestResolveXUIDClosure_429DeuxFoisPuisResolu(t *testing.T) {
	slept := noSleep(t)
	before := observability.LoadCounter(metricResolveRetry)
	f := &throttledResolver{throttles: 2}
	rx := ResolveXUIDClosure(context.Background(), f, nil)
	if got := rx("Adversaire"); got != "x-Adversaire" {
		t.Fatalf("résolution %q, want x-Adversaire", got)
	}
	if f.calls != 3 {
		t.Errorf("%d appels, want 3", f.calls)
	}
	if want := resolveBackoffFirst + 2*resolveBackoffFirst; *slept != want {
		t.Errorf("attente %v, want %v (recul doublé)", *slept, want)
	}
	if got := observability.LoadCounter(metricResolveRetry) - before; got != 2 {
		t.Errorf("compteur des attentes +%d, want +2", got)
	}
}

// Une autre erreur qu'un 429 n'attend pas.
func TestResolveXUIDClosure_ErreurNon429SansAttente(t *testing.T) {
	slept := noSleep(t)
	rx := ResolveXUIDClosure(context.Background(), fakeResolver{err: errors.New("gamertag absent")}, nil)
	if got := rx("X"); got != "" || *slept != 0 {
		t.Errorf("résolution %q après %v, want vide et sans attente", got, *slept)
	}
}

// Le budget d'attente de la passe borne la patience : un service saturé rend l'erreur, comptée.
func TestResolveXUIDClosure_BudgetDAttenteEpuise(t *testing.T) {
	slept := noSleep(t)
	before := observability.LoadCounter(metricResolveGiveUp)
	f := &throttledResolver{throttles: 1 << 30}
	rx := ResolveXUIDClosure(context.Background(), f, nil)
	if got := rx("X"); got != "" {
		t.Fatalf("résolution %q, want vide (budget épuisé)", got)
	}
	if *slept > resolveRetryBudget || *slept < resolveRetryBudget-resolveBackoffMax {
		t.Errorf("attente cumulée %v, want proche de %v sans le dépasser", *slept, resolveRetryBudget)
	}
	if got := observability.LoadCounter(metricResolveGiveUp) - before; got != 1 {
		t.Errorf("compteur des abandons +%d, want +1", got)
	}
	// Le budget est celui de la passe : un second gamertag n'attend plus que le reste, jamais au-delà.
	_ = rx("Y")
	if *slept > resolveRetryBudget {
		t.Errorf("attente cumulée %v après deux résolutions, want au plus %v", *slept, resolveRetryBudget)
	}
}

// La cadence espace les requêtes d'un compte sous la limite observée, sans attendre la première.
func TestPacedResolver_EspaceLesRequetes(t *testing.T) {
	slept := noSleep(t)
	p := newPacedResolver(fakeResolver{m: map[string]string{"A": "1", "B": "2", "C": "3"}})
	for _, gt := range []string{"A", "B", "C"} {
		if _, err := p.ResolveXUID(context.Background(), gt); err != nil {
			t.Fatal(err)
		}
	}
	if want := 2 * resolveMinGap; *slept != want {
		t.Errorf("attente %v pour trois requêtes, want %v (la première part tout de suite)", *slept, want)
	}
	if per15s := float64(15*time.Second) / float64(resolveMinGap); per15s >= 10 {
		t.Errorf("%.1f requêtes par 15 s, want moins de 10 (limite observée)", per15s)
	}
}
