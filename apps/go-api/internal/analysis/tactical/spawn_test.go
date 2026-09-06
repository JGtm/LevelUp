package tactical

import (
	"math"
	"testing"
)

// pointsAutour pose `n` reapparitions d'un match chacune, serrees autour de (x, y) : elles
// tombent dans des cellules VOISINES, jamais dans la meme — c'est la situation reelle, et
// c'est elle qui decide ou porte le plancher.
func pointsAutour(x, y float64, matchs ...string) []PointSpawn {
	out := make([]PointSpawn, 0, len(matchs))
	for i, m := range matchs {
		out = append(out, PointSpawn{MatchID: m, X: x + float64(i)*0.4, Y: y})
	}
	return out
}

// TestGrappes_DeuxNuagesSepares — LE CAS DE REFERENCE : deux amas distants de trois matchs
// chacun rendent DEUX grappes.
func TestGrappes_DeuxNuagesSepares(t *testing.T) {
	g := GrilleParDefaut()
	pts := append(
		pointsAutour(0.25, 0.25, "m1", "m2", "m3"),
		pointsAutour(50.25, 50.25, "m1", "m2", "m3")...,
	)
	out := GrappesDeSpawn(g, pts, nil)
	if len(out) != 2 {
		t.Fatalf("grappes = %d, attendu 2 : %+v", len(out), out)
	}
	for _, gr := range out {
		if gr.Matchs != 3 {
			t.Fatalf("grappe %s vue dans %d matchs, attendu 3", gr.ID, gr.Matchs)
		}
	}
	if out[0].ID == out[1].ID {
		t.Fatalf("deux grappes distinctes partagent un identifiant : %s", out[0].ID)
	}
}

// TestGrappes_SousLePlancher — un amas vu dans DEUX matchs n'existe pas. Il n'est pas rendu
// « faible » : il est absent. Un amas vu deux fois ne dit pas ou le jeu fait naitre.
func TestGrappes_SousLePlancher(t *testing.T) {
	out := GrappesDeSpawn(GrilleParDefaut(), pointsAutour(0.25, 0.25, "m1", "m2"), nil)
	if len(out) != 0 {
		t.Fatalf("grappes = %+v, attendu aucune sous le plancher de %d matchs distincts",
			out, PlancherMatchsParCellule)
	}
}

// TestGrappes_LePlancherPorteSurLAMAS_PasSurLaCellule — LA LECTURE QUI FAIT QUE LA MESURE
// MESURE QUELQUE CHOSE.
//
// Sur une grille de 0,5 m, trois reapparitions de trois matchs differents tombent dans
// trois cellules VOISINES. Si le plancher s'appliquait cellule par cellule AVANT de
// connecter, chaque cellule ne compterait qu'un match et il ne resterait rien : tous les
// spawns du jeu disparaitraient. Ce test fige la lecture retenue.
func TestGrappes_LePlancherPorteSurLAMAS_PasSurLaCellule(t *testing.T) {
	g := GrilleParDefaut()
	pts := pointsAutour(0.25, 0.25, "m1", "m2", "m3")
	// Verification de la premisse : les trois points sont bien dans des cellules
	// DIFFERENTES, sans quoi le test ne prouverait pas ce qu'il annonce.
	cellules := map[Cellule]bool{}
	for _, p := range pts {
		c, _ := g.Cellule(p.X, p.Y)
		cellules[c] = true
	}
	if len(cellules) < 2 {
		t.Fatalf("les points de la fixture tombent dans %d cellule(s) : le test ne prouve rien",
			len(cellules))
	}
	out := GrappesDeSpawn(g, pts, nil)
	if len(out) != 1 || out[0].Matchs != 3 {
		t.Fatalf("grappes = %+v, attendu UNE grappe a 3 matchs (le plancher porte sur l'amas)", out)
	}
}

// TestGrappes_NommageParLeCalloutLePlusProche — le nom vient du callout le plus proche du
// barycentre, jamais d'un catalogue manuel.
func TestGrappes_NommageParLeCalloutLePlusProche(t *testing.T) {
	zones := []ZoneNommee{
		{Nom: "Base rouge", X: 0, Y: 0},
		{Nom: "Rampe", X: 40, Y: 40},
		{Nom: "Base bleue", X: 100, Y: 100},
	}
	out := GrappesDeSpawn(GrilleParDefaut(), pointsAutour(50.25, 50.25, "m1", "m2", "m3"), zones)
	if len(out) != 1 {
		t.Fatalf("grappes = %+v, attendu 1", out)
	}
	if out[0].Nom != "Rampe" {
		t.Fatalf("nom = %q, attendu « Rampe » (le callout le plus proche du barycentre)", out[0].Nom)
	}
}

