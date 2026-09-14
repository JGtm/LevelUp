package halo_infinite

// medal_icons_test.go — garde-rail du référentiel d'icônes de médailles.
//
// La page Médailles rend `/static/medals/halo_infinite/{medal_id}.png` pour
// chaque médaille de la taxonomie (medalCategoryTable). Un identifiant sans
// fichier = une vignette cassée en production (constaté le 2026-09-13 sur la
// médaille VIP 1053114074, absente du catalogue officiel GameCMS).
//
// Compléter le référentiel : `go run ./cmd/refresh-metadata medal-images
// --player <GT> --download` (médailles du catalogue officiel) ou `--pin
// <id>:<spriteIndex>` (médailles présentes dans la feuille de sprites mais
// absentes du catalogue). Voir docs/COMMANDS.md.

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"levelup/go-api/internal/testfixtures"
)

func TestMedalCategoryTable_ToutesLesIconesExistent(t *testing.T) {
	dir := filepath.Join(testfixtures.RepoRoot(), "static", "medals", TitleSlug)
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("dossier d'icônes absent (%s): %v", dir, err)
	}
	var manquantes []int64
	for id := range medalCategoryTable {
		path := filepath.Join(dir, strconv.FormatInt(id, 10)+".png")
		if _, err := os.Stat(path); err != nil {
			manquantes = append(manquantes, id)
		}
	}
	if len(manquantes) != 0 {
		t.Errorf("%d médaille(s) de medalCategoryTable sans icône dans %s : %v",
			len(manquantes), dir, manquantes)
	}
}
