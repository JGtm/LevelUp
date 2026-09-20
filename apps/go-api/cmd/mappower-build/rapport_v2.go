package main

// rapport_v2.go — LES SECTIONS DU RAPPORT AJOUTEES PAR LA V2 (item 2bis.B, 2026-09-20) :
// la couverture du rang, la dispersion des signaux angulaires, et ce que la selection a
// fait des cellules retenues (tailles des composantes avant et apres fermeture).
//
// Ces sections existent parce que le verdict v1 n'a pu etre diagnostique qu'en rejouant
// le score sur les CSV : la passe ne disait ni combien de cellules passaient le seuil, ni
// la taille des amas qu'elles formaient. Le reglage v2 se choisit sur ces comptes.

import (
	"fmt"
	"math"
	"strings"

	"levelup/go-api/internal/analysis/powerpos"
)

// ecrisSectionRangs : quelle part des kills a un tueur au rang connu, carte par carte.
func ecrisSectionRangs(b *strings.Builder, cibles []*Cible) {
	fmt.Fprint(b, titreSection("Couverture du rang du tueur (match_csrs_latest, base partagee)"))
	if len(cibles) > 0 && cibles[0].Rangs != nil {
		fmt.Fprintf(b, "Corpus : %s.\n\n", cibles[0].Rangs.Resume())
	}
	fmt.Fprintln(b, "| Carte | Kills lus | Kills exclus (variante) | Kills a rang connu | part | Matchs | Matchs avec >= 1 rang | part |")
	fmt.Fprintln(b, "|---|---|---|---|---|---|---|---|")
	for _, c := range cibles {
		matchsAvecRang := 0
		for id := range c.Matchs {
			if c.Rangs != nil && c.Rangs.MatchConnu(id) {
				matchsAvecRang++
			}
		}
		fmt.Fprintf(b, "| %s | %d | %d | %d | %s | %d | %d | %s |\n", c.Carte, c.KillsLus, c.KillsExclus,
			c.KillsRangConnu, partEnPourcent(c.KillsRangConnu, c.KillsLus), len(c.Matchs), matchsAvecRang,
			partEnPourcent(matchsAvecRang, len(c.Matchs)))
	}
	fmt.Fprintln(b)
}

// pourcent formate une part, ou « — » quand le denominateur est nul.
func partEnPourcent(num, den int) string {
	if den == 0 {
		return "—"
	}
	return fmt.Sprintf("%.0f%%", 100*float64(num)/float64(den))
}

// ecrisSectionAngles : la dispersion des directions par DISQUE scorable — nombre de
// directions, couverture (dispersion sortante) et abri (1 - dispersion entrante) — et les
// correlations entre axes, pour savoir s'ils disent la meme chose.
func ecrisSectionAngles(b *strings.Builder, cibles []*Cible) {
	fmt.Fprint(b, titreSection("Signaux angulaires par disque scorable (dispersion corrigee du biais, cf. powerpos/angles.go)"))
	fmt.Fprintln(b, "| Carte | Disques | N sortant p10 / p50 / p90 | N entrant p10 / p50 / p90 | couverture p10 / p50 / p90 | abri p10 / p50 / p90 | asymetrie (couv + abri) / 2 p10 / p50 / p90 | corr(couv, abri) | corr(couv, avantage) | corr(abri, avantage) | corr(asym, avantage) | corr(asym, hauteur) |")
	fmt.Fprintln(b, "|---|---|---|---|---|---|---|---|---|---|---|")
	for _, c := range cibles {
		var nS, nE, couv, abri, asym, avantage, hauteur []float64
		for _, s := range c.Scorees {
			nS = append(nS, float64(s.DirSortanteDisque.N))
			nE = append(nE, float64(s.DirEntranteDisque.N))
			if s.DirSortanteDisque.N < 2 || s.DirEntranteDisque.N < 2 {
				continue
			}
			couv = append(couv, s.DirSortanteDisque.Dispersion())
			abri = append(abri, 1-s.DirEntranteDisque.Dispersion())
			asym = append(asym, (couv[len(couv)-1]+abri[len(abri)-1])/2)
			avantage = append(avantage, s.Avantage)
			hauteur = append(hauteur, s.Hauteur)
		}
		fmt.Fprintf(b, "| %s | %d | %s | %s | %s | %s | %s | %.2f | %.2f | %.2f | %.2f | %.2f |\n",
			c.Carte, len(c.Scorees), trio(nS, "%.0f"), trio(nE, "%.0f"), trio(couv, "%.2f"),
			trio(abri, "%.2f"), trio(asym, "%.2f"), pearson(couv, abri), pearson(couv, avantage),
			pearson(abri, avantage), pearson(asym, avantage), pearson(asym, hauteur))
	}
	fmt.Fprintln(b)
}

