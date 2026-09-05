package tactical

import (
	"errors"
	"math"
	"testing"

	"levelup/go-api/internal/domain"
)

func TestNouvelleGrille_RefusePasNonFiniOuNul(t *testing.T) {
	for _, pas := range []float64{0, -0.5, math.NaN(), math.Inf(1)} {
		if _, err := NouvelleGrille(pas); !errors.Is(err, ErrPasInvalide) {
			t.Fatalf("NouvelleGrille(%v) : err = %v, attendu ErrPasInvalide", pas, err)
		}
	}
	g, err := NouvelleGrille(0.25)
	if err != nil {
		t.Fatalf("NouvelleGrille(0.25) : %v", err)
	}
	if g.PasM() != 0.25 {
		t.Fatalf("PasM = %v, attendu 0.25", g.PasM())
	}
}

func TestGrilleParDefaut_Pas05(t *testing.T) {
	if got := GrilleParDefaut().PasM(); got != 0.5 {
		t.Fatalf("PasM = %v, attendu 0.5", got)
	}
	// Une Grille zero doit rester adressable (pas de division par zero).
	if got := (Grille{}).PasM(); got != PasParDefautM {
		t.Fatalf("Grille zero : PasM = %v, attendu %v", got, PasParDefautM)
	}
}

// TestCellule_AncrageMonde : l'adressage est ancre sur l'origine du monde et borne par le
// BAS de chaque cellule. Les negatifs sont le cas qui distingue un floor d'une troncature
// (int(-0.2) vaut 0, floor(-0.2) vaut -1) : sans lui, deux points de part et d'autre de
// l'origine tomberaient dans la meme cellule.
func TestCellule_AncrageMonde(t *testing.T) {
	g := GrilleParDefaut()
	cas := []struct {
		x, y     float64
		col, lig int
	}{
		{0, 0, 0, 0},
		{0.49, 0.49, 0, 0},
		{0.5, 0.5, 1, 1},
		{0.99, 1.0, 1, 2},
		{-0.01, -0.01, -1, -1},
		{-0.5, -0.5, -1, -1},
		{-0.51, -0.51, -2, -2},
		{12.3, -7.8, 24, -16},
	}
	for _, c := range cas {
		got, ok := g.Cellule(c.x, c.y)
		if !ok {
			t.Fatalf("Cellule(%v, %v) : ok = false", c.x, c.y)
		}
		if got.Col != c.col || got.Lig != c.lig {
			t.Fatalf("Cellule(%v, %v) = (%d, %d), attendu (%d, %d)", c.x, c.y, got.Col, got.Lig, c.col, c.lig)
		}
	}
}

func TestCellule_EcarteLesPositionsNonFinies(t *testing.T) {
	g := GrilleParDefaut()
	cas := [][2]float64{
		{math.NaN(), 0},
		{0, math.NaN()},
		{math.Inf(1), 0},
		{0, math.Inf(-1)},
	}
	for _, c := range cas {
		if _, ok := g.Cellule(c[0], c[1]); ok {
			t.Fatalf("Cellule(%v, %v) : ok = true, attendu false", c[0], c[1])
		}
	}
}

func TestCentreEtBornesDe(t *testing.T) {
	g := GrilleParDefaut()
	x, y := g.Centre(Cellule{Col: 0, Lig: 0})
	if x != 0.25 || y != 0.25 {
		t.Fatalf("Centre(0,0) = (%v, %v), attendu (0.25, 0.25)", x, y)
	}
	x, y = g.Centre(Cellule{Col: -1, Lig: 3})
	if x != -0.25 || y != 1.75 {
		t.Fatalf("Centre(-1,3) = (%v, %v), attendu (-0.25, 1.75)", x, y)
	}

	b := g.BornesDe(Cellule{Col: -1, Lig: 2})
	attendu := domain.BornesMonde{MinX: -0.5, MinY: 1, MaxX: 0, MaxY: 1.5, Valide: true}
	if b != attendu {
		t.Fatalf("BornesDe(-1,2) = %+v, attendu %+v", b, attendu)
	}
}

func TestUnionBornes(t *testing.T) {
	a := domain.BornesMonde{MinX: -2, MinY: 0, MaxX: 3, MaxY: 4, Valide: true}
	b := domain.BornesMonde{MinX: 1, MinY: -5, MaxX: 10, MaxY: 2, Valide: true}

	got := UnionBornes(a, b)
	attendu := domain.BornesMonde{MinX: -2, MinY: -5, MaxX: 10, MaxY: 4, Valide: true}
	if got != attendu {
		t.Fatalf("UnionBornes = %+v, attendu %+v", got, attendu)
	}

	// Des bornes non valides sont neutres : elles ne deplacent aucun bord.
	if got := UnionBornes(domain.BornesMonde{}, a); got != a {
		t.Fatalf("UnionBornes(vide, a) = %+v, attendu %+v", got, a)
	}
	if got := UnionBornes(a, domain.BornesMonde{}); got != a {
		t.Fatalf("UnionBornes(a, vide) = %+v, attendu %+v", got, a)
	}
	if got := UnionBornes(domain.BornesMonde{}, domain.BornesMonde{}); got.Valide {
		t.Fatalf("UnionBornes(vide, vide) : Valide = true, attendu false")
	}
	// Une borne « vide » n'est PAS le rectangle a l'origine : sans le drapeau Valide,
	// l'union ci-dessus tirerait le cadre vers (0,0).
	if got := UnionBornes(domain.BornesMonde{}, b); got.MinY != -5 || got.MaxY != 2 {
		t.Fatalf("UnionBornes(vide, b) = %+v, attendu les bornes de b", got)
	}
}
