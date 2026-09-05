package tactical

import (
	"errors"
	"math"
	"testing"

	"levelup/go-api/internal/domain"
)

// cellule : retrouve une cellule de sortie par son adresse.
func cellule(t *testing.T, cellules []domain.CelluleTactique, col, lig int) domain.CelluleTactique {
	t.Helper()
	for _, c := range cellules {
		if c.Col == col && c.Lig == lig {
			return c
		}
	}
	t.Fatalf("cellule (%d,%d) absente de %+v", col, lig, cellules)
	return domain.CelluleTactique{}
}

func absente(t *testing.T, cellules []domain.CelluleTactique, col, lig int) {
	t.Helper()
	for _, c := range cellules {
		if c.Col == col && c.Lig == lig {
			t.Fatalf("cellule (%d,%d) presente alors qu'elle devait etre retiree : %+v", col, lig, c)
		}
	}
}

// ordreDesCellules verifie l'ordre (Col, Lig) d'une sortie.
func ordreDesCellules(t *testing.T, cellules []domain.CelluleTactique, attendu []Cellule, contexte string) {
	t.Helper()
	if len(cellules) != len(attendu) {
		t.Fatalf("%s : len = %d, attendu %d (%+v)", contexte, len(cellules), len(attendu), cellules)
	}
	for i, a := range attendu {
		if cellules[i].Col != a.Col || cellules[i].Lig != a.Lig {
			t.Fatalf("%s : [%d] = (%d,%d), attendu (%d,%d)",
				contexte, i, cellules[i].Col, cellules[i].Lig, a.Col, a.Lig)
		}
	}
}

// TestSomme_DeuxRasters : sommer un raster par match doit rendre EXACTEMENT ce que le
// rasterisage direct de l'union rend. C'est l'invariant qui autorise le stockage d'un
// raster par match a la cuisson.
func TestSomme_DeuxRasters(t *testing.T) {
	g := GrilleParDefaut()
	univers := []string{"m1", "m2"}
	tous := []domain.PositionSample{
		pt("m1", 0.1, 0.1), pt("m1", 0.2, 0.2), pt("m1", 1.2, 0.2),
		pt("m2", 0.3, 0.3), pt("m2", -0.2, -0.2),
	}
	direct := rasteriseOk(t, g, univers, tous)
	somme, err := Somme(
		rasteriseOk(t, g, univers, tous[:3]),
		rasteriseOk(t, g, univers, tous[3:]),
	)
	if err != nil {
		t.Fatalf("Somme : %v", err)
	}

	if somme.NbMatchs() != direct.NbMatchs() || somme.NbMatchs() != 2 {
		t.Fatalf("NbMatchs : somme = %d, direct = %d, attendu 2", somme.NbMatchs(), direct.NbMatchs())
	}
	if somme.NbCellules() != direct.NbCellules() || somme.NbCellules() != 3 {
		t.Fatalf("NbCellules : somme = %d, direct = %d, attendu 3", somme.NbCellules(), direct.NbCellules())
	}
	for _, c := range []Cellule{{Col: 0, Lig: 0}, {Col: 2, Lig: 0}, {Col: -1, Lig: -1}} {
		if somme.Occurrences(c) != direct.Occurrences(c) {
			t.Fatalf("Occurrences%+v : somme = %d, direct = %d", c, somme.Occurrences(c), direct.Occurrences(c))
		}
		if somme.MatchsDistincts(c) != direct.MatchsDistincts(c) {
			t.Fatalf("MatchsDistincts%+v : somme = %d, direct = %d", c, somme.MatchsDistincts(c), direct.MatchsDistincts(c))
		}
	}
	// La cellule (0,0) : m1 deux fois, m2 une fois -> 3 passages, 2 matchs distincts.
	if got := somme.Occurrences(Cellule{}); got != 3 {
		t.Fatalf("Occurrences(0,0) = %d, attendu 3", got)
	}
	if got := somme.MatchsDistincts(Cellule{}); got != 2 {
		t.Fatalf("MatchsDistincts(0,0) = %d, attendu 2", got)
	}
}

