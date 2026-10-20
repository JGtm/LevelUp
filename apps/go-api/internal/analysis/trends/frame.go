package trends

import (
	"time"

	"levelup/go-api/internal/analysis/temporal"
)

// Fenêtres et seuils de la page.
const (
	// minMatchesCompare : matchs requis de part et d'autre pour comparer un horizon à la période d'avant.
	minMatchesCompare = 10
	// minMatchesMonth : matchs requis pour qu'un mois porte une valeur.
	minMatchesMonth = 5
	// matchSeriesDays : profondeur de la série par match.
	matchSeriesDays = 30
	// seriesDays : profondeur des séries par jour, semaine et mois.
	seriesDays  = 365
	monthCount  = 12
	hoursPerDay = 24
)

// Minimum de matchs par point de série, par pas.
const (
	minPointDay   = 2
	minPointWeek  = 3
	minPointMonth = 5
)

// horizonDays retourne les horizons de la page, du plus long au plus court.
func horizonDays() [4]int { return [4]int{365, 90, 30, 7} }

// daysBefore retourne l'instant situé d jours avant t, en durée exacte.
func daysBefore(t time.Time, d int) time.Time {
	return t.Add(-time.Duration(d) * hoursPerDay * time.Hour)
}

// inWindow retourne les matchs de ms dans ]from, to].
func inWindow(ms []Match, from, to time.Time) []Match {
	out := make([]Match, 0)
	for _, m := range ms {
		if m.Start.After(from) && !m.Start.After(to) {
			out = append(out, m)
		}
	}
	return out
}

// monthSpan est un mois local avec ses matchs.
type monthSpan struct {
	key     string
	matches []Match
}

// horizonWindow porte les matchs d'un horizon et de la période d'avant.
type horizonWindow struct {
	days int
	cur  []Match
	prev []Match
}

// frame est le jeu de fenêtres calculé une fois et partagé par tous les indicateurs.
type frame struct {
	now    time.Time
	loc    *time.Location
	months []monthSpan
	hor    []horizonWindow
	last30 []Match
	day    []temporal.Bucket[Match]
	week   []temporal.Bucket[Match]
	month  []temporal.Bucket[Match]
}

// newFrame découpe scope (trié, sans match après now) en mois, horizons et
// intervalles de séries, dans le fuseau loc.
func newFrame(scope []Match, now time.Time, loc *time.Location) *frame {
	f := &frame{now: now, loc: loc}
	f.months = monthSpans(scope, now, loc)
	for _, d := range horizonDays() {
		f.hor = append(f.hor, horizonWindow{
			days: d,
			cur:  inWindow(scope, daysBefore(now, d), now),
			prev: inWindow(scope, daysBefore(now, 2*d), daysBefore(now, d)),
		})
	}
	f.last30 = inWindow(scope, daysBefore(now, matchSeriesDays), now)
	year := inWindow(scope, daysBefore(now, seriesDays), now)
	f.day = temporal.BucketByGranularity(year, temporal.GranDay, temporal.PeriodAll)
	f.week = temporal.BucketByGranularity(year, temporal.GranWeek, temporal.PeriodAll)
	f.month = temporal.BucketByGranularity(year, temporal.GranMonth, temporal.PeriodAll)
	return f
}

// monthSpans retourne les 12 derniers mois locaux (le mois courant en dernier)
// avec leurs matchs.
func monthSpans(scope []Match, now time.Time, loc *time.Location) []monthSpan {
	ln := now.In(loc)
	first := time.Date(ln.Year(), ln.Month(), 1, 0, 0, 0, 0, loc)
	out := make([]monthSpan, 0, monthCount)
	for i := monthCount - 1; i >= 0; i-- {
		start := first.AddDate(0, -i, 0)
		end := start.AddDate(0, 1, 0)
		span := monthSpan{key: start.Format(monthKeyLayout), matches: make([]Match, 0)}
		for _, m := range scope {
			if !m.Start.Before(start) && m.Start.Before(end) {
				span.matches = append(span.matches, m)
			}
		}
		out = append(out, span)
	}
	return out
}

// monthKeys retourne les clés AAAA-MM des mois du cadre.
func (f *frame) monthKeys() []string {
	keys := make([]string, 0, len(f.months))
	for _, m := range f.months {
		keys = append(keys, m.key)
	}
	return keys
}
