//go:build research

package grammar

// mouvement_5_3_5_oracle_research_test.go — L ORACLE DU DEPLACEMENT, LA DISTRIBUTION, LES
// IMPULSIONS (lot 5.3.5). Scission par deplacement pur d avec
// `mouvement_5_3_5_vitesse_research_test.go` (ratchet de taille).

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"testing"
)

// m535Oracle confronte la vitesse DECODEE au DEPLACEMENT de la meme vie.
//
// LA METHODE, ET CE QU ELLE NE SUPPOSE PAS. Pour chaque paire de positions successives d un
// MEME slot, on calcule le deplacement par seconde ; on le divise par la vitesse decodee la plus
// proche en temps DANS l intervalle. Si le decodage est juste, ce rapport est le FACTEUR
// D UNITE — constant —, et sa dispersion (p90/p10) est proche de 1. S il est du bruit, elle
// s etale sur des ordres de grandeur. Aucune unite n est supposee : c est la DISPERSION qui
// tranche.
func m535Oracle(t *testing.T, rec *m535Rec) {
	t.Helper()
	posParSlot := map[uint32][]m535Pos{}
	for _, p := range rec.pos {
		posParSlot[p.slot] = append(posParSlot[p.slot], p)
	}
	vitParSlot := map[uint32][]m535Vit{}
	for _, v := range rec.vit {
		vitParSlot[v.slot] = append(vitParSlot[v.slot], v)
	}
	for s := range posParSlot {
		g := posParSlot[s]
		sort.Slice(g, func(i, j int) bool { return g[i].ts < g[j].ts })
		posParSlot[s] = g
	}
	for s := range vitParSlot {
		g := vitParSlot[s]
		sort.Slice(g, func(i, j int) bool { return g[i].ts < g[j].ts })
		vitParSlot[s] = g
	}
	for _, kind := range []PosKind{PosKindRaw, PosKindAbsolute, PosKindAbsFallback} {
		m535OracleParNature(t, posParSlot, vitParSlot, kind)
	}
}

// m535OracleParNature joue l oracle sur UNE nature d echantillon de position : les trois
// natures ABSOLUES sont les seules comparables entre elles (un delta n est pas une coordonnee,
// et le brut de 96 bits n est pas dans la meme boite que le quantifie — l en-tete de
// `position_capture.go` le dit).
func m535OracleParNature(t *testing.T, pos map[uint32][]m535Pos, vit map[uint32][]m535Vit,
	kind PosKind) {
	t.Helper()
	var rapports, deplacements []float64
	var paires int
	for slot, g := range pos {
		var prev *m535Pos
		for i := range g {
			if g[i].kind != kind {
				continue
			}
			cur := g[i]
			if prev != nil && cur.ts > prev.ts {
				dt := float64(cur.ts-prev.ts) / 1e6
				d := math.Sqrt(m535Carre(cur.vec[0]-prev.vec[0]) +
					m535Carre(cur.vec[1]-prev.vec[1]) + m535Carre(cur.vec[2]-prev.vec[2]))
				if dt > 0 && dt < 1.0 && d > 0 {
					paires++
					vitesse := d / dt
					deplacements = append(deplacements, vitesse)
					if dec, ok := m535VitesseDans(vit[slot], prev.ts, cur.ts); ok && dec > 0 {
						rapports = append(rapports, vitesse/dec)
					}
				}
			}
			p := cur
			prev = &p
		}
	}
	if paires == 0 {
		t.Logf("ORACLE [%s] : aucune paire de positions successives", kind)
		return
	}
	t.Logf("ORACLE [%s] : %d paires · deplacement/s %s", kind, paires,
		m535Quant(deplacements))
	if len(rapports) < 20 {
		t.Logf("  rapport deplacement/vitesse : %d paires appariees — trop peu pour trancher",
			len(rapports))
		return
	}
	sort.Float64s(rapports)
	p10 := rapports[len(rapports)/10]
	p50 := rapports[len(rapports)/2]
	p90 := rapports[(len(rapports)*9)/10]
	etal := math.Inf(1)
	if p10 > 0 {
		etal = p90 / p10
	}
	t.Logf("  RAPPORT deplacement/vitesse decodee : %d paires · p10 %.3f · mediane %.3f · "+
		"p90 %.3f · **dispersion p90/p10 = %.1f**", len(rapports), p10, p50, p90, etal)
	switch {
	case etal < 3:
		t.Logf("  VERDICT : dispersion ETROITE — la vitesse decodee SUIT le deplacement, et le "+
			"facteur d unite vaut %.3f", p50)
	default:
		t.Logf("  VERDICT : dispersion LARGE (%.1f ordres de grandeur sur 80 %% de la "+
			"population) — la vitesse decodee NE SUIT PAS le deplacement", etal)
	}
}

