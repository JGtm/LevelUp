package himap

import (
	"math"
	"testing"
)

func presque(a, b float64) bool { return math.Abs(a-b) < 1e-5 }

func matPresque(a, b [9]float64) bool {
	for i := range a {
		if !presque(a[i], b[i]) {
			return false
		}
	}
	return true
}

// TestQuatVersRot : quaternion identite -> matrice identite ; 90 deg autour de Z -> rotation connue.
func TestQuatVersRot(t *testing.T) {
	id := quatVersRot([4]float64{0, 0, 0, 1})
	if !matPresque(id, [9]float64{1, 0, 0, 0, 1, 0, 0, 0, 1}) {
		t.Fatalf("quat identite -> %v", id)
	}
	// Quaternion nul (noeud sans rotation valide) -> identite (repli).
	if z := quatVersRot([4]float64{0, 0, 0, 0}); !matPresque(z, [9]float64{1, 0, 0, 0, 1, 0, 0, 0, 1}) {
		t.Fatalf("quat nul -> %v", z)
	}
	s := math.Sqrt2 / 2
	rz := quatVersRot([4]float64{0, 0, s, s}) // +90 deg autour de Z
	if !matPresque(rz, [9]float64{0, -1, 0, 1, 0, 0, 0, 0, 1}) {
		t.Fatalf("quat 90degZ -> %v", rz)
	}
	// +X doit aller vers +Y.
	p := Transforme{Scale: 1, Rot: rz}.Applique([3]float64{1, 0, 0})
	if !presque(p[0], 0) || !presque(p[1], 1) || !presque(p[2], 0) {
		t.Fatalf("rot 90degZ (1,0,0) -> %v", p)
	}
}

// TestCompose : verifie que la composition reproduit T(p)=Scale*(Rot.p)+Trans et
// (a∘b)(p)=a(b(p)), avec echelle multipliee et translation mise a l'echelle (Ghidra FUN_140474790).
func TestCompose(t *testing.T) {
	a := Transforme{Scale: 2, Rot: [9]float64{1, 0, 0, 0, 1, 0, 0, 0, 1}, Trans: [3]float64{1, 0, 0}}
	b := Transforme{Scale: 3, Rot: [9]float64{1, 0, 0, 0, 1, 0, 0, 0, 1}, Trans: [3]float64{0, 1, 0}}
	c := Compose(a, b)
	if !presque(c.Scale, 6) {
		t.Fatalf("echelle composee = %f, attendu 6", c.Scale)
	}
	if !presque(c.Trans[0], 1) || !presque(c.Trans[1], 2) || !presque(c.Trans[2], 0) {
		t.Fatalf("translation composee = %v, attendu (1,2,0)", c.Trans)
	}
	p := [3]float64{1, 1, 1}
	viaCompose := c.Applique(p)
	viaChaine := a.Applique(b.Applique(p))
	for i := 0; i < 3; i++ {
		if !presque(viaCompose[i], viaChaine[i]) {
			t.Fatalf("compose != chaine : %v vs %v", viaCompose, viaChaine)
		}
	}
}

// TestNodeModelTransform : une chaine racine->enfant avec echelle et translation compose
// correctement la transformee model-space (echelles multipliees, translations mises a l'echelle).
func TestNodeModelTransform(t *testing.T) {
	nodes := []Noeud{
		{Name: 1, Parent: -1, Pos: [3]float64{1, 0, 0}, Quat: [4]float64{0, 0, 0, 1}, Scale: 2},
		{Name: 2, Parent: 0, Pos: [3]float64{1, 0, 0}, Quat: [4]float64{0, 0, 0, 1}, Scale: 3},
	}
	m := NodeModelTransform(nodes, 1)
	if !presque(m.Scale, 6) {
		t.Fatalf("echelle model = %f, attendu 6", m.Scale)
	}
	// enfant a pos (1,0,0) dans le repere du parent (echelle 2) + translation racine (1,0,0) = 3.
	if !presque(m.Trans[0], 3) || !presque(m.Trans[1], 0) || !presque(m.Trans[2], 0) {
		t.Fatalf("translation model = %v, attendu (3,0,0)", m.Trans)
	}
	// Noeud hors borne -> identite.
	if id := NodeModelTransform(nodes, 99); !presque(id.Scale, 1) {
		t.Fatalf("hors borne -> echelle %f, attendu 1", id.Scale)
	}
}

// TestNodeModelTransformAntiCycle : un parent cyclique ne boucle pas indefiniment.
func TestNodeModelTransformAntiCycle(t *testing.T) {
	nodes := []Noeud{
		{Name: 1, Parent: 1, Scale: 1, Quat: [4]float64{0, 0, 0, 1}},
		{Name: 2, Parent: 0, Scale: 1, Quat: [4]float64{0, 0, 0, 1}},
	}
	_ = NodeModelTransform(nodes, 1) // ne doit pas boucler
}

func TestEchelleEstUnitaire(t *testing.T) {
	if !EchelleEstUnitaire(1.0) || !EchelleEstUnitaire(1.00005) {
		t.Fatal("1.0 doit etre unitaire")
	}
	if EchelleEstUnitaire(0.5) || EchelleEstUnitaire(1.5) {
		t.Fatal("0.5/1.5 ne sont pas unitaires")
	}
}
