package himap

import (
	"math"
	"testing"
)

// rendulDeuxDalles pose deux dalles horizontales cote a cote, separees par une marche de
// `marche` metres. Meme normale, altitudes differentes : c'est le cas que l'ombrage de
// `rendu.go` ne sait PAS montrer, et que les aretes doivent reveler.
// L'emprise DEBORDE les deux dalles a droite : il faut de la place VIDE pour qu'une cellule
// aberrante puisse exister. Posee sous une dalle, elle serait masquee par le z-buffer — qui
// garde la surface la plus haute — et le temoin des centiles ne testerait rien.
func rendulDeuxDalles(marche float64) *Rendu {
	r := NewRendu([2]float64{0, 0}, [2]float64{16, 4}, 1)
	in := instanceIdentite()
	gauche := &Mesh{
		Vertices:  [][3]float64{{0, 0, 0}, {4, 0, 0}, {4, 4, 0}, {0, 4, 0}},
		Triangles: [][3]int{{0, 1, 2}, {0, 2, 3}},
	}
	droite := &Mesh{
		Vertices:  [][3]float64{{4, 0, marche}, {8, 0, marche}, {8, 4, marche}, {4, 4, marche}},
		Triangles: [][3]int{{0, 1, 2}, {0, 2, 3}},
	}
	r.AddMesh(gauche, in)
	r.AddMesh(droite, in)
	return r
}

// TestAreteRevele Ce QueLOmbrageCache — deux dalles de MEME inclinaison a deux hauteurs ont le
// meme eclairement ; seule la rupture d'altitude les separe.
//
// MUTATION QUI LE FAIT ROUGIR : tester la normale au lieu de la difference d'altitude dans
// `Arete`, ou relever `SeuilAreteMetres` au-dessus de la marche.
func TestAreteReveleCeQueLOmbrageCache(t *testing.T) {
	r := rendulDeuxDalles(2.0)
	eg, _ := r.Eclairement(2, 2)
	ed, _ := r.Eclairement(6, 2)
	if math.Abs(eg-ed) > 1e-9 {
		t.Fatalf("les deux dalles doivent avoir le MEME eclairement (%.6f vs %.6f) — sinon le "+
			"temoin ne teste pas ce qu'il pretend", eg, ed)
	}
	if !r.Arete(3, 2) {
		t.Error("le pixel qui borde la marche de 2 m doit etre une arete")
	}
	if r.Arete(1, 2) {
		t.Error("un pixel au milieu d'une dalle plate n'est PAS une arete")
	}
}

// TestAreteIgnoreLesMarchesFranchissables — une marche sous le seuil physique n'est pas un bord.
func TestAreteIgnoreLesMarchesFranchissables(t *testing.T) {
	r := rendulDeuxDalles(SeuilAreteMetres / 2)
	if r.Arete(3, 2) {
		t.Errorf("une marche de %.2f m se franchit sans sauter : ce n'est pas un rebord",
			SeuilAreteMetres/2)
	}
}

// TestEclairementPlatQuantifie — les valeurs rendues doivent tomber sur la grille des paliers.
func TestEclairementPlatQuantifie(t *testing.T) {
	r := rendulDeuxDalles(2.0)
	e, ok := r.EclairementPlat(2, 2)
	if !ok {
		t.Fatal("pixel sans matiere")
	}
	n := float64(PaliersEclairement - 1)
	if math.Abs(e*n-math.Round(e*n)) > 1e-9 {
		t.Fatalf("eclairement %.6f hors de la grille de %d paliers", e, PaliersEclairement)
	}
}

