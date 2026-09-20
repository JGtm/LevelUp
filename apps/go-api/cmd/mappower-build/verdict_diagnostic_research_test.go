//go:build research

package main

// verdict_diagnostic_research_test.go — POURQUOI UNE ZONE ATTENDUE A ECHAPPE (item 2.3).
//
// Le plan exige, en cas de NO-GO, un diagnostic precis : « quel signal manque, quelles
// positions echappent et pourquoi — score sous le seuil ? non scorable faute d'engagements ?
// composante trop petite ? ». Ce fichier repond a cette question pour CHAQUE zone `forte`
// manquee, en rejouant le score FIGE sur le CSV par cellule de l'etape 1 et en regardant
// UNIQUEMENT les cellules dont le centre tombe dans la zone.
//
// AUCUN SEUIL N'EST TOUCHE. Le reglage est `powerpos.ReglageV1()` tel quel, et le seuil de
// carte est recalcule exactement comme `powerpos.Selectionne` le fait : le plus exigeant du
// p90 des cellules scorables et du plancher absolu. Le diagnostic DIT pourquoi la zone n'est
// pas sortie ; il ne propose pas de l'y faire entrer.

import (
	"fmt"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/analysis/powerpos"
	"levelup/go-api/internal/analysis/tactical"
)

// diagnosticZone est l'autopsie d'une zone `forte` manquee.
type diagnosticZone struct {
	Carte, Zone string

	CellulesDansZone  int
	CellulesScorables int
	ScoreMax          float64
	ScoreP50          float64
	SeuilCarte        float64
	AuDessusDuSeuil   int
	PlusGrandeCompo   int

	Cause string
}

// diagnostique autopsie les zones `forte` manquees d'une carte.
func diagnostique(dossier string, e entreeCarte, b bilanCarte) []diagnosticZone {
	manquees := zonesManquees(b)
	if len(manquees) == 0 {
		return nil
	}
	cellules, err := litCSVCellules(filepath.Join(dossier,
		nomDeFichier(e.Carte.Carte, e.Carte.Axe)+".csv"))
	if err != nil {
		return []diagnosticZone{{Carte: b.Carte, Zone: "(toutes)", Cause: err.Error()}}
	}
	g := tactical.GrilleParDefaut()
	r := powerpos.ReglageV1()
	scorees := powerpos.Score(g, cellules, r)
	seuil := seuilCommeSelectionne(scorees, r)

	index := indexeZones(e.Zones)
	var out []diagnosticZone
	for _, nom := range manquees {
		out = append(out, autopsie(b.Carte, nom, index[strings.ToLower(nom)], scorees, seuil, r))
	}
	return out
}

// zonesManquees rend les libelles des zones `forte` resolues et non retrouvees.
func zonesManquees(b bilanCarte) []string {
	var out []string
	for _, z := range b.Attendues {
		if z.Confiance == confianceForte && z.Resolue && !z.Retrouvee {
			out = append(out, z.NomEN)
		}
	}
	return out
}

// autopsie mesure ce que le score a rendu dans l'emprise d'une zone.
func autopsie(carte, nom string, zone *zoneIndexee, scorees []powerpos.CelluleScoree,
	seuil float64, r powerpos.Reglage) diagnosticZone {
	d := diagnosticZone{Carte: carte, Zone: nom, SeuilCarte: seuil}
	if zone == nil {
		d.Cause = "zone sans geometrie au catalogue"
		return d
	}
	var scores []float64
	var retenues []tactical.Cellule
	for _, c := range scorees {
		if !zone.contient(c.CentreX, c.CentreY) {
			continue
		}
		d.CellulesScorables++
		scores = append(scores, c.Score)
		if c.Score > d.ScoreMax {
			d.ScoreMax = c.Score
		}
		if c.Score >= seuil {
			d.AuDessusDuSeuil++
			retenues = append(retenues, tactical.Cellule{Col: c.Col, Lig: c.Lig})
		}
	}
	d.CellulesDansZone = int(zone.aireM2 / (tactical.GrilleParDefaut().PasM() *
		tactical.GrilleParDefaut().PasM()))
	d.ScoreP50 = powerpos.Quantile(scores, 0.50)
	for _, comp := range powerpos.Composantes(retenues) {
		if len(comp) > d.PlusGrandeCompo {
			d.PlusGrandeCompo = len(comp)
		}
	}
	d.Cause = cause(d, r)
	return d
}

