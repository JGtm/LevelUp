package powerpos

import (
	"math"
	"math/rand"
	"testing"
)

// sommeDe accumule des directions donnees en radians.
func sommeDe(angles ...float64) SommeAngulaire {
	var s SommeAngulaire
	for _, a := range angles {
		s.Ajoute(math.Cos(a), math.Sin(a))
	}
	return s
}

// TestAnglesConcentres : toutes les directions identiques — resultante 1, dispersion 0.
func TestAnglesConcentres(t *testing.T) {
	s := sommeDe(0.3, 0.3, 0.3, 0.3, 0.3, 0.3)
	if got := s.Resultante(); math.Abs(got-1) > 1e-9 {
		t.Errorf("resultante = %.6f, attendu 1", got)
	}
	if got := s.ResultanteCorrigee(); math.Abs(got-1) > 1e-9 {
		t.Errorf("resultante corrigee = %.6f, attendu 1", got)
	}
	if got := s.Dispersion(); math.Abs(got) > 1e-9 {
		t.Errorf("dispersion = %.6f, attendu 0", got)
	}
}

// TestAnglesOpposes : deux directions opposees s'annulent — dispersion maximale.
func TestAnglesOpposes(t *testing.T) {
	s := sommeDe(0, math.Pi, 0, math.Pi)
	if got := s.Resultante(); got > 1e-9 {
		t.Errorf("resultante = %.6f, attendu 0", got)
	}
	if got := s.Dispersion(); math.Abs(got-1) > 1e-9 {
		t.Errorf("dispersion = %.6f, attendu 1", got)
	}
}

// TestAnglesUniformes : un eventail regulier de directions est isotrope — dispersion 1,
// et la correction de biais ne cree pas de concentration la ou il n'y en a pas.
func TestAnglesUniformes(t *testing.T) {
	const n = 12
	angles := make([]float64, n)
	for i := range angles {
		angles[i] = 2 * math.Pi * float64(i) / n
	}
	s := sommeDe(angles...)
	if got := s.Resultante(); got > 1e-9 {
		t.Errorf("resultante = %.6f, attendu 0", got)
	}
	if got := s.Dispersion(); math.Abs(got-1) > 1e-9 {
		t.Errorf("dispersion = %.6f, attendu 1", got)
	}
}

// TestAnglesCorrectionDuPetitEchantillon : la resultante BRUTE d'un tirage isotrope
// n'est pas nulle (sa moyenne quadratique vaut 1/sqrt(n)), la resultante CORRIGEE l'est
// en moyenne. Verifie sur 400 tirages de 10 directions, generateur a graine fixe.
func TestAnglesCorrectionDuPetitEchantillon(t *testing.T) {
	const n = 10
	const tirages = 400
	gen := rand.New(rand.NewSource(20260920))
	var brute, corrigee float64
	for k := 0; k < tirages; k++ {
		var s SommeAngulaire
		for i := 0; i < n; i++ {
			a := 2 * math.Pi * gen.Float64()
			s.Ajoute(math.Cos(a), math.Sin(a))
		}
		brute += s.Resultante() * s.Resultante()
		r := s.ResultanteCorrigee()
		corrigee += r * r
	}
	brute /= tirages
	corrigee /= tirages
	// E[R^2] = 1/n sous l'isotropie : la brute doit etre proche de 0,1. La corrigee est
	// tronquee a zero sur les tirages sous-disperses, donc sa moyenne n'est pas exactement
	// nulle (esperance de (nR^2 - 1)+ / (n - 1), soit ~e^-1 / 9 = 0,04), mais elle doit
	// rester nettement sous la brute.
	if brute < 0.07 || brute > 0.14 {
		t.Errorf("R^2 brut moyen = %.3f, attendu ~0,1 pour n = 10", brute)
	}
	if corrigee > 0.6*brute {
		t.Errorf("R^2 corrige moyen = %.3f, attendu < %.3f (60 %% du brut)", corrigee, 0.6*brute)
	}
}

// TestAnglesVideEtSingleton : une somme vide ou a une seule direction ne dit rien.
func TestAnglesVideEtSingleton(t *testing.T) {
	var vide SommeAngulaire
	if got := vide.Resultante(); got != 0 {
		t.Errorf("resultante vide = %.3f, attendu 0", got)
	}
	if got := vide.ResultanteCorrigee(); got != 0 {
		t.Errorf("resultante corrigee vide = %.3f, attendu 0", got)
	}
	seule := sommeDe(1.0)
	if got := seule.ResultanteCorrigee(); got != 0 {
		t.Errorf("resultante corrigee d'une direction seule = %.3f, attendu 0", got)
	}
	if seule.N != 1 {
		t.Errorf("N = %d, attendu 1", seule.N)
	}
}

// TestAnglesVecteurNulIgnore : deux positions confondues ne portent aucune direction.
func TestAnglesVecteurNulIgnore(t *testing.T) {
	var s SommeAngulaire
	s.Ajoute(0, 0)
	s.Ajoute(math.NaN(), 1)
	s.Ajoute(math.Inf(1), 0)
	if s.N != 0 {
		t.Errorf("N = %d, attendu 0 (vecteur nul, NaN et Inf ignores)", s.N)
	}
}

// TestAnglesPlusEstAdditif : la somme de deux cellules est la somme de leurs sommes —
// c'est ce qui permet le lissage sur un disque sans re-parcourir les eliminations.
func TestAnglesPlusEstAdditif(t *testing.T) {
	a := sommeDe(0, 0.5)
	b := sommeDe(1.0, 1.5, 2.0)
	tout := sommeDe(0, 0.5, 1.0, 1.5, 2.0)
	got := a.Plus(b)
	if got.N != tout.N || math.Abs(got.Cos-tout.Cos) > 1e-12 || math.Abs(got.Sin-tout.Sin) > 1e-12 {
		t.Errorf("a.Plus(b) = %+v, attendu %+v", got, tout)
	}
}

// TestAnglesNormalise : la norme du vecteur n'entre pas dans la direction.
func TestAnglesNormalise(t *testing.T) {
	var court, long SommeAngulaire
	court.Ajoute(1, 0)
	long.Ajoute(100, 0)
	if math.Abs(court.Cos-long.Cos) > 1e-12 || math.Abs(court.Sin-long.Sin) > 1e-12 {
		t.Errorf("la norme influence la direction : %+v contre %+v", court, long)
	}
}
