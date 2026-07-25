package duckdb

// weapon_resolver_source_test.go — GARDE-RAIL V72-06 (source unique des noms
// d'armes). Fige le fait que le resolver ne SERT plus le nom « inventé » du registre
// (w.name_fr / w.name) et lit bien la source canonique keyée par weapon_key
// (weapon_name_labels). Sans ce ratchet, on pourrait ré-ajouter le repli registre au
// COALESCE du label et re-diverger. Test source-level (pas de DuckDB) → tourne sans tag.

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWeaponResolverUsesWeaponKeyNameSource(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller indisponible")
	}
	src := filepath.Join(filepath.Dir(thisFile), "weapon_resolver.go")
	raw, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("lecture %s: %v", src, err)
	}
	code := string(raw)

	// La source canonique keyée par weapon_key DOIT être jointe.
	if !strings.Contains(code, "weapon_name_labels") {
		t.Error("weapon_resolver.go ne joint plus weapon_name_labels (source unique des noms cassée)")
	}
	// Le nom « inventé » du registre ne doit PLUS servir de nom d'affichage.
	for _, forbidden := range []string{"w.name_fr", "NULLIF(w.name,"} {
		if strings.Contains(code, forbidden) {
			t.Errorf("weapon_resolver.go référence %q : le repli nom du registre a été ré-introduit (interdit V72-06)", forbidden)
		}
	}
}
