package himap

import "testing"

// Temoins unitaires du lecteur sddt et de la pose d'eau — donnees synthetiques, aucun
// fichier du jeu. Les temoins sur les octets reels vivent dans `sddt_gamefiles_test.go`.

// volumeEauBoite : un volume d'eau en boite [x0;x1] x [y0;y1] x [z0;z1], plans orientes
// comme dans le tag (interieur = n.p <= d, normales vers l'EXTERIEUR).
func volumeEauBoite(x0, x1, y0, y1, z0, z1 float64) VolumeEauSddt {
	return VolumeEauSddt{
		AABBMin: [3]float64{x0, y0, z0},
		AABBMax: [3]float64{x1, y1, z1},
		Plans: [][4]float64{
			{1, 0, 0, x1}, {-1, 0, 0, -x0},
			{0, 1, 0, y1}, {0, -1, 0, -y0},
			{0, 0, 1, z1}, {0, 0, -1, -z0},
		},
	}
}

// TestVolumeEauOrientation — l'interieur d'un volume est `n.p <= d`, et le sens INVERSE ne
// doit PAS l'accepter. C'est le temoin qui a etabli l'orientation sur les 233 prismes reels
// (233/233 dans un sens, 0/233 dans l'autre) : la version synthetique JOUE la mutation en
// inversant chaque plan.
func TestVolumeEauOrientation(t *testing.T) {
	v := volumeEauBoite(0, 4, 0, 4, -4.5, -3.5)
	centre := [3]float64{2, 2, -4}
	if !v.Contient(centre, 0) {
		t.Fatal("le centre du volume doit etre dedans")
	}
	if v.Contient([3]float64{2, 2, 0}, 0) {
		t.Fatal("un point au-dessus du toit ne doit pas etre dedans")
	}
	if v.Contient([3]float64{5, 2, -4}, 0) {
		t.Fatal("un point hors de l'emprise ne doit pas etre dedans")
	}
	// LA MUTATION, JOUEE : chaque plan retourne (sens inverse). Le centre doit sortir.
	inverse := v
	inverse.Plans = nil
	for _, p := range v.Plans {
		inverse.Plans = append(inverse.Plans, [4]float64{-p[0], -p[1], -p[2], -p[3]})
	}
	if inverse.Contient(centre, 0) {
		t.Fatal("le sens inverse accepte le centre : le temoin d'orientation ne separe pas")
	}
}

// TestCoquilleContientEtDedoublonne — l'interieur de la coquille est `n.p >= d` (normales
// vers l'INTERIEUR, orientation opposee aux volumes), et deux triangles-frontieres portes
// par le meme plan ne donnent qu'UN plan.
func TestCoquilleContientEtDedoublonne(t *testing.T) {
	s := Sddt{Frontieres: []FrontiereSddt{
		{Normale: [3]float64{0, 0, 1}, D: -10},  // plancher : z >= -10
		{Normale: [3]float64{0, 0, -1}, D: -20}, // plafond : -z >= -20 soit z <= 20
		{Normale: [3]float64{0, 0, 1}, D: -10},  // le meme plancher, second triangle
	}}
	c := s.Coquille()
	if len(c) != 2 {
		t.Fatalf("2 plans distincts attendus apres dedoublonnage, obtenu %d", len(c))
	}
	if !c.Contient([3]float64{0, 0, 0}, 0) {
		t.Fatal("un point entre plancher et plafond doit etre dans la coquille")
	}
	if c.Contient([3]float64{0, 0, -15}, 0) {
		t.Fatal("un point sous le plancher ne doit pas etre dans la coquille")
	}
	if c.Contient([3]float64{0, 0, 25}, 0) {
		t.Fatal("un point au-dessus du plafond ne doit pas etre dans la coquille")
	}
}

// TestPoseEauHabillePasLaMatiere — les trois regles de la pose d'eau, et l'invariant qui
// garde le banc : le z-buffer ne bouge PAS.
//
//  1. la matiere SOUS le toit du volume (le lit de la riviere) devient de l'eau ;
//  2. la matiere AU-DESSUS (un pont) garde son pixel ;
//  3. une cellule SANS matiere dans le volume est de l'eau quand meme — l'eau n'est pas
//     dans les instances de rendu, elle comble ses propres trous.
//
// MUTATION QUI DOIT FAIRE ROUGIR : PoseEau qui ecrirait dans r.z (la comparaison
// avant/apres tombe), ou la marge supprimee (le pont devient de l'eau).
func TestPoseEauHabillePasLaMatiere(t *testing.T) {
	r := NewRendu([2]float64{0, 0}, [2]float64{4, 4}, 1)
	// Le lit : un sol a -4 m sur la moitie est (x dans [2;4]).
	lit := &Mesh{
		Vertices:  [][3]float64{{2, 0, -4}, {4, 0, -4}, {4, 4, -4}, {2, 4, -4}},
		Triangles: [][3]int{{0, 1, 2}, {0, 2, 3}},
	}
	r.AddMesh(lit, instanceIdentite())
	// Le pont : une dalle a +2 m sur le quart sud-ouest.
	pont := &Mesh{
		Vertices:  [][3]float64{{0, 0, 2}, {2, 0, 2}, {2, 2, 2}, {0, 2, 2}},
		Triangles: [][3]int{{0, 1, 2}, {0, 2, 3}},
	}
	r.AddMesh(pont, instanceIdentite())

	avant := make([]float64, len(r.z))
	copy(avant, r.z)

	r.PoseEau([]VolumeEauSddt{volumeEauBoite(0, 4, 0, 4, -4.5, -3.5)})

	for k := range avant {
		if avant[k] != r.z[k] {
			t.Fatalf("PoseEau a modifie le z-buffer (cellule %d) : le terrain doit rester intact", k)
		}
	}
	if !r.Eau(3, 3) {
		t.Error("le lit sous le toit du volume doit etre de l'eau")
	}
	if r.Eau(1, 1) {
		t.Error("le pont au-dessus du toit doit garder son pixel de matiere")
	}
	if !r.Eau(1, 3) {
		t.Error("une cellule sans matiere dans le volume doit etre de l'eau")
	}
	if r.Eau(-1, 0) || r.Eau(0, r.NY) {
		t.Error("hors grille, jamais d'eau")
	}
}

// TestBordEau — la berge est une cellule d'eau dont un voisin n'en est pas.
func TestBordEau(t *testing.T) {
	r := NewRendu([2]float64{0, 0}, [2]float64{4, 4}, 1)
	r.PoseEau([]VolumeEauSddt{volumeEauBoite(0, 3, 0, 3, -1, 1)})
	if !r.BordEau(2, 1) {
		t.Error("la lisiere de l'eau doit etre une berge")
	}
	if r.BordEau(1, 1) {
		t.Error("la pleine eau n'est pas une berge")
	}
	if r.BordEau(4, 4) {
		t.Error("une cellule sans eau n'est pas une berge")
	}
}
