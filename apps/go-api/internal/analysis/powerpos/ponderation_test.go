package powerpos

import (
	"math"
	"testing"
)

func rang(v float64) *float64 { return &v }

func ponderationTest() Ponderation {
	return Ponderation{
		RangBas: 1000, RangHaut: 1500, RangMedian: 1200,
		PoidsBas: 0.5, PoidsHaut: 1.5, PoidsInconnu: 1,
	}
}

// TestPonderationNeutreNePencheRien : le defaut rend 1,0 pour tout rang, connu ou non.
func TestPonderationNeutreNePencheRien(t *testing.T) {
	p := PonderationNeutre()
	if p.Active() {
		t.Fatal("la ponderation neutre se declare active")
	}
	for _, r := range []*float64{nil, rang(0), rang(800), rang(2000)} {
		if got := p.Poids(r); got != 1 {
			t.Errorf("poids neutre pour %v = %.3f, attendu 1", r, got)
		}
		if p.EstFort(r) {
			t.Errorf("ponderation neutre : %v declare fort", r)
		}
	}
}

// TestPonderationRampe : lineaire entre les deux bornes, bornee au-dela.
func TestPonderationRampe(t *testing.T) {
	p := ponderationTest()
	if !p.Active() {
		t.Fatal("ponderation inactive")
	}
	cas := []struct {
		rang  float64
		poids float64
	}{
		{500, 0.5},  // sous la borne basse : borne
		{1000, 0.5}, // borne basse
		{1250, 1.0}, // milieu de la rampe
		{1500, 1.5}, // borne haute
		{2000, 1.5}, // au-dela : borne
	}
	for _, c := range cas {
		if got := p.Poids(rang(c.rang)); math.Abs(got-c.poids) > 1e-9 {
			t.Errorf("poids(%.0f) = %.3f, attendu %.3f", c.rang, got, c.poids)
		}
	}
}

// TestPonderationInconnuEstNeutre : un rang absent pese 1,0 et n'est pas « fort ».
func TestPonderationInconnuEstNeutre(t *testing.T) {
	p := ponderationTest()
	if got := p.Poids(nil); got != 1 {
		t.Errorf("poids d'un rang inconnu = %.3f, attendu 1", got)
	}
	if p.EstFort(nil) {
		t.Error("un rang inconnu est declare fort")
	}
	p.PoidsInconnu = 0 // valeur non renseignee : repli sur 1, jamais 0
	if got := p.Poids(nil); got != 1 {
		t.Errorf("poids d'un rang inconnu sans PoidsInconnu = %.3f, attendu 1", got)
	}
}

// TestPonderationEstFort : la mediane separe, inclusivement.
func TestPonderationEstFort(t *testing.T) {
	p := ponderationTest()
	if p.EstFort(rang(1199)) {
		t.Error("1199 declare fort sous une mediane de 1200")
	}
	if !p.EstFort(rang(1200)) {
		t.Error("1200 non fort a une mediane de 1200")
	}
	if !p.EstFort(rang(1700)) {
		t.Error("1700 non fort")
	}
}

// TestPonderationBornesInversees : bornes egales ou inversees = inactive, poids 1.
func TestPonderationBornesInversees(t *testing.T) {
	p := ponderationTest()
	p.RangHaut = p.RangBas
	if p.Active() {
		t.Error("bornes egales : ponderation active")
	}
	if got := p.Poids(rang(1300)); got != 1 {
		t.Errorf("poids avec bornes egales = %.3f, attendu 1", got)
	}
}
