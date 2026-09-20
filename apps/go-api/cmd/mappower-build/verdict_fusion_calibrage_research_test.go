//go:build research

package main

// verdict_fusion_calibrage_research_test.go — LE CHOIX DU REGLAGE DE FUSION, SUR LES TROIS
// CARTES DE CALIBRAGE SEULEMENT (item 2bis.D, seconde moitie, 2026-09-20 ; plan D11).
//
// C'est le seul endroit du chantier ou l'oracle sert a CHOISIR : une petite grille de
// (poids, valeurs d'absence, quantiles d'hysteresis) est balayee sur Recharge, Aquarius et
// Streets, chaque candidat est juge avec les regles du verdict v2 (memes fonctions, memes
// seuils), et le gagnant est celui qui maximise le RAPPEL moyen, puis la PRECISION moyenne,
// puis le moins de pieges purs, puis — a egalite — le plus proche de l'equilibre (a = 0,5)
// et le plus conservateur (croissance la plus haute). L'ordre est total : deux executions
// rendent le meme gagnant.
//
// La grille entiere et ses scores sont ECRITS (`fusion_2026-09-20/_calibrage_fusion.md`) :
// on doit voir que le choix n'est pas fait sur les cartes de validation, et ce que les
// voisins du gagnant valent. Le reglage fige (`fusion.ReglageFusionV1`) doit etre le
// gagnant : le test le verifie, et echoue tant que le gel n'est pas fait.
//
// INVOCATION (depuis apps/go-api) :
//
//	MAPPOWER_DATA_ROOT=<racine contenant data/> \
//	  go test -tags research ./cmd/mappower-build/ -run FusionCalibrage -v

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/powerpos"
	"levelup/go-api/internal/analysis/powerpos/fusion"
	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// Dossiers et fichiers de la fusion.
const (
	verdictGeometrie = "geometrie_2026-09-20"
	verdictFusion    = "fusion_2026-09-20"
	calibrageNom     = "_calibrage_fusion.md"
)

// La grille balayee. Petite et ecrite d'avance : chaque axe a une raison.
var (
	// poidsGeoBalayes : de « empirique seul » (0) a « geometrie seule » (1), pour lire ce
	// que la fusion apporte a chacune des deux voies sous la MEME selection.
	poidsGeoBalayes = []float64{0, 0.3, 0.4, 0.5, 0.6, 0.7, 1}
	// empAbsentBalayes : neutre (0,5) ou penalise (0,25) — une cellule du sol sans
	// engagement mesure.
	empAbsentBalayes = []float64{0.25, 0.5}
	// geoAbsentBalayes : nul (le sol derive ne la connait pas) ou neutre — une cellule
	// scorable hors du sol derive.
	geoAbsentBalayes = []float64{0, 0.5}
	// hysteresisBalayees : (amorce, croissance) ; la paire v2 en premier.
	hysteresisBalayees = [][2]float64{{0.95, 0.90}, {0.95, 0.85}, {0.90, 0.85}, {0.90, 0.80}}
)

// candidatFusion est un reglage balaye et son bilan de calibrage.
type candidatFusion struct {
	Reglage fusion.Reglage
	// RappelMoyen, PrecisionMoyenne : moyennes sur les cartes de calibrage.
	RappelMoyen      float64
	PrecisionMoyenne float64
	PiegesPurs       int
	Tiennent         int
	// AireMaxM2, Couloirs : la plus grande position produite et le nombre de positions de
	// plus de `seuilCouloirCellules` cellules, toutes cartes de calibrage confondues. NE
	// PARTICIPENT PAS AU TRI (le critere ecrit est rappel puis precision) ; ils sont la
	// pour qu'on voie ce que le critere a achete — une position de 500 m2 « retrouve »
	// tout ce qu'elle couvre.
	AireMaxM2 float64
	Couloirs  int
	ParCarte  map[string]bilanV2
}

// carteCalibrage porte les entrees fusionnables d'une carte de calibrage.
type carteCalibrage struct {
	Identite SortieCartePos
	Geo      []fusion.CelluleGeo
	Emp      []powerpos.CelluleScoree
	Entree   entreeCarte
}

