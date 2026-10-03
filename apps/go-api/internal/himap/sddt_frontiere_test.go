package himap

import "testing"

// boiteFermee rend les 12 triangles d'un pave axe. Normales et `d` sont calcules comme le fait
// le jeu (interieur = n.p >= d), pour que les DEUX lectures soient comparables.
func boiteFermee(lo, hi [3]float64) []FrontiereSddt {
	c := [8][3]float64{
		{lo[0], lo[1], lo[2]}, {hi[0], lo[1], lo[2]}, {hi[0], hi[1], lo[2]}, {lo[0], hi[1], lo[2]},
		{lo[0], lo[1], hi[2]}, {hi[0], lo[1], hi[2]}, {hi[0], hi[1], hi[2]}, {lo[0], hi[1], hi[2]},
	}
	faces := [12][3]int{
		{0, 2, 1}, {0, 3, 2}, {4, 5, 6}, {4, 6, 7},
		{0, 1, 5}, {0, 5, 4}, {2, 3, 7}, {2, 7, 6},
		{1, 2, 6}, {1, 6, 5}, {0, 4, 7}, {0, 7, 3},
	}
	var out []FrontiereSddt
	for _, f := range faces {
		out = append(out, FrontiereSddt{Sommets: [3][3]float64{c[f[0]], c[f[1]], c[f[2]]}})
	}
	return out
}

// TestFrontierePariteEstExacteSurUneBoite — le cas convexe, ou les deux lectures s'accordent.
func TestFrontierePariteEstExacteSurUneBoite(t *testing.T) {
	s := Sddt{Frontieres: boiteFermee([3]float64{0, 0, 0}, [3]float64{10, 10, 10})}
	if !s.ContientFrontiere([3]float64{5, 5, 5}) {
		t.Error("le centre de la boite est DEDANS")
	}
	if s.ContientFrontiere([3]float64{15, 5, 5}) {
		t.Error("un point a droite de la boite est DEHORS")
	}
	if s.ContientFrontiere([3]float64{-5, 5, 5}) {
		t.Error("un point a gauche de la boite est DEHORS")
	}
}

// TestFrontiereParitieSepareSurUnNonConVEXE — LE temoin.
//
// Deux boites disjointes forment une frontiere NON CONVEXE. Un point situe entre les deux est
// DEHORS. L'intersection des demi-espaces (`CoquilleSddt`) ne peut PAS le voir : elle rend
// meme un volume VIDE, donc elle repond « dehors » partout, y compris au centre des boites —
// et c'est exactement ce qui effacait 100 % de behemoth.
//
// MUTATION QUI LE FAIT ROUGIR : remplacer `ContientFrontiere` par la lecture par demi-espaces.
func TestFrontiereParitieSepareSurUnNonConVEXE(t *testing.T) {
	var fs []FrontiereSddt
	fs = append(fs, boiteFermee([3]float64{0, 0, 0}, [3]float64{10, 10, 10})...)
	fs = append(fs, boiteFermee([3]float64{20, 0, 0}, [3]float64{30, 10, 10})...)
	s := Sddt{Frontieres: fs}

	if !s.ContientFrontiere([3]float64{5, 5, 5}) {
		t.Error("le centre de la premiere boite est DEDANS")
	}
	if !s.ContientFrontiere([3]float64{25, 5, 5}) {
		t.Error("le centre de la seconde boite est DEDANS")
	}
	if s.ContientFrontiere([3]float64{15, 5, 5}) {
		t.Error("le vide ENTRE les deux boites est DEHORS")
	}

	// La lecture concurrente, elle, se trompe — et c'est ce contraste qui fait le temoin.
	coq := Sddt{Frontieres: fs}.CoquilleDepuisPlans()
	if coq.Contient([3]float64{5, 5, 5}, 0) {
		t.Error("le temoin ne separe pas : l'intersection des demi-espaces devrait ECHOUER ici, " +
			"sinon elle n'aurait jamais efface behemoth")
	}
}
