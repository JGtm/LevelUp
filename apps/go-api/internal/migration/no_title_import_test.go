package migration

import (
	"os"
	"strings"
	"testing"
)

// TestMigrationDoesNotImportTitlePackage verrouille le découplage MT-07 : le
// package migration (title-agnostique) ne doit JAMAIS importer un package de
// titre (internal/games/halo_infinite/...). La donnée Halo-spécifique (grille de
// rangs) vit dans le package de titre et remonte via le seam provider
// (SetCareerRankTranslationsProvider). Un import direct ré-introduirait la dette.
func TestMigrationDoesNotImportTitlePackage(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	const forbidden = "levelup/go-api/internal/games/"
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if i := strings.Index(string(data), forbidden); i >= 0 {
			t.Errorf("%s importe un package de titre (%s…) — interdit : le package migration "+
				"doit rester title-agnostique (cf. MT-07, seam SetCareerRankTranslationsProvider)",
				name, forbidden)
		}
	}
}
