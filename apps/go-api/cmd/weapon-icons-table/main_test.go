package main

import (
	"bytes"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/testfixtures"
)

func indexPath() string {
	return filepath.Join(testfixtures.RepoRoot(),
		"static", "weapons-assets", "halo_infinite", "jeu", "index.json")
}

func generatedPath() string {
	return filepath.Join(testfixtures.RepoRoot(),
		"apps", "go-api", "internal", "games", "halo_infinite", "weapon_icons_table.go")
}

// TestTableGenereeEstAJour est LE garde-rail de ce chemin : il rejoue la génération
// et compare octet à octet au fichier versionné. Sans lui, la table serait une COPIE
// d'index.json qui vieillit en silence — exactement le défaut vécu par les tables de
// la page de nommage, recopiées à la main (cf. cmd/weapon-icons-build/page.go).
//
// Il tourne partout, y compris en CI : la génération ne lit qu'index.json, versionné.
func TestTableGenereeEstAJour(t *testing.T) {
	entries, err := loadEntries(indexPath())
	if err != nil {
		t.Fatalf("index.json : %v", err)
	}
	table, err := BuildTable(entries)
	if err != nil {
		t.Fatalf("BuildTable : %v", err)
	}
	attendu, err := format.Source(Render(table))
	if err != nil {
		t.Fatalf("gofmt : %v", err)
	}
	obtenu, err := os.ReadFile(generatedPath())
	if err != nil {
		t.Fatalf("fichier généré : %v", err)
	}
	if !bytes.Equal(bytes.ReplaceAll(obtenu, []byte("\r\n"), []byte("\n")), attendu) {
		t.Error("weapon_icons_table.go diverge d'index.json — rejouer " +
			"`cd apps/go-api && go run ./cmd/weapon-icons-table` et committer le résultat")
	}
}

// TestSeulLAtlasContourEstLu verrouille la décision qui évite l'icône fausse :
// index.json porte `tags_weap` sur les TROIS styles, mais ce champ est indexé par
// `sprite index`, qui ne vaut que pour les atlas d'armes. L'atlas du kill feed a sa
// propre numérotation — son entrée 0 est le Battle Rifle là où le contour 0 est le
// fusil d'assaut. Lire le kill feed rendrait une icône fausse pour CHAQUE arme.
func TestSeulLAtlasContourEstLu(t *testing.T) {
	entries, err := loadEntries(indexPath())
	if err != nil {
		t.Fatalf("index.json : %v", err)
	}
	table, err := BuildTable(entries)
	if err != nil {
		t.Fatalf("BuildTable : %v", err)
	}
	if len(table) == 0 {
		t.Fatal("table vide")
	}
	for _, tf := range table {
		if !strings.HasPrefix(tf.File, atlasStyle) {
			t.Errorf("tag %08x -> %q : hors atlas %q", tf.Tag, tf.File, atlasStyle)
		}
	}
	// Contrôle de non-vacuité : les autres styles existent bien dans la source, donc
	// le filtre travaille réellement.
	autres := 0
	for _, e := range entries {
		if e.Style != atlasStyle {
			autres++
		}
	}
	if autres == 0 {
		t.Fatal("index.json ne porte qu'un style : le filtre ne prouve rien")
	}
}
