package trends

import (
	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/util/pointers"
)

// buildIndicators évalue chaque définition sur le cadre ; une ligne sans
// aucune valeur (mois ni horizon) est omise.
func buildIndicators(defs []indicator, f *frame) []domain.TrendsIndicator {
	out := make([]domain.TrendsIndicator, 0, len(defs))
	for _, d := range defs {
		ind := domain.TrendsIndicator{
			Key: d.key, Variant: d.variant, Group: d.group, Unit: d.unit,
			Decimals: d.decimals, Better: d.better, InMatrix: d.inMatrix,
		}
		var sd float64
		ind.Months, sd = monthCells(d, f)
		ind.Horizons = horizonCells(d, f, sd)
		if !hasValue(ind) {
			continue
		}
		ind.Series = buildSeries(d, f)
		out = append(out, ind)
	}
	return out
}

// hasValue indique si au moins une cellule du mois ou d'horizon porte une valeur.
func hasValue(ind domain.TrendsIndicator) bool {
	for _, c := range ind.Months {
		if c.Value != nil {
			return true
		}
	}
	for _, c := range ind.Horizons {
		if c.Value != nil {
			return true
		}
	}
	return false
}

// monthCells retourne les 12 cellules mensuelles et l'écart-type des mois
// (déjà remplacé par 1 s'il est nul). Un mois porte une valeur dès 5 matchs ;
// son z est better x (valeur - moyenne des mois) / écart-type des mois.
func monthCells(d indicator, f *frame) ([]domain.TrendsMonthCell, float64) {
	cells := make([]domain.TrendsMonthCell, len(f.months))
	var vals []float64
	for i, span := range f.months {
		cells[i].Matches = len(span.matches)
		if len(span.matches) < minMatchesMonth {
			continue
		}
		if v, ok := d.cells()(span.matches); ok {
			cells[i].Value = pointers.Ptr(v)
			vals = append(vals, v)
		}
	}
	mean := meanOf(vals)
	sd := sdOrOne(stdPop(vals, mean))
	for i := range cells {
		if cells[i].Value != nil {
			cells[i].Z = pointers.Ptr(zScore(d.better, *cells[i].Value-mean, sd))
		}
	}
	return cells, sd
}

// zScore : better x écart / sd ; better = 0 donne 0.
func zScore(better int, diff, sd float64) float64 {
	if better == 0 {
		return 0
	}
	return float64(better) * diff / sd
}

// horizonCells retourne les 4 cellules d'horizon (365, 90, 30, 7). La valeur
// existe dès 1 match ; la période d'avant et z exigent 10 matchs de part et
// d'autre. z = better x (valeur - valeur d'avant) / écart-type des mois.
func horizonCells(d indicator, f *frame, monthSD float64) []domain.TrendsHorizonCell {
	cells := make([]domain.TrendsHorizonCell, len(f.hor))
	for i, w := range f.hor {
		c := domain.TrendsHorizonCell{Days: w.days, Matches: len(w.cur), PrevMatches: len(w.prev)}
		eval := d.cells()
		cur, curOK := eval(w.cur)
		if curOK {
			c.Value = pointers.Ptr(cur)
		}
		if curOK && len(w.cur) >= minMatchesCompare && len(w.prev) >= minMatchesCompare {
			if prev, ok := eval(w.prev); ok {
				c.PrevValue = pointers.Ptr(prev)
				c.Z = pointers.Ptr(zScore(d.better, cur-prev, monthSD))
			}
		}
		cells[i] = c
	}
	return cells
}

// buildSeries retourne les courbes d'un indicateur : par match sur 30 jours
// (sauf indicateur d'ensemble), puis par jour, semaine et mois sur 365 jours.
func buildSeries(d indicator, f *frame) domain.TrendsSeries {
	s := domain.TrendsSeries{Match: []domain.TrendsPoint{}}
	if !d.noMatchSeries {
		for _, m := range f.last30 {
			if v, ok := d.eval([]Match{m}); ok {
				s.Match = append(s.Match, domain.TrendsPoint{T: m.Start, Value: v, Matches: 1})
			}
		}
	}
	s.Day = bucketPoints(d.eval, f.day, minPointDay)
	s.Week = bucketPoints(d.eval, f.week, minPointWeek)
	s.Month = bucketPoints(d.cells(), f.month, minPointMonth)
	return s
}

// bucketPoints évalue l'indicateur par intervalle ; un point est daté du début
// de son intervalle et exige min matchs.
func bucketPoints(eval evalFunc, buckets []temporal.Bucket[Match], minMatches int) []domain.TrendsPoint {
	pts := make([]domain.TrendsPoint, 0, len(buckets))
	for _, b := range buckets {
		if len(b.Items) < minMatches {
			continue
		}
		if v, ok := eval(b.Items); ok {
			pts = append(pts, domain.TrendsPoint{T: b.Start, Value: v, Matches: len(b.Items)})
		}
	}
	return pts
}
