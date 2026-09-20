package main

// rapport.go — LE TABLEAU DE MESURE : ce qu'il faut pour CHOISIR une formule de score, et
// rien de plus.
//
// TROIS FAMILLES DE CHIFFRES, ET POURQUOI CELLES-LA :
//
//  1. LE RAYON DU NUAGE EN FONCTION DU PLANCHER. C'est la mesure qui a fixe le plancher de
//     la grille tactique (cmd/mappos-build, 2026-08-30 : 268 m sans filtre, 27 m a deux
//     matchs, 19,4 m a trois). On la rejoue ici sur les cellules d'ELIMINATION, qui ne sont
//     pas les cellules de passage : rien ne dit que le meme plancher convienne.
//  2. LA TAILLE D'ECHANTILLON PAR CELLULE. Le rapport de duel d'une cellule a trois
//     engagements ne vaut rien ; savoir combien de cellules en ont six, douze ou vingt-cinq
//     dit si la mesure a de quoi trancher, et calibre le retrecissement.
//  3. LA DISPERSION DES SIGNAUX. Sans elle, tout seuil est une devinette.
//
// Il n'y a AUCUN score dans ce rapport : le score se choisit APRES, sur ces chiffres.

import (
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"strings"
	"time"

	"levelup/go-api/internal/analysis/powerpos"
)

// planchersEtudies : les valeurs de plancher (en matchs distincts par cellule) dont on
// mesure l'effet sur le rayon du nuage.
var planchersEtudies = []int{1, 2, 3, 5}

// minEngagementsRapport est la taille d'echantillon a partir de laquelle le rapport de duel
// d'une cellule entre dans les quantiles. Sous ce seuil la valeur ne prend que quelques
// niveaux discrets (0, 1/3, 1/2...) et la dispersion mesuree serait celle du comptage, pas
// celle du terrain.
const minEngagementsRapport = 6

// titreSection rend un titre de section Markdown suivi d'une ligne vide.
func titreSection(titre string) string { return "## " + titre + "\n\n" }

// EcrisRapport ecrit le tableau de mesure de toutes les cartes.
func EcrisRapport(chemin string, cibles []*Cible, r powerpos.Reglage, nomReglage string) error {
	f, err := os.Create(chemin)
	if err != nil {
		return fmt.Errorf("creation du rapport (%s) : %w", chemin, err)
	}
	defer func() {
		if errFerme := f.Close(); errFerme != nil {
			slog.Error("mappower: fermeture du rapport", "err", errFerme, "path", chemin)
		}
	}()
	var b strings.Builder
	fmt.Fprintf(&b, "<!-- mappower-build --mesure --reglage %s, %s -->\n\n", nomReglage,
		time.Now().UTC().Format("2006-01-02"))
	ecrisSectionCorpus(&b, cibles)
	ecrisSectionRangs(&b, cibles)
	ecrisSectionRayon(&b, cibles)
	ecrisSectionEchantillon(&b, cibles)
	ecrisSectionSignaux(&b, cibles)
	ecrisSectionAngles(&b, cibles)
	ecrisSectionOccupation(&b, cibles)
	ecrisSectionAxes(&b, cibles)
	ecrisSectionScore(&b, cibles, r, nomReglage)
	ecrisSectionSelection(&b, cibles, r)
	ecrisSectionPositions(&b, cibles)
	if _, err := io.WriteString(f, b.String()); err != nil {
		return fmt.Errorf("ecriture du rapport (%s) : %w", chemin, err)
	}
	return nil
}

// ecrisSectionCorpus : ce que chaque carte apporte.
func ecrisSectionCorpus(b *strings.Builder, cibles []*Cible) {
	fmt.Fprint(b, titreSection("Corpus mesure"))
	fmt.Fprintln(b, "| Carte | Axe | Matchs | Kills | Cellules peuplees | Artefacts | Points de piste | Pistes sans issue | Ecart variantes (m) |")
	fmt.Fprintln(b, "|---|---|---|---|---|---|---|---|---|")
	for _, c := range cibles {
		fmt.Fprintf(b, "| %s | %s | %d | %d | %d | %d | %d | %d | %.2f |\n",
			c.Carte, c.Axe, c.Acc.NbMatchs(), c.KillsLus, c.Acc.NbCellules(),
			c.ArtefactsLus, c.PointsLus, c.PistesSansIssue, c.EcartVariantesM)
	}
	fmt.Fprintln(b)
}

