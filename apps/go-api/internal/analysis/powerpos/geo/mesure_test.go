package geo

import (
	"math"
	"testing"
)

// TestVariablesSurLaScene : H, V, E, M lisent la scene comme un joueur.
func TestVariablesSurLaScene(t *testing.T) {
	res, b := mesureTemoin(t)
	g, noeuds := res.Graphe, res.Noeuds
	if b.Cibles == 0 || b.Rayons == 0 {
		t.Fatalf("cibles %d, rayons %d", b.Cibles, b.Rayons)
	}
	dalle, sol := noeudEn(t, g, 17, 17, 2), noeudEn(t, g, 7, 12, 0)
	// H : la dalle surplombe, le sol est en creux ou neutre.
	if noeuds[dalle].Brut.H <= 0.5 {
		t.Errorf("H sur la dalle = %.2f, > 0,5 attendu", noeuds[dalle].Brut.H)
	}
	if noeuds[sol].Brut.H > 0.1 {
		t.Errorf("H au sol ouvert = %.2f, <= 0,1 attendu", noeuds[sol].Brut.H)
	}
	// V : depuis la dalle on voit plus que depuis un recoin derriere le mur.
	recoin := noeudEn(t, g, 9, 6, 0)
	if noeuds[dalle].Brut.V <= noeuds[recoin].Brut.V {
		t.Errorf("V dalle %.2f <= V recoin %.2f", noeuds[dalle].Brut.V, noeuds[recoin].Brut.V)
	}
	// E : le recoin est vu de moins de secteurs que le milieu du terrain ouvert.
	milieu := noeudEn(t, g, 6, 12, 0)
	if noeuds[recoin].Brut.E >= noeuds[milieu].Brut.E {
		t.Errorf("E recoin %.2f >= E milieu %.2f", noeuds[recoin].Brut.E, noeuds[milieu].Brut.E)
	}
	// M : pres du mur, le couvert est plus proche qu'au milieu du terrain.
	presDuMur := noeudEn(t, g, 11.5, 6, 0)
	if noeuds[presDuMur].DCouvert >= noeuds[milieu].DCouvert {
		t.Errorf("couvert pres du mur %.1f m >= milieu %.1f m", noeuds[presDuMur].DCouvert, noeuds[milieu].DCouvert)
	}
	// R : la distance a l'arme forte est finie partout et nulle a son pied.
	pied := noeudEn(t, g, 12, 12, 0)
	if noeuds[pied].DArmeForte > 0.8 {
		t.Errorf("distance a l'arme forte a son pied = %.2f", noeuds[pied].DArmeForte)
	}
	for i, n := range noeuds {
		if math.IsInf(n.DArmeForte, 1) || math.IsInf(n.DObjectif, 1) {
			t.Fatalf("noeud %d (%v, %v, %v) sans chemin vers une ressource", i, n.X, n.Y, n.Z)
		}
	}
	t.Logf("noeuds %d, cibles %d, rayons %d, durees %v", len(noeuds), b.Cibles, b.Rayons, b.Durees)
}

// TestScoreEtSelection : le score fige retient la dalle haute, et rien sous le seuil.
func TestScoreEtSelection(t *testing.T) {
	res, _ := mesureTemoin(t)
	Score(res.Noeuds, ReglageGeoV1())
	for _, n := range res.Noeuds {
		for _, v := range []float64{n.Norm.H, n.Norm.V, n.Norm.E, n.Norm.R, n.Norm.M} {
			if v < 0 || v > 1 || math.IsNaN(v) {
				t.Fatalf("variable normalisee hors [0,1] : %+v", n.Norm)
			}
		}
	}
	positions := Selectionne(res.Graphe, res.Noeuds, ReglageGeoV1())
	if len(positions) == 0 {
		t.Fatal("aucune position retenue sur la scene")
	}
	p := positions[0]
	// La dalle, avec sa rampe (qui passe aussi le seuil : elle monte et voit) — le centre
	// est dans le coin nord-est et le sommet a 2 m.
	if p.ZMax < 1.9 || p.CentreX < 10 || p.CentreY < 14 {
		t.Errorf("la premiere position n'est pas sur la dalle (centre %.1f, %.1f ; z max %.2f) : %+v",
			p.CentreX, p.CentreY, p.ZMax, p.Norm)
	}
	if len(p.Polygone) < 3 || p.AireM2 <= 0 {
		t.Errorf("polygone %v, aire %.2f", p.Polygone, p.AireM2)
	}
	for _, q := range positions {
		if len(q.Noeuds) < ReglageGeoV1().TailleMiniComposante {
			t.Errorf("position de %d noeuds sous la taille minimale", len(q.Noeuds))
		}
	}
	t.Logf("%d positions ; premiere : centre (%.1f, %.1f) z [%.1f, %.1f] score %.3f aire %.1f m2 norm %+v",
		len(positions), p.CentreX, p.CentreY, p.ZMin, p.ZMax, p.ScoreMoyen, p.AireM2, p.Norm)
}

// TestNormalise : min-max robuste, serie plate a 0,5, NaN a 0.
func TestNormalise(t *testing.T) {
	out := Normalise([]float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	if out[0] != 0 || out[10] != 1 || math.Abs(out[5]-0.5) > 0.01 {
		t.Errorf("normalise = %v", out)
	}
	if plat := Normalise([]float64{3, 3, 3}); plat[0] != 0.5 {
		t.Errorf("serie plate = %v", plat)
	}
	if avecNaN := Normalise([]float64{math.NaN(), 1, 2}); avecNaN[0] != 0 {
		t.Errorf("NaN normalise = %v", avecNaN)
	}
}

// TestProximite : sous le pied 1, au bout 0, infini 0.
func TestProximite(t *testing.T) {
	if Proximite(0, 20) != 1 || Proximite(20, 20) != 0 || Proximite(40, 20) != 0 || Proximite(math.Inf(1), 20) != 0 {
		t.Error("proximite")
	}
	if p := Proximite(5, 20); math.Abs(p-0.75) > 1e-9 {
		t.Errorf("proximite(5, 20) = %v", p)
	}
}

// TestCorrelation : identique 1, opposee -1, plate NaN.
func TestCorrelation(t *testing.T) {
	a := []float64{1, 2, 3, 4}
	if c := Correlation(a, a); math.Abs(c-1) > 1e-9 {
		t.Errorf("corr(a, a) = %v", c)
	}
	if c := Correlation(a, []float64{4, 3, 2, 1}); math.Abs(c+1) > 1e-9 {
		t.Errorf("corr(a, -a) = %v", c)
	}
	if c := Correlation(a, []float64{1, 1, 1, 1}); !math.IsNaN(c) {
		t.Errorf("corr(a, plat) = %v", c)
	}
}

// TestRefusSansTriangle : rien a mesurer est une erreur, pas un zero.
func TestRefusSansTriangle(t *testing.T) {
	e := entreesTemoin(t)
	e.Triangles = nil
	if _, _, err := Mesure(t.Context(), e, ParametresParDefaut()); err == nil {
		t.Error("mesure sans triangle acceptee")
	}
}