// TestFusionCalibrage balaye la grille, ecrit la table, et verifie que le reglage fige est
// le gagnant.
func TestFusionCalibrage(t *testing.T) {
	racineDonnees := cheminDonnees()
	if racineDonnees == "" {
		t.Skipf("%s absent : le calibrage a besoin de data/ (zones nommees)", verdictDataRootEnv)
	}
	racine := racineDepot(t)
	dossier := filepath.Join(racine, verdictDossier)
	cartes := chargeCartesCalibrage(t, racineDonnees, dossier)
	candidats := balaye(cartes)
	trieCandidats(candidats)
	if err := os.MkdirAll(filepath.Join(dossier, verdictFusion), 0o755); err != nil {
		t.Fatal(err)
	}
	sortie := filepath.Join(dossier, verdictFusion, calibrageNom)
	if err := os.WriteFile(sortie, []byte(rendCalibrage(candidats, cartes)), 0o644); err != nil {
		t.Fatal(err)
	}
	gagnant := candidats[0]
	t.Logf("gagnant : %+v — rappel %.3f precision %.3f pieges %d (table : %s)",
		gagnant.Reglage, gagnant.RappelMoyen, gagnant.PrecisionMoyenne, gagnant.PiegesPurs, sortie)
	if fige := fusion.ReglageFusionV1(); !memeReglage(fige, gagnant.Reglage) {
		t.Fatalf("le reglage fige %+v n'est pas le gagnant du balayage %+v : figer le gagnant"+
			" (reglage_v1.go) avant tout verdict", fige, gagnant.Reglage)
	}
}

// chargeCartesCalibrage lit, pour chaque carte de calibrage, ses deux CSV et son entree
// de verdict (zones nommees, lignes de l'oracle).
func chargeCartesCalibrage(t *testing.T, racineDonnees, dossier string) []carteCalibrage {
	t.Helper()
	oracle, pieges := litOracleEntre(t, filepath.Join(dossier, verdictV2OracleNom), marqueursOracleV2)
	pieges = eclateZones(pieges)
	cat, err := replay.LoadMapCallouts(title.NewPathResolver(racineDonnees).MapCalloutsPath(verdictTitleSlug))
	if err != nil {
		t.Fatalf("catalogue de zones nommees illisible : %v", err)
	}
	geoDoc := litPositions(t, filepath.Join(dossier, verdictGeometrie, positionsGeoNom))
	g := tactical.GrilleParDefaut()
	var out []carteCalibrage
	for _, ident := range geoDoc.Cartes {
		if !cartesCalibrage[ident.Carte] {
			continue
		}
		noeuds, _, err := litCSVNoeudsGeo(filepath.Join(dossier, verdictGeometrie, nomGeo(ident.Carte)+".csv"), g)
		if err != nil {
			t.Fatal(err)
		}
		cellules, err := litCSVCellules(filepath.Join(dossier, verdictV2Mesures, nomDeFichier(ident.Carte, axeFusion)+".csv"))
		if err != nil {
			t.Fatal(err)
		}
		cle, lignes := attentesDe(oracle, ident)
		_, piegesDeCarte := attentesDe(pieges, ident)
		zones := zonesDeCarte(cat, ident)
		if cle == "" || len(zones) == 0 {
			t.Fatalf("carte de calibrage %s sans oracle ou sans zones", ident.Carte)
		}
		ident.Axe = axeFusion
		out = append(out, carteCalibrage{
			Identite: ident, Geo: fusion.AgregeCellules(noeuds),
			Emp:    powerpos.Score(g, cellules, powerpos.ReglageV2()),
			Entree: entreeCarte{Carte: ident, Cle: cle, Zones: zones, Lignes: lignes, Pieges: piegesDeCarte},
		})
	}
	if len(out) != len(cartesCalibrage) {
		t.Fatalf("%d cartes de calibrage chargees, %d attendues", len(out), len(cartesCalibrage))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Identite.Carte < out[j].Identite.Carte })
	return out
}

// balaye juge chaque reglage de la grille sur les cartes de calibrage.
func balaye(cartes []carteCalibrage) []candidatFusion {
	var out []candidatFusion
	for _, a := range poidsGeoBalayes {
		for _, empAbsent := range empAbsentBalayes {
			for _, geoAbsent := range geoAbsentBalayes {
				for _, h := range hysteresisBalayees {
					r := reglageCandidat(a, empAbsent, geoAbsent, h)
					out = append(out, jugeCandidat(r, cartes))
				}
			}
		}
	}
	return out
}