// ecrisSectionRayon rejoue la mesure de plancher de cmd/mappos-build sur les cellules
// d'elimination : rayon du nuage (p99 de la distance au barycentre) par plancher.
func ecrisSectionRayon(b *strings.Builder, cibles []*Cible) {
	fmt.Fprint(b, titreSection("Plancher de rarete — rayon du nuage (p99 de la distance au barycentre, en m)"))
	fmt.Fprint(b, "| Carte |")
	for _, p := range planchersEtudies {
		fmt.Fprintf(b, " >=%d match(s) |", p)
	}
	fmt.Fprint(b, "\n|---|")
	for range planchersEtudies {
		fmt.Fprint(b, "---|")
	}
	fmt.Fprintln(b)
	for _, c := range cibles {
		cellules := c.Acc.Cellules()
		fmt.Fprintf(b, "| %s |", c.Carte)
		for _, p := range planchersEtudies {
			rayon, n := rayonNuage(cellules, p)
			fmt.Fprintf(b, " %.1f (%d) |", rayon, n)
		}
		fmt.Fprintln(b)
	}
	fmt.Fprint(b, "\n_Entre parentheses : le nombre de cellules retenues par ce plancher._\n\n")
}

// rayonNuage rend le p99 de la distance au barycentre des cellules passant le plancher, et
// leur nombre.
func rayonNuage(cellules []powerpos.Cellule, plancher int) (float64, int) {
	var xs, ys []float64
	for _, c := range cellules {
		if c.MatchsKills >= plancher {
			xs = append(xs, c.CentreX)
			ys = append(ys, c.CentreY)
		}
	}
	if len(xs) == 0 {
		return 0, 0
	}
	var sx, sy float64
	for i := range xs {
		sx, sy = sx+xs[i], sy+ys[i]
	}
	bx, by := sx/float64(len(xs)), sy/float64(len(ys))
	distances := make([]float64, len(xs))
	for i := range xs {
		distances[i] = math.Hypot(xs[i]-bx, ys[i]-by)
	}
	return powerpos.Quantile(distances, 0.99), len(distances)
}

// ecrisSectionEchantillon : combien d'engagements par cellule, et donc quelle confiance.
func ecrisSectionEchantillon(b *strings.Builder, cibles []*Cible) {
	fmt.Fprint(b, titreSection("Taille d'echantillon par cellule (engagements = kills depuis + morts dedans)"))
	fmt.Fprintln(b, "| Carte | p50 | p75 | p90 | p99 | max | cellules >=6 | cellules >=12 | cellules >=25 |")
	fmt.Fprintln(b, "|---|---|---|---|---|---|---|---|---|")
	for _, c := range cibles {
		cellules := c.Acc.Cellules()
		n := make([]float64, 0, len(cellules))
		maxi := 0.0
		for _, cel := range cellules {
			v := float64(cel.TotalEngagements())
			n = append(n, v)
			maxi = math.Max(maxi, v)
		}
		fmt.Fprintf(b, "| %s | %.0f | %.0f | %.0f | %.0f | %.0f | %d | %d | %d |\n", c.Carte,
			powerpos.Quantile(n, 0.5), powerpos.Quantile(n, 0.75), powerpos.Quantile(n, 0.9),
			powerpos.Quantile(n, 0.99), maxi,
			compteAuMoins(cellules, 6), compteAuMoins(cellules, 12), compteAuMoins(cellules, 25))
	}
	fmt.Fprintln(b)
}

// compteAuMoins compte les cellules a au moins `seuil` engagements.
func compteAuMoins(cellules []powerpos.Cellule, seuil int) int {
	k := 0
	for _, cel := range cellules {
		if cel.TotalEngagements() >= seuil {
			k++
		}
	}
	return k
}

// ecrisSectionSignaux : dispersion du rapport de duel, de la portee et du denivele.
func ecrisSectionSignaux(b *strings.Builder, cibles []*Cible) {
	fmt.Fprint(b, titreSection(fmt.Sprintf(
		"Dispersion des signaux (cellules a >=%d engagements)", minEngagementsRapport)))
	fmt.Fprintln(b, "| Carte | n | duel p10 | p25 | p50 | p75 | p90 | portee p50 (m) | p90 | denivele p10 (m) | p50 | p90 |")
	fmt.Fprintln(b, "|---|---|---|---|---|---|---|---|---|---|---|---|")
	for _, c := range cibles {
		var duels, portees, denivelees []float64
		for _, cel := range c.Acc.Cellules() {
			if cel.TotalEngagements() < minEngagementsRapport {
				continue
			}
			duels = append(duels, float64(cel.KillsDepuis)/float64(cel.TotalEngagements()))
			if cel.KillsDepuis > 0 {
				portees = append(portees, cel.PorteeMedianeM)
				denivelees = append(denivelees, cel.DeniveleMedianM)
			}
		}
		fmt.Fprintf(b, "| %s | %d | %.2f | %.2f | %.2f | %.2f | %.2f | %.1f | %.1f | %.1f | %.1f | %.1f |\n",
			c.Carte, len(duels),
			powerpos.Quantile(duels, 0.10), powerpos.Quantile(duels, 0.25),
			powerpos.Quantile(duels, 0.50), powerpos.Quantile(duels, 0.75),
			powerpos.Quantile(duels, 0.90),
			powerpos.Quantile(portees, 0.50), powerpos.Quantile(portees, 0.90),
			powerpos.Quantile(denivelees, 0.10), powerpos.Quantile(denivelees, 0.50),
			powerpos.Quantile(denivelees, 0.90))
	}
	fmt.Fprintln(b)
}

