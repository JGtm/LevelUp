//go:build research

package main

// verdict_v2_rapport_tables_research_test.go — LA SUITE DU DOCUMENT DE VERDICT V2 : les
// couloirs, le diagnostic des fortes manquees avec la preuve de fidelite du rejeu, le
// progres v1 -> v2 a regles egales, la section « geometrie et fusion » (vide tant que la
// geometrie n'est pas jugee) et la synthese des causes.

import (
	"fmt"
	"sort"
	"strings"
)

// tableCouloirs : les positions de plus de `seuilCouloirCellules` cellules, jugees.
func tableCouloirs(b *strings.Builder, ordre []string, reels map[string]bilanV2) {
	fmt.Fprintf(b, "## 6. Couloirs — les positions de plus de %d cellules : position ou salle ?\n\n", seuilCouloirCellules)
	fmt.Fprintf(b, "Le document de mesure v2 signale deux positions de 85 et 88 cellules retenues d'un"+
		" bloc par la croissance au p90, et laisse au verdict de dire si c'est une position ou une"+
		" salle. Lecture mecanique : SALLE ENTIERE si la zone dominante est couverte a plus de la"+
		" moitie par la position ; COULOIR TRAVERSANT si au moins deux zones portent chacune un"+
		" quart ou plus de la position (elle enjambe une frontiere) ; POSITION LARGE sinon (une"+
		" partie d'une seule zone, ce que D1 appelle une position).\n\n")
	fmt.Fprintln(b, "| Carte | Position | Cellules | Aire m2 | Zones traversees (part de la position / part de la zone) | Lecture |")
	fmt.Fprintln(b, "|---|---|---|---|---|---|")
	n := 0
	for _, nom := range ordre {
		for _, k := range reels[nom].Couloirs {
			n++
			fmt.Fprintf(b, "| %s | `%s` | %d | %.1f | %s | %s |\n", nom, k.ID, k.NbCellules, k.AireM2,
				strings.Join(k.Zones, " ; "), k.Lecture)
		}
	}
	if n == 0 {
		fmt.Fprintln(b, "| — | — | — | — | aucune position au-dessus du seuil | — |")
	}
	fmt.Fprintln(b)
}

// tableDiagnosticV2 imprime la fidelite du rejeu puis l'autopsie des fortes manquees.
func tableDiagnosticV2(b *strings.Builder, v verdictV2Rendu) {
	fmt.Fprintf(b, "## 7. Diagnostic — pourquoi chaque forte manquee a echappe a la v2\n\n")
	if v.Doc.Reglage.QuantileAmorce <= 0 {
		fmt.Fprintln(b, "Diagnostic non rejoue : le fichier juge ne porte pas le reglage empirique v2"+
			" (pas d'hysteresis declaree). La geometrie et la fusion ont leurs propres outils.")
		fmt.Fprintln(b)
		return
	}
	fmt.Fprintf(b, "**Fidelite du rejeu.** Le score et la selection v2 sont rejoues sur le CSV par"+
		" cellule de chaque carte avec le reglage serialise dans le fichier juge, et les positions"+
		" rejouees sont comparees a celles du fichier (barycentres a %.2f m pres). Sans cette"+
		" preuve, le diagnostic ne dirait rien de la passe reelle.\n\n", toleranceRejeuM)
	fmt.Fprintln(b, "| Carte | Positions au fichier | Positions au rejeu | Retrouvees | Fidele |")
	fmt.Fprintln(b, "|---|---|---|---|---|")
	for _, f := range v.Fidelites {
		fidele := "oui"
		if f.PositionsJSON != f.PositionsRejeu || f.Retrouvees != f.PositionsJSON {
			fidele = "NON"
		}
		fmt.Fprintf(b, "| %s | %d | %d | %d | %s |\n", f.Carte, f.PositionsJSON, f.PositionsRejeu, f.Retrouvees, fidele)
	}
	fmt.Fprintln(b)
	if len(v.Diagnostics) == 0 {
		fmt.Fprintln(b, "Aucune zone `forte` manquee.")
		fmt.Fprintln(b)
		return
	}
	fmt.Fprintf(b, "Seules les cellules dont le CENTRE tombe dans la zone sont regardees. « Scorables »"+
		" compte celles qui passent le plancher de matchs ET les %d engagements du disque ;"+
		" « >= croissance » et « >= amorce » les cellules au-dessus de chaque seuil de l'hysteresis ;"+
		" « retenues » celles reliees a une amorce avant fermeture ; « composante » la plus grande"+
		" composante 8-connexe apres fermeture qui touche la zone, et la part de la zone que son"+
		" enveloppe couvre.\n\n", v.Doc.Reglage.MinEngagementsDisque)
	fmt.Fprintln(b, "| Carte | Zone manquee | Raison (oracle) | Scorables | Score p50 | Score max | Croissance (p90) |"+
		" Amorce (p95) | >= croissance | >= amorce | Retenues | Composante | Couverture zone | Cause |")
	fmt.Fprintln(b, "|---|---|---|---|---|---|---|---|---|---|---|---|---|---|")
	for _, d := range v.Diagnostics {
		fmt.Fprintf(b, "| %s | %s | %s | %d | %.3f | %.3f | %.3f | %.3f | %d | %d | %d | %d | %.0f %% | %s |\n",
			d.Carte, d.Zone, d.Raison, d.CellulesScorables, d.ScoreP50, d.ScoreMax, d.SeuilCroissance,
			d.SeuilAmorce, d.AuDessusCroissance, d.AuDessusAmorce, d.RetenuesHysteresis,
			d.PlusGrandeCompo, d.CouvertureZone*100, d.Cause)
	}
	fmt.Fprintln(b)
}

// unionTriee rend l'union de deux listes, triee.
func unionTriee(a, b []string) []string {
	vus := map[string]bool{}
	var out []string
	for _, l := range [][]string{a, b} {
		for _, x := range l {
			if !vus[x] {
				vus[x] = true
				out = append(out, x)
			}
		}
	}
	sort.Strings(out)
	return out
}

// syntheseV2 compte les causes d'echec, comptees sur les diagnostics, jamais saisies.
func syntheseV2(b *strings.Builder, diags []diagnosticV2) {
	fmt.Fprintf(b, "## 10. Ce que ces chiffres disent\n\n")
	if len(diags) == 0 {
		fmt.Fprintln(b, "Aucune zone `forte` manquee : rien a expliquer.")
		return
	}
	causes := map[string]int{}
	for _, d := range diags {
		causes[strings.SplitN(d.Cause, " :", 2)[0]]++
	}
	var cles []string
	for c := range causes {
		cles = append(cles, c)
	}
	sort.Strings(cles)
	fmt.Fprintf(b, "Sur **%d zones `forte` manquees** (calibrage et validation confondus), la repartition des causes est :\n\n", len(diags))
	for _, c := range cles {
		fmt.Fprintf(b, "- **%s** : %d\n", c, causes[c])
	}
	fmt.Fprintf(b, "\nChaque cause nomme le PREMIER filtre de la selection v2 qui a arrete la zone."+
		" Rien n'est retouche ici : toute retouche de seuil apres le verdict l'invalide (protocole"+
		" du plan, D12). Ce que ces causes disent a la fusion (2bis.D, seconde moitie) est ecrit"+
		" dans le rapport de l'agent, pas dans ce document genere.\n")
}
