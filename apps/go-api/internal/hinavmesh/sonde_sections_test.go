package hinavmesh

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// SONDE DES OCCURRENCES DE SECTION. `parcoursSections` retient la PREMIERE occurrence de chaque
// etiquette — « les etiquettes sont uniques dans un fichier-tag », dit son commentaire. Si cette
// hypothese tombe sur la generation TSTR/FSTR, le decodeur lit les corps de types d un bloc avec
// la table de chaines d un AUTRE, et l indice de champ deborde : exactement le symptome
// d Absolution.
func TestSondeOccurrencesDeSection(t *testing.T) {
	const racine = `C:/Users/Guillaume/Projects/LevelUp/.ai/re_dump/navmesh`
	for _, cas := range []struct{ nom, blob string }{
		{"Isolation", "01af558d-53ab-4f05-ba68-92d805fc6260.blob"},
		{"Absolution", "78da545f-a168-4a5e-9c8d-dd379067c352.blob"},
	} {
		brut, err := os.ReadFile(filepath.Join(racine, cas.blob))
		if err != nil {
			t.Skipf("blob absent (%v)", err)
		}
		charge, err := decompresse(brut)
		if err != nil {
			t.Fatalf("%s : %v", cas.nom, err)
		}
		decoupe, err := regions(charge)
		if err != nil {
			t.Fatalf("%s : %v", cas.nom, err)
		}
		t.Logf("%s : %d region(s)", cas.nom, len(decoupe))
		for ri, region := range decoupe {
			if len(region) < 8 || string(region[4:8]) != sectionTAG0 {
				continue
			}
			compte := map[string]int{}
			var visite func(debut, fin, prof int)
			visite = func(debut, fin, prof int) {
				if prof > profondeurSectionsMax {
					return
				}
				for p := debut; p+8 <= fin; {
					entete := binary.BigEndian.Uint32(region[p:])
					taille := int(entete & 0x3FFFFFFF)
					if taille < 8 || p+taille > fin {
						return
					}
					compte[string(region[p+4:p+8])]++
					if entete&0x40000000 == 0 {
						visite(p+8, p+taille, prof+1)
					}
					p += taille
				}
			}
			visite(0, len(region), 0)
			for _, tag := range []string{"TYPE", "TST1", "FST1", "TSTR", "FSTR", "TNA1", "TBDY", "ITEM", "DATA", "INDX"} {
				if n := compte[tag]; n > 1 {
					t.Logf("  region %d : %s apparait %d FOIS", ri+1, tag, n)
				} else if n == 1 {
					t.Logf("  region %d : %s x1", ri+1, tag)
				}
			}
		}
	}
}
