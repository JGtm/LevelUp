//go:build research

package main

// verdict_v2_diagnostic_research_test.go — POURQUOI UNE FORTE A ECHAPPE A LA V2, et la
// preuve que le rejeu est fidele.
//
// Le score et la selection v2 (`ReglageV2` tel que SERIALISE dans le fichier de positions
// juge) sont rejoues sur le CSV par cellule de la carte, exactement comme la passe de mesure
// les a joues : score sur disque, hysteresis amorce / croissance, fermeture morphologique,
// composantes 8-connexes, taille minimale. Pour chaque zone `forte` manquee, on regarde
// UNIQUEMENT les cellules dont le centre tombe dans la zone, et on nomme le PREMIER filtre
// qui l'a arretee — dans l'ordre ou les filtres s'appliquent.
//
// AUCUN SEUIL N'EST TOUCHE. Le rejeu est prouve fidele en comparant ses positions a celles
// du fichier juge (meme nombre, memes barycentres a 0,75 m pres) ; sans cette preuve, un
// diagnostic ne dirait rien de la passe reelle.
//
// Le diagnostic ne se rejoue que sur un fichier de reglage EMPIRIQUE v2 (hysteresis
// declaree) : la geometrie et la fusion ont leurs propres outils.

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/powerpos"
	"levelup/go-api/internal/analysis/tactical"
)

// toleranceRejeuM : ecart de barycentre tolere entre une position du fichier et son rejeu
// (le CSV ecrit trois decimales ; le pas de grille est de 0,5 m).
const toleranceRejeuM = 0.75

// diagnosticV2 est l'autopsie d'une zone `forte` manquee par la v2.
type diagnosticV2 struct {
	Carte, Zone, Raison string

	CellulesScorables int
	ScoreP50          float64
	ScoreMax          float64
	SeuilAmorce       float64
	SeuilCroissance   float64
	// AuDessusCroissance / AuDessusAmorce : cellules de la zone au-dessus de chaque seuil.
	AuDessusCroissance int
	AuDessusAmorce     int
	// RetenuesHysteresis : cellules de la zone retenues AVANT fermeture (reliees a une amorce).
	RetenuesHysteresis int
	// PlusGrandeCompo : la plus grande composante 8-connexe APRES fermeture qui touche la
	// zone ; CouvertureZone : la part de la zone que son enveloppe couvre.
	PlusGrandeCompo int
	CouvertureZone  float64

	Cause string
}

// fideliteRejeu compare les positions rejouees a celles du fichier juge.
type fideliteRejeu struct {
	Carte          string
	PositionsJSON  int
	PositionsRejeu int
	Retrouvees     int
}

// rejeuV2 est l'etat intermediaire d'une selection v2 rejouee.
type rejeuV2 struct {
	g          tactical.Grille
	r          powerpos.Reglage
	scorees    []powerpos.CelluleScoree
	amorce     float64
	croissance float64
	retenues   map[tactical.Cellule]bool
	comps      [][]tactical.Cellule
}

// diagnostiqueV2 autopsie les fortes manquees de chaque carte et mesure la fidelite du rejeu.
func diagnostiqueV2(t *testing.T, dossier string, src sourcesV2, doc SortiePositions,
	reels map[string]bilanV2) ([]diagnosticV2, []fideliteRejeu) {
	t.Helper()
	if doc.Reglage.QuantileAmorce <= 0 {
		t.Logf("diagnostic non rejoue : le reglage du fichier juge n'est pas l'empirique v2")
		return nil, nil
	}
	var diags []diagnosticV2
	var fids []fideliteRejeu
	for _, c := range doc.Cartes {
		b, ok := reels[c.Carte]
		if !ok || b.Role == roleHorsOracle {
			continue
		}
		manquees := zonesManquees(b.bilanCarte)
		cellules, err := litCSVCellules(filepath.Join(dossier, nomDeFichier(c.Carte, c.Axe)+".csv"))
		if err != nil {
			t.Logf("carte %s : %v", c.Carte, err)
			if len(manquees) > 0 {
				diags = append(diags, diagnosticV2{Carte: c.Carte, Zone: "(toutes)", Cause: err.Error()})
			}
			continue
		}
		rj := rejoue(cellules, doc.Reglage)
		fids = append(fids, fidelite(rj, c))
		index := indexeZones(zonesDeCarte(src.Catalogue, c))
		for _, nom := range manquees {
			diags = append(diags, autopsieV2(b, nom, index[strings.ToLower(nom)], rj))
		}
	}
	sort.SliceStable(diags, func(i, j int) bool { return diags[i].Carte < diags[j].Carte })
	sort.SliceStable(fids, func(i, j int) bool { return fids[i].Carte < fids[j].Carte })
	return diags, fids
}

// rejoue applique score et selection v2 a un CSV, en gardant les etats intermediaires.
func rejoue(cellules []powerpos.Cellule, r powerpos.Reglage) rejeuV2 {
	g := tactical.GrilleParDefaut()
	rj := rejeuV2{g: g, r: r, scorees: powerpos.Score(g, cellules, r), retenues: map[tactical.Cellule]bool{}}
	d := powerpos.Diagnostique(rj.scorees, r)
	rj.amorce, rj.croissance = d.SeuilAmorce, d.SeuilCroissance
	germes := map[tactical.Cellule]bool{}
	var candidates []tactical.Cellule
	for _, c := range rj.scorees {
		adr := tactical.Cellule{Col: c.Col, Lig: c.Lig}
		if c.Score >= rj.amorce {
			germes[adr] = true
		}
		if c.Score >= rj.croissance {
			candidates = append(candidates, adr)
		}
	}
	var avant []tactical.Cellule
	for _, comp := range powerpos.Composantes(candidates) {
		if !toucheUnGerme(comp, germes) {
			continue
		}
		for _, adr := range comp {
			rj.retenues[adr] = true
		}
		avant = append(avant, comp...)
	}
	apres := powerpos.Fermeture(avant, r.FermetureRayonCellules)
	rj.comps = powerpos.ComposantesConnexite(apres, r.Connexite8)
	return rj
}

