package hinavmesh

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// SONDE DES MEMBRES DE hkaiNavMesh dans les deux generations. Le decodeur exige des membres
// nommes ; si la generation TSTR/FSTR n en declare pas certains, il faut savoir LESQUELS avant de
// decider ce qui est indispensable et ce qui ne l est pas.
func TestSondeMembresDuNavMesh(t *testing.T) {
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
				continue
			}
			f, err := lireFichierTag(region)
			if err != nil {
				continue
			}
			for ti, ty := range f.types {
				if ty.Nom != classeNavMesh {
					continue
				}
				var noms []string
				for _, m := range ty.Membres {
					noms = append(noms, m.Nom)
				}
				t.Logf("%s region %d : %s (type %d), %d membres : %s",
					cas.nom, i+1, ty.Nom, ti, len(ty.Membres), strings.Join(noms, ", "))
			}
		}
	}
}
