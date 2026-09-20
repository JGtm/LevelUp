package geo

import (
	"math"
	"testing"
)

// TestSolPraticable : le sol, la rampe et la dalle sont des noeuds ; le dessous du plafond
// bas, l'interieur du mur et le dessus de la caisse (inaccessible) n'en sont pas.
func TestSolPraticable(t *testing.T) {
	res, b := mesureTemoin(t)
	g := res.Graphe
	// Les germes sont les 3 ancres ET les 2 ressources.
	if b.Sol.AncresPlacees != 5 {
		t.Fatalf("germes places = %d/5 (sans noeud %d)", b.Sol.AncresPlacees, b.Sol.AncresSansNoeud)
	}
	p := ParametresParDefaut()
	// Le sol ouvert.
	noeudEn(t, g, 7, 7, 0)
	// La dalle, par la rampe.
	if n := noeudEn(t, g, 17, 17, 2); math.Abs(g.Noeuds[n].Z-2) > 0.01 {
		t.Errorf("dalle : z = %v, 2 attendu", g.Noeuds[n].Z)
	}
	// Sous le plafond bas : aucun noeud au sol (1,2 m de hauteur libre < 1,8) — le DESSUS
	// du plafond, lui, est un sol ou l'on saute (1,2 m <= saut), et c'est voulu.
	if n, ok := g.NoeudLePlusProche(2.5, 2.5, 0, p); ok && g.Noeuds[n].Z < 0.5 {
		t.Errorf("un noeud existe sous le plafond bas (z = %v)", g.Noeuds[n].Z)
	}
	// L'interieur de la dalle pleine (sa face inferieure a z = 0) n'est pas un sol.
	for _, n := range g.Noeuds {
		if n.X > 14 && n.Y > 14 && n.Z < 1 {
			t.Errorf("un noeud existe DANS la dalle pleine (%v)", n)
		}
	}
	// Dans le mur : aucun noeud au sol (la colonne est occupee).
	if n, ok := g.NoeudLePlusProche(10, 6, 0, p); ok && math.Abs(g.Noeuds[n].X-10) < 0.2 {
		t.Errorf("un noeud existe dans le mur (x = %v)", g.Noeuds[n].X)
	}
	// Le dessus de la caisse (1 m : plus qu'une marche, moins qu'un saut) est atteint.
	if n := noeudEn(t, g, 5.5, 15.5, 1); math.Abs(g.Noeuds[n].Z-1) > 0.01 {
		t.Errorf("caisse : z = %v, 1 attendu", g.Noeuds[n].Z)
	}
	// Le dessus du pilier (2,5 m : plus qu'un saut) n'est atteint par personne, meme si l'on
	// peut en tomber — l'atteignabilite suit les arcs sortants des ancres.
	for _, n := range g.Noeuds {
		if n.X > 2 && n.X < 3 && n.Y > 17 && n.Y < 18 && n.Z > 2 {
			t.Errorf("le dessus du pilier est un noeud (%v)", n)
		}
	}
	// Une seule composante ancree, puisque tout se rejoint.
	if b.Sol.Composantes != 1 {
		t.Errorf("composantes ancrees = %d, 1 attendue", b.Sol.Composantes)
	}
	t.Logf("candidats %d, libres %d, noeuds %d", b.Sol.Candidats, b.Sol.SolsLibres, b.Sol.Noeuds)
}

// TestDistanceDeDeplacementContourneLeMur : entre deux points de part et d'autre du mur, la
// distance de deplacement est bien plus longue qu'a vol d'oiseau.
func TestDistanceDeDeplacementContourneLeMur(t *testing.T) {
	res, _ := mesureTemoin(t)
	g := res.Graphe
	a, b := noeudEn(t, g, 8, 6, 0), noeudEn(t, g, 12, 6, 0)
	d := DistancesDepuis(g, []int{a})[b]
	if math.IsInf(d, 1) {
		t.Fatal("les deux cotes du mur ne sont pas relies")
	}
	if d < 8 {
		t.Errorf("distance autour du mur = %.1f m, au moins 8 attendus (4 a vol d'oiseau)", d)
	}
	// Meme cote du mur : la distance est proche de la ligne droite.
	c := noeudEn(t, g, 8, 12, 0)
	if d := DistancesDepuis(g, []int{a})[c]; d > 6.5 || d < 6 {
		t.Errorf("distance en terrain ouvert = %.2f m, 6 a 6,5 attendus", d)
	}
}

// TestGrapheOriente : tomber de la dalle est court, y remonter passe par la rampe ou coute
// le saut — les deux sens ne se valent pas.
func TestGrapheOriente(t *testing.T) {
	res, _ := mesureTemoin(t)
	g := res.Graphe
	haut, bas := noeudEn(t, g, 14.5, 14.5, 2), noeudEn(t, g, 13.5, 13.5, 0)
	descente := DistancesDepuis(g, []int{haut})[bas]
	montee := DistancesVers(g, []int{haut})[bas]
	if descente > 3 {
		t.Errorf("tomber de la dalle coute %.1f m, moins de 3 attendus", descente)
	}
	if montee <= descente {
		t.Errorf("remonter (%.1f m) devrait couter plus que tomber (%.1f m)", montee, descente)
	}
}

// TestDedoublonne : deux sols a moins d'un pas de voxel fusionnent, le plus haut gagne.
func TestDedoublonne(t *testing.T) {
	out := dedoublonne([]float64{2.0, 0.0, 0.1, 2.2}, 0.25)
	if len(out) != 2 || out[0] != 0.1 || out[1] != 2.2 {
		t.Errorf("dedoublonne = %v", out)
	}
}
