package trends

import (
	"sort"
	"time"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/util/pointers"
)

// buildCalendar retourne un élément par jour local joué des 365 derniers
// jours, trié par date. Le taux de victoire est victoires / tous les matchs du
// jour ; le score de performance est la moyenne des scores connus.
func buildCalendar(f *frame) []domain.TrendsCalendarDay {
	out := make([]domain.TrendsCalendarDay, 0, len(f.day))
	for _, b := range f.day {
		day := domain.TrendsCalendarDay{Date: b.Label, Matches: len(b.Items)}
		var perf []float64
		for _, m := range b.Items {
			switch m.Outcome {
			case canonical.OutcomeWin:
				day.Wins++
			case canonical.OutcomeLoss:
				day.Losses++
			}
			if m.PerformanceScore != nil {
				perf = append(perf, *m.PerformanceScore)
			}
		}
		day.WinRate = pointers.Ptr(float64(day.Wins) / float64(day.Matches))
		if len(perf) > 0 {
			day.PerformanceScore = pointers.Ptr(meanOf(perf))
		}
		out = append(out, day)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out
}

// buildGameTypes retourne les chaînes jouées sur 365 jours, triées par nombre
// de matchs décroissant puis par clé. all contient tous les matchs, sans filtre
// de type de partie.
func buildGameTypes(all []Match, now time.Time) []domain.TrendsGameType {
	counts := map[string]int{}
	for _, m := range inWindow(all, daysBefore(now, seriesDays), now) {
		counts[chainKey(m)]++
	}
	out := make([]domain.TrendsGameType, 0, len(counts))
	for k, n := range counts {
		out = append(out, domain.TrendsGameType{Key: k, Matches: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Matches != out[j].Matches {
			return out[i].Matches > out[j].Matches
		}
		return out[i].Key < out[j].Key
	})
	return out
}

// buildMix compte les matchs de l'année par chaîne, à trois pas (jour,
// semaine, mois) ; all contient tous les matchs, sans filtre de type de partie.
func buildMix(all []Match, now time.Time) domain.TrendsMix {
	year := inWindow(all, daysBefore(now, seriesDays), now)
	return domain.TrendsMix{
		Day:   mixBuckets(year, temporal.GranDay),
		Week:  mixBuckets(year, temporal.GranWeek),
		Month: mixBuckets(year, temporal.GranMonth),
	}
}

func mixBuckets(year []Match, gran temporal.Granularity) []domain.TrendsMixBucket {
	buckets := temporal.BucketByGranularity(year, gran, temporal.PeriodAll)
	out := make([]domain.TrendsMixBucket, 0, len(buckets))
	for _, b := range buckets {
		counts := map[string]int{}
		for _, m := range b.Items {
			counts[chainKey(m)]++
		}
		out = append(out, domain.TrendsMixBucket{T: b.Start, Counts: counts})
	}
	return out
}
