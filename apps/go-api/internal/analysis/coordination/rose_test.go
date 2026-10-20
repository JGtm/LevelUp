package coordination

import (
	"math"
	"testing"

	"levelup/go-api/internal/domain"
)

// versAngle rend le vecteur unitaire (dx, dy) d'un angle en degres depuis le haut, sens horaire.
func versAngle(deg float64) (dx, dy float64) {
	rad := deg * math.Pi / 180
	return math.Sin(rad), math.Cos(rad)
}

// TestRose8_LesHuitFrontieres : les secteurs sont centres sur les axes et les diagonales ; chaque
// frontiere 45 k + 22,5 degres ouvre le secteur k + 1 (borne basse incluse). Un cheveu avant : le
// secteur k. Un cheveu apres : k + 1.
func TestRose8_LesHuitFrontieres(t *testing.T) {
	for k := 0; k < 8; k++ {
		frontiere := float64(k)*45 + 22.5
		avant, _ := Rose8(versAngle(frontiere - 0.001))
		apres, _ := Rose8(versAngle(frontiere + 0.001))
		if avant != k || apres != (k+1)%8 {
			t.Errorf("frontiere a %v deg : avant %d (attendu %d), apres %d (attendu %d)",
				frontiere, avant, k, apres, (k+1)%8)
		}
	}
}

// TestRose8_LesAxesEtLesDiagonalesSontAuCentre : un tueur pile au nord / est / sud / ouest, ou sur
// une diagonale, est au centre de son secteur — exactement, sans trigonometrie.
func TestRose8_LesAxesEtLesDiagonalesSontAuCentre(t *testing.T) {
	centres := [][2]float64{{0, 1}, {1, 1}, {1, 0}, {1, -1}, {0, -1}, {-1, -1}, {-1, 0}, {-1, 1}}
	for k, c := range centres {
		if got, ok := Rose8(c[0], c[1]); !ok || got != k {
			t.Errorf("centre %d (%v) -> secteur %d ok=%v", k, c, got, ok)
		}
	}
	if _, ok := Rose8(0, 0); ok {
		t.Fatal("deplacement nul : pas de direction")
	}
}

func TestRose8_NordEtOuestDeChaqueCoteDuHaut(t *testing.T) {
	// 10 degres a gauche et a droite du haut : tous deux dans le secteur 0 (qui chevauche le haut).
	for _, a := range []float64{-10, 10, -22.4, 22.4} {
		if got, _ := Rose8(versAngle(a)); got != 0 {
			t.Errorf("%v deg -> secteur %d, attendu 0", a, got)
		}
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
