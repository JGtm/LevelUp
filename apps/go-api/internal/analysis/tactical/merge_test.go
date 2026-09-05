package tactical

import (
	"errors"
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

// TestSomme_DeuxRasters : sommer un raster par match doit rendre EXACTEMENT ce que le
// rasterisage direct de l'union rend. C'est l'invariant qui autorise le stockage d'un
// raster par match a la cuisson.
func TestSomme_DeuxRasters(t *testing.T) {
	g := GrilleParDefaut()
	tous := []domain.PositionSample{
		pt("m1", 0.1, 0.1), pt("m1", 0.2, 0.2), pt("m1", 1.2, 0.2),
		pt("m2", 0.3, 0.3), pt("m2", -0.2, -0.2),
	}
	direct := Rasterise(g, tous)
	somme, err := Somme(
		Rasterise(g, tous[:3]),
		Rasterise(g, tous[3:]),
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
		Rasterise(g, []domain.PositionSample{pt("m1", 0.2, 0.2)}),
		Rasterise(g, []domain.PositionSample{pt("m1", 0.3, 0.3)}),
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

func TestSomme_Erreurs(t *testing.T) {
	if _, err := Somme(); !errors.Is(err, ErrAucunRaster) {
		t.Fatalf("Somme() : err = %v, attendu ErrAucunRaster", err)
	}

	fine, err := NouvelleGrille(0.25)
	if err != nil {
		t.Fatalf("NouvelleGrille : %v", err)
	}
	_, err = Somme(
		Rasterise(GrilleParDefaut(), []domain.PositionSample{pt("m1", 0.2, 0.2)}),
		Rasterise(fine, []domain.PositionSample{pt("m2", 0.2, 0.2)}),
	)
	if !errors.Is(err, ErrPasIncompatible) {
		t.Fatalf("Somme(pas differents) : err = %v, attendu ErrPasIncompatible", err)
	}

	g := GrilleParDefaut()
	_, err = Somme(
		RasteriseAvecResultats(g, []domain.PositionSample{pt("m1", 0.2, 0.2)}, map[string]int{"m1": domain.OutcomeWin}),
		RasteriseAvecResultats(g, []domain.PositionSample{pt("m1", 0.2, 0.2)}, map[string]int{"m1": domain.OutcomeLoss}),
	)
	if !errors.Is(err, ErrResultatIncoherent) {
		t.Fatalf("Somme(resultats contradictoires) : err = %v, attendu ErrResultatIncoherent", err)
	}
}

// TestCellules_PlancherEtValeurParMatch : jeu pose a la main, comptes exacts.
//
//	cellule (0,0) : m1 x2, m2 x1, m3 x3 -> 6 passages, 3 matchs distincts -> RETENUE
//	cellule (2,0) : m1 x5, m2 x5        -> 10 passages, 2 matchs          -> RETIREE
//	cellule (4,0) : m4 x1               -> 1 passage,  1 match            -> RETIREE
//
// Le raster compte 4 matchs, donc la valeur par match de (0,0) vaut 6/4 = 1,5 — et NON
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

	r := Rasterise(GrilleParDefaut(), points)
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

	r := RasteriseAvecResultats(GrilleParDefaut(), points, resultats)
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

	cellules := RasteriseAvecResultats(GrilleParDefaut(), points, resultats).CellulesSignees()
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

	r := RasteriseAvecResultats(GrilleParDefaut(), points, resultats)
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
	var points []domain.PositionSample
	for _, m := range []string{"m1", "m2", "m3"} {
		points = append(points,
			pt(m, 2.2, 0.2), pt(m, 0.2, 1.2), pt(m, 0.2, 0.2), pt(m, -1.2, 0.2))
	}
	cellules := Rasterise(GrilleParDefaut(), points).Cellules()
	attendu := []Cellule{{Col: -3, Lig: 0}, {Col: 0, Lig: 0}, {Col: 0, Lig: 2}, {Col: 4, Lig: 0}}
	if len(cellules) != len(attendu) {
		t.Fatalf("len(Cellules) = %d, attendu %d", len(cellules), len(attendu))
	}
	for i, a := range attendu {
		if cellules[i].Col != a.Col || cellules[i].Lig != a.Lig {
			t.Fatalf("Cellules[%d] = (%d,%d), attendu (%d,%d)", i, cellules[i].Col, cellules[i].Lig, a.Col, a.Lig)
		}
	}
}
