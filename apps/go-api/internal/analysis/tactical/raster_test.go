package tactical

import (
	"math"
	"testing"

	"levelup/go-api/internal/domain"
)

// pt : raccourci de lecture pour les jeux poses a la main.
func pt(match string, x, y float64) domain.PositionSample {
	return domain.PositionSample{MatchID: match, X: x, Y: y}
}

// TestRasterise_ComptesExacts : cellules posees a la main, comptes verifies un par un.
//
// Disposition (pas de 0,5 m) :
//
//	cellule (0,0)  : m1 x2, m2 x1, m3 x1  -> 4 passages, 3 matchs distincts
//	cellule (2,0)  : m1 x1                -> 1 passage,  1 match
//	cellule (-1,-1): m2 x2                -> 2 passages, 1 match
func TestRasterise_ComptesExacts(t *testing.T) {
	r := Rasterise(GrilleParDefaut(), []domain.PositionSample{
		pt("m1", 0.1, 0.1), pt("m1", 0.4, 0.2), pt("m2", 0.0, 0.49), pt("m3", 0.3, 0.3),
		pt("m1", 1.2, 0.2),
		pt("m2", -0.1, -0.1), pt("m2", -0.4, -0.5),
	})

	if got := r.NbCellules(); got != 3 {
		t.Fatalf("NbCellules = %d, attendu 3", got)
	}
	if got := r.NbMatchs(); got != 3 {
		t.Fatalf("NbMatchs = %d, attendu 3", got)
	}

	cas := []struct {
		c           Cellule
		occurrences int
		matchs      int
	}{
		{Cellule{Col: 0, Lig: 0}, 4, 3},
		{Cellule{Col: 2, Lig: 0}, 1, 1},
		{Cellule{Col: -1, Lig: -1}, 2, 1},
	}
	for _, k := range cas {
		if got := r.Occurrences(k.c); got != k.occurrences {
			t.Fatalf("Occurrences%+v = %d, attendu %d", k.c, got, k.occurrences)
		}
		if got := r.MatchsDistincts(k.c); got != k.matchs {
			t.Fatalf("MatchsDistincts%+v = %d, attendu %d", k.c, got, k.matchs)
		}
	}
}

// TestRasterise_CelluleJamaisAtteinteNExistePas : une cellule au milieu du nuage, sans
// aucun passage, ne doit ni exister ni etre comptee. C'est la difference entre « pas de
// donnee » et « zero », et elle se voit ici : NbCellules ne compte que les alimentees.
func TestRasterise_CelluleJamaisAtteinteNExistePas(t *testing.T) {
	r := Rasterise(GrilleParDefaut(), []domain.PositionSample{
		pt("m1", 0.2, 0.2), // cellule (0,0)
		pt("m1", 1.2, 0.2), // cellule (2,0) — la cellule (1,0) est enjambee
	})

	vide := Cellule{Col: 1, Lig: 0}
	if _, existe := r.cellules[vide]; existe {
		t.Fatalf("la cellule %+v jamais atteinte existe dans la table", vide)
	}
	if got := r.Occurrences(vide); got != 0 {
		t.Fatalf("Occurrences%+v = %d, attendu 0", vide, got)
	}
	if got := r.NbCellules(); got != 2 {
		t.Fatalf("NbCellules = %d, attendu 2 (la cellule enjambee ne compte pas)", got)
	}
}

// TestRasterise_EcarteLesPositionsNonFinies : les points illisibles sont comptes a part, et
// un match qui n'a place aucun point valide n'entre pas au denominateur « par match ».
func TestRasterise_EcarteLesPositionsNonFinies(t *testing.T) {
	r := Rasterise(GrilleParDefaut(), []domain.PositionSample{
		pt("m1", 0.2, 0.2),
		pt("m1", math.NaN(), 0.2),
		pt("mort", math.Inf(1), math.Inf(-1)),
	})

	if got := r.PointsIgnores(); got != 2 {
		t.Fatalf("PointsIgnores = %d, attendu 2", got)
	}
	if got := r.NbCellules(); got != 1 {
		t.Fatalf("NbCellules = %d, attendu 1", got)
	}
	if got := r.NbMatchs(); got != 1 {
		t.Fatalf("NbMatchs = %d, attendu 1 (un match sans point valide ne compte pas)", got)
	}
}

func TestRasteriseAvecResultats_CompteLesCotes(t *testing.T) {
	r := RasteriseAvecResultats(GrilleParDefaut(), []domain.PositionSample{
		pt("v1", 0.2, 0.2), pt("v2", 0.2, 0.2), pt("d1", 0.2, 0.2), pt("n1", 0.2, 0.2),
	}, map[string]int{
		"v1": domain.OutcomeWin,
		"v2": domain.OutcomeWin,
		"d1": domain.OutcomeLoss,
		"n1": domain.OutcomeDraw,
	})

	if got := r.NbMatchsResultat(domain.OutcomeWin); got != 2 {
		t.Fatalf("NbMatchsResultat(win) = %d, attendu 2", got)
	}
	if got := r.NbMatchsResultat(domain.OutcomeLoss); got != 1 {
		t.Fatalf("NbMatchsResultat(loss) = %d, attendu 1", got)
	}
	// Un match sans resultat connu compte dans les passages, dans aucun cote.
	r2 := Rasterise(GrilleParDefaut(), []domain.PositionSample{pt("m1", 0.2, 0.2)})
	if got := r2.NbMatchsResultat(domain.OutcomeWin); got != 0 {
		t.Fatalf("NbMatchsResultat(win) sans table = %d, attendu 0", got)
	}
	if got := r2.NbMatchsResultat(domain.OutcomeUnknown); got != 1 {
		t.Fatalf("NbMatchsResultat(unknown) = %d, attendu 1", got)
	}
}

func TestRaster_Bornes(t *testing.T) {
	r := Rasterise(GrilleParDefaut(), []domain.PositionSample{
		pt("m1", 0.2, 0.2),   // cellule (0,0)   -> [0, 0.5[ x [0, 0.5[
		pt("m1", -0.6, 1.25), // cellule (-2,2)  -> [-1, -0.5[ x [1, 1.5[
	})
	got := r.Bornes()
	attendu := domain.BornesMonde{MinX: -1, MinY: 0, MaxX: 0.5, MaxY: 1.5, Valide: true}
	if got != attendu {
		t.Fatalf("Bornes = %+v, attendu %+v", got, attendu)
	}

	if vide := Rasterise(GrilleParDefaut(), nil).Bornes(); vide.Valide {
		t.Fatalf("Bornes d'un raster vide : Valide = true, attendu false")
	}
}
