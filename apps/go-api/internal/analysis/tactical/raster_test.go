package tactical

import (
	"errors"
	"math"
	"testing"

	"levelup/go-api/internal/domain"
)

// pt : raccourci de lecture pour les jeux poses a la main.
func pt(match string, x, y float64) domain.PositionSample {
	return domain.PositionSample{MatchID: match, X: x, Y: y}
}

// rasteriseOk : rasterise et echoue le test si l'univers est viole.
func rasteriseOk(t *testing.T, g Grille, matchs []string, points []domain.PositionSample) *Raster {
	t.Helper()
	r, err := Rasterise(g, matchs, points)
	if err != nil {
		t.Fatalf("Rasterise : %v", err)
	}
	return r
}

// rasteriseResOk : idem, avec les resultats de match (la table EST l'univers).
func rasteriseResOk(t *testing.T, g Grille, resultats map[string]int, points []domain.PositionSample) *Raster {
	t.Helper()
	r, err := RasteriseAvecResultats(g, resultats, points)
	if err != nil {
		t.Fatalf("RasteriseAvecResultats : %v", err)
	}
	return r
}

// TestRasterise_ComptesExacts : cellules posees a la main, comptes verifies un par un.
//
// Disposition (pas de 0,5 m) :
//
//	cellule (0,0)  : m1 x2, m2 x1, m3 x1  -> 4 passages, 3 matchs distincts
//	cellule (2,0)  : m1 x1                -> 1 passage,  1 match
//	cellule (-1,-1): m2 x2                -> 2 passages, 1 match
func TestRasterise_ComptesExacts(t *testing.T) {
	r := rasteriseOk(t, GrilleParDefaut(), []string{"m1", "m2", "m3"}, []domain.PositionSample{
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
	r := rasteriseOk(t, GrilleParDefaut(), []string{"m1"}, []domain.PositionSample{
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

// TestRasterise_EcarteLesPositionsNonFinies : les points illisibles sont comptes a part —
// ils ne sont PAS un zero legitime, ils sont un decodage rate. Le match reste dans l'univers
// et continue de compter au denominateur, parce que c'est l'appelant qui declare l'univers.
func TestRasterise_EcarteLesPositionsNonFinies(t *testing.T) {
	r := rasteriseOk(t, GrilleParDefaut(), []string{"m1", "illisible"}, []domain.PositionSample{
		pt("m1", 0.2, 0.2),
		pt("m1", math.NaN(), 0.2),
		pt("illisible", math.Inf(1), math.Inf(-1)),
	})

	if got := r.PointsIgnores(); got != 2 {
		t.Fatalf("PointsIgnores = %d, attendu 2", got)
	}
	if got := r.NbCellules(); got != 1 {
		t.Fatalf("NbCellules = %d, attendu 1", got)
	}
	if got := r.NbMatchs(); got != 2 {
		t.Fatalf("NbMatchs = %d, attendu 2 (l'univers est declare, pas deduit des points)", got)
	}
}

// TestRasterise_UnMatchSansPointCompteAuDenominateur : le coeur de la regle. Univers de dix
// matchs, cinq passages dans une cellule, un match (au moins) sans le moindre point : la
// valeur par match vaut 5/10 = 0,5. Deduire l'univers des points rendrait 5/9 ou 5/5.
func TestRasterise_UnMatchSansPointCompteAuDenominateur(t *testing.T) {
	univers := []string{"m1", "m2", "m3", "m4", "m5", "m6", "m7", "m8", "m9", "muet"}
	var points []domain.PositionSample
	for _, m := range []string{"m1", "m2", "m3", "m4", "m5"} {
		points = append(points, pt(m, 0.2, 0.2)) // cellule (0,0) : 5 passages, 5 matchs
	}
	for _, m := range []string{"m6", "m7", "m8", "m9"} {
		points = append(points, pt(m, 1.2, 0.2)) // cellule (2,0) : 4 passages, 4 matchs
	}
	// "muet" a ete joue et retenu, mais rien ne s'y est passe sur cette lecture.

	r := rasteriseOk(t, GrilleParDefaut(), univers, points)
	if got := r.NbMatchs(); got != 10 {
		t.Fatalf("NbMatchs = %d, attendu 10 (le match muet compte)", got)
	}

	cellules := r.Cellules()
	if got := cellule(t, cellules, 0, 0).Valeur; got != 0.5 {
		t.Fatalf("Valeur (0,0) = %v, attendu 0.5 (5 passages / 10 matchs retenus)", got)
	}
	if got := cellule(t, cellules, 2, 0).Valeur; got != 0.4 {
		t.Fatalf("Valeur (2,0) = %v, attendu 0.4 (4 passages / 10 matchs retenus)", got)
	}
}

// TestRasterise_MatchHorsUniversEstUneErreur : un point d'un match que l'appelant n'a pas
// retenu est un bug de filtre. Ni avale, ni compte en silence.
func TestRasterise_MatchHorsUniversEstUneErreur(t *testing.T) {
	_, err := Rasterise(GrilleParDefaut(), []string{"m1"}, []domain.PositionSample{
		pt("m1", 0.2, 0.2),
		pt("intrus", 0.2, 0.2),
	})
	if !errors.Is(err, ErrMatchHorsUnivers) {
		t.Fatalf("err = %v, attendu ErrMatchHorsUnivers", err)
	}

	_, err = RasteriseAvecResultats(GrilleParDefaut(), map[string]int{"m1": domain.OutcomeWin},
		[]domain.PositionSample{pt("intrus", 0.2, 0.2)})
	if !errors.Is(err, ErrMatchHorsUnivers) {
		t.Fatalf("err = %v, attendu ErrMatchHorsUnivers", err)
	}
}

func TestRasteriseAvecResultats_CompteLesCotesSurLUnivers(t *testing.T) {
	r := rasteriseResOk(t, GrilleParDefaut(), map[string]int{
		"v1":     domain.OutcomeWin,
		"v2":     domain.OutcomeWin,
		"muette": domain.OutcomeWin, // retenue, aucun point sur cette lecture
		"d1":     domain.OutcomeLoss,
		"n1":     domain.OutcomeDraw,
	}, []domain.PositionSample{
		pt("v1", 0.2, 0.2), pt("v2", 0.2, 0.2), pt("d1", 0.2, 0.2), pt("n1", 0.2, 0.2),
	})

	if got := r.NbMatchsResultat(domain.OutcomeWin); got != 3 {
		t.Fatalf("NbMatchsResultat(win) = %d, attendu 3 (la victoire muette compte)", got)
	}
	if got := r.NbMatchsResultat(domain.OutcomeLoss); got != 1 {
		t.Fatalf("NbMatchsResultat(loss) = %d, attendu 1", got)
	}

	// Un univers declare sans resultats : tout le monde est OutcomeUnknown.
	r2 := rasteriseOk(t, GrilleParDefaut(), []string{"m1"}, []domain.PositionSample{pt("m1", 0.2, 0.2)})
	if got := r2.NbMatchsResultat(domain.OutcomeWin); got != 0 {
		t.Fatalf("NbMatchsResultat(win) sans table = %d, attendu 0", got)
	}
	if got := r2.NbMatchsResultat(domain.OutcomeUnknown); got != 1 {
		t.Fatalf("NbMatchsResultat(unknown) = %d, attendu 1", got)
	}
}

// TestRaster_BornesSurLesCellulesLisibles : le cadre suit ce que le peintre pose, donc les
// cellules qui passent le plancher de rarete. Sur Dredge, des cellules vues une fois dans un
// seul match s'etendent a 268 m du centre : les inclure rendrait un cadre quatorze fois trop
// large pour la carte peinte.
func TestRaster_BornesSurLesCellulesLisibles(t *testing.T) {
	univers := []string{"m1", "m2", "m3"}
	var points []domain.PositionSample
	for _, m := range univers {
		points = append(points,
			pt(m, 0.2, 0.2),   // cellule (0,0)  -> [0, 0.5[ x [0, 0.5[
			pt(m, -0.6, 1.25), // cellule (-2,2) -> [-1, -0.5[ x [1, 1.5[
		)
	}
	points = append(points, pt("m1", 200, 200)) // vue dans UN seul match : hors plancher

	r := rasteriseOk(t, GrilleParDefaut(), univers, points)
	if got := r.NbCellules(); got != 3 {
		t.Fatalf("NbCellules = %d, attendu 3 (la cellule isolee est bien alimentee)", got)
	}

	got := r.Bornes()
	attendu := domain.BornesMonde{MinX: -1, MinY: 0, MaxX: 0.5, MaxY: 1.5, Valide: true}
	if got != attendu {
		t.Fatalf("Bornes = %+v, attendu %+v (la cellule a 200 m ne passe pas le plancher)", got, attendu)
	}

	vide, err := Rasterise(GrilleParDefaut(), nil, nil)
	if err != nil {
		t.Fatalf("Rasterise(vide) : %v", err)
	}
	if vide.Bornes().Valide {
		t.Fatalf("Bornes d'un raster vide : Valide = true, attendu false")
	}
}
