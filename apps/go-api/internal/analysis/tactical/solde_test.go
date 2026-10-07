package tactical

// solde_test.go — la lecture « Solde frags − morts » : plancher sur l'union des deux faces,
// valeur ramenée aux matchs de l'univers, comptes par face, somme de rasters.

import (
	"math"
	"testing"

	"levelup/go-api/internal/domain"
)

func soldeOk(t *testing.T, matchs []string, frags, morts []domain.PositionSample) *Raster {
	t.Helper()
	r, err := RasteriseSolde(GrilleParDefaut(), matchs, frags, morts)
	if err != nil {
		t.Fatalf("RasteriseSolde : %v", err)
	}
	return r
}

func repete(match string, x, y float64, n int) []domain.PositionSample {
	out := make([]domain.PositionSample, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, pt(match, x, y))
	}
	return out
}

// TestCellulesSolde_PlancherSurLUnionEtValeurParMatch : jeu posé à la main.
//
//	cellule (0,0) : frags m1 x2, m2 x1 ; morts m3 x1 -> 3 matchs distincts (union) -> RETENUE
//	cellule (2,0) : frags m1 x1 ; morts m2 x4        -> 2 matchs distincts         -> RETIRÉE
//
// L'univers compte 5 matchs (m4 et m5 sans aucun point) : la valeur de (0,0) vaut
// 3/5 − 1/5 = 0,4, et non (3 − 1)/3 qui serait la moyenne sur les seuls matchs de la cellule.
// Aucune des deux faces n'atteint seule trois matchs : un plancher par face retirerait (0,0).
func TestCellulesSolde_PlancherSurLUnionEtValeurParMatch(t *testing.T) {
	var frags, morts []domain.PositionSample
	frags = append(frags, repete("m1", 0.2, 0.2, 2)...)
	frags = append(frags, repete("m2", 0.2, 0.2, 1)...)
	morts = append(morts, repete("m3", 0.2, 0.2, 1)...)
	frags = append(frags, repete("m1", 1.2, 0.2, 1)...)
	morts = append(morts, repete("m2", 1.2, 0.2, 4)...)

	r := soldeOk(t, []string{"m1", "m2", "m3", "m4", "m5"}, frags, morts)
	cellules := r.CellulesSolde()
	if len(cellules) != 1 {
		t.Fatalf("len(CellulesSolde) = %d, attendu 1 : %+v", len(cellules), cellules)
	}
	absente(t, cellules, 2, 0)
	c := cellule(t, cellules, 0, 0)
	if c.Frags != 3 || c.Morts != 1 {
		t.Fatalf("faces = %d frags / %d morts, attendu 3 / 1", c.Frags, c.Morts)
	}
	if c.Brut != 2 {
		t.Fatalf("Brut = %v, attendu 2 (frags − morts)", c.Brut)
	}
	if c.Matchs != 3 {
		t.Fatalf("Matchs = %d, attendu 3 (union des deux faces)", c.Matchs)
	}
	if math.Abs(c.Valeur-0.4) > 1e-12 {
		t.Fatalf("Valeur = %v, attendu 0,4 (3/5 − 1/5)", c.Valeur)
	}
}

// TestCellulesSolde_SigneNegatif : plus de morts que de frags donne une valeur NÉGATIVE — c'est
// une lecture signée, l'échelle symétrique la peint du côté bas.
func TestCellulesSolde_SigneNegatif(t *testing.T) {
	var frags, morts []domain.PositionSample
	frags = append(frags, repete("m1", 0.2, 0.2, 1)...)
	morts = append(morts, repete("m2", 0.2, 0.2, 2)...)
	morts = append(morts, repete("m3", 0.2, 0.2, 2)...)

	c := cellule(t, soldeOk(t, []string{"m1", "m2", "m3", "m4"}, frags, morts).CellulesSolde(), 0, 0)
	if c.Brut != -3 || math.Abs(c.Valeur-(-0.75)) > 1e-12 {
		t.Fatalf("Brut / Valeur = %v / %v, attendu −3 / −0,75", c.Brut, c.Valeur)
	}
}

// TestRasteriseSolde_PositionsNonFiniesEcartees : un point illisible n'entre dans aucune face et
// se compte dans PointsIgnores, comme pour toute lecture.
func TestRasteriseSolde_PositionsNonFiniesEcartees(t *testing.T) {
	frags := append(repete("m1", 0.2, 0.2, 1), pt("m2", math.NaN(), 0.2))
	morts := append(repete("m2", 0.2, 0.2, 1), repete("m3", 0.2, 0.2, 1)...)
	r := soldeOk(t, []string{"m1", "m2", "m3"}, frags, morts)
	if r.PointsIgnores() != 1 {
		t.Fatalf("PointsIgnores = %d, attendu 1", r.PointsIgnores())
	}
	c := cellule(t, r.CellulesSolde(), 0, 0)
	if c.Frags != 1 || c.Morts != 2 {
		t.Fatalf("faces = %d / %d, attendu 1 / 2", c.Frags, c.Morts)
	}
}

// TestRasteriseSolde_MatchHorsUnivers : même refus que Rasterise.
func TestRasteriseSolde_MatchHorsUnivers(t *testing.T) {
	if _, err := RasteriseSolde(GrilleParDefaut(), []string{"m1"}, nil, repete("m9", 0.2, 0.2, 1)); err == nil {
		t.Fatal("un point d'un match hors univers doit être refusé")
	}
}

// TestSomme_ConserveLesFacesDuSolde : la somme de deux rasters de solde du même univers additionne
// aussi les comptes par face.
func TestSomme_ConserveLesFacesDuSolde(t *testing.T) {
	univers := []string{"m1", "m2", "m3"}
	a := soldeOk(t, univers, repete("m1", 0.2, 0.2, 2), repete("m2", 0.2, 0.2, 1))
	b := soldeOk(t, univers, repete("m3", 0.2, 0.2, 1), repete("m3", 0.2, 0.2, 3))
	s, err := Somme(a, b)
	if err != nil {
		t.Fatalf("Somme : %v", err)
	}
	c := cellule(t, s.CellulesSolde(), 0, 0)
	if c.Frags != 3 || c.Morts != 4 || c.Matchs != 3 {
		t.Fatalf("somme = %d frags / %d morts / %d matchs, attendu 3 / 4 / 3", c.Frags, c.Morts, c.Matchs)
	}
}
