package himap

import "testing"

// solPlat rend un maillage carre horizontal a l'altitude z, place par une instance identite.
func solPlat(z float64) (*Mesh, Instance) {
	m := &Mesh{
		Vertices:  [][3]float64{{0, 0, z}, {4, 0, z}, {4, 4, z}, {0, 4, z}},
		Triangles: [][3]int{{0, 1, 2}, {0, 2, 3}},
	}
	return m, instanceIdentite()
}

// TestTrancheEcarteLeDecorSousLaCarte — le PLANCHER existe parce que sans lui la carte est
// noyee par le relief situe dessous.
//
// Mesure qui l'a impose (ridgeline, comparaison pixel a pixel avec la carte validee) : sous
// -10 m, 96 % de la matiere dessinee est en trop. Le prototype le portait depuis toujours
// (`s31_raster.py`, ZB0 = -12) ; la traduction en z-buffer ne gardait que le plafond.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : retirer le test de plancher dans `accepteZ` — le sol
// enterre reapparait et la premiere assertion tombe.
func TestTrancheEcarteLeDecorSousLaCarte(t *testing.T) {
	cas := []struct {
		nom     string
		zSol    float64
		tranche bool
		visible bool
	}{
		{"sol enterre, tranche posee", -30, true, false},
		{"sol enterre, aucune tranche", -30, false, true},
		{"sol dans la tranche", 0, true, true},
		{"sol au-dessus du plafond", 40, true, false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			r := NewRendu([2]float64{0, 0}, [2]float64{4, 4}, 1)
			if c.tranche {
				r.Tranche(TrancheDeJeuMin, TrancheDeJeuMax)
			}
			m, in := solPlat(c.zSol)
			r.AddMesh(m, in)
			if _, ok := r.Altitude(1, 1); ok != c.visible {
				t.Fatalf("sol a %+.0f m : visible=%v, attendu %v", c.zSol, ok, c.visible)
			}
		})
	}
}

// TestTrancheNeCoupeQueParLAltitude verifie que le plancher ne tronque pas un sol qui le
// traverse : chaque pixel se juge sur SON altitude, pas sur celle du maillage entier.
func TestTrancheNeCoupeQueParLAltitude(t *testing.T) {
	r := NewRendu([2]float64{0, 0}, [2]float64{4, 4}, 1)
	r.Tranche(-1, 1)
	// Une rampe qui plonge de +0,5 m a -3 m le long de X.
	m := &Mesh{
		Vertices:  [][3]float64{{0, 0, 0.5}, {4, 0, -3}, {4, 4, -3}, {0, 4, 0.5}},
		Triangles: [][3]int{{0, 1, 2}, {0, 2, 3}},
	}
	r.AddMesh(m, instanceIdentite())
	if _, ok := r.Altitude(0, 1); !ok {
		t.Error("le haut de la rampe est dans la tranche, il doit etre dessine")
	}
	if _, ok := r.Altitude(3, 1); ok {
		t.Error("le bas de la rampe est sous le plancher, il ne doit PAS etre dessine")
	}
}