// cause nomme la raison de l'absence, dans l'ordre ou les filtres s'appliquent.
func cause(d diagnosticZone, r powerpos.Reglage) string {
	switch {
	case d.CellulesScorables == 0:
		return fmt.Sprintf("NON SCORABLE : aucune cellule de la zone n'atteint %d matchs"+
			" distincts ET %d engagements dans son disque", r.PlancherMatchs, r.MinEngagementsDisque)
	case d.AuDessusDuSeuil == 0:
		return fmt.Sprintf("SCORE SOUS LE SEUIL : max %.3f < %.3f (p90 de la carte)",
			d.ScoreMax, d.SeuilCarte)
	case d.PlusGrandeCompo < r.TailleMiniComposante:
		return fmt.Sprintf("COMPOSANTE TROP PETITE : %d cellules au-dessus du seuil, la plus"+
			" grande composante 4-connexe en fait %d (minimum %d)",
			d.AuDessusDuSeuil, d.PlusGrandeCompo, r.TailleMiniComposante)
	default:
		return fmt.Sprintf("RETENUE AILLEURS : %d cellules passent et forment une composante"+
			" de %d, mais la position construite ne couvre pas %.0f %% de la zone et son"+
			" barycentre tombe dehors", d.AuDessusDuSeuil, d.PlusGrandeCompo,
			seuilRecouvrement*100)
	}
}

// seuilCommeSelectionne reproduit le seuil de `powerpos.Selectionne` (le plus exigeant du
// quantile de carte et du plancher absolu). Le paquet le garde prive ; le recopier ici est
// la 2e et derniere copie tolerable — une 3e imposerait de l'exporter.
func seuilCommeSelectionne(scorees []powerpos.CelluleScoree, r powerpos.Reglage) float64 {
	scores := make([]float64, 0, len(scorees))
	for _, c := range scorees {
		scores = append(scores, c.Score)
	}
	if q := powerpos.Quantile(scores, r.QuantileSeuil); q > r.SeuilScoreMin {
		return q
	}
	return r.SeuilScoreMin
}

// tableDiagnostic imprime l'autopsie des zones manquees.
func tableDiagnostic(b *strings.Builder, diags []diagnosticZone) {
	fmt.Fprintf(b, "## 6. Diagnostic — pourquoi chaque zone `forte` manquee a echappe\n\n")
	if len(diags) == 0 {
		fmt.Fprintln(b, "Aucune zone `forte` manquee.")
		fmt.Fprintln(b)
		return
	}
	fmt.Fprintf(b, "Le score FIGE est rejoue sur le CSV par cellule de l'etape 1, et seules"+
		" les cellules dont le CENTRE tombe dans la zone sont regardees. « Cellules zone » est"+
		" l'emprise de la zone en cellules de 0,5 m ; « scorables » compte celles qui passent"+
		" le plancher de matchs ET les %d engagements de leur disque.\n\n",
		powerpos.ReglageV1().MinEngagementsDisque)
	fmt.Fprintln(b, "| Carte | Zone manquee | Cellules zone | Scorables | Score p50 | Score max |"+
		" Seuil carte | Au-dessus | Plus grande composante | Cause |")
	fmt.Fprintln(b, "|---|---|---|---|---|---|---|---|---|---|")
	for _, d := range diags {
		fmt.Fprintf(b, "| %s | %s | %d | %d | %.3f | %.3f | %.3f | %d | %d | %s |\n",
			d.Carte, d.Zone, d.CellulesDansZone, d.CellulesScorables, d.ScoreP50,
			d.ScoreMax, d.SeuilCarte, d.AuDessusDuSeuil, d.PlusGrandeCompo, d.Cause)
	}
	fmt.Fprintln(b)
}
