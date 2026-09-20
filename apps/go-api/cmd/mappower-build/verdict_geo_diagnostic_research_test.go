//go:build research

package main

// verdict_geo_diagnostic_research_test.go — POURQUOI UNE FORTE A ECHAPPE A LA GEOMETRIE
// SEULE (item 2bis.D, seconde moitie, 2026-09-20).
//
// Le score geometrique est relu sur le CSV par noeud (`cmd/mapgeo-build`, colonne `score`,
// reglage `geo.ReglageGeoV1`), et pour chaque zone `forte` manquee on regarde les noeuds
// dont la cellule tombe dans la zone. Les causes, dans l'ordre ou la selection
// geometrique filtre :
//
//	SOL ABSENT          aucun noeud du sol derive dans la zone (le sol est DERIVE du rendu ;
//	                    ses trous sont la limite 1 de GEOMETRIE_2026-09-20.md §5) ;
//	SOUS LE SEUIL       aucun noeud de la zone n'atteint le p90 du score de la carte ;
//	RETENUE AILLEURS    une position geometrique touche la zone sans la couvrir a 30 % ni
//	                    y poser son barycentre ;
//	SANS MAXIMUM LOCAL  des noeuds passent le seuil, mais aucun ne domine sa boule
//	                    euclidienne 3D de 3 m — la selection reelle domine sur 3 m de MARCHE
//	                    (graphe absent du CSV) ; la boule euclidienne contient tout ce que la
//	                    marche atteint, donc ce test est PLUS STRICT : un maximum de marche
//	                    peut subsister, et la cause suivante s'applique alors ;
//	HORS PLAFOND / TROP PETITE  un maximum existe mais aucune position ne touche la zone :
//	                    composante < 12 noeuds (croissance bornee a 4 m) ou au-dela des 8
//	                    positions de la carte — indecidable sans le graphe ; le score du
//	                    meilleur noeud est compare a la 8e position publiee.
//
// AUCUN REGLAGE N'EST TOUCHE : ce fichier explique, il ne corrige pas.

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/powerpos"
	"levelup/go-api/internal/analysis/powerpos/fusion"
	"levelup/go-api/internal/analysis/tactical"
)

// rayonMaximumLocalDiagM : le rayon de dominance de `geo.ReglageGeoV1` (3 m).
const rayonMaximumLocalDiagM = 3.0

// diagnosticGeo est l'autopsie d'une zone `forte` manquee par la geometrie.
type diagnosticGeo struct {
	Carte, Zone, Raison string

	CellulesZone      int
	CellulesAvecNoeud int
	NoeudsDansZone    int
	ScoreP50          float64
	ScoreMax          float64
	Seuil             float64
	AuDessus          int
	MaximaLocaux      int
	// CouverturePosition : la plus grande part de la zone couverte par une position.
	CouverturePosition float64
	// ScoreDernierePosition : le score moyen de la position la moins bien classee publiee.
	ScoreDernierePosition float64

	Cause string
}

// diagnostiqueGeo autopsie les fortes manquees de chaque carte du fichier geometrique.
func diagnostiqueGeo(t *testing.T, dossier string, src sourcesV2, doc SortiePositions,
	reels map[string]bilanV2) []diagnosticGeo {
	t.Helper()
	g := tactical.GrilleParDefaut()
	var out []diagnosticGeo
	for _, c := range doc.Cartes {
		b, ok := reels[c.Carte]
		if !ok || b.Role == roleHorsOracle {
			continue
		}
		manquees := zonesManquees(b.bilanCarte)
		if len(manquees) == 0 {
			continue
		}
		noeuds, _, err := litCSVNoeudsGeo(filepath.Join(dossier, nomGeo(c.Carte)+".csv"), g)
		if err != nil {
			t.Logf("carte %s : %v", c.Carte, err)
			out = append(out, diagnosticGeo{Carte: c.Carte, Zone: "(toutes)", Cause: err.Error()})
			continue
		}
		seuil := seuilGeo(noeuds)
		index := indexeZones(zonesDeCarte(src.Catalogue, c))
		for _, nom := range manquees {
			out = append(out, autopsieGeo(g, b, nom, index[strings.ToLower(nom)], noeuds, seuil))
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Carte < out[j].Carte })
	return out
}

// seuilGeo rend le p90 du score des noeuds (le seuil de `geo.ReglageGeoV1`).
func seuilGeo(noeuds []fusion.NoeudGeo) float64 {
	scores := make([]float64, len(noeuds))
	for i, n := range noeuds {
		scores[i] = n.Score
	}
	return powerpos.Quantile(scores, 0.90)
}

// autopsieGeo mesure ce que la geometrie a rendu dans l'emprise d'une zone manquee.
func autopsieGeo(g tactical.Grille, b bilanV2, nom string, zone *zoneIndexee,
	noeuds []fusion.NoeudGeo, seuil float64) diagnosticGeo {
	d := diagnosticGeo{Carte: b.Carte, Zone: nom, Raison: raisonDe(b, nom), Seuil: seuil}
	if zone == nil {
		d.Cause = "zone sans geometrie au catalogue"
		return d
	}
	pas := g.PasM()
	d.CellulesZone = int(zone.aireM2 / (pas * pas))
	var dedans []int
	cellules := map[tactical.Cellule]bool{}
	var scores []float64
	for i, n := range noeuds {
		x, y := g.Centre(tactical.Cellule{Col: n.Col, Lig: n.Lig})
		if !zone.contient(x, y) {
			continue
		}
		dedans = append(dedans, i)
		cellules[tactical.Cellule{Col: n.Col, Lig: n.Lig}] = true
		scores = append(scores, n.Score)
		d.ScoreMax = math.Max(d.ScoreMax, n.Score)
		if n.Score >= seuil {
			d.AuDessus++
			if domineBoule(g, noeuds, i, rayonMaximumLocalDiagM) {
				d.MaximaLocaux++
			}
		}
	}
	d.NoeudsDansZone, d.CellulesAvecNoeud = len(dedans), len(cellules)
	d.ScoreP50 = powerpos.Quantile(scores, 0.50)
	for _, p := range b.Positions {
		d.CouverturePosition = math.Max(d.CouverturePosition, partCouverte(zone.points, p.forme))
		if d.ScoreDernierePosition == 0 || p.ScoreMoyen < d.ScoreDernierePosition {
			d.ScoreDernierePosition = p.ScoreMoyen
		}
	}
	d.Cause = causeGeo(d, len(b.Positions))
	return d
}

