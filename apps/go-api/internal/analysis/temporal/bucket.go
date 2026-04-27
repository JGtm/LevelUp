package temporal

import (
	"fmt"
	"sort"
	"time"
)

// Granularity represente une largeur de bucket temporel.
type Granularity string

// Constantes des granularites admises. GranAdaptive est resolue via
// ResolveAdaptive(period) au moment du bucketing.
const (
	GranDay      Granularity = "1d"
	GranWeek     Granularity = "1w"
	GranMonth    Granularity = "1m"
	GranAdaptive Granularity = "adaptive"
)

// IsValid retourne true si la valeur est connue.
func (g Granularity) IsValid() bool {
	switch g {
	case GranDay, GranWeek, GranMonth, GranAdaptive:
		return true
	}
	return false
}

// ResolveAdaptive choisit une granularite basee sur la longueur de la periode.
// Mapping :
//
//	1w / 1m  -> 1d
//	1y       -> 1w
//	2y / all -> 1m
//
// Le defaut sur valeur inconnue est GranDay (defensive).
func ResolveAdaptive(period Period) Granularity {
	switch period {
	case Period1W, Period1M:
		return GranDay
	case Period1Y:
		return GranWeek
	case Period2Y, PeriodAll:
		return GranMonth
	}
	return GranDay
}

// Bucket regroupe des rows partageant un meme intervalle temporel [Start, End[.
// Label est une string lisible (ex. "2026-04-27", "2026-W17", "2026-04").
type Bucket[T any] struct {
	Start time.Time
	End   time.Time
	Label string
	Items []T
}

// BucketByGranularity regroupe les rows par bucket. La granularite GranAdaptive
// est resolue via ResolveAdaptive(period). Les buckets vides ne sont pas crees.
// Les buckets retournes sont tries par Start ascendant.
//
// Le calcul du bucket Start preserve la timezone des rows entrantes : les rows
// avec des locations differentes peuvent generer des buckets distincts meme
// pour le meme moment absolu. C'est un choix volontaire : le bucketing UI doit
// refleter le decoupage local du joueur.
func BucketByGranularity[T HasStartTime](rows []T, gran Granularity, period Period) []Bucket[T] {
	if gran == GranAdaptive {
		gran = ResolveAdaptive(period)
	}
	grouped := make(map[time.Time][]T)
	for _, r := range rows {
		start := bucketStart(r.GetStartTime(), gran)
		grouped[start] = append(grouped[start], r)
	}
	out := make([]Bucket[T], 0, len(grouped))
	for start, items := range grouped {
		end := bucketEnd(start, gran)
		out = append(out, Bucket[T]{
			Start: start,
			End:   end,
			Label: bucketLabel(start, gran),
			Items: items,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out
}

// bucketStart retourne le debut du bucket contenant t, selon la granularite.
// Pour la semaine, l'origine est le lundi (ISO 8601).
func bucketStart(t time.Time, gran Granularity) time.Time {
	y, m, d := t.Date()
	loc := t.Location()
	switch gran {
	case GranDay:
		return time.Date(y, m, d, 0, 0, 0, 0, loc)
	case GranWeek:
		// ISO 8601 : lundi est le premier jour. time.Weekday() retourne 0 pour
		// dimanche : on remappe vers 7 pour calculer l'offset depuis lundi.
		wd := int(t.Weekday())
		if wd == 0 {
			wd = 7
		}
		offset := wd - 1
		return time.Date(y, m, d-offset, 0, 0, 0, 0, loc)
	case GranMonth:
		return time.Date(y, m, 1, 0, 0, 0, 0, loc)
	}
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}

// bucketEnd retourne la borne superieure exclusive du bucket.
func bucketEnd(start time.Time, gran Granularity) time.Time {
	switch gran {
	case GranDay:
		return start.AddDate(0, 0, 1)
	case GranWeek:
		return start.AddDate(0, 0, 7)
	case GranMonth:
		return start.AddDate(0, 1, 0)
	}
	return start.AddDate(0, 0, 1)
}

// bucketLabel formate le label lisible du bucket selon la granularite.
//
//	day   -> "2006-01-02"
//	week  -> "2006-W02" (numero ISO sur 2 chiffres)
//	month -> "2006-01"
func bucketLabel(start time.Time, gran Granularity) string {
	switch gran {
	case GranDay:
		return start.Format("2006-01-02")
	case GranWeek:
		y, w := start.ISOWeek()
		return fmt.Sprintf("%d-W%02d", y, w)
	case GranMonth:
		return start.Format("2006-01")
	}
	return start.Format("2006-01-02")
}
