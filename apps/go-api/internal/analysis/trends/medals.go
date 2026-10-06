package trends

import (
	"math"
	"sort"
	"strconv"

	"levelup/go-api/internal/domain"
)

// medalRowsMax : nombre de médailles listées par horizon.
const medalRowsMax = 10

// MedalCount est le nombre de médailles d'un type obtenues par le joueur dans un match.
type MedalCount struct {
	MatchID string
	MedalID int64
	Count   int
}

// medalRates donne, par médaille, le nombre moyen obtenu par match d'une fenêtre.
func medalRates(window []Match, byMatch map[string][]MedalCount) map[int64]float64 {
	sums := map[int64]int{}
	for i := range window {
		for _, c := range byMatch[window[i].ID] {
			sums[c.MedalID] += c.Count
		}
	}
	out := make(map[int64]float64, len(sums))
	for id, n := range sums {
		out[id] = float64(n) / float64(len(window))
	}
	return out
}

// buildMedals construit un bloc par horizon (365, 90, 30, 7 jours). Les taux
// sont des médailles par match de la fenêtre, sur les matchs du périmètre.
func buildMedals(f *frame, counts []MedalCount, names map[int64]string) []domain.TrendsMedalsBlock {
	byMatch := map[string][]MedalCount{}
	for _, c := range counts {
		byMatch[c.MatchID] = append(byMatch[c.MatchID], c)
	}
	out := make([]domain.TrendsMedalsBlock, 0, len(f.hor))
	for _, w := range f.hor {
		blk := domain.TrendsMedalsBlock{
			Days:     w.days,
			Compared: len(w.cur) >= minMatchesCompare && len(w.prev) >= minMatchesCompare,
			Rows:     []domain.TrendsMedalRow{},
		}
		if len(w.cur) > 0 {
			cur := medalRates(w.cur, byMatch)
			var prev map[int64]float64
			if blk.Compared {
				prev = medalRates(w.prev, byMatch)
			}
			blk.Rows = medalRows(cur, prev, blk.Compared, names)
		}
		out = append(out, blk)
	}
	return out
}

// medalRows classe les médailles d'un horizon : les plus fortes variations de
// taux face à la période d'avant quand elle est comparée (prev non nil), sinon
// les taux les plus élevés. Égalités départagées par identifiant croissant.
func medalRows(cur, prev map[int64]float64, compared bool, names map[int64]string) []domain.TrendsMedalRow {
	ids := make([]int64, 0, len(cur)+len(prev))
	for id := range cur {
		ids = append(ids, id)
	}
	if compared {
		for id := range prev {
			if _, ok := cur[id]; !ok {
				ids = append(ids, id)
			}
		}
	}
	score := func(id int64) float64 {
		if compared {
			return math.Abs(cur[id] - prev[id])
		}
		return cur[id]
	}
	sort.Slice(ids, func(i, j int) bool {
		si, sj := score(ids[i]), score(ids[j])
		if si != sj {
			return si > sj
		}
		return ids[i] < ids[j]
	})
	if len(ids) > medalRowsMax {
		ids = ids[:medalRowsMax]
	}
	rows := make([]domain.TrendsMedalRow, 0, len(ids))
	for _, id := range ids {
		row := domain.TrendsMedalRow{MedalID: id, Name: medalName(names, id), Rate: cur[id]}
		if compared {
			p := prev[id]
			row.PrevRate = &p
		}
		rows = append(rows, row)
	}
	return rows
}

// medalName retourne le nom de la médaille, à défaut son identifiant en chiffres.
func medalName(names map[int64]string, id int64) string {
	if n := names[id]; n != "" {
		return n
	}
	return strconv.FormatInt(id, 10)
}
