//go:build research

package grammar

// mouvement_5_3_4_tableaux_research_test.go — LES TABLEAUX DE LA MESURE FINALE (lot 5.3.4).
//
// Scission par DEPLACEMENT PUR d avec `mouvement_5_3_4_etats_research_test.go` : le ratchet de
// taille (500 lignes) tient les deux moities separees, et aucune ligne de calcul n a change.
// Ici vivent les reductions — cadences, intervalles, instants, quantiles, histogrammes — et
// elles ne connaissent pas les etats : elles ne voient que des lectures datees et un predicat.

import (
	"fmt"
	"sort"
	"strings"
)

// m534Cadences rend la cadence des huit slots les plus actifs, en records par seconde.
func m534Cadences(parSlot map[uint32][]m534Ev, duree float64) string {
	if duree <= 0 || len(parSlot) == 0 {
		return "(duree nulle ou aucune lecture)"
	}
	type p struct {
		slot uint32
		n    int
	}
	ps := make([]p, 0, len(parSlot))
	for s, g := range parSlot {
		ps = append(ps, p{s, len(g)})
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].n > ps[j].n })
	var out []string
	for i, x := range ps {
		if i == 8 {
			break
		}
		out = append(out, fmt.Sprintf("slot %d %.3f rec/s (%d)", x.slot,
			float64(x.n)/duree, x.n))
	}
	return strings.Join(out, " · ")
}

// m534Intervalles rend les quantiles des ecarts entre deux lectures RETENUES du meme slot, en
// secondes. C est la mesure d une transition a la suivante, par vie.
func m534Intervalles(parSlot map[uint32][]m534Ev, garde func(m534Ev) bool) string {
	var ecarts []float64
	var retenues int
	for _, g := range parSlot {
		var prev uint64
		for _, e := range g {
			if !garde(e) {
				continue
			}
			retenues++
			if prev != 0 && e.ts > prev {
				ecarts = append(ecarts, float64(e.ts-prev)/1e6)
			}
			prev = e.ts
		}
	}
	if len(ecarts) == 0 {
		return fmt.Sprintf("%d lecture(s) retenue(s), aucun intervalle", retenues)
	}
	return fmt.Sprintf("%d retenues, %d intervalles · %s", retenues, len(ecarts),
		m534Quantiles(ecarts))
}

// m534Instants rend CINQ instants en temps de barre Theater, un par episode : les lectures
// retenues sont parcourues dans l ordre du temps, et deux instants du meme episode (moins de
// deux secondes d ecart sur le meme slot) ne comptent qu une fois.
func m534Instants(parSlot map[uint32][]m534Ev, tmin uint64, garde func(m534Ev) bool) string {
	type inst struct {
		slot uint32
		ts   uint64
	}
	var all []inst
	for s, g := range parSlot {
		var prev uint64
		for _, e := range g {
			if !garde(e) {
				continue
			}
			if prev != 0 && e.ts < prev+2_000_000 {
				continue
			}
			prev = e.ts
			all = append(all, inst{s, e.ts})
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ts < all[j].ts })
	var out []string
	for _, x := range all {
		if len(out) == 5 {
			break
		}
		out = append(out, fmt.Sprintf("%s (slot %d)", m532Barre(x.ts, tmin), x.slot))
	}
	if len(out) == 0 {
		return "(aucun instant retenu)"
	}
	return strings.Join(out, " · ")
}

// m534Quantiles rend mediane, p90 et max d une serie.
func m534Quantiles(xs []float64) string {
	if len(xs) == 0 {
		return "(aucune valeur)"
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	return fmt.Sprintf("mediane %.2f · p90 %.2f · max %.2f",
		s[len(s)/2], s[(len(s)*9)/10], s[len(s)-1])
}

// m534Histo rend un histogramme en classes de 1 unite, plafonne a douze classes. C est le
// TEMOIN DU SPRINT : si le sprint se voyait a la vitesse, la distribution aurait deux maxima
// locaux separes.
func m534Histo(xs []float64) string {
	if len(xs) == 0 {
		return "(aucune valeur)"
	}
	classes := map[int]int{}
	maxC := 0
	for _, x := range xs {
		c := int(x)
		if c > 11 {
			c = 11
		}
		classes[c]++
		if c > maxC {
			maxC = c
		}
	}
	var out []string
	for c := 0; c <= maxC; c++ {
		out = append(out, fmt.Sprintf("%d-%d:%d", c, c+1, classes[c]))
	}
	return strings.Join(out, " ")
}
