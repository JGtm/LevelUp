package tactical

// merge_cellules_test.go — les lectures agregees (Cellules, CellulesSignees) : planchers,
// valeur par match, ecart de taux, tri. Les tests de Somme et les helpers partages vivent
// dans merge_test.go (seuil de 500 lignes par fichier).

import (
	"testing"

	"levelup/go-api/internal/domain"
)

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
