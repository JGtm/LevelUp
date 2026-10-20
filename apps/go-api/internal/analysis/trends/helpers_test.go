package trends

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/util/pointers"
)

// testNow est l'instant de référence des tests : 04:00 à Paris le 27/09/2026.
var testNow = time.Date(2026, 9, 27, 2, 0, 0, 0, time.UTC)

func paris(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Skipf("fuseau Europe/Paris indisponible : %v", err)
	}
	return loc
}

// mk crée un match à l'instant at (UTC), exprimé dans loc, avec des valeurs neutres.
func mk(loc *time.Location, at time.Time, mods ...func(*Match)) Match {
	m := Match{
		Start:   at.In(loc),
		Outcome: canonical.OutcomeWin,
		Kills:   10, Deaths: 5, Assists: 3,
		Seconds: 600,
		Chain:   "arena",
	}
	for _, f := range mods {
		f(&m)
	}
	return m
}

// ago retourne testNow moins days jours (durée exacte) plus un décalage.
func ago(days int, plus time.Duration) time.Time {
	return testNow.Add(-time.Duration(days) * 24 * time.Hour).Add(plus)
}

// many crée n matchs espacés d'une minute à partir de start.
func many(loc *time.Location, n int, start time.Time, mods ...func(*Match)) []Match {
	out := make([]Match, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, mk(loc, start.Add(time.Duration(i)*time.Minute), mods...))
	}
	return out
}

func outcome(o canonical.Outcome) func(*Match) { return func(m *Match) { m.Outcome = o } }

func opts(loc *time.Location) Options {
	return Options{Now: testNow, Loc: loc, HpToKill: 225}
}

func indicatorOf(t *testing.T, resp domain.TrendsPageResponse, key, variant string) domain.TrendsIndicator {
	t.Helper()
	for _, ind := range resp.Indicators {
		if ind.Key == key && ind.Variant == variant {
			return ind
		}
	}
	t.Fatalf("indicateur %s/%q absent", key, variant)
	return domain.TrendsIndicator{}
}

func hasIndicator(resp domain.TrendsPageResponse, key string) bool {
	for _, ind := range resp.Indicators {
		if ind.Key == key {
			return true
		}
	}
	return false
}

// horizon retourne la cellule d'un horizon (en jours).
func horizon(t *testing.T, ind domain.TrendsIndicator, days int) domain.TrendsHorizonCell {
	t.Helper()
	for _, c := range ind.Horizons {
		if c.Days == days {
			return c
		}
	}
	t.Fatalf("horizon %d absent", days)
	return domain.TrendsHorizonCell{}
}

func near(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 1e-9
}

func f64(v float64) *float64 { return pointers.Ptr(v) }
