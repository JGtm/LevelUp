package sync

import (
	"os"
	"strings"
	"testing"
)

// TestTransformsDoNotImportTitlePackage verrouille le découplage MT-14 :
// l'extraction JSON→DB (transforms.go + transforms_helpers.go) ne doit plus
// importer le package de titre halo_infinite. La valeur mode_category écrite dans
// match_registry est une colonne opaque (write-only) → constantes locales sync
// (mode_category.go). Les AUTRES fichiers de sync peuvent encore importer
// halo_infinite (hors scope MT-14/MT-15) — d'où le guard ciblé par fichier.
func TestTransformsDoNotImportTitlePackage(t *testing.T) {
	const forbidden = "levelup/go-api/internal/games/halo_infinite"
	for _, f := range []string{"transforms.go", "transforms_helpers.go"} {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if strings.Contains(string(data), forbidden) {
			t.Errorf("%s importe %s — interdit (MT-14) : la catégorie de mode est une "+
				"valeur de colonne opaque, utiliser les constantes locales de mode_category.go", f, forbidden)
		}
	}
}
