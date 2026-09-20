package main

// rapport.go — LE TABLEAU DE CALIBRAGE : couts, comptes, distributions et correlations par
// carte, tel qu'il se colle dans GEOMETRIE_2026-09-20.md. Les poids se choisissent en le
// lisant, jamais en regardant l'oracle.

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis/powerpos/geo"
)

// nomsVariables : l'ordre des colonnes de distribution.
var nomsVariables = []string{"H", "V", "E", "R", "M"}

// EcrisRapport ecrit le rapport Markdown.
func EcrisRapport(chemin string, p geo.Parametres, r geo.Reglage, cuites []*Cuite) error {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Mesure geometrique — %s\n\n", time.Now().UTC().Format("2006-01-02"))
	fmt.Fprintf(&sb, "Parametres : `%+v`\n\nReglage : `%+v`\n\n", p, r)
	ecrisCouts(&sb, cuites)
	for _, c := range cuites {
		ecrisCarte(&sb, c)
	}
	return os.WriteFile(chemin, []byte(sb.String()), 0o644)
}

// ecrisCouts ecrit le tableau des couts et des comptes.
func ecrisCouts(sb *strings.Builder, cuites []*Cuite) {
	fmt.Fprint(sb, "## Couts et comptes\n\n")
	fmt.Fprintln(sb, "| carte | module | cellules | triangles | voxels occupes | candidats sol | noeuds | composantes (tailles) | sans retour | arcs coupes | germes places | cibles | rayons | voxelisation | sol | visibilite | couvert | total | mem sys Mo | frontiere | positions |")
	fmt.Fprintln(sb, "|---|---|---:|---:|---:|---:|---:|---|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---:|")
	for _, c := range cuites {
		b := c.Bilan
		fmt.Fprintf(sb, "| `%s` | `%s` | %d | %d | %d | %d | %d | %d (%s) | %d | %d | %d/%d | %d | %d | %s | %s | %s | %s | %s | %.0f | %v | %d |\n",
			c.Cible.Carte, c.Cible.Module, c.Resultat.Graphe.Cadre().NbCellules(), c.Triangles.Triangles,
			b.VoxelsOccupes, b.Sol.Candidats, len(c.Resultat.Noeuds), b.Sol.Composantes, tailles(b.Sol.TaillesComposantes),
			b.Sol.SansRetour, b.Sol.ArcsCoupes, b.Sol.AncresPlacees, b.Sol.AncresPlacees+b.Sol.AncresSansNoeud,
			b.Cibles, b.Rayons, duree(b.Durees["voxelisation"]), duree(b.Durees["sol"]), duree(b.Durees["visibilite"]),
			duree(b.Durees["couvert"]), duree(c.DureeTotale), c.MemoireSysMo, c.FrontiereAppliquee, len(c.Positions))
	}
	fmt.Fprintln(sb)
}

// tailles ecrit les tailles de composantes, les six premieres.
func tailles(t []int) string {
	if len(t) > 6 {
		t = t[:6]
	}
	parts := make([]string, len(t))
	for i, v := range t {
		parts[i] = fmt.Sprint(v)
	}
	return strings.Join(parts, ", ")
}

func duree(d time.Duration) string { return fmt.Sprintf("%.1f s", d.Seconds()) }

