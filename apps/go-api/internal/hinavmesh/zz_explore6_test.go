package hinavmesh

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

func TestZZSectionsEtChainesRegion3(t *testing.T) {
	region := chargeRegionsTemoin(t, "01af558d-53ab-4f05-ba68-92d805fc6260")[2]
	sections := map[string][2]int{}
	if err := parcoursSections(region, 0, len(region), sections, 0); err != nil {
		t.Fatalf("sections: %v", err)
	}
	noms := []string{}
	for n := range sections {
		noms = append(noms, n)
	}
	sort.Strings(noms)
	t.Logf("=== SECTIONS region 3 (%d octets) ===", len(region))
	for _, n := range noms {
		t.Logf("  %-6s offset=%-8d taille=%d", n, sections[n][0], sections[n][1])
	}
	for _, sec := range []string{"TST1", "FST1"} {
		c := chaines(region, sections[sec])
		t.Logf("=== %s : %d chaines ===", sec, len(c))
		t.Logf("  %s", strings.Join(c, " | "))
	}
	if s, ok := sections["SDKV"]; ok {
		t.Logf("  SDKV = %q", region[s[0]:s[0]+s[1]])
	}
}

// TestZZChasseAuxChaines balaie TOUTE la charge inflatee (les 5 regions) a la recherche
// de suites ASCII imprimables, et signale celles qui tombent HORS des sections de noms
// de types/champs Havok (TST1/FST1) — les seules ou une chaine est attendue.
func TestZZChasseAuxChaines(t *testing.T) {
	for _, asset := range []string{"01af558d-53ab-4f05-ba68-92d805fc6260", "df7dbf08-b8de-4ade-9d7f-1947128c9ae4"} {
		d := chargeRegionsTemoin(t, asset)
		t.Logf("=== CHASSE AUX CHAINES, %s ===", asset)
		for i, r := range d {
			zonesNoms := [][2]int{}
			if len(r) >= 8 && string(r[4:8]) == "TAG0" {
				sec := map[string][2]int{}
				if err := parcoursSections(r, 0, len(r), sec, 0); err == nil {
					for _, n := range []string{"TST1", "FST1", "SDKV"} {
						if s, ok := sec[n]; ok {
							zonesNoms = append(zonesNoms, [2]int{s[0], s[0] + s[1]})
						}
					}
				}
			}
			dansNoms := func(p int) bool {
				for _, z := range zonesNoms {
					if p >= z[0] && p < z[1] {
						return true
					}
				}
				return false
			}
			var trouvees []string
			debut := -1
			for p := 0; p <= len(r); p++ {
				imprimable := p < len(r) && (r[p] == '_' || r[p] == ':' || r[p] == ' ' || r[p] == '-' ||
					(r[p] >= 'A' && r[p] <= 'Z') || (r[p] >= 'a' && r[p] <= 'z') || (r[p] >= '0' && r[p] <= '9'))
				if imprimable {
					if debut < 0 {
						debut = p
					}
					continue
				}
				if debut >= 0 && p-debut >= 5 && !dansNoms(debut) {
					if len(trouvees) < 25 {
						trouvees = append(trouvees, fmt.Sprintf("@%d %q", debut, r[debut:p]))
					}
				}
				debut = -1
			}
			t.Logf("  region %d (%d octets, %d zones de noms Havok) : %d suites ASCII >=5 hors TST1/FST1/SDKV",
				i+1, len(r), len(zonesNoms), len(trouvees))
			for _, s := range trouvees {
				t.Logf("      %s", s)
			}
		}
	}
}
