package hinavmesh

import (
	_ "encoding/binary"
	"fmt"
	_ "math"
	"os"
	"path/filepath"
	_ "sort"
	_ "strings"
	"testing"
)

func chargeRegionsTemoin(t *testing.T, assetID string) [][]byte {
	t.Helper()
	blob, err := os.ReadFile(filepath.Join("testdata", assetID+".navmesh.blob"))
	if err != nil {
		t.Fatalf("lecture: %v", err)
	}
	charge, err := decompresse(blob)
	if err != nil {
		t.Fatalf("decompresse: %v", err)
	}
	d, err := regions(charge)
	if err != nil {
		t.Fatalf("regions: %v", err)
	}
	return d
}

func TestZZExploreRegions(t *testing.T) {
	for _, asset := range []string{"01af558d-53ab-4f05-ba68-92d805fc6260", "df7dbf08-b8de-4ade-9d7f-1947128c9ae4"} {
		d := chargeRegionsTemoin(t, asset)
		t.Logf("=== ASSET %s ===", asset)
		for i, r := range d {
			tag := ""
			if len(r) >= 8 {
				tag = fmt.Sprintf("%q", r[4:8])
			}
			if len(r) < 8 || string(r[4:8]) != "TAG0" {
				t.Logf("region %d: %d octets, PAS un tagfile (tete=%s, hexa=% x)", i+1, len(r), tag, r[:min(32, len(r))])
				continue
			}
			f, err := lireFichierTag(r)
			if err != nil {
				t.Logf("region %d: %d octets, tagfile ILLISIBLE: %v", i+1, len(r), err)
				continue
			}
			rac, err := f.racine()
			if err != nil {
				t.Logf("region %d: racine: %v", i+1, err)
				continue
			}
			t.Logf("region %d: %d octets, DATA=%d, %d types, %d items, racine=%s (item[1] off=%d compte=%d)",
				i+1, len(r), len(f.data), len(f.types), len(f.items), f.nomType(rac.Type), rac.Offset, rac.Compte)
		}
	}
}