// reglageCandidat assemble un reglage de la grille (selection v2 hors quantiles).
func reglageCandidat(a, empAbsent, geoAbsent float64, h [2]float64) fusion.Reglage {
	return fusion.Reglage{PoidsGeo: a, PoidsEmp: 1 - a, EmpAbsent: empAbsent, GeoAbsent: geoAbsent,
		Selection: powerpos.Reglage{QuantileAmorce: h[0], QuantileCroissance: h[1],
			FermetureRayonCellules: 1, Connexite8: true, TailleMiniComposante: 10, MaxComposantes: 8}}
}

// jugeCandidat fusionne, selectionne et juge chaque carte de calibrage avec un reglage.
func jugeCandidat(r fusion.Reglage, cartes []carteCalibrage) candidatFusion {
	g := tactical.GrilleParDefaut()
	c := candidatFusion{Reglage: r, ParCarte: map[string]bilanV2{}}
	for _, k := range cartes {
		cf := &carteFusionnee{Identite: k.Identite}
		cf.Cellules = fusion.Fusionne(g, k.Geo, k.Emp, r)
		cf.Positions = fusion.Selectionne(g, cf.Cellules, r)
		for _, p := range cf.Positions {
			cf.Bilans = append(cf.Bilans, fusion.BilanDe(p, cf.Cellules))
			c.AireMaxM2 = math.Max(c.AireMaxM2, p.AireM2)
			if len(p.Cellules) >= seuilCouloirCellules {
				c.Couloirs++
			}
		}
		e := k.Entree
		e.Carte = sortieFusionDe(cf, len(k.Geo), len(k.Emp), 0).SortieCartePos
		b := evalueV2(e, false)
		c.ParCarte[k.Identite.Carte] = b
		c.RappelMoyen += b.Rappel
		c.PrecisionMoyenne += b.PrecisionForte
		c.PiegesPurs += len(b.ContreExemplesPurs)
		if tient(b) {
			c.Tiennent++
		}
	}
	n := float64(len(cartes))
	c.RappelMoyen, c.PrecisionMoyenne = c.RappelMoyen/n, c.PrecisionMoyenne/n
	return c
}

// trieCandidats ordonne du meilleur au moins bon (ordre total, cf. en-tete).
func trieCandidats(c []candidatFusion) {
	sort.SliceStable(c, func(i, j int) bool {
		a, b := c[i], c[j]
		switch {
		case a.RappelMoyen != b.RappelMoyen:
			return a.RappelMoyen > b.RappelMoyen
		case a.PrecisionMoyenne != b.PrecisionMoyenne:
			return a.PrecisionMoyenne > b.PrecisionMoyenne
		case a.PiegesPurs != b.PiegesPurs:
			return a.PiegesPurs < b.PiegesPurs
		case math.Abs(a.Reglage.PoidsGeo-0.5) != math.Abs(b.Reglage.PoidsGeo-0.5):
			return math.Abs(a.Reglage.PoidsGeo-0.5) < math.Abs(b.Reglage.PoidsGeo-0.5)
		case a.Reglage.Selection.QuantileCroissance != b.Reglage.Selection.QuantileCroissance:
			return a.Reglage.Selection.QuantileCroissance > b.Reglage.Selection.QuantileCroissance
		case a.Reglage.Selection.QuantileAmorce != b.Reglage.Selection.QuantileAmorce:
			return a.Reglage.Selection.QuantileAmorce > b.Reglage.Selection.QuantileAmorce
		case a.Reglage.EmpAbsent != b.Reglage.EmpAbsent:
			return a.Reglage.EmpAbsent > b.Reglage.EmpAbsent
		default:
			return a.Reglage.GeoAbsent > b.Reglage.GeoAbsent
		}
	})
}

