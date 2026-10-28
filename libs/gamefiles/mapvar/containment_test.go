package mapvar

import (
	"errors"
	"math"
	"testing"
)

// containment_test.go — chaque test ci-dessous doit VIRER AU ROUGE si on retire
// la regle qu'il garde. Les points d'epreuve sont choisis pour cela, pas pour
// illustrer : un point « evidemment dedans » ne garde rien.

func boxShape(halfX, halfY, upZ, downZ float64, fwd Vec3) *Shape {
	hx, hy := halfX, halfY
	return &Shape{
		Family: ShapeBox, HalfX: &hx, HalfY: &hy,
		UpZ: upZ, DownZ: downZ,
		Forward: fwd, Up: Vec3{Z: 1},
	}
}

func cylinderShape(radius, upZ, downZ float64, up Vec3) *Shape {
	r := radius
	return &Shape{
		Family: ShapeCylinder, Radius: &r,
		UpZ: upZ, DownZ: downZ,
		Forward: Vec3{X: 1}, Up: up,
	}
}

func mustVolume(t *testing.T, center Vec3, s *Shape) Volume {
	t.Helper()
	v, err := NewVolume(center, s)
	if err != nil {
		t.Fatalf("NewVolume: %v", err)
	}
	return v
}

// TestBoiteOrienteeNeSeRabatPasSurLesAxesDuMonde est LE test de la lecon payee
// par le chantier : ignorer Forward declarait « dedans » 31 % de positions qui
// sont dehors. Les deux points ci-dessous inversent leur verdict selon qu'on
// tient compte de l'orientation ou non — un test axis-aligned les rate tous les
// deux.
func TestBoiteOrienteeNeSeRabatPasSurLesAxesDuMonde(t *testing.T) {
	const c = math.Sqrt2 / 2 // 45 degres
	v := mustVolume(t, Vec3{}, boxShape(2, 0.5, 3, 0.5, Vec3{X: c, Y: c}))

	// Le long de la diagonale de la boite, a 1,8 m : DEDANS. Un test aligne sur
	// les axes du monde y lirait ly = 1,27 > 0,5 et dirait dehors.
	surLAxeLong := Vec3{X: 1.8 * c, Y: 1.8 * c}
	if !v.Contains(surLAxeLong) {
		t.Errorf("point sur l'axe long de la boite tournee : attendu dedans, obtenu dehors")
	}
	// Le long de +X du monde, a 1,8 m : DEHORS (c'est la largeur de 0,5 m qui
	// s'y presente). Un test aligne sur les axes dirait dedans.
	surLAxeDuMonde := Vec3{X: 1.8}
	if v.Contains(surLAxeDuMonde) {
		t.Errorf("point sur +X du monde : attendu dehors, obtenu dedans")
	}
}

// TestForwardEstRedresseDansLePlanPerpendiculaireAUp garde le Gram-Schmidt.
// Sans lui, l'axe lateral n'est plus unitaire (norme 0,894) et la boite se
// comporte comme si elle etait 12 % plus large : le point d'epreuve, a 1,05 m
// d'un demi-cote de 1,00 m, serait declare dedans.
func TestForwardEstRedresseDansLePlanPerpendiculaireAUp(t *testing.T) {
	nonOrthogonal := Vec3{X: 1, Z: 0.5}
	v := mustVolume(t, Vec3{}, boxShape(2, 1, 3, 0.5, nonOrthogonal))

	if v.Contains(Vec3{Y: 1.05}) {
		t.Errorf("1,05 m sur un demi-cote de 1,00 m : attendu dehors, obtenu dedans "+
			"(base non redressee ? |right| = %.4f)", math.Sqrt(dot(v.right, v.right)))
	}
	if !v.Contains(Vec3{Y: 0.95}) {
		t.Errorf("0,95 m sur un demi-cote de 1,00 m : attendu dedans, obtenu dehors")
	}
}

// TestHauteurEstAsymetrique garde la distinction UpZ / DownZ. Les zones de
// Bastion du catalogue montent de 1,0 a 4,0 m et ne descendent que de 0,0 a
// 1,0 m : confondre les deux bornes, ou les echanger, change le verdict d'un
// joueur situe a l'etage du dessous.
func TestHauteurEstAsymetrique(t *testing.T) {
	v := mustVolume(t, Vec3{}, boxShape(2, 2, 3, 0.5, Vec3{X: 1}))

	cas := []struct {
		nom    string
		z      float64
		dedans bool
	}{
		{"juste sous le plafond", 2.9, true},
		{"juste au-dessus du plafond", 3.1, false},
		{"juste au-dessus du plancher", -0.4, true},
		{"juste sous le plancher", -0.6, false},
		{"a la hauteur du plafond de l'autre borne", -2.9, false},
	}
	for _, c := range cas {
		if got := v.Contains(Vec3{Z: c.z}); got != c.dedans {
			t.Errorf("%s (z = %.1f) : attendu dedans=%v, obtenu %v", c.nom, c.z, c.dedans, got)
		}
	}
}

