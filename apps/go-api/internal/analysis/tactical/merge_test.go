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
