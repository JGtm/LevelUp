package tactical

import (
	"math"
	"testing"
)

// echantillonLigne : des candidats alignes sur l'axe x, le reel a l'indice `reel`, au point `au`.
func echantillonLigne(xs []float64, reel int, au float64) Echantillon {
	pos := make([]Point3, len(xs))
	for i, x := range xs {
		pos[i] = Point3{X: x}
	}
	return Echantillon{Feats: make([][NbVariables]float64, len(xs)), Pos: pos, Reel: reel, Au: Point3{X: au}, ZoneReelle: -1}
}

func procheDe(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestPartTouchee_CasCalculesAMain(t *testing.T) {
	cas := []struct {
		nom     string
		n, k, m int
		attendu float64
	}{
		{"un tirage", 5, 2, 1, 2.0 / 5},
		{"trois tirages parmi 5, 2 dedans : 1 - C(3,3)/C(5,3)", 5, 2, 3, 1 - 1.0/10},
		{"trois tirages parmi 10, 1 dedans : 1 - C(9,3)/C(10,3)", 10, 1, 3, 1 - 84.0/120},
		{"moins de trois candidats : les tirages les epuisent", 3, 1, 3, 1},
		{"deux candidats seulement", 2, 1, 3, 1},
		{"aucun candidat dans le rayon", 6, 0, 3, 0},
		{"tous dans le rayon", 6, 6, 3, 1},
		{"plus de candidats hors rayon que de tirages", 4, 1, 3, 3.0 / 4},
	}
	for _, c := range cas {
		if got := partTouchee(c.n, c.k, c.m); !procheDe(got, c.attendu) {
			t.Errorf("%s : partTouchee(%d, %d, %d) = %v, attendu %v", c.nom, c.n, c.k, c.m, got, c.attendu)
		}
	}
}

// Le hasard du top 3 a 8 m n'est pas la chance d'UN tirage (k/n) : pour 5 candidats dont 2 dans
// le rayon, un tirage touche 40 %, trois tirages 90 %.
func TestEvaluer_HasardDuRayonParEchelle(t *testing.T) {
	s := echantillonLigne([]float64{0, 5, 100, 200, 300}, 0, 0)
	m := Evaluer([]Echantillon{s}, [][]float64{{0, 0, 0, 0, 0}})
	if !procheDe(m.HasardRayon1, 40) || !procheDe(m.HasardRayon3, 90) {
		t.Errorf("hasard du rayon = %v / %v, attendu 40 / 90", m.HasardRayon1, m.HasardRayon3)
	}
	// Moyenne sur les evenements : 90 % puis 1 candidat sur 10 dans le rayon (30 %).
	xs := []float64{0, 100, 200, 300, 400, 500, 600, 700, 800, 900}
	m = Evaluer([]Echantillon{s, echantillonLigne(xs, 0, 0)}, [][]float64{{0, 0, 0, 0, 0}, make([]float64, 10)})
	if !procheDe(m.HasardRayon3, 60) || !procheDe(m.HasardRayon1, 25) {
		t.Errorf("hasard moyen = %v / %v, attendu 60 / 25", m.HasardRayon1, m.HasardRayon3)
	}
}

// A scores egaux, le rayon se tire au hasard entre ex aequo : le reel, quel que soit son rang
// dans la liste des candidats, n'est pas favorise.
func TestEvaluer_EgalitesDuRayonNeFavorisentPasLeReel(t *testing.T) {
	// Reel en tete de liste puis en queue de liste : meme resultat.
	for _, reel := range []int{0, 3} {
		xs := []float64{100, 200, 300, 400}
		xs[reel] = 0
		m := Evaluer([]Echantillon{echantillonLigne(xs, reel, 0)}, [][]float64{{1, 1, 1, 1}})
		if !procheDe(m.Rayon1, 25) || !procheDe(m.Rayon3, 75) {
			t.Errorf("reel en %d, quatre ex aequo : rayon = %v / %v, attendu 25 / 75", reel, m.Rayon1, m.Rayon3)
		}
	}
	// Un groupe au-dessus (hors rayon), puis trois ex aequo dont le reel, une seule place restante
	// a la frontiere : un tirage parmi trois, dont un dedans, 1/3.
	s := echantillonLigne([]float64{900, 100, 0, 300}, 2, 0)
	if got := toucheRayon(s, []float64{5, 3, 3, 3}, 2); !procheDe(got, 1.0/3) {
		t.Errorf("une place pour trois ex aequo dont un dedans : %v, attendu 1/3", got)
	}
	// Un candidat dans le rayon dans le groupe du dessus : touche a coup sur.
	s = echantillonLigne([]float64{3, 100, 200, 300}, 0, 0)
	if got := toucheRayon(s, []float64{5, 3, 3, 3}, 1); got != 1 {
		t.Errorf("le premier est dans le rayon : %v, attendu 1", got)
	}
	// Deux places sur trois ex aequo dont un dedans : 1 - C(2,2)/C(3,2) = 2/3.
	s = echantillonLigne([]float64{900, 0, 300, 400}, 1, 0)
	if got := toucheRayon(s, []float64{5, 3, 3, 3}, 3); !procheDe(got, 1-1.0/3) {
		t.Errorf("deux places pour trois ex aequo dont un dedans : %v, attendu 2/3", got)
	}
}

// Sans egalite, l'ordre des scores decide : le premier est le plus haut.
func TestEvaluer_RayonSansEgalite(t *testing.T) {
	s := echantillonLigne([]float64{0, 100, 200}, 0, 0)
	if toucheRayon(s, []float64{3, 2, 1}, 1) != 1 || toucheRayon(s, []float64{1, 3, 2}, 1) != 0 {
		t.Error("le premier choix est le score le plus haut")
	}
	if toucheRayon(s, []float64{1, 3, 2}, 2) != 0 || toucheRayon(s, []float64{1, 3, 2}, 3) != 1 {
		t.Error("le reel est le dernier : touche seulement au troisieme choix")
	}
}
