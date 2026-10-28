package himap

import (
	"math"
	"testing"
)

// carreau rend un maillage carre horizontal a l'altitude z, de cote 10 m.
func carreau(z float64) *Mesh {
	return &Mesh{
		Vertices:  [][3]float64{{0, 0, z}, {10, 0, z}, {10, 10, z}, {0, 10, z}},
		Triangles: [][3]int{{0, 1, 2}, {0, 2, 3}},
	}
}

// TestFloorsExigeLeDegagement — un sol couvert de trop pres n'est pas praticable.
//
// Mutation qui doit le faire rougir : rendre Floors sans masquer les bandes surmontees.
func TestFloorsExigeLeDegagement(t *testing.T) {
	v := NewVolumeZ([2]float64{0, 0}, [2]float64{10, 10}, 0, 10)
	v.AddMesh(carreau(0.25), instanceIdentite()) // sol
	v.AddMesh(carreau(1.25), instanceIdentite()) // plafond a 1 m au-dessus
	v.AddMesh(carreau(0.25), instanceIdentite()) // idem, ecriture idempotente
	// 4 bandes = 2 m exiges : le sol du bas est DISQUALIFIE par le plafond, et c'est le
	// PLAFOND qui devient praticable — le mecanisme exact que l'oracle a mesure en grand
	// (l'ajout des modules globaux fait chuter la justesse des altitudes de 63,7 % a 35,9 %).
	sol := v.Floors(4)
	if z, _, ok := sol.SolLePlusProche(5, 5, 0.25); !ok || math.Abs(z-1.25) > 0.26 {
		t.Errorf("degagement 2 m : sol rendu %v (ok=%v), attendu le PLAFOND a 1,25 m", z, ok)
	}
	// 1 bande = 0,5 m exiges : le vrai sol redevient praticable et redevient le plus proche.
	large := v.Floors(1)
	if z, _, ok := large.SolLePlusProche(5, 5, 0.25); !ok || math.Abs(z-0.25) > 0.26 {
		t.Errorf("degagement 0,5 m : sol rendu %v (ok=%v), attendu le vrai sol a 0,25 m", z, ok)
	}
}

// TestSolLePlusProcheChoisitLaBonneBande — entre deux etages, c'est le plus proche des pieds
// qui est rendu, et l'ecart est SIGNE.
//
// Mutation qui doit le faire rougir : rendre la premiere bande occupee au lieu de la plus proche.
func TestSolLePlusProcheChoisitLaBonneBande(t *testing.T) {
	v := NewVolumeZ([2]float64{0, 0}, [2]float64{10, 10}, 0, 12)
	v.AddMesh(carreau(0.25), instanceIdentite())
	v.AddMesh(carreau(6.25), instanceIdentite())

	sol, ecart, ok := v.SolLePlusProche(5, 5, 6.2)
	if !ok || math.Abs(sol-6.25) > 0.26 {
		t.Errorf("a 6,2 m le sol rendu est %v (ok=%v), attendu l'etage haut", sol, ok)
	}
	if ecart > 0.3 || ecart < -0.3 {
		t.Errorf("ecart %v, attendu proche de zero", ecart)
	}
	if _, e, _ := v.SolLePlusProche(5, 5, 0.3); e < -0.3 || e > 0.3 {
		t.Errorf("ecart %v au ras du sol bas, attendu proche de zero", e)
	}
	if _, _, ok := v.SolLePlusProche(50, 50, 0); ok {
		t.Error("hors emprise doit rendre faux")
	}
}

// TestAddMeshBorneEcarteCeQuiDeborde — un maillage qui sort de la boite monde declaree de son
// instance ne depose que la part qui est dedans.
//
// Mutation qui doit le faire rougir : ignorer la boite dans AddMeshBorne. C'est ce bornage qui
// fait tomber les trous de la carte de 11,1 % a 0,6 % (mesure oracle du 2026-08-08).
func TestAddMeshBorneEcarteCeQuiDeborde(t *testing.T) {
	in := instanceIdentite()
	in.AABBMin = [3]float64{0, 0, 0}
	in.AABBMax = [3]float64{5, 10, 1}

	borne := NewVolumeZ([2]float64{0, 0}, [2]float64{10, 10}, 0, 4)
	borne.AddMeshBorne(carreau(0.25), in, 0)
	libre := NewVolumeZ([2]float64{0, 0}, [2]float64{10, 10}, 0, 4)
	libre.AddMesh(carreau(0.25), in)

	if _, _, ok := borne.SolLePlusProche(2, 5, 0.25); !ok {
		t.Error("la part DANS la boite doit etre rasterisee")
	}
	if _, _, ok := borne.SolLePlusProche(8, 5, 0.25); ok {
		t.Error("la part HORS de la boite doit etre ecartee — sinon le bornage ne borne rien")
	}
	if _, _, ok := libre.SolLePlusProche(8, 5, 0.25); !ok {
		t.Error("sans bornage, la meme part doit etre rasterisee (temoin de contraste)")
	}
}

// TestDequantSepareLesDeuxLectures — les deux lectures ne coincident qu'a l'origine du
// quantum ; le temoin existe pour qu'un refactor ne les rende pas identiques en silence.
func TestDequantSepareLesDeuxLectures(t *testing.T) {
	const mn, mx = -1.0, 1.0
	if DequantBrut(0, mn, mx) != mn {
		t.Error("u16 brut : le quantum nul est la borne basse")
	}
	if math.Abs(DequantSigne(0, mn, mx)) > 1e-4 {
		t.Error("i16 decale : le quantum nul est le MILIEU de la plage")
	}
	if DequantBrut(32768, mn, mx) == DequantSigne(32768, mn, mx) {
		t.Error("les deux lectures doivent differer — sinon aucun temoin ne peut les departager")
	}
}
