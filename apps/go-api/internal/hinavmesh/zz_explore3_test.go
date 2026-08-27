package hinavmesh

import (
	"fmt"
	"testing"
)

func TestZZItemsRegion3(t *testing.T) {
	for _, asset := range []string{"01af558d-53ab-4f05-ba68-92d805fc6260", "df7dbf08-b8de-4ade-9d7f-1947128c9ae4"} {
		f := region3(t, asset)
		t.Logf("=== ITEMS region 3, %s (%d items, DATA=%d) ===", asset, len(f.items), len(f.data))
		// Regroupe par type pour ne pas noyer.
		parType := map[string]struct {
			n, total int
			exemples []string
		}{}
		for i, it := range f.items {
			nom := f.nomType(it.Type)
			e := parType[nom]
			e.n++
			e.total += it.Compte
			if len(e.exemples) < 4 {
				e.exemples = append(e.exemples, fmt.Sprintf("#%d off=%d n=%d", i, it.Offset, it.Compte))
			}
			parType[nom] = e
		}
		for nom, e := range parType {
			t.Logf("  type %-45s : %d entrees ITEM, %d elements cumules  ex: %v", nom, e.n, e.total, e.exemples)
		}
	}
}