// TestSomme_MemeMatchDansDeuxRasters : deux vues partielles d'un meme match s'additionnent
// en passages mais ne comptent qu'une fois en matchs distincts.
func TestSomme_MemeMatchDansDeuxRasters(t *testing.T) {
	g := GrilleParDefaut()
	somme, err := Somme(
		rasteriseOk(t, g, []string{"m1"}, []domain.PositionSample{pt("m1", 0.2, 0.2)}),
		rasteriseOk(t, g, []string{"m1"}, []domain.PositionSample{pt("m1", 0.3, 0.3)}),
	)
	if err != nil {
		t.Fatalf("Somme : %v", err)
	}
	if got := somme.Occurrences(Cellule{}); got != 2 {
		t.Fatalf("Occurrences = %d, attendu 2", got)
	}
	if got := somme.MatchsDistincts(Cellule{}); got != 1 {
		t.Fatalf("MatchsDistincts = %d, attendu 1", got)
	}
	if got := somme.NbMatchs(); got != 1 {
		t.Fatalf("NbMatchs = %d, attendu 1", got)
	}
}

// TestSomme_ResultatInconnuNEstPasUneContradiction : le cas nominal annonce par la doc de
// Somme — les morts (rasterisees sans resultats) et les kills (avec) du meme filtre. Un
// OutcomeUnknown est une ABSENCE d'information, pas un resultat concurrent : la valeur
// connue l'emporte, dans les deux sens de la somme.
func TestSomme_ResultatInconnuNEstPasUneContradiction(t *testing.T) {
	g := GrilleParDefaut()
	morts := rasteriseOk(t, g, []string{"m1"}, []domain.PositionSample{pt("m1", 0.2, 0.2)})
	kills := rasteriseResOk(t, g, map[string]int{"m1": domain.OutcomeWin},
		[]domain.PositionSample{pt("m1", 0.3, 0.3)})

	for _, ordre := range [][]*Raster{{morts, kills}, {kills, morts}} {
		somme, err := Somme(ordre...)
		if err != nil {
			t.Fatalf("Somme : %v", err)
		}
		if got := somme.NbMatchsResultat(domain.OutcomeWin); got != 1 {
			t.Fatalf("NbMatchsResultat(win) = %d, attendu 1 (la valeur connue l'emporte)", got)
		}
		if got := somme.Occurrences(Cellule{}); got != 2 {
			t.Fatalf("Occurrences = %d, attendu 2", got)
		}
	}
}

func TestSomme_Erreurs(t *testing.T) {
	g := GrilleParDefaut()
	if _, err := Somme(); !errors.Is(err, ErrAucunRaster) {
		t.Fatalf("Somme() : err = %v, attendu ErrAucunRaster", err)
	}

	fine, err := NouvelleGrille(0.25)
	if err != nil {
		t.Fatalf("NouvelleGrille : %v", err)
	}
	_, err = Somme(
		rasteriseOk(t, g, []string{"m1"}, []domain.PositionSample{pt("m1", 0.2, 0.2)}),
		rasteriseOk(t, fine, []string{"m1"}, []domain.PositionSample{pt("m1", 0.2, 0.2)}),
	)
	if !errors.Is(err, ErrPasIncompatible) {
		t.Fatalf("Somme(pas differents) : err = %v, attendu ErrPasIncompatible", err)
	}

	// Deux resultats CONNUS et differents pour le meme match : contradiction.
	_, err = Somme(
		rasteriseResOk(t, g, map[string]int{"m1": domain.OutcomeWin}, nil),
		rasteriseResOk(t, g, map[string]int{"m1": domain.OutcomeLoss}, nil),
	)
	if !errors.Is(err, ErrResultatIncoherent) {
		t.Fatalf("Somme(resultats contradictoires) : err = %v, attendu ErrResultatIncoherent", err)
	}

	// Univers de tailles differentes, puis de meme taille mais d'ensembles differents.
	_, err = Somme(
		rasteriseOk(t, g, []string{"m1", "m2"}, nil),
		rasteriseOk(t, g, []string{"m1"}, nil),
	)
	if !errors.Is(err, ErrUniversIncompatible) {
		t.Fatalf("Somme(univers de tailles differentes) : err = %v, attendu ErrUniversIncompatible", err)
	}
	_, err = Somme(
		rasteriseOk(t, g, []string{"m1", "m2"}, nil),
		rasteriseOk(t, g, []string{"m1", "m3"}, nil),
	)
	if !errors.Is(err, ErrUniversIncompatible) {
		t.Fatalf("Somme(univers differents) : err = %v, attendu ErrUniversIncompatible", err)
	}
}

