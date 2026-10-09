package himap

import "testing"

// LE SEUIL DE COUVERTURE PAR CARTE — temoin de la campagne du 2026-08-30.
//
// Ce reglage existe parce que la couverture, seule, classe mal certaines cartes : Dredge la
// donne a 28,7 % — dans la population des cartes VALIDEES a l oeil (19 a 28 %) — alors que son
// ecart median entre une ancre et la surface dessinee vaut -19,84 m, c est-a-dire qu on regarde
// le terrain de vingt metres au-dessus. L utilisateur, lui, voit « que des toits ».
//
// Ce que ce temoin fige : le defaut reste le seuil universel, et un reglage par carte ne
// s applique QUE s il est strictement positif. Un zero, un negatif ou une carte sans reglage
// doivent tous rendre `SeuilCarteCouverte` — sinon une carte non declaree changerait de voie de
// rendu sans que personne l ait demande.
func TestSeuilCouvertureParCarte(t *testing.T) {
	precedent := SeuilCouvertureCarte
	defer func() { SeuilCouvertureCarte = precedent }()

	for _, inerte := range []float64{0, -1, -0.25} {
		SeuilCouvertureCarte = inerte
		if got := SeuilCouvertureEffectif(); got != SeuilCarteCouverte {
			t.Fatalf("seuil %v devrait etre inerte et rendre %v, rendu %v",
				inerte, SeuilCarteCouverte, got)
		}
	}

	SeuilCouvertureCarte = 0.25
	if got := SeuilCouvertureEffectif(); got != 0.25 {
		t.Fatalf("seuil arme a 0,25 non pris en compte : %v", got)
	}
	if 0.25 >= SeuilCarteCouverte {
		t.Fatalf("le cas d usage est d ABAISSER le seuil ; 0,25 doit rester sous %v",
			SeuilCarteCouverte)
	}
}