// TestBornesAltitudeResistentAUneCelluleAberrante — LE temoin des centiles.
//
// Il existe parce que cette exacte regression a deja eu lieu sur ce chantier : une cellule a
// -131 m ecrasait toute la carte dans deux nuances de blanc. Bornes min/max = carte illisible.
//
// MUTATION QUI LE FAIT ROUGIR : remplacer les centiles 2/98 par min/max dans
// `BornesAltitudeRobustes`.
func TestBornesAltitudeResistentAUneCelluleAberrante(t *testing.T) {
	// L'echantillon doit avoir la TAILLE D'UNE VRAIE CARTE. Sur cinquante cellules, le 2e
	// centile EST la cellule aberrante — le centile ne protege rien et le temoin condamnerait
	// du code juste. La carte reelle porte ~1,6 M de pixels ; on en simule 20 000.
	r := NewRendu([2]float64{0, 0}, [2]float64{200, 100}, 1)
	in := instanceIdentite()
	// Une rampe et non un plan : des bornes n'ont de sens que sur une carte qui a du relief.
	r.AddMesh(&Mesh{
		Vertices:  [][3]float64{{0, 0, 0}, {180, 0, 10}, {180, 100, 10}, {0, 100, 0}},
		Triangles: [][3]int{{0, 1, 2}, {0, 2, 3}},
	}, in)
	basAvant, hautAvant, ok := r.BornesAltitudeRobustes()
	if !ok {
		t.Fatal("bornes indisponibles")
	}
	// Un unique triangle abyssal, dans la zone VIDE a droite — sinon le z-buffer le masque
	// (il garde la surface la plus HAUTE) et le temoin ne teste rien.
	r.AddMesh(&Mesh{
		Vertices:  [][3]float64{{190, 0, -131}, {198, 0, -131}, {198, 8, -131}},
		Triangles: [][3]int{{0, 1, 2}},
	}, in)
	if _, vu := r.Altitude(195, 2); !vu {
		t.Fatal("la cellule aberrante doit etre VISIBLE, sinon ce temoin est tautologique")
	}
	basApres, hautApres, _ := r.BornesAltitudeRobustes()
	if math.Abs(basApres-basAvant) > 0.5 || math.Abs(hautApres-hautAvant) > 0.5 {
		t.Fatalf("une cellule aberrante a deplace les bornes : [%.2f ; %.2f] -> [%.2f ; %.2f]",
			basAvant, hautAvant, basApres, hautApres)
	}
}

// TestTeinteAltitudeEstOrdonnee — la rampe doit etre monotone en luminance : une altitude plus
// haute ne peut pas rendre une couleur plus sombre, sinon les paliers se lisent a l'envers.
func TestTeinteAltitudeEstOrdonnee(t *testing.T) {
	lum := func(x float64) float64 {
		c := TeinteAltitude(x, 1)
		return 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
	}
	precedent := -1.0
	for i := 0; i <= 20; i++ {
		l := lum(float64(i) / 20)
		if l < precedent-1e-9 {
			t.Fatalf("rampe non monotone a t=%.2f : %.1f apres %.1f", float64(i)/20, l, precedent)
		}
		precedent = l
	}
	// Ecretage : hors [0,1] on borne, on n'extrapole pas.
	if TeinteAltitude(-5, 1) != TeinteAltitude(0, 1) || TeinteAltitude(5, 1) != TeinteAltitude(1, 1) {
		t.Error("les valeurs hors bornes doivent etre ecretees")
	}
}

// TestSeuilAreteParCarte — le seuil d'arete se regle PAR CARTE, et le defaut ne bouge pas.
//
// Le gribouillis d'Isolation (2026-08-27) vient de la : le predicat compare deux voisins a
// 50 cm, et sur une carte en pieces organiques il est vrai presque partout. Un seuil releve
// doit faire taire un denivele modere et laisser parler une vraie marche.
func TestSeuilAreteParCarte(t *testing.T) {
	// Un sol plein de 3 x 3, et une marche de 1 m sur la colonne de droite. La cellule
	// testee est INTERIEURE : ses quatre voisins existent, sans quoi le predicat rend vrai
	// par manque de matiere et ne mesure plus rien.
	r := NewRendu([2]float64{0, 0}, [2]float64{3, 3}, 1)
	sol, in := quadPlat(0, 0, 3, 3, 0)
	r.AddMesh(sol, in)
	marche, inM := quadPlat(2, 0, 3, 3, 1)
	r.AddMesh(marche, inM)

	if !r.Arete(1, 1) {
		t.Fatal("denivele de 1 m : le seuil par defaut (0,5 m) doit tracer un bord")
	}
	r.SeuilArete = 2
	if r.Arete(1, 1) {
		t.Fatal("seuil releve a 2 m : un denivele de 1 m ne doit plus tracer de bord")
	}
	r.SeuilArete = 0
	if !r.Arete(1, 1) {
		t.Fatal("seuil remis a zero : le defaut doit reprendre")
	}
}