// ecrisSectionOccupation : ce que la presence par equipe apporte, et sur quel corpus.
func ecrisSectionOccupation(b *strings.Builder, cibles []*Cible) {
	fmt.Fprint(b, titreSection("Occupation par equipe (artefacts de rejeu)"))
	fmt.Fprintln(b, "| Carte | Artefacts | Cellules avec presence | Cellules a >=3 matchs | ecart p10 | p50 | p90 |")
	fmt.Fprintln(b, "|---|---|---|---|---|---|---|")
	for _, c := range cibles {
		var ecarts []float64
		avec, plancher := 0, 0
		for _, cel := range c.Acc.Cellules() {
			total := cel.TotalOccupationMS()
			if total <= 0 {
				continue
			}
			avec++
			if cel.MatchsPresence < 3 {
				continue
			}
			plancher++
			ecarts = append(ecarts, (cel.OccupationGagnantsMS-cel.OccupationPerdantsMS)/total)
		}
		fmt.Fprintf(b, "| %s | %d | %d | %d | %.2f | %.2f | %.2f |\n", c.Carte, c.ArtefactsLus,
			avec, plancher, powerpos.Quantile(ecarts, 0.10), powerpos.Quantile(ecarts, 0.50),
			powerpos.Quantile(ecarts, 0.90))
	}
	fmt.Fprintln(b)
}

// ecrisSectionScore : ce que le score FIGE rend sur chaque carte.
func ecrisSectionScore(b *strings.Builder, cibles []*Cible, r powerpos.Reglage, nom string) {
	fmt.Fprint(b, titreSection(fmt.Sprintf("Score (reglage %s) — cellules scorables", nom)))
	fmt.Fprintln(b, "| Carte | Cellules peuplees | Scorables | part | score p50 | p75 | p90 | p99 | max | seuil applique |")
	fmt.Fprintln(b, "|---|---|---|---|---|---|---|---|---|---|")
	for _, c := range cibles {
		scores := make([]float64, 0, len(c.Scorees))
		for _, s := range c.Scorees {
			scores = append(scores, s.Score)
		}
		part := 0.0
		if n := c.Acc.NbCellules(); n > 0 {
			part = 100 * float64(len(c.Scorees)) / float64(n)
		}
		seuil := powerpos.Diagnostique(c.Scorees, r).SeuilAmorce
		fmt.Fprintf(b, "| %s | %d | %d | %.0f%% | %.3f | %.3f | %.3f | %.3f | %.3f | %.3f |\n",
			c.Carte, c.Acc.NbCellules(), len(c.Scorees), part,
			powerpos.Quantile(scores, 0.50), powerpos.Quantile(scores, 0.75),
			powerpos.Quantile(scores, 0.90), powerpos.Quantile(scores, 0.99),
			powerpos.Quantile(scores, 1.0), seuil)
	}
	fmt.Fprintln(b)
}

// ecrisSectionPositions : les positions retenues, carte par carte.
func ecrisSectionPositions(b *strings.Builder, cibles []*Cible) {
	fmt.Fprint(b, titreSection("Positions retenues"))
	fmt.Fprintln(b, "| Carte | Positions | Cellules (min/med/max) | Aire m2 (min/max) | Score moyen (min/max) | Centres (x, y) |")
	fmt.Fprintln(b, "|---|---|---|---|---|---|")
	for _, c := range cibles {
		if len(c.Positions) == 0 {
			fmt.Fprintf(b, "| %s | 0 | — | — | — | — |\n", c.Carte)
			continue
		}
		var tailles, aires, scores []float64
		var centres []string
		for _, p := range c.Positions {
			tailles = append(tailles, float64(len(p.Cellules)))
			aires = append(aires, p.AireM2)
			scores = append(scores, p.ScoreMoyen)
			centres = append(centres, fmt.Sprintf("(%.0f, %.0f)", p.CentreX, p.CentreY))
		}
		fmt.Fprintf(b, "| %s | %d | %.0f / %.0f / %.0f | %.1f / %.1f | %.3f / %.3f | %s |\n",
			c.Carte, len(c.Positions),
			powerpos.Quantile(tailles, 0), powerpos.Quantile(tailles, 0.5), powerpos.Quantile(tailles, 1),
			powerpos.Quantile(aires, 0), powerpos.Quantile(aires, 1),
			powerpos.Quantile(scores, 0), powerpos.Quantile(scores, 1),
			strings.Join(centres, " "))
	}
	fmt.Fprintln(b)
}
