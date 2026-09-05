package coordination

import (
	"math"
	"testing"
)

func TestMesurer_EchantillonFaibleSousLePlancher(t *testing.T) {
	// Huit morts, toutes vengees : le taux vaut 1, et il ne classe personne.
	faible := Mesurer(8, 8, 4)
	if !faible.EchantillonFaible {
		t.Fatalf("8 morts : EchantillonFaible = false, attendu true (plancher %d)", SeuilEchantillonFaible)
	}
	if faible.Taux != 1 {
		t.Fatalf("Taux = %v, attendu 1", faible.Taux)
	}
	if faible.Brut != 8 || faible.N != 8 {
		t.Fatalf("Brut/N = %d/%d, attendu 8/8", faible.Brut, faible.N)
	}
	if faible.ParMatch != 2 {
		t.Fatalf("ParMatch = %v, attendu 2 (8 morts vengees sur 4 matchs)", faible.ParMatch)
	}

	// Trente morts : le plancher est atteint, la reserve tombe.
	solide := Mesurer(12, 30, 10)
	if solide.EchantillonFaible {
		t.Fatalf("30 morts : EchantillonFaible = true, attendu false")
	}
	if math.Abs(solide.Taux-0.4) > 1e-12 {
		t.Fatalf("Taux = %v, attendu 0.4", solide.Taux)
	}
	if solide.ParMatch != 1.2 {
		t.Fatalf("ParMatch = %v, attendu 1.2", solide.ParMatch)
	}

	// Le plancher est bien a 30 et non a 29 : 29 morts restent un echantillon faible.
	if !Mesurer(10, SeuilEchantillonFaible-1, 10).EchantillonFaible {
		t.Fatalf("%d morts : EchantillonFaible = false, attendu true", SeuilEchantillonFaible-1)
	}
}

// TestMesurer_TauxEnUnite0a1 : ADR 0006 — un taux sort en 0..1, jamais en pourcentage.
func TestMesurer_TauxEnUnite0a1(t *testing.T) {
	c := Mesurer(1, 4, 2)
	if c.Taux != 0.25 {
		t.Fatalf("Taux = %v, attendu 0.25 (unite 0..1, pas 25)", c.Taux)
	}
}

// TestMesurer_AucuneDonneeNEstPasZeroPourCent : un denominateur vide ne doit pas se lire
// comme une performance nulle, et surtout pas diviser par zero.
func TestMesurer_AucuneDonneeNEstPasZeroPourCent(t *testing.T) {
	for _, n := range []int{0, -3} {
		c := Mesurer(0, n, 0)
		if c.Taux != 0 {
			t.Fatalf("n = %d : Taux = %v, attendu 0", n, c.Taux)
		}
		if c.ParMatch != 0 {
			t.Fatalf("n = %d : ParMatch = %v, attendu 0", n, c.ParMatch)
		}
		if !c.EchantillonFaible {
			t.Fatalf("n = %d : EchantillonFaible = false, attendu true", n)
		}
	}
}
