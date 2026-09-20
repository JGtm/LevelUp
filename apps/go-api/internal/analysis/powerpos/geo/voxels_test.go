package geo

import (
	"testing"

	"levelup/go-api/internal/analysis/tactical"
)

func cadreTest(t *testing.T, minX, minY, maxX, maxY float64) Cadre {
	t.Helper()
	c, err := NouveauCadre(tactical.GrilleParDefaut(), minX, minY, maxX, maxY)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// TestVoxeliseSolHorizontal : un sol a z = 1 marque la couche qui contient z = 1 sous
// chaque cellule couverte, et rien au-dessus.
func TestVoxeliseSolHorizontal(t *testing.T) {
	c := cadreTest(t, 0, 0, 4, 4)
	v := NouveauVolume(c, 0, 4, 0.25, 0.25)
	if n := v.Voxelise(quad(0, 0, 4, 4, 1)); n != 2 {
		t.Fatalf("triangles utiles = %d, 2 attendus", n)
	}
	k := v.Couche(1.0)
	// Le volume deborde le sol d'une demi-cellule tactique (le cadre s'aligne sur la grille
	// de 0,5 m) : on ne controle que les colonnes SOUS le sol.
	iMax, jMax := v.ColonneDe(3.99, 3.99)
	for i := 0; i <= iMax; i++ {
		for j := 0; j <= jMax; j++ {
			if !v.Occupe(i, j, k) {
				t.Errorf("cellule (%d,%d) couche %d non marquee", i, j, k)
			}
			if v.Occupe(i, j, k+2) {
				t.Errorf("cellule (%d,%d) couche %d marquee a tort", i, j, k+2)
			}
		}
	}
}

// TestVoxeliseMurVertical : un mur mince marque toute sa colonne, et seulement elle.
func TestVoxeliseMurVertical(t *testing.T) {
	c := cadreTest(t, 0, 0, 4, 4)
	v := NouveauVolume(c, 0, 4, 0.25, 0.25)
	v.Voxelise(boite(1.9, 0, 0, 2.1, 4, 3))
	i, j := v.ColonneDe(2.0, 3.0) // le mur, epais de 20 cm, tient dans cette colonne et sa voisine
	for k := 0; k < v.Couche(3.0); k++ {
		if !v.Occupe(i, j, k) && !v.Occupe(i-1, j, k) {
			t.Errorf("couche %d : mur absent des colonnes %d et %d", k, i-1, i)
		}
	}
	if v.Occupe(0, 0, 4) || v.Occupe(v.NX()-1, v.NY()-1, 4) {
		t.Error("le mur a marque des voxels loin de lui")
	}
}

// TestRayonLibre : un rayon traverse le vide, s'arrete sur un mur, et n'est pas gene par un
// sol sous lui.
func TestRayonLibre(t *testing.T) {
	c := cadreTest(t, 0, 0, 10, 10)
	v := NouveauVolume(c, -1, 5, 0.25, 0.25)
	v.Voxelise(quad(0, 0, 10, 10, 0))
	if !v.RayonLibre([3]float64{1, 5, 1.7}, [3]float64{9, 5, 1.7}) {
		t.Error("rayon a 1,7 m au-dessus d'un sol plat bloque a tort")
	}
	v.Voxelise(boite(4.9, 0, 0, 5.1, 10, 3))
	if v.RayonLibre([3]float64{1, 5, 1.7}, [3]float64{9, 5, 1.7}) {
		t.Error("rayon a travers un mur de 3 m declare libre")
	}
	if !v.RayonLibre([3]float64{1, 5, 3.5}, [3]float64{9, 5, 3.5}) {
		t.Error("rayon au-dessus du mur bloque a tort")
	}
	if !v.RayonLibre([3]float64{1, 1, 1.7}, [3]float64{1, 9, 1.7}) {
		t.Error("rayon parallele au mur, de son cote, bloque a tort")
	}
}

// TestNormaleZ : un sol regarde vers le haut, un mur non.
func TestNormaleZ(t *testing.T) {
	if n := normaleZ(quad(0, 0, 1, 1, 0)[0]); n < 0.999 {
		t.Errorf("normale d'un sol = %v", n)
	}
	mur := Triangle{{0, 0, 0}, {0, 1, 0}, {0, 1, 1}}
	if n := normaleZ(mur); n > 1e-9 {
		t.Errorf("normale d'un mur = %v", n)
	}
}