// m535VitesseDans rend la vitesse decodee dont l horodatage tombe dans [t0, t1].
func m535VitesseDans(g []m535Vit, t0, t1 uint64) (float64, bool) {
	for _, v := range g {
		if v.ts >= t0 && v.ts <= t1 {
			return math.Sqrt(m535Carre(v.v[0]) + m535Carre(v.v[1]) + m535Carre(v.v[2])), true
		}
	}
	return 0, false
}

// m535Distribution publie l histogramme de la vitesse AU SOL, et cherche les DEUX BOSSES.
func m535Distribution(t *testing.T, rec *m535Rec) {
	t.Helper()
	if len(rec.vit) == 0 {
		t.Logf("DISTRIBUTION : aucune vitesse dequantifiee")
		return
	}
	var sol []float64
	for _, v := range rec.vit {
		sol = append(sol, v.sol)
	}
	t.Logf("VITESSE AU SOL : %d lectures · %s", len(sol), m535Quant(sol))
	classes := make([]int, 26) // 0..24 m/s par pas de 1, la derniere = >= 25
	for _, x := range sol {
		c := int(x)
		if c > 25 {
			c = 25
		}
		classes[c]++
	}
	var parts []string
	for i, n := range classes {
		if i == 25 {
			parts = append(parts, fmt.Sprintf(">=25:%d", n))
			continue
		}
		parts = append(parts, fmt.Sprintf("%d:%d", i, n))
	}
	t.Logf("  HISTOGRAMME (pas de 1 m/s) : %s", strings.Join(parts, " "))
	t.Logf("  BOSSES (maxima locaux sur 0..24, au moins 2 %% de la population) : %s",
		m535Bosses(classes[:25], len(sol)))
}

// m535Bosses nomme les maxima locaux significatifs. Une marche a ~5,5 m/s et un sprint ~20 %
// au-dessus donneraient DEUX bosses separees ; une seule bosse, ou aucune, ne prouve rien.
func m535Bosses(classes []int, total int) string {
	seuil := total / 50
	var out []string
	for i := 1; i < len(classes)-1; i++ {
		if classes[i] > classes[i-1] && classes[i] >= classes[i+1] && classes[i] >= seuil {
			out = append(out, fmt.Sprintf("%d-%d m/s (%d, %.1f %%)", i, i+1, classes[i],
				100*float64(classes[i])/float64(total)))
		}
	}
	if len(out) == 0 {
		return "aucun maximum local au-dessus du seuil"
	}
	return strings.Join(out, " · ")
}

// m535Impulsions cherche le SAUT : une composante verticale qui devient positive puis negative
// sur la MEME vie. C est la forme d une impulsion, et sa duree mediane la signe.
func m535Impulsions(t *testing.T, rec *m535Rec) {
	t.Helper()
	parSlot := map[uint32][]m535Vit{}
	for _, v := range rec.vit {
		parSlot[v.slot] = append(parSlot[v.slot], v)
	}
	var durees, pics, dureesFortes []float64
	var impulsions, slots int
	for _, g := range parSlot {
		sort.Slice(g, func(i, j int) bool { return g[i].ts < g[j].ts })
		var debut uint64
		var pic float64
		var vu bool
		var ici int
		for _, v := range g {
			switch {
			case v.vz > 0:
				if !vu {
					debut, pic, vu = v.ts, v.vz, true
				} else if v.vz > pic {
					pic = v.vz
				}
			case v.vz < 0 && vu:
				if v.ts > debut {
					d := float64(v.ts-debut) / 1e6
					durees = append(durees, d)
					pics = append(pics, pic)
					if pic >= m535PicSaut {
						dureesFortes = append(dureesFortes, d)
					}
					impulsions++
					ici++
				}
				vu = false
			}
		}
		if ici > 0 {
			slots++
		}
	}
	t.Logf("IMPULSIONS VERTICALES (vz > 0 puis vz < 0 sur la meme vie) : %d sur %d slots · "+
		"duree %s", impulsions, slots, m535Quant(durees))
	t.Logf("  PIC DE MONTEE par impulsion (m/s) : %s", m535Quant(pics))
	t.Logf("  DUREE des impulsions a pic >= %.1f m/s : %d impulsions · %s",
		m535PicSaut, len(dureesFortes), m535Quant(dureesFortes))
}

// m535PicSaut : le pic de composante verticale au-dela duquel une impulsion est un CANDIDAT
// SAUT plutot qu une chute ou un relief. 3 m/s — un Spartan quitte le sol autour de 5 a 6 m/s,
// une descente de pente reste sous 2. SEUIL D INSTRUMENT, pas une grammaire.
const m535PicSaut = 3.0

// m535Carre rend le carre d un float32 en double.
func m535Carre(f float32) float64 { return float64(f) * float64(f) }

// m535Quant rend mediane, p10, p90 et max d une serie.
func m535Quant(xs []float64) string {
	if len(xs) == 0 {
		return "(aucune valeur)"
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	return fmt.Sprintf("p10 %.3f · mediane %.3f · p90 %.3f · max %.3f",
		s[len(s)/10], s[len(s)/2], s[(len(s)*9)/10], s[len(s)-1])
}