// toucheUnGerme dit si une composante contient au moins une amorce.
func toucheUnGerme(comp []tactical.Cellule, germes map[tactical.Cellule]bool) bool {
	for _, c := range comp {
		if germes[c] {
			return true
		}
	}
	return false
}

// fidelite compare `Selectionne` rejoue sur le CSV aux positions du fichier juge.
func fidelite(rj rejeuV2, c SortieCartePos) fideliteRejeu {
	rejouees := powerpos.Selectionne(rj.g, rj.scorees, rj.r)
	f := fideliteRejeu{Carte: c.Carte, PositionsJSON: len(c.Positions), PositionsRejeu: len(rejouees)}
	for _, p := range c.Positions {
		for _, q := range rejouees {
			if math.Hypot(p.CentreX-q.CentreX, p.CentreY-q.CentreY) <= toleranceRejeuM {
				f.Retrouvees++
				break
			}
		}
	}
	return f
}

// autopsieV2 mesure ce que la selection v2 a fait des cellules d'une zone manquee.
func autopsieV2(b bilanV2, nom string, zone *zoneIndexee, rj rejeuV2) diagnosticV2 {
	d := diagnosticV2{Carte: b.Carte, Zone: nom, Raison: raisonDe(b, nom),
		SeuilAmorce: rj.amorce, SeuilCroissance: rj.croissance}
	if zone == nil {
		d.Cause = "zone sans geometrie au catalogue"
		return d
	}
	var scores []float64
	for _, c := range rj.scorees {
		if !zone.contient(c.CentreX, c.CentreY) {
			continue
		}
		d.CellulesScorables++
		scores = append(scores, c.Score)
		d.ScoreMax = math.Max(d.ScoreMax, c.Score)
		if c.Score >= rj.croissance {
			d.AuDessusCroissance++
		}
		if c.Score >= rj.amorce {
			d.AuDessusAmorce++
		}
		if rj.retenues[tactical.Cellule{Col: c.Col, Lig: c.Lig}] {
			d.RetenuesHysteresis++
		}
	}
	d.ScoreP50 = powerpos.Quantile(scores, 0.50)
	if comp := plusGrandeComposanteDans(rj, zone); len(comp) > 0 {
		d.PlusGrandeCompo = len(comp)
		d.CouvertureZone = partCouverte(zone.points, nouvelleSurface(powerpos.Enveloppe(rj.g, comp)))
	}
	d.Cause = causeV2(d, rj.r)
	return d
}

// plusGrandeComposanteDans rend la plus grande composante (apres fermeture) dont au moins
// une cellule a son centre dans la zone.
func plusGrandeComposanteDans(rj rejeuV2, zone *zoneIndexee) []tactical.Cellule {
	var meilleure []tactical.Cellule
	for _, comp := range rj.comps {
		if len(comp) <= len(meilleure) {
			continue
		}
		for _, adr := range comp {
			if x, y := rj.g.Centre(adr); zone.contient(x, y) {
				meilleure = comp
				break
			}
		}
	}
	return meilleure
}

// causeV2 nomme le premier filtre qui a arrete la zone, dans l'ordre de la selection.
func causeV2(d diagnosticV2, r powerpos.Reglage) string {
	switch {
	case d.CellulesScorables == 0:
		return fmt.Sprintf("NON SCORABLE : aucune cellule de la zone n'atteint %d matchs"+
			" distincts ET %d engagements dans son disque", r.PlancherMatchs, r.MinEngagementsDisque)
	case d.AuDessusCroissance == 0:
		return fmt.Sprintf("SCORE SOUS LA CROISSANCE : max %.3f < %.3f (p90 de la carte)",
			d.ScoreMax, d.SeuilCroissance)
	case d.RetenuesHysteresis == 0:
		return fmt.Sprintf("SANS AMORCE : %d cellule(s) passe(nt) la croissance (max %.3f) mais"+
			" aucune n'atteint l'amorce %.3f (p95) ni n'est reliee a une amorce",
			d.AuDessusCroissance, d.ScoreMax, d.SeuilAmorce)
	case d.PlusGrandeCompo < r.TailleMiniComposante:
		return fmt.Sprintf("COMPOSANTE TROP PETITE : %d cellules retenues, la plus grande"+
			" composante 8-connexe apres fermeture en fait %d (minimum %d)",
			d.RetenuesHysteresis, d.PlusGrandeCompo, r.TailleMiniComposante)
	default:
		return fmt.Sprintf("RETENUE AILLEURS : une composante de %d cellules touche la zone,"+
			" mais son enveloppe n'en couvre que %.0f %% (< %.0f %%) et son barycentre tombe"+
			" dehors", d.PlusGrandeCompo, d.CouvertureZone*100, seuilRecouvrement*100)
	}
}

// raisonDe rend la raison de l'oracle pour une zone attendue de la carte.
func raisonDe(b bilanV2, nom string) string {
	for _, z := range b.Attendues {
		if z.NomEN == nom {
			return z.Raison
		}
	}
	return ""
}
