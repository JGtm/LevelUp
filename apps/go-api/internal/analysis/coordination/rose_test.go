package coordination

import (
	"math"
	"testing"

	"levelup/go-api/internal/domain"
)

// TestRose8_LesHuitFrontieres : chaque frontiere k * 45 degres ouvre le secteur k (borne basse
// incluse), et un cheveu avant tombe dans le secteur precedent.
func TestRose8_LesHuitFrontieres(t *testing.T) {
	// (dx, dy) pile sur les huit frontieres, dans l'ordre horaire depuis le haut.
	frontieres := [][2]float64{{0, 1}, {1, 1}, {1, 0}, {1, -1}, {0, -1}, {-1, -1}, {-1, 0}, {-1, 1}}
	for k, f := range frontieres {
		got, ok := Rose8(f[0], f[1])
		if !ok || got != k {
			t.Errorf("frontiere %d (%v) -> secteur %d ok=%v, attendu %d", k, f, got, ok, k)
		}
		// un cheveu AVANT la frontiere (angle - epsilon) : secteur precedent.
		a := float64(k)*45 - 0.001
		rad := a * math.Pi / 180
		avant, _ := Rose8(math.Sin(rad), math.Cos(rad))
		if want := (k + 7) % 8; avant != want {
			t.Errorf("juste avant la frontiere %d : secteur %d, attendu %d", k, avant, want)
		}
	}
}

func TestRose8_MilieuxEtDeplacementNul(t *testing.T) {
	for k := 0; k < 8; k++ {
		a := (float64(k)*45 + 22.5) * math.Pi / 180
		if got, ok := Rose8(math.Sin(a), math.Cos(a)); !ok || got != k {
			t.Errorf("milieu du secteur %d -> %d", k, got)
		}
	}
	if _, ok := Rose8(0, 0); ok {
		t.Fatal("deplacement nul : pas de direction")
	}
}

func TestRoseDepuis_RepereDuPlan(t *testing.T) {
	victime := domain.TacticalPoint{X: 100, Y: 50}
	// le tueur est au nord (+Y) de la victime : secteur 0 ; a l'est (+X) : secteur 2.
	if s, _ := RoseDepuis(victime, domain.TacticalPoint{X: 100.5, Y: 60}); s != 0 {
		t.Errorf("nord : %d", s)
	}
	if s, _ := RoseDepuis(victime, domain.TacticalPoint{X: 110, Y: 49}); s != 2 {
		t.Errorf("est : %d", s)
	}
}

func TestStatutDeLaMort_QuatreStatuts(t *testing.T) {
	d := func(v float64) *float64 { return &v }
	cas := []struct {
		nom                 string
		visibles, horsDeVue int
		proche              *float64
		want                string
	}{
		{"a la portee exacte : accompagnee", 1, 0, d(18), domain.TacticalStatutAccompagnee},
		{"juste au-dela : visible hors rayon", 1, 0, d(18.01), domain.TacticalStatutVisibleHorsRayon},
		{"aucun visible, un hors de vue", 0, 1, nil, domain.TacticalStatutAucunVisible},
		{"personne", 0, 0, nil, domain.TacticalStatutEquipeATerre},
	}
	for _, c := range cas {
		if got := StatutDeLaMort(c.visibles, c.horsDeVue, c.proche, 18); got != c.want {
			t.Errorf("%s : %s, attendu %s", c.nom, got, c.want)
		}
	}
}