// ecrisCarte ecrit les distributions et les correlations d'une carte, puis ses positions.
func ecrisCarte(sb *strings.Builder, c *Cuite) {
	fmt.Fprintf(sb, "## %s (`%s`)\n\n", c.Cible.Carte, c.Cible.Module)
	brut, norm := seriesDe(c.Resultat.Noeuds)
	fmt.Fprint(sb, "Distributions BRUTES par noeud (H en m, V et E en fraction, R et M en proximite) :\n\n")
	fmt.Fprintln(sb, "| variable | p5 | p25 | p50 | p75 | p95 | moyenne |")
	fmt.Fprintln(sb, "|---|---:|---:|---:|---:|---:|---:|")
	for i, nom := range nomsVariables {
		q := quantiles(brut[i], 0.05, 0.25, 0.5, 0.75, 0.95)
		fmt.Fprintf(sb, "| %s | %.3f | %.3f | %.3f | %.3f | %.3f | %.3f |\n", nom, q[0], q[1], q[2], q[3], q[4], moyenne(brut[i]))
	}
	fmt.Fprint(sb, "\nCorrelations (Pearson) des variables NORMALISEES :\n\n")
	fmt.Fprintln(sb, "| | H | V | E | R | M |")
	fmt.Fprintln(sb, "|---|---:|---:|---:|---:|---:|")
	for i, nom := range nomsVariables {
		fmt.Fprintf(sb, "| %s |", nom)
		for j := range nomsVariables {
			fmt.Fprintf(sb, " %.2f |", geo.Correlation(norm[i], norm[j]))
		}
		fmt.Fprintln(sb)
	}
	scores := make([]float64, len(c.Resultat.Noeuds))
	for i, n := range c.Resultat.Noeuds {
		scores[i] = n.Score
	}
	q := quantiles(scores, 0.5, 0.9, 0.99)
	fmt.Fprintf(sb, "\nScore : p50 %.3f, p90 %.3f (seuil), p99 %.3f. Distances de deplacement : %s.\n\n",
		q[0], q[1], q[2], distancesDe(c.Resultat.Noeuds))
	fmt.Fprintln(sb, "| rang | centre | z | noeuds | cellules | aire m2 | score moyen | H | V | E | R | M |")
	fmt.Fprintln(sb, "|---:|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|")
	for i, p := range c.Positions {
		fmt.Fprintf(sb, "| %d | (%.1f, %.1f) | %.1f..%.1f | %d | %d | %.1f | %.3f | %.2f | %.2f | %.2f | %.2f | %.2f |\n",
			i+1, p.CentreX, p.CentreY, p.ZMin, p.ZMax, len(p.Noeuds), len(p.Cellules), p.AireM2, p.ScoreMoyen,
			p.Norm.H, p.Norm.V, p.Norm.E, p.Norm.R, p.Norm.M)
	}
	fmt.Fprintln(sb)
}

// seriesDe rend les cinq series brutes et les cinq normalisees.
func seriesDe(noeuds []geo.NoeudMesure) (brut, norm [5][]float64) {
	for _, n := range noeuds {
		b, m := n.Brut, n.Norm
		for i, v := range []float64{b.H, b.V, b.E, b.R, b.M} {
			brut[i] = append(brut[i], v)
		}
		for i, v := range []float64{m.H, m.V, m.E, m.R, m.M} {
			norm[i] = append(norm[i], v)
		}
	}
	return brut, norm
}

// distancesDe resume les distances de deplacement (medianes et part infinie).
func distancesDe(noeuds []geo.NoeudMesure) string {
	var arme, obj, couv []float64
	sansArme, sansObj := 0, 0
	for _, n := range noeuds {
		couv = append(couv, n.DCouvert)
		if math.IsInf(n.DArmeForte, 1) {
			sansArme++
		} else {
			arme = append(arme, n.DArmeForte)
		}
		if math.IsInf(n.DObjectif, 1) {
			sansObj++
		} else {
			obj = append(obj, n.DObjectif)
		}
	}
	return fmt.Sprintf("arme forte p50 %.1f m (p95 %.1f, %d noeuds sans chemin), objectif p50 %.1f m (p95 %.1f, %d sans chemin), couvert p50 %.1f m (p95 %.1f)",
		quantiles(arme, 0.5)[0], quantiles(arme, 0.95)[0], sansArme, quantiles(obj, 0.5)[0], quantiles(obj, 0.95)[0], sansObj,
		quantiles(couv, 0.5)[0], quantiles(couv, 0.95)[0])
}

// quantiles rend les quantiles demandes d'une serie (NaN si vide).
func quantiles(vals []float64, qs ...float64) []float64 {
	out := make([]float64, len(qs))
	if len(vals) == 0 {
		for i := range out {
			out[i] = math.NaN()
		}
		return out
	}
	tries := append([]float64(nil), vals...)
	sort.Float64s(tries)
	for i, q := range qs {
		pos := q * float64(len(tries)-1)
		k := int(pos)
		if k+1 >= len(tries) {
			out[i] = tries[len(tries)-1]
			continue
		}
		f := pos - float64(k)
		out[i] = tries[k]*(1-f) + tries[k+1]*f
	}
	return out
}

func moyenne(vals []float64) float64 {
	if len(vals) == 0 {
		return math.NaN()
	}
	s := 0.0
	for _, v := range vals {
		s += v
	}
	return s / float64(len(vals))
}