// TestSomme_ConservePasEtPointsIgnores : la somme garde le pas de ses rasters (ici 0,25 m,
// pas la valeur par defaut) et CUMULE les points illisibles — un decodage qui derape sur
// deux matchs se voit sur l'agregat.
func TestSomme_ConservePasEtPointsIgnores(t *testing.T) {
	fine, err := NouvelleGrille(0.25)
	if err != nil {
		t.Fatalf("NouvelleGrille : %v", err)
	}
	univers := []string{"m1", "m2", "m3"}
	somme, err := Somme(
		rasteriseOk(t, fine, univers, []domain.PositionSample{
			pt("m1", 0.1, 0.1), pt("m1", math.NaN(), 0.1),
		}),
		rasteriseOk(t, fine, univers, []domain.PositionSample{
			pt("m2", 0.2, 0.2), pt("m3", 0.05, 0.24), pt("m3", 0.1, math.Inf(1)),
		}),
	)
	if err != nil {
		t.Fatalf("Somme : %v", err)
	}

	if somme.PasM() != 0.25 {
		t.Fatalf("PasM = %v, attendu 0.25", somme.PasM())
	}
	if got := somme.PointsIgnores(); got != 2 {
		t.Fatalf("PointsIgnores = %d, attendu 2 (un par raster)", got)
	}
	c := cellule(t, somme.Cellules(), 0, 0)
	if c.CentreX != 0.125 || c.CentreY != 0.125 {
		t.Fatalf("Centre = (%v, %v), attendu (0.125, 0.125) au pas de 0,25 m", c.CentreX, c.CentreY)
	}
	if c.Matchs != 3 || c.Brut != 3 {
		t.Fatalf("cellule = %+v, attendu 3 matchs et 3 passages", c)
	}
}

// TestCellules_PlancherEtValeurParMatch : jeu pose a la main, comptes exacts.
//
//	cellule (0,0) : m1 x2, m2 x1, m3 x3 -> 6 passages, 3 matchs distincts -> RETENUE
//	cellule (2,0) : m1 x5, m2 x5        -> 10 passages, 2 matchs          -> RETIREE
//	cellule (4,0) : m4 x1               -> 1 passage,  1 match            -> RETIREE
//
// L'univers compte 4 matchs, donc la valeur par match de (0,0) vaut 6/4 = 1,5 — et NON
// 6/3 = 2, qui serait la moyenne sur les seuls matchs de la cellule.
func TestCellules_PlancherEtValeurParMatch(t *testing.T) {
	var points []domain.PositionSample
	ajoute := func(match string, x, y float64, n int) {
		for i := 0; i < n; i++ {
			points = append(points, pt(match, x, y))
		}
	}
	ajoute("m1", 0.2, 0.2, 2)
	ajoute("m2", 0.2, 0.2, 1)
	ajoute("m3", 0.2, 0.2, 3)
	ajoute("m1", 1.2, 0.2, 5)
	ajoute("m2", 1.2, 0.2, 5)
	ajoute("m4", 2.2, 0.2, 1)

	r := rasteriseOk(t, GrilleParDefaut(), []string{"m1", "m2", "m3", "m4"}, points)
	if got := r.NbMatchs(); got != 4 {
		t.Fatalf("NbMatchs = %d, attendu 4", got)
	}

	cellules := r.Cellules()
	if len(cellules) != 1 {
		t.Fatalf("len(Cellules) = %d, attendu 1 : %+v", len(cellules), cellules)
	}
	absente(t, cellules, 2, 0)
	absente(t, cellules, 4, 0)

	c := cellule(t, cellules, 0, 0)
	if c.Brut != 6 {
		t.Fatalf("Brut = %v, attendu 6", c.Brut)
	}
	if c.Matchs != 3 {
		t.Fatalf("Matchs = %d, attendu 3", c.Matchs)
	}
	if c.Valeur != 1.5 {
		t.Fatalf("Valeur = %v, attendu 1.5 (6 passages / 4 matchs retenus)", c.Valeur)
	}
	if c.CentreX != 0.25 || c.CentreY != 0.25 {
		t.Fatalf("Centre = (%v, %v), attendu (0.25, 0.25)", c.CentreX, c.CentreY)
	}
}

