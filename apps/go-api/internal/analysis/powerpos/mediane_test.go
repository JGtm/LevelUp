package powerpos

import (
	"math"
	"testing"
)

func TestMedianeSerieImpaireEtPaire(t *testing.T) {
	cas := []struct {
		nom    string
		serie  []float64
		attend float64
	}{
		{"vide", nil, 0},
		{"un element", []float64{7.5}, 7.5},
		{"impaire non triee", []float64{9, 1, 5}, 5},
		{"paire non triee", []float64{9, 1, 5, 3}, 4},
		{"negatifs", []float64{-8, -2, -5}, -5},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := Mediane(c.serie); math.Abs(got-c.attend) > 1e-9 {
				t.Errorf("Mediane(%v) = %v, attendu %v", c.serie, got, c.attend)
			}
		})
	}
}

// TestMedianeNeReordonnePasLEntree : une fonction pure qui trie la tranche de son appelant
// est un effet de bord cache.
func TestMedianeNeReordonnePasLEntree(t *testing.T) {
	serie := []float64{9, 1, 5, 3}
	_ = Mediane(serie)
	attendu := []float64{9, 1, 5, 3}
	for i := range attendu {
		if serie[i] != attendu[i] {
			t.Fatalf("entree reordonnee : %v, attendu %v", serie, attendu)
		}
	}
}

// TestMedianeResisteAuxExtremes : la raison d'etre de la mediane (cf. mediane.go) — une
// valeur extreme legitime ne doit pas deplacer la mesure.
func TestMedianeResisteAuxExtremes(t *testing.T) {
	serie := []float64{4, 5, 6, 5, 4, 300}
	if got := Mediane(serie); got != 5 {
		t.Errorf("Mediane = %v, attendu 5 (la valeur a 300 ne deplace pas le centre)", got)
	}
}

func TestQuantile(t *testing.T) {
	serie := []float64{10, 20, 30, 40, 50}
	cas := []struct {
		ordre  float64
		attend float64
	}{
		{0, 10},
		{0.25, 20},
		{0.5, 30},
		{0.95, 48},
		{1, 50},
		{1.5, 50},
		{-1, 10},
	}
	for _, c := range cas {
		if got := Quantile(serie, c.ordre); math.Abs(got-c.attend) > 1e-9 {
			t.Errorf("Quantile(%v) = %v, attendu %v", c.ordre, got, c.attend)
		}
	}
	if got := Quantile(nil, 0.5); got != 0 {
		t.Errorf("Quantile(vide) = %v, attendu 0", got)
	}
	if got := Quantile([]float64{3}, 0.9); got != 3 {
		t.Errorf("Quantile(un element) = %v, attendu 3", got)
	}
}