// TestCylindreEstRadialEtSuitSonAxe garde deux choses : le rayon se compare a la
// distance dans le plan (pas au plus grand des deux ecarts), et l'axe du
// cylindre est Up, pas la verticale du monde.
func TestCylindreEstRadialEtSuitSonAxe(t *testing.T) {
	v := mustVolume(t, Vec3{}, cylinderShape(2, 3, 0.5, Vec3{Z: 1}))

	if !v.Contains(Vec3{X: 1.9}) {
		t.Errorf("1,9 m d'un rayon de 2 m : attendu dedans")
	}
	if v.Contains(Vec3{X: 2.1}) {
		t.Errorf("2,1 m d'un rayon de 2 m : attendu dehors")
	}
	// Diagonale : chaque composante vaut 1,5 (< 2) mais la distance vaut 2,12.
	// Un test qui bornerait chaque axe separement dirait dedans.
	if v.Contains(Vec3{X: 1.5, Y: 1.5}) {
		t.Errorf("diagonale a 2,12 m d'un rayon de 2 m : attendu dehors, obtenu dedans")
	}

	// Cylindre couche : son axe part sur +X. Un point a 2,9 m sur +X reste dans
	// la hauteur (UpZ = 3) ; le meme ecart sur +Z sort du rayon.
	formeCouchee := cylinderShape(2, 3, 0.5, Vec3{X: 1})
	formeCouchee.Forward = Vec3{Z: 1} // doit rester non colineaire a Up
	couche := mustVolume(t, Vec3{}, formeCouchee)
	if !couche.Contains(Vec3{X: 2.9}) {
		t.Errorf("cylindre couche : 2,9 m le long de son axe, attendu dedans")
	}
	if couche.Contains(Vec3{Z: 2.9}) {
		t.Errorf("cylindre couche : 2,9 m perpendiculaire a son axe (rayon 2 m), attendu dehors")
	}
}

// TestLesBornesSontInclusives fige le choix documente : a la limite, on est
// dedans. Le quantum du decodeur vaut ~1,4 cm, la question ne se pose pas sur
// des donnees reelles — mais le contrat doit etre ecrit quelque part.
func TestLesBornesSontInclusives(t *testing.T) {
	v := mustVolume(t, Vec3{}, boxShape(2, 1, 3, 0.5, Vec3{X: 1}))
	for _, p := range []Vec3{{X: 2}, {Y: 1}, {Z: 3}, {Z: -0.5}} {
		if !v.Contains(p) {
			t.Errorf("point exactement sur la limite %+v : attendu dedans", p)
		}
	}
}

// TestDistanceEstNulleDedansEtExacteDehors garde l'equivalence documentee
// DistanceTo(p) == 0 <=> Contains(p), et la mesure a la BOITE (pas au centre).
func TestDistanceEstNulleDedansEtExacteDehors(t *testing.T) {
	v := mustVolume(t, Vec3{}, boxShape(2, 1, 3, 0.5, Vec3{X: 1}))

	cas := []struct {
		nom  string
		p    Vec3
		want float64
	}{
		{"au centre", Vec3{}, 0},
		{"sur la limite", Vec3{X: 2}, 0},
		{"a 1 m au-dela du demi-cote long", Vec3{X: 3}, 1},
		{"a 2 m au-dela du demi-cote court", Vec3{Y: 3}, 2},
		{"a 1 m au-dessus du plafond", Vec3{Z: 4}, 1},
		{"a 1 m sous le plancher", Vec3{Z: -1.5}, 1},
		// En diagonale : la distance est la norme des DEUX depassements (3-4-5), pas le
		// plus grand des deux. Un test qui prendrait le max lirait 4.
		{"en diagonale, depassements 3 et 4", Vec3{X: 5, Y: 5}, 5},
		// Le long de la boite mais au-dela en hauteur : la composante deja dans les
		// bornes doit etre ANNULEE, sinon on mesurerait une distance au centre.
		{"dans les bornes en x, dehors en z", Vec3{X: 1.5, Z: 4}, 1},
	}
	for _, c := range cas {
		got := v.DistanceTo(c.p)
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%s : distance = %.4f, attendu %.4f", c.nom, got, c.want)
		}
		if dedans := v.Contains(c.p); dedans != (got == 0) {
			t.Errorf("%s : Contains = %v mais distance = %.4f — les deux doivent concorder",
				c.nom, dedans, got)
		}
	}
}

// TestDistanceCylindreEstRadiale : le cylindre mesure a sa PAROI, pas a son axe carre.
func TestDistanceCylindreEstRadiale(t *testing.T) {
	v := mustVolume(t, Vec3{}, cylinderShape(2, 3, 0.5, Vec3{Z: 1}))
	if got := v.DistanceTo(Vec3{X: 5}); math.Abs(got-3) > 1e-9 {
		t.Errorf("a 5 m d'un rayon de 2 : distance = %.4f, attendu 3", got)
	}
	// Diagonale a 5 m du centre (3-4-5) : 5 - 2 = 3 de la paroi.
	if got := v.DistanceTo(Vec3{X: 3, Y: 4}); math.Abs(got-3) > 1e-9 {
		t.Errorf("diagonale a 5 m du centre : distance = %.4f, attendu 3", got)
	}
}