// TestCellulesSignees_PlancherParCote : trois victoires et zero defaite -> la cellule est
// RETIREE, pas peinte en positif.
func TestCellulesSignees_PlancherParCote(t *testing.T) {
	resultats := map[string]int{
		"v1": domain.OutcomeWin, "v2": domain.OutcomeWin, "v3": domain.OutcomeWin,
		"d1": domain.OutcomeLoss, "d2": domain.OutcomeLoss, "d3": domain.OutcomeLoss,
	}
	var points []domain.PositionSample
	// Cellule (0,0) : 3 victoires, 0 defaite -> retiree.
	for _, m := range []string{"v1", "v2", "v3"} {
		points = append(points, pt(m, 0.2, 0.2))
	}
	// Cellule (2,0) : 3 victoires x2 passages, 3 defaites x1 passage -> retenue.
	for _, m := range []string{"v1", "v2", "v3"} {
		points = append(points, pt(m, 1.2, 0.2), pt(m, 1.3, 0.2))
	}
	for _, m := range []string{"d1", "d2", "d3"} {
		points = append(points, pt(m, 1.2, 0.2))
	}

	r := rasteriseResOk(t, GrilleParDefaut(), resultats, points)
	cellules := r.CellulesSignees()
	if len(cellules) != 1 {
		t.Fatalf("len(CellulesSignees) = %d, attendu 1 : %+v", len(cellules), cellules)
	}
	absente(t, cellules, 0, 0)

	c := cellule(t, cellules, 2, 0)
	if c.MatchsVictoire != 3 || c.MatchsDefaite != 3 {
		t.Fatalf("cotes = %d V / %d D, attendu 3 V / 3 D", c.MatchsVictoire, c.MatchsDefaite)
	}
	if c.Matchs != 6 {
		t.Fatalf("Matchs = %d, attendu 6", c.Matchs)
	}
	if c.Brut != 3 {
		t.Fatalf("Brut = %v, attendu 3 (6 passages en victoire - 3 en defaite)", c.Brut)
	}
	if c.Valeur != 1 {
		t.Fatalf("Valeur = %v, attendu 1 (6/3 victoires - 3/3 defaites)", c.Valeur)
	}
}

// TestCellulesSignees_LesDeuxCotesDuPlancherDecident : la moitie « 3 victoires » doit etre
// decidante elle aussi. Univers largement fourni des deux cotes ; ce sont les comptes DE LA
// CELLULE qui la retirent.
func TestCellulesSignees_LesDeuxCotesDuPlancherDecident(t *testing.T) {
	resultats := map[string]int{}
	for _, m := range []string{"v1", "v2", "v3", "v4", "v5"} {
		resultats[m] = domain.OutcomeWin
	}
	for _, m := range []string{"d1", "d2", "d3", "d4"} {
		resultats[m] = domain.OutcomeLoss
	}

	cas := []struct {
		nom        string
		victoires  []string
		defaites   []string
		col        int
		x          float64
		attendueOK bool
	}{
		{"1 victoire, 4 defaites", []string{"v1"}, []string{"d1", "d2", "d3", "d4"}, 0, 0.2, false},
		{"2 victoires, 3 defaites", []string{"v1", "v2"}, []string{"d1", "d2", "d3"}, 2, 1.2, false},
		{"3 victoires, 3 defaites", []string{"v1", "v2", "v3"}, []string{"d1", "d2", "d3"}, 4, 2.2, true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			var points []domain.PositionSample
			for _, m := range append(append([]string{}, c.victoires...), c.defaites...) {
				points = append(points, pt(m, c.x, 0.2))
			}
			cellules := rasteriseResOk(t, GrilleParDefaut(), resultats, points).CellulesSignees()
			if c.attendueOK {
				cellule(t, cellules, c.col, 0)
				return
			}
			absente(t, cellules, c.col, 0)
			if len(cellules) != 0 {
				t.Fatalf("CellulesSignees = %+v, attendu vide", cellules)
			}
		})
	}
}

