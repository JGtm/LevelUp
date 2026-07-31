package himap

import (
	"math"
	"os"
	"testing"
)

// Ces tests lisent les fichiers du jeu installé. Ils sont IGNORÉS quand l'installation
// est absente (CI) : ils servent de garde-rail sur poste de développement, là où la
// régression serait sinon invisible.
const (
	gameRoot     = `D:/SteamLibrary/steamapps/common/Halo Infinite/deploy`
	ridgelinePC  = gameRoot + `/pc/levels/multi/ridgeline/ridgeline-rtx-new.module`
	ridgelineDS  = gameRoot + `/ds/levels/multi/ridgeline/ridgeline-rtx-new.module`
	catalystAny  = gameRoot + `/any/levels/multi/catalyst/catalyst-rtx-new.module`
	minInstances = 10000
)

func requireModule(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Skipf("module absent (%s)", path)
	}
}

// TestPluginInstanceStride verrouille l'accord entre le plugin sbsp.xml et la taille
// d'enregistrement câblée : toute dérive du plugin doit casser ici, pas en silence.
func TestPluginInstanceStride(t *testing.T) {
	got, err := PluginInstanceStride()
	if err != nil {
		t.Fatal(err)
	}
	if got != instanceStride {
		t.Fatalf("stride plugin=%#x, attendu %#x", got, instanceStride)
	}
}

// TestRidgelineInstancesGeometry vérifie les invariants géométriques du bloc
// `instanced geometry instances` : base orthonormée et encadrement sphère/AABB.
// Ce dernier est l'ORACLE INTERNE par instance — il ne peut passer que si les offsets
// de la base, de l'AABB et de la sphère sont tous corrects simultanément.
func TestRidgelineInstancesGeometry(t *testing.T) {
	requireModule(t, ridgelinePC)
	bsps, err := ReadModuleInstances(ridgelinePC)
	if err != nil {
		t.Fatal(err)
	}
	var ins []Instance
	best := math.Inf(1)
	for _, b := range bsps {
		if len(b.Instances) == 0 || !b.Bounds.Valid() {
			continue
		}
		if v := b.Bounds.Extent(0) * b.Bounds.Extent(1) * b.Bounds.Extent(2); v < best {
			best, ins = v, b.Instances
		}
	}
	if len(ins) < minInstances {
		t.Fatalf("%d instances, attendu >= %d", len(ins), minInstances)
	}
	for _, in := range ins {
		for _, v := range [3][3]float64{in.Forward, in.Left, in.Up} {
			if n := math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2]); math.Abs(n-1) > 1e-3 {
				t.Fatalf("instance %d : base non unitaire (|v|=%f)", in.Index, n)
			}
		}
		var maxHalf float64
		for a := 0; a < 3; a++ {
			if h := (in.AABBMax[a] - in.AABBMin[a]) / 2; h > maxHalf {
				maxHalf = h
			}
		}
		if in.Radius < maxHalf-1e-3 {
			t.Fatalf("instance %d : rayon %f < demi-extension max %f (offsets AABB/rayon incohérents)",
				in.Index, in.Radius, maxHalf)
		}
	}
}