// TestGrappes_SansCallout_AucunNomInvente — une carte hors catalogue rend des grappes
// MUETTES. Un nom de repli afficherait un lieu que le jeu ne prononce pas.
func TestGrappes_SansCallout_AucunNomInvente(t *testing.T) {
	out := GrappesDeSpawn(GrilleParDefaut(), pointsAutour(0.25, 0.25, "m1", "m2", "m3"), nil)
	if len(out) != 1 || out[0].Nom != "" {
		t.Fatalf("grappes = %+v, attendu une grappe sans nom", out)
	}
	// Une zone SANS libelle ne nomme rien non plus (le catalogue en porte : Forge).
	muettes := []ZoneNommee{{Nom: "", X: 0, Y: 0}}
	out = GrappesDeSpawn(GrilleParDefaut(), pointsAutour(0.25, 0.25, "m1", "m2", "m3"), muettes)
	if len(out) != 1 || out[0].Nom != "" {
		t.Fatalf("grappes = %+v, attendu une grappe sans nom", out)
	}
}

// TestGrappes_IdentifiantStableEntreDeuxAppels — L'ID EST UNE POSITION, PAS UN RANG.
//
// Un index de tableau change des qu'un match entre dans le filtre, et le lien `?spawn=<id>`
// d'un utilisateur designerait alors un autre amas. Ici le MEME amas est lu deux fois, avec
// des points donnes dans un ORDRE different et un second amas ajoute : son identifiant ne
// doit pas bouger.
func TestGrappes_IdentifiantStableEntreDeuxAppels(t *testing.T) {
	g := GrilleParDefaut()
	amas := pointsAutour(0.25, 0.25, "m1", "m2", "m3")
	premier := GrappesDeSpawn(g, amas, nil)
	if len(premier) != 1 {
		t.Fatalf("grappes = %+v", premier)
	}

	// LE SECOND AMAS EST POSE EN COORDONNEES NEGATIVES, ET C'EST LE POINT DU TEST : il
	// passe DEVANT l'amas d'origine dans TOUS les ordres possibles — l'ordre des cellules
	// (colonnes negatives d'abord) comme celui du tri final (4 matchs contre 3). Un
	// identifiant derive d'un RANG changerait donc forcement ; un identifiant derive de la
	// position ne bouge pas. Pose a (80, 80), l'amas d'origine serait reste premier et la
	// mutation « id = index » aurait survecu — verifie en la jouant.
	melange := []PointSpawn{amas[2], amas[0], amas[1]}
	avecUnAutreAmas := append(melange, pointsAutour(-50.25, -50.25, "m1", "m2", "m3", "m4")...)
	second := GrappesDeSpawn(g, avecUnAutreAmas, nil)
	if len(second) != 2 {
		t.Fatalf("grappes = %+v, attendu 2", second)
	}
	if second[0].Matchs != 4 {
		t.Fatalf("l'amas a 4 matchs n'est pas en tete : %+v", second)
	}
	if second[1].ID != premier[0].ID {
		t.Fatalf("identifiant de l'amas d'origine = %q puis %q : il a bouge alors que l'amas "+
			"n'a pas change", premier[0].ID, second[1].ID)
	}
}

// TestGrappes_HuitVoisinage — un amas pose EN DIAGONALE est UN amas. En 4-voisinage il se
// scinderait en autant de grappes que de cellules, et un couloir de spawn oblique
// disparaitrait sous le plancher.
func TestGrappes_HuitVoisinage(t *testing.T) {
	g := GrilleParDefaut()
	// Trois points strictement en diagonale, un par match : cellules (0,0), (1,1), (2,2).
	pts := []PointSpawn{
		{MatchID: "m1", X: 0.25, Y: 0.25},
		{MatchID: "m2", X: 0.75, Y: 0.75},
		{MatchID: "m3", X: 1.25, Y: 1.25},
	}
	out := GrappesDeSpawn(g, pts, nil)
	if len(out) != 1 || out[0].Matchs != 3 {
		t.Fatalf("grappes = %+v, attendu UNE grappe a 3 matchs (8-voisinage)", out)
	}
	if len(out[0].Cellules) != 3 {
		t.Fatalf("cellules de l'amas = %+v, attendu 3", out[0].Cellules)
	}
}

// TestGrappes_PositionNonFinie — un decodage qui derape ne fabrique pas d'amas.
func TestGrappes_PositionNonFinie(t *testing.T) {
	nan := math.NaN()
	pts := []PointSpawn{
		{MatchID: "m1", X: nan, Y: nan},
		{MatchID: "m2", X: nan, Y: nan},
		{MatchID: "m3", X: nan, Y: nan},
	}
	if out := GrappesDeSpawn(GrilleParDefaut(), pts, nil); len(out) != 0 {
		t.Fatalf("grappes = %+v, attendu aucune", out)
	}
}