// memeReglage compare ce que le balayage fait varier (le plancher absolu, pose apres le
// gel sous l'amorce la plus basse, n'en fait pas partie).
func memeReglage(a, b fusion.Reglage) bool {
	return a.PoidsGeo == b.PoidsGeo && a.PoidsEmp == b.PoidsEmp &&
		a.EmpAbsent == b.EmpAbsent && a.GeoAbsent == b.GeoAbsent &&
		a.Selection.QuantileAmorce == b.Selection.QuantileAmorce &&
		a.Selection.QuantileCroissance == b.Selection.QuantileCroissance &&
		a.Selection.TailleMiniComposante == b.Selection.TailleMiniComposante &&
		a.Selection.MaxComposantes == b.Selection.MaxComposantes &&
		a.Selection.FermetureRayonCellules == b.Selection.FermetureRayonCellules &&
		a.Selection.Connexite8 == b.Selection.Connexite8
}

// rendCalibrage ecrit la table complete du balayage, du meilleur au moins bon.
func rendCalibrage(candidats []candidatFusion, cartes []carteCalibrage) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Balayage du reglage de fusion — cartes de CALIBRAGE seulement (2026-09-20)\n\n")
	fmt.Fprintf(&b, "> Genere par `go test -tags research ./cmd/mappower-build/ -run FusionCalibrage`."+
		" %d candidats = poids geo %v x emp absent %v x geo absent %v x (amorce, croissance) %v."+
		" Juges avec les regles du verdict v2 (rappel sur les fortes resolues, precision « touche une"+
		" forte », pieges purs au barycentre) sur %s. Tri : rappel moyen, puis precision moyenne, puis"+
		" pieges purs, puis |a - 0,5|, puis croissance et amorce les plus hautes, puis absences les plus"+
		" neutres. Les cartes de VALIDATION ne sont jamais lues ici.\n\n",
		len(candidats), poidsGeoBalayes, empAbsentBalayes, geoAbsentBalayes, hysteresisBalayees,
		liste(clesDeMap(cartesCalibrage)))
	fmt.Fprintf(&b, "Les deux colonnes « aire max » et « couloirs » (positions de %d cellules ou plus)"+
		" ne participent pas au tri : elles montrent ce que le critere achete.\n\n", seuilCouloirCellules)
	fmt.Fprintf(&b, "| rang | a (geo) | b (emp) | emp absent | geo absent | amorce | croissance | rappel moyen | precision moyenne | pieges purs | tiennent | aire max m2 | couloirs |")
	for _, k := range cartes {
		fmt.Fprintf(&b, " %s (rappel / precision / positions) |", k.Identite.Carte)
	}
	fmt.Fprintln(&b)
	fmt.Fprint(&b, "|---|---|---|---|---|---|---|---|---|---|---|---|---|")
	for range cartes {
		fmt.Fprint(&b, "---|")
	}
	fmt.Fprintln(&b)
	for i, c := range candidats {
		r := c.Reglage
		fmt.Fprintf(&b, "| %d | %.1f | %.1f | %.2f | %.1f | p%.0f | p%.0f | %.3f | %.3f | %d | %d | %.0f | %d |", i+1,
			r.PoidsGeo, r.PoidsEmp, r.EmpAbsent, r.GeoAbsent, r.Selection.QuantileAmorce*100,
			r.Selection.QuantileCroissance*100, c.RappelMoyen, c.PrecisionMoyenne, c.PiegesPurs, c.Tiennent,
			c.AireMaxM2, c.Couloirs)
		for _, k := range cartes {
			bk := c.ParCarte[k.Identite.Carte]
			fmt.Fprintf(&b, " %.2f / %.2f / %d |", bk.Rappel, bk.PrecisionForte, len(bk.Positions))
		}
		fmt.Fprintln(&b)
	}
	fmt.Fprintln(&b)
	g := candidats[0]
	fmt.Fprintf(&b, "**Gagnant** : a = %.1f, b = %.1f, emp absent %.2f, geo absent %.1f, amorce p%.0f,"+
		" croissance p%.0f — rappel moyen %.3f, precision moyenne %.3f, %d piege(s) pur(s).\n",
		g.Reglage.PoidsGeo, g.Reglage.PoidsEmp, g.Reglage.EmpAbsent, g.Reglage.GeoAbsent,
		g.Reglage.Selection.QuantileAmorce*100, g.Reglage.Selection.QuantileCroissance*100,
		g.RappelMoyen, g.PrecisionMoyenne, g.PiegesPurs)
	return b.String()
}