// domineBoule dit si le noeud i domine (au sens large) tous les noeuds a moins de `rayon`
// metres en distance euclidienne 3D.
func domineBoule(g tactical.Grille, noeuds []fusion.NoeudGeo, i int, rayon float64) bool {
	xi, yi := g.Centre(tactical.Cellule{Col: noeuds[i].Col, Lig: noeuds[i].Lig})
	for j, n := range noeuds {
		if j == i || n.Score <= noeuds[i].Score {
			continue
		}
		if math.Abs(float64(n.Col-noeuds[i].Col))*g.PasM() > rayon ||
			math.Abs(float64(n.Lig-noeuds[i].Lig))*g.PasM() > rayon {
			continue
		}
		xj, yj := g.Centre(tactical.Cellule{Col: n.Col, Lig: n.Lig})
		dz := n.Z - noeuds[i].Z
		if math.Sqrt((xj-xi)*(xj-xi)+(yj-yi)*(yj-yi)+dz*dz) <= rayon {
			return false
		}
	}
	return true
}

// causeGeo nomme le premier filtre qui a arrete la zone.
func causeGeo(d diagnosticGeo, nbPositions int) string {
	switch {
	case d.NoeudsDansZone == 0:
		return fmt.Sprintf("SOL ABSENT : aucun noeud du sol derive dans la zone (%d cellules)", d.CellulesZone)
	case d.AuDessus == 0:
		return fmt.Sprintf("SOUS LE SEUIL : max %.3f < %.3f (p90 de la carte)", d.ScoreMax, d.Seuil)
	case d.CouverturePosition > 0:
		return fmt.Sprintf("RETENUE AILLEURS : une position touche la zone mais n'en couvre que"+
			" %.0f %% (< %.0f %%) et son barycentre tombe dehors", d.CouverturePosition*100, seuilRecouvrement*100)
	case d.MaximaLocaux == 0:
		return fmt.Sprintf("SANS MAXIMUM LOCAL : %d noeud(s) passe(nt) le seuil (max %.3f) mais aucun"+
			" ne domine sa boule de %.0f m (test euclidien, plus strict que la marche)",
			d.AuDessus, d.ScoreMax, rayonMaximumLocalDiagM)
	case nbPositions >= 8 && d.ScoreMax < d.ScoreDernierePosition:
		return fmt.Sprintf("HORS PLAFOND : %d maximum(s) local(aux) mais le meilleur noeud (%.3f) est sous"+
			" la 8e position publiee (%.3f)", d.MaximaLocaux, d.ScoreMax, d.ScoreDernierePosition)
	default:
		return fmt.Sprintf("TROP PETITE OU HORS PLAFOND : %d maximum(s) local(aux) (max %.3f), aucune"+
			" position ne touche la zone — composante < 12 noeuds a 4 m de marche, ou plafond de"+
			" 8 (indecidable sans le graphe)", d.MaximaLocaux, d.ScoreMax)
	}
}

// tableDiagnosticGeo imprime l'autopsie geometrique.
func tableDiagnosticGeo(b *strings.Builder, diags []diagnosticGeo) {
	fmt.Fprintf(b, "**Diagnostic geometrique.** Le score de `geo.ReglageGeoV1` est relu sur le CSV par"+
		" noeud ; seuls les noeuds dont la cellule tombe dans la zone sont regardes. « Cellules zone »"+
		" est l'emprise de la zone en cellules de 0,5 m, « avec noeud » celles que le sol derive"+
		" couvre ; « maxima » compte les noeuds au-dessus du seuil qui dominent leur boule euclidienne"+
		" 3D de %.0f m (test plus strict que la dominance a %.0f m de marche de la selection reelle).\n\n",
		rayonMaximumLocalDiagM, rayonMaximumLocalDiagM)
	if len(diags) == 0 {
		fmt.Fprintln(b, "Aucune zone `forte` manquee.")
		fmt.Fprintln(b)
		return
	}
	fmt.Fprintln(b, "| Carte | Zone manquee | Raison (oracle) | Cellules zone | Avec noeud | Noeuds | Score p50 |"+
		" Score max | Seuil (p90) | Au-dessus | Maxima | Couverture par une position | Cause |")
	fmt.Fprintln(b, "|---|---|---|---|---|---|---|---|---|---|---|---|---|")
	for _, d := range diags {
		fmt.Fprintf(b, "| %s | %s | %s | %d | %d | %d | %.3f | %.3f | %.3f | %d | %d | %.0f %% | %s |\n",
			d.Carte, d.Zone, d.Raison, d.CellulesZone, d.CellulesAvecNoeud, d.NoeudsDansZone,
			d.ScoreP50, d.ScoreMax, d.Seuil, d.AuDessus, d.MaximaLocaux, d.CouverturePosition*100, d.Cause)
	}
	fmt.Fprintln(b)
}
