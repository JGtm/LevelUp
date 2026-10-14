package equipenergy

import (
	"path/filepath"
	"testing"
)

// cheminDuCatalogue : le catalogue versionne, depuis ce paquet.
func cheminDuCatalogue() string {
	return filepath.Join("..", "..", "..", "..", "..", "..", "data", "titles", "halo_infinite", "reference",
		"equipment_energy.json")
}

// TestUsagesRestantsSuitLeHUD : la regle de `FUN_1407e9384` sur les energies que le film ecrit
// (i24 dequantifie, 14 bits aux extremites exactes) — le repulseur de 0d9a9af9 et 72b0a25e :
// 0,6 -> 3, puis 2, 1, 0 aux activations.
func TestUsagesRestantsSuitLeHUD(t *testing.T) {
	const pas = float32(1) / 16382
	cas := []struct {
		q    uint32
		want int
	}{{9830, 3}, {6553, 2}, {3277, 1}, {1, 0}, {0, 0}, {16383, 5}}
	for _, c := range cas {
		e := float32(c.q-1)*pas + (pas*0.5 + 0)
		switch c.q {
		case 0:
			e = 0
		case 16383:
			e = 1
		}
		n, ok := UsagesRestants(e, 0.2, -0.2)
		if !ok || n != c.want {
			t.Errorf("q=%d (e=%v) : %d usages (%v), attendu %d", c.q, e, n, ok, c.want)
		}
	}
	if _, ok := UsagesRestants(0.6, 0, 0); ok {
		t.Error("une capacite a seuil nul ne compte pas par l energie")
	}
	if _, ok := UsagesRestants(0.6, 0.2, 0); ok {
		t.Error("une variation nulle n est pas un cout")
	}
}

// TestCatalogueCommisSuitLeParc : le catalogue versionne porte les constantes que les baisses
// d energie mesurees au parc exigent (238 films, 2026-10-14).
func TestCatalogueCommisSuitLeParc(t *testing.T) {
	c, err := Load(cheminDuCatalogue())
	if err != nil {
		t.Fatal(err)
	}
	par := map[uint32]Entry{}
	for _, e := range c.Entries {
		if id, ok := e.PickupID(); ok {
			par[id] = e
		}
	}
	attendus := map[uint32][3]float32{ // energie du ramassable, variation, usages a la prise
		0x7ca85adc: {0.6, -0.2, 3},   // repulseur (releve Theater 72b0a25e : 3 charges)
		0x72199cba: {0.5, -0.25, 2},  // capteur
		0x4396db42: {0.5, -0.25, 2},  // ecran occultant
		0x4744d742: {0.5, -0.25, 2},  // traqueur
		0x2974c233: {0.66, -0.33, 2}, // mur
		0x32d97758: {0.5, -0.5, 1},   // champ de reparation
		0x730dc70f: {1, -1, 1},       // translocateur
	}
	for id, a := range attendus {
		e, ok := par[id]
		if !ok {
			t.Errorf("0x%08x absent du catalogue", id)
			continue
		}
		n, okN := UsagesRestants(e.PickupEnergy, e.Threshold, e.Delta)
		if e.PickupEnergy != a[0] || e.Delta != a[1] || !okN || n != int(a[2]) {
			t.Errorf("0x%08x : energie %v, variation %v, %d usages ; attendu %v, %v, %v", id, e.PickupEnergy,
				e.Delta, n, a[0], a[1], a[2])
		}
	}
}

func TestLoadRefuseUneVersionInconnue(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "absent.json")); err == nil {
		t.Fatal("un catalogue absent doit echouer")
	}
}