// TestVolumeRefuseUneFormeInutilisable garde le refus explicite. Mesure sur le
// catalogue : 0 forme de zone degeneree sur 224 — une occurrence serait une
// nouveaute a expliquer, et un repli silencieux sur les axes du monde
// l'effacerait.
func TestVolumeRefuseUneFormeInutilisable(t *testing.T) {
	if _, err := NewVolume(Vec3{}, nil); !errors.Is(err, ErrNoShape) {
		t.Errorf("objectif ponctuel : attendu ErrNoShape, obtenu %v", err)
	}
	upNul := boxShape(1, 1, 1, 1, Vec3{X: 1})
	upNul.Up = Vec3{}
	if _, err := NewVolume(Vec3{}, upNul); !errors.Is(err, ErrDegenerateFrame) {
		t.Errorf("Up nul : attendu ErrDegenerateFrame, obtenu %v", err)
	}
	colineaire := boxShape(1, 1, 1, 1, Vec3{Z: 1})
	if _, err := NewVolume(Vec3{}, colineaire); !errors.Is(err, ErrDegenerateFrame) {
		t.Errorf("Forward colineaire a Up : attendu ErrDegenerateFrame, obtenu %v", err)
	}
	sansDemiCotes := &Shape{Family: ShapeBox, UpZ: 1, DownZ: 1, Forward: Vec3{X: 1}, Up: Vec3{Z: 1}}
	if _, err := NewVolume(Vec3{}, sansDemiCotes); !errors.Is(err, ErrMissingExtent) {
		t.Errorf("boite sans demi-cotes : attendu ErrMissingExtent, obtenu %v", err)
	}
	sansRayon := &Shape{Family: ShapeCylinder, UpZ: 1, DownZ: 1, Forward: Vec3{X: 1}, Up: Vec3{Z: 1}}
	if _, err := NewVolume(Vec3{}, sansRayon); !errors.Is(err, ErrMissingExtent) {
		t.Errorf("cylindre sans rayon : attendu ErrMissingExtent, obtenu %v", err)
	}
	inconnue := &Shape{Family: "triangle", UpZ: 1, DownZ: 1, Forward: Vec3{X: 1}, Up: Vec3{Z: 1}}
	v, err := NewVolume(Vec3{}, inconnue)
	if err != nil {
		t.Fatalf("famille inconnue : la construction ne doit pas echouer, %v", err)
	}
	if v.Contains(Vec3{}) {
		t.Errorf("famille inconnue : aucun point ne doit etre declare dedans")
	}
}

// TestLeCentreEstPrisEnCompte garde le fait qu'un volume est pose QUELQUE PART.
// Un test qui oublierait de soustraire le centre passerait tous les precedents
// (ils sont centres sur l'origine) et se tromperait sur toutes les vraies
// zones, dont aucune n'est a l'origine.
func TestLeCentreEstPrisEnCompte(t *testing.T) {
	centre := Vec3{X: 100, Y: -50, Z: 12}
	v := mustVolume(t, centre, boxShape(2, 1, 3, 0.5, Vec3{X: 1}))
	if !v.Contains(Vec3{X: 101.5, Y: -50.5, Z: 13}) {
		t.Errorf("point pres du centre reel : attendu dedans")
	}
	if v.Contains(Vec3{}) {
		t.Errorf("l'origine du monde n'est pas dans une zone posee en %+v", centre)
	}
}

// TestTranslateDeplaceLeVolumeSansLeDeformer garde le temoin negatif : la meme
// forme, ailleurs. Si Translate deformait la zone, le temoin ne serait plus
// comparable a la vraie et la mesure de signal/temoin perdrait son sens.
func TestTranslateDeplaceLeVolumeSansLeDeformer(t *testing.T) {
	v := mustVolume(t, Vec3{X: 10}, boxShape(2, 1, 3, 0.5, Vec3{X: 1}))
	d := Vec3{X: 12, Y: 12}
	temoin := v.Translate(d)

	if got, want := temoin.Center(), (Vec3{X: 22, Y: 12}); got != want {
		t.Errorf("centre du temoin : attendu %+v, obtenu %+v", want, got)
	}
	if v.Center().X != 10 {
		t.Errorf("Translate a modifie le volume d'origine (centre %.1f)", v.Center().X)
	}
	// Le meme point RELATIF au centre doit donner le meme verdict des deux
	// cotes : c'est la definition d'un temoin non deforme.
	for _, rel := range []Vec3{{X: 1.9}, {X: 2.1}, {Y: 0.9}, {Z: 3.1}} {
		aVrai := v.Contains(Vec3{X: v.Center().X + rel.X, Y: v.Center().Y + rel.Y, Z: v.Center().Z + rel.Z})
		aTemoin := temoin.Contains(Vec3{X: temoin.Center().X + rel.X, Y: temoin.Center().Y + rel.Y, Z: temoin.Center().Z + rel.Z})
		if aVrai != aTemoin {
			t.Errorf("point relatif %+v : vraie zone dit %v, temoin dit %v", rel, aVrai, aTemoin)
		}
	}
}