// TestCellulesSignees_DenominateurSurLUnivers : le cas P0. Douze victoires et huit defaites,
// dont DEUX victoires sans le moindre point sur cette lecture. La cellule est vue 6 fois en
// victoire et 4 fois en defaite : au rythme reel, 6/12 = 0,5 contre 4/8 = 0,5, soit une
// cellule NEUTRE. Un denominateur deduit des points rendrait 6/10 - 4/8 = +0,10 et peindrait
// une zone gagnante qui n'est que le silence de deux victoires.
func TestCellulesSignees_DenominateurSurLUnivers(t *testing.T) {
	resultats := map[string]int{}
	var points []domain.PositionSample
	for i := 1; i <= 12; i++ {
		m := "v" + string(rune('a'+i-1))
		resultats[m] = domain.OutcomeWin
		if i <= 6 {
			points = append(points, pt(m, 0.2, 0.2))
		} else if i <= 10 {
			// Joue, retenu, present ailleurs sur la carte : ni muet, ni dans la cellule.
			points = append(points, pt(m, 5.2, 5.2))
		}
		// i = 11 et 12 : victoires SANS aucun point sur cette lecture.
	}
	for i := 1; i <= 8; i++ {
		m := "d" + string(rune('a'+i-1))
		resultats[m] = domain.OutcomeLoss
		if i <= 4 {
			points = append(points, pt(m, 0.2, 0.2))
		} else {
			points = append(points, pt(m, 5.2, 5.2))
		}
	}

	r := rasteriseResOk(t, GrilleParDefaut(), resultats, points)
	if got := r.NbMatchsResultat(domain.OutcomeWin); got != 12 {
		t.Fatalf("NbMatchsResultat(win) = %d, attendu 12 (les deux victoires muettes comptent)", got)
	}
	if got := r.NbMatchsResultat(domain.OutcomeLoss); got != 8 {
		t.Fatalf("NbMatchsResultat(loss) = %d, attendu 8", got)
	}

	c := cellule(t, r.CellulesSignees(), 0, 0)
	if c.MatchsVictoire != 6 || c.MatchsDefaite != 4 {
		t.Fatalf("cotes = %d V / %d D, attendu 6 V / 4 D", c.MatchsVictoire, c.MatchsDefaite)
	}
	if c.Brut != 2 {
		t.Fatalf("Brut = %v, attendu 2 (6 passages en victoire - 4 en defaite)", c.Brut)
	}
	presque(t, "Valeur", c.Valeur, 0)
}

// TestCellulesSignees_CelluleIncompleteEtCotesAsymetriques : la cellule ne contient PAS tous
// les matchs de l'univers, et ses deux cotes n'ont ni le meme nombre de matchs ni le meme
// nombre de passages. Trois erreurs distinctes echouent ici : diviser par les matchs de la
// cellule (8/4 - 3/3 = 1), faire une difference brute (5), et echanger MatchsVictoire avec
// MatchsDefaite.
func TestCellulesSignees_CelluleIncompleteEtCotesAsymetriques(t *testing.T) {
	resultats := map[string]int{}
	var points []domain.PositionSample
	for _, m := range []string{"v1", "v2", "v3", "v4", "v5"} {
		resultats[m] = domain.OutcomeWin
	}
	for _, m := range []string{"d1", "d2", "d3", "d4"} {
		resultats[m] = domain.OutcomeLoss
	}
	// Cellule (0,0) : 4 victoires a 2 passages (8), 3 defaites a 1 passage (3).
	for _, m := range []string{"v1", "v2", "v3", "v4"} {
		points = append(points, pt(m, 0.2, 0.2), pt(m, 0.3, 0.3))
	}
	for _, m := range []string{"d1", "d2", "d3"} {
		points = append(points, pt(m, 0.2, 0.2))
	}
	// v5 et d4 sont retenus et joues, mais ailleurs sur la carte.
	points = append(points, pt("v5", 5.2, 5.2), pt("d4", 5.2, 5.2))

	c := cellule(t, rasteriseResOk(t, GrilleParDefaut(), resultats, points).CellulesSignees(), 0, 0)
	if c.MatchsVictoire != 4 {
		t.Fatalf("MatchsVictoire = %d, attendu 4", c.MatchsVictoire)
	}
	if c.MatchsDefaite != 3 {
		t.Fatalf("MatchsDefaite = %d, attendu 3", c.MatchsDefaite)
	}
	if c.Matchs != 7 {
		t.Fatalf("Matchs = %d, attendu 7", c.Matchs)
	}
	if c.Brut != 5 {
		t.Fatalf("Brut = %v, attendu 5 (8 - 3)", c.Brut)
	}
	// 8/5 - 3/4 = 1,60 - 0,75 = 0,85 (et non 8/4 - 3/3 = 1).
	presque(t, "Valeur", c.Valeur, 0.85)
}