// trio formate p10 / p50 / p90 d'une serie.
func trio(serie []float64, format string) string {
	return fmt.Sprintf(format+" / "+format+" / "+format,
		powerpos.Quantile(serie, 0.10), powerpos.Quantile(serie, 0.50), powerpos.Quantile(serie, 0.90))
}

// pearson rend le coefficient de correlation lineaire de deux series de meme longueur, ou
// 0 si l'une est constante ou vide.
func pearson(x, y []float64) float64 {
	n := len(x)
	if n < 2 || len(y) != n {
		return 0
	}
	var mx, my float64
	for i := range x {
		mx += x[i]
		my += y[i]
	}
	mx /= float64(n)
	my /= float64(n)
	var sxy, sxx, syy float64
	for i := range x {
		dx, dy := x[i]-mx, y[i]-my
		sxy += dx * dy
		sxx += dx * dx
		syy += dy * dy
	}
	if sxx <= 0 || syy <= 0 {
		return 0
	}
	return sxy / math.Sqrt(sxx*syy)
}

// ecrisSectionSelection : ce que la selection a fait, chiffre par chiffre.
func ecrisSectionSelection(b *strings.Builder, cibles []*Cible, r powerpos.Reglage) {
	fmt.Fprint(b, titreSection("Selection — cellules retenues et composantes (avant / apres fermeture)"))
	fmt.Fprintln(b, "| Carte | Seuil amorce | Seuil croissance | Cellules amorce | Retenues | Apres fermeture | Composantes avant (tailles, 8 plus grandes) | Composantes apres (tailles) | Retenues (>= taille min) |")
	fmt.Fprintln(b, "|---|---|---|---|---|---|---|---|---|")
	for _, c := range cibles {
		d := powerpos.Diagnostique(c.Scorees, r)
		fmt.Fprintf(b, "| %s | %.3f | %.3f | %d | %d | %d | %d : %s | %d : %s | %d |\n", c.Carte,
			d.SeuilAmorce, d.SeuilCroissance, d.CellulesAmorce, d.CellulesRetenues, d.CellulesApresFermeture,
			len(d.TaillesAvantFermeture), tetes(d.TaillesAvantFermeture, 8),
			len(d.TaillesComposantes), tetes(d.TaillesComposantes, 8), d.ComposantesRetenues)
	}
	fmt.Fprintln(b)
}

// tetes formate les n premieres valeurs d'une liste d'entiers.
func tetes(vs []int, n int) string {
	if len(vs) == 0 {
		return "—"
	}
	if len(vs) > n {
		vs = vs[:n]
	}
	parts := make([]string, 0, len(vs))
	for _, v := range vs {
		parts = append(parts, fmt.Sprint(v))
	}
	return strings.Join(parts, " ")
}

// ecrisSectionAxes : la dispersion de CHAQUE axe du score par disque scorable. C'est sur
// l'etalement (p10-p90) de chaque axe que se choisit son poids : un axe plat ne departage
// rien quel que soit son poids, un axe etale domine le score au moindre poids.
func ecrisSectionAxes(b *strings.Builder, cibles []*Cible) {
	fmt.Fprint(b, titreSection("Axes du score par disque scorable (p10 / p50 / p90, etalement p90 - p10)"))
	fmt.Fprintln(b, "| Carte | avantage | intensite | hauteur | portee | couverture | abri | score |")
	fmt.Fprintln(b, "|---|---|---|---|---|---|---|---|")
	for _, c := range cibles {
		var av, in, ha, po, co, ab, sc []float64
		for _, s := range c.Scorees {
			av = append(av, s.Avantage)
			in = append(in, s.Intensite)
			ha = append(ha, s.Hauteur)
			po = append(po, s.Portee)
			co = append(co, s.Couverture)
			ab = append(ab, s.Abri)
			sc = append(sc, s.Score)
		}
		fmt.Fprintf(b, "| %s | %s | %s | %s | %s | %s | %s | %s |\n", c.Carte,
			etalement(av), etalement(in), etalement(ha), etalement(po), etalement(co), etalement(ab), etalement(sc))
	}
	fmt.Fprintln(b)
}

// etalement formate p10 / p50 / p90 et l'ecart p90 - p10 entre crochets.
func etalement(serie []float64) string {
	p10, p90 := powerpos.Quantile(serie, 0.10), powerpos.Quantile(serie, 0.90)
	return fmt.Sprintf("%.2f / %.2f / %.2f [%.2f]", p10, powerpos.Quantile(serie, 0.50), p90, p90-p10)
}
