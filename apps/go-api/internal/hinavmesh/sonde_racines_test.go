package hinavmesh

import (
	"os"
	"path/filepath"
	"testing"
)

// SONDE DES RACINES. `Decode` retient la region dont la racine est un `hkaiNavMesh`. Absolution
// n en a apparemment aucune : cette sonde dit ce que chaque region porte VRAIMENT, et combien
// d entrees TBDY ont du etre franchies pour y arriver.
func TestSondeRacinesDesRegions(t *testing.T) {
	for _, cas := range []struct{ nom, blob string }{
		{"Isolation", "01af558d-53ab-4f05-ba68-92d805fc6260.blob"},
		{"Absolution", "78da545f-a168-4a5e-9c8d-dd379067c352.blob"},
	} {
		brut, err := os.ReadFile(filepath.Join(
			`C:/Users/Guillaume/Projects/LevelUp/.ai/re_dump/navmesh`, cas.blob))
		if err != nil {
			t.Skipf("blob absent (%v)", err)
		}
		charge, err := decompresse(brut)
		if err != nil {
			t.Fatal(err)
		}
		decoupe, err := regions(charge)
		if err != nil {
			t.Fatal(err)
		}
		for i, region := range decoupe {
			if len(region) < 8 || string(region[4:8]) != sectionTAG0 {
				t.Logf("%s region %d : pas de TAG0", cas.nom, i+1)
				continue
			}
			f, err := lireFichierTag(region)
			if err != nil {
				t.Logf("%s region %d : ILLISIBLE : %v", cas.nom, i+1, err)
				continue
			}
			racine, err := f.racine()
			if err != nil {
				t.Logf("%s region %d : pas de racine : %v", cas.nom, i+1, err)
				continue
			}
			var opaques []string
			for _, ty := range f.types {
				if ty.Opaque {
					opaques = append(opaques, ty.Nom)
				}
			}
			t.Logf("%s region %d : racine = %s | %d types (%d franchis : %v) | %d items",
				cas.nom, i+1, f.nomType(racine.Type), len(f.types), f.recuperations, opaques, len(f.items))
		}
	}
}
