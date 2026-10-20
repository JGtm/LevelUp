package himap

import (
	"math"
	"testing"
)

// Temoins de la voie de reference (rendu_reference.go) — chacun departage : la mutation qui
// retire la clause testee fait rougir l'assertion citee.

// quadPlat rend un maillage rectangulaire horizontal a l'altitude z.
func quadPlat(x0, y0, x1, y1, z float64) (*Mesh, Instance) {
	m := &Mesh{
		Vertices:  [][3]float64{{x0, y0, z}, {x1, y0, z}, {x1, y1, z}, {x0, y1, z}},
		Triangles: [][3]int{{0, 1, 2}, {0, 2, 3}},
	}
	return m, instanceIdentite()
}

// TestReferenceMontreLeSolSousLePlafond — une arene entierement couverte (sol a 0, toit a 8,
// ancre au sol) doit montrer le SOL. MUTATION QUI DOIT FAIRE ROUGIR : ne pas appeler
// AppliqueReference, ou inverser son critere — l'altitude lue redevient 8.
func TestReferenceMontreLeSolSousLePlafond(t *testing.T) {
	ancres := [][3]float64{{2, 2, AncrageDecalageSol}}
	s := NewSurfaceReference(ancres)
	r := NewRendu([2]float64{0, 0}, [2]float64{4, 4}, 1)
	r.ArmeReference(s)
	m, in := quadPlat(0, 0, 4, 4, 0)
	r.AddMesh(m, in)
	toit, _ := quadPlat(0, 0, 4, 4, 8)
	r.AddMesh(toit, in)

	matAvant := compteMatiere(r)
	taux, substituees, couverte := r.AppliqueReference(s, false)
	if !couverte || taux < 0.99 {
		t.Fatalf("arene entierement couverte : taux %.2f, couverte %v — attendu ~1 et true", taux, couverte)
	}
	if substituees == 0 {
		t.Fatal("aucune cellule substituee sur une arene couverte")
	}
	if z, ok := r.Altitude(1, 1); !ok || z != 0 {
		t.Fatalf("altitude (1,1) = %v (ok=%v) : le SOL (0) doit remplacer le toit (8)", z, ok)
	}
	if matApres := compteMatiere(r); matApres != matAvant {
		t.Fatalf("la substitution a change la silhouette : %d -> %d cellules de matiere", matAvant, matApres)
	}
}

// TestReferenceEpargneLaCarteNonCouverte — un surplomb qui ne couvre qu'un quart de la matiere
// (les rochers de Cliffhanger) ne declenche rien : l'image reste STRICTEMENT la meme.
// MUTATION QUI DOIT FAIRE ROUGIR : abaisser SeuilCarteCouverte sous 25 % — le rocher disparait.
func TestReferenceEpargneLaCarteNonCouverte(t *testing.T) {
	ancres := [][3]float64{{2, 2, AncrageDecalageSol}}
	s := NewSurfaceReference(ancres)
	r := NewRendu([2]float64{0, 0}, [2]float64{4, 4}, 1)
	r.ArmeReference(s)
	m, in := quadPlat(0, 0, 4, 4, 0)
	r.AddMesh(m, in)
	rocher, _ := quadPlat(0, 0, 2, 2, 8)
	r.AddMesh(rocher, in)

	taux, substituees, couverte := r.AppliqueReference(s, false)
	if couverte || substituees != 0 {
		t.Fatalf("un quart de couverture (taux %.2f) ne doit PAS declencher (couverte %v, %d substituees)",
			taux, couverte, substituees)
	}
	if z, ok := r.Altitude(1, 1); !ok || z != 8 {
		t.Fatalf("altitude (1,1) = %v (ok=%v) : le surplomb valide (8) doit rester", z, ok)
	}
}

// TestReferenceNeTouchePasHorsPortee — au-dela de PorteeAncre, la reference extrapole sans
// rien savoir : le decor y reste rendu par la surface la plus haute, meme sur carte couverte.
func TestReferenceNeTouchePasHorsPortee(t *testing.T) {
	ancres := [][3]float64{{2, 2, AncrageDecalageSol}}
	s := NewSurfaceReference(ancres)
	r := NewRendu([2]float64{0, 0}, [2]float64{40, 4}, 1)
	r.ArmeReference(s)
	m, in := quadPlat(0, 0, 40, 4, 0)
	r.AddMesh(m, in)
	toit, _ := quadPlat(0, 0, 40, 4, 8)
	r.AddMesh(toit, in)

	if _, _, couverte := r.AppliqueReference(s, false); !couverte {
		t.Fatal("toit integral : la carte doit etre couverte")
	}
	if z, ok := r.Altitude(10, 1); !ok || z != 0 {
		t.Fatalf("altitude (10,1) = %v (ok=%v) : dans la portee de l'ancre, le sol (0) doit se montrer", z, ok)
	}
	if z, ok := r.Altitude(35, 1); !ok || z != 8 {
		t.Fatalf("altitude (35,1) = %v (ok=%v) : hors de portee (%.0f m), le toit (8) doit rester", z, ok, PorteeAncre)
	}
}

// TestReferenceNonArmeeNeFaitRien — sans ArmeReference, AppliqueReference est un no-op declare.
func TestReferenceNonArmeeNeFaitRien(t *testing.T) {
	r := NewRendu([2]float64{0, 0}, [2]float64{4, 4}, 1)
	m, in := quadPlat(0, 0, 4, 4, 8)
	r.AddMesh(m, in)
	s := NewSurfaceReference([][3]float64{{2, 2, AncrageDecalageSol}})
	if taux, substituees, couverte := r.AppliqueReference(s, false); taux != 0 || substituees != 0 || couverte {
		t.Fatalf("reference non armee : (%v, %d, %v), attendu (0, 0, false)", taux, substituees, couverte)
	}
	if z, ok := r.Altitude(1, 1); !ok || z != 8 {
		t.Fatalf("altitude (1,1) = %v (ok=%v) : rien ne doit bouger", z, ok)
	}
}

func compteMatiere(r *Rendu) int {
	n := 0
	for k := range r.z {
		if !math.IsInf(r.z[k], -1) {
			n++
		}
	}
	return n
}

// TestReferenceSansPorteeCouvreToutLePlateau — avec sansPortee, la substitution ne s'arrete
// plus a PorteeAncre : c'est le reglage par carte `substitutionSansPortee`, arme sur Chasm
// dont l'arene est plus longue que la portee d'une ancre (130 m de matiere utile). Le meme
// decor qu'a TestReferenceNeTouchePasHorsPortee doit alors montrer le sol partout.
func TestReferenceSansPorteeCouvreToutLePlateau(t *testing.T) {
	ancres := [][3]float64{{2, 2, AncrageDecalageSol}}
	s := NewSurfaceReference(ancres)
	r := NewRendu([2]float64{0, 0}, [2]float64{40, 4}, 1)
	r.ArmeReference(s)
	m, in := quadPlat(0, 0, 40, 4, 0)
	r.AddMesh(m, in)
	toit, _ := quadPlat(0, 0, 40, 4, 8)
	r.AddMesh(toit, in)

	if _, _, couverte := r.AppliqueReference(s, true); !couverte {
		t.Fatal("toit integral : la carte doit etre couverte")
	}
	if z, ok := r.Altitude(35, 1); !ok || z != 0 {
		t.Fatalf("altitude (35,1) = %v (ok=%v) : sans portee, le sol (0) doit se montrer a 35 m de l'ancre", z, ok)
	}
}