// TestCellulesSignees_ChaqueCoteRameneAuSien : le cas qui distingue une difference brute
// d'un ecart de taux. Six victoires, trois defaites, une cellule traversee une fois par
// match des deux cotes : le rythme est IDENTIQUE, la valeur doit etre nulle. Une difference
// brute lirait +3 et peindrait une zone gagnante qui n'est que le taux de victoire global.
func TestCellulesSignees_ChaqueCoteRameneAuSien(t *testing.T) {
	resultats := map[string]int{}
	var points []domain.PositionSample
	for _, m := range []string{"v1", "v2", "v3", "v4", "v5", "v6"} {
		resultats[m] = domain.OutcomeWin
		points = append(points, pt(m, 0.2, 0.2))
	}
	for _, m := range []string{"d1", "d2", "d3"} {
		resultats[m] = domain.OutcomeLoss
		points = append(points, pt(m, 0.2, 0.2))
	}

	cellules := rasteriseResOk(t, GrilleParDefaut(), resultats, points).CellulesSignees()
	c := cellule(t, cellules, 0, 0)
	if c.Brut != 3 {
		t.Fatalf("Brut = %v, attendu 3 (la difference brute, servie a titre d'audit)", c.Brut)
	}
	if c.Valeur != 0 {
		t.Fatalf("Valeur = %v, attendu 0 (6/6 - 3/3 : meme rythme des deux cotes)", c.Valeur)
	}
}

// TestCellulesSignees_NulsEtInconnusHorsLecture : un nul ou un resultat inconnu ne
// participe a aucun cote, mais reste compte dans la lecture non signee.
func TestCellulesSignees_NulsEtInconnusHorsLecture(t *testing.T) {
	resultats := map[string]int{
		"v1": domain.OutcomeWin, "v2": domain.OutcomeWin, "v3": domain.OutcomeWin,
		"d1": domain.OutcomeLoss, "d2": domain.OutcomeLoss,
		"n1": domain.OutcomeDraw, "x1": domain.OutcomeUnknown,
	}
	var points []domain.PositionSample
	for m := range resultats {
		points = append(points, pt(m, 0.2, 0.2))
	}

	r := rasteriseResOk(t, GrilleParDefaut(), resultats, points)
	// 2 defaites seulement : la cellule ne passe pas le plancher par cote.
	if got := r.CellulesSignees(); len(got) != 0 {
		t.Fatalf("CellulesSignees = %+v, attendu vide (2 defaites < %d)", got, PlancherMatchsParCote)
	}
	// La meme cellule reste lisible en lecture non signee : 7 passages sur 7 matchs.
	c := cellule(t, r.Cellules(), 0, 0)
	if c.Brut != 7 || c.Matchs != 7 || c.Valeur != 1 {
		t.Fatalf("lecture non signee = %+v, attendu Brut 7, Matchs 7, Valeur 1", c)
	}
	if c.MatchsVictoire != 0 || c.MatchsDefaite != 0 {
		t.Fatalf("lecture non signee : cotes = %d/%d, attendu 0/0", c.MatchsVictoire, c.MatchsDefaite)
	}
}

// TestCellules_TriStable : le parcours d'une map est aleatoire ; la sortie ne doit pas
// bouger d'un appel a l'autre.
func TestCellules_TriStable(t *testing.T) {
	univers := []string{"m1", "m2", "m3"}
	var points []domain.PositionSample
	for _, m := range univers {
		points = append(points,
			pt(m, 2.2, 0.2), pt(m, 0.2, 1.2), pt(m, 0.2, 0.2), pt(m, -1.2, 0.2))
	}
	attendu := []Cellule{{Col: -3, Lig: 0}, {Col: 0, Lig: 0}, {Col: 0, Lig: 2}, {Col: 4, Lig: 0}}
	for essai := 0; essai < 5; essai++ {
		cellules := rasteriseOk(t, GrilleParDefaut(), univers, points).Cellules()
		ordreDesCellules(t, cellules, attendu, "Cellules")
	}
}

// TestCellulesSignees_TriStable : meme exigence sur la lecture signee, qui a son propre
// parcours de map.
func TestCellulesSignees_TriStable(t *testing.T) {
	resultats := map[string]int{
		"v1": domain.OutcomeWin, "v2": domain.OutcomeWin, "v3": domain.OutcomeWin,
		"d1": domain.OutcomeLoss, "d2": domain.OutcomeLoss, "d3": domain.OutcomeLoss,
	}
	var points []domain.PositionSample
	for m := range resultats {
		points = append(points,
			pt(m, 2.2, 0.2), pt(m, 0.2, 1.2), pt(m, 0.2, 0.2), pt(m, -1.2, 0.2))
	}
	attendu := []Cellule{{Col: -3, Lig: 0}, {Col: 0, Lig: 0}, {Col: 0, Lig: 2}, {Col: 4, Lig: 0}}
	for essai := 0; essai < 5; essai++ {
		cellules := rasteriseResOk(t, GrilleParDefaut(), resultats, points).CellulesSignees()
		ordreDesCellules(t, cellules, attendu, "CellulesSignees")
	}
}
