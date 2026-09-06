package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadManifestValide — le manifeste versionne reel doit rester chargeable : ce test le
// relit tel quel (pas une fixture), donc rougit si son format derive sans que ce fichier ne
// suive.
func TestLoadManifestValide(t *testing.T) {
	// Depuis apps/go-api/cmd/replay-corpus-gate/, la racine du depot est trois niveaux au-dessus.
	path := filepath.Join("..", "..", "..", "..", "config", "replay_corpus.toml")
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("le manifeste versionne doit rester chargeable : %v", err)
	}
	if m.Meta.TitleSlug == "" {
		t.Fatal("title_slug absent du manifeste versionne")
	}
	if len(m.Temoins) == 0 {
		t.Fatal("le manifeste versionne ne porte aucun temoin")
	}
	vus := map[string]bool{}
	for _, tm := range m.Temoins {
		if vus[tm.ID] {
			t.Errorf("temoin %s liste plusieurs fois", tm.ID)
		}
		vus[tm.ID] = true
		if tm.Raison == "" {
			t.Errorf("temoin %s sans raison — le manifeste doit justifier chaque entree", tm.ID)
		}
	}
}

// TestLoadManifestSansTitleSlug — un manifeste sans title_slug est une erreur de config, pas
// un titre par defaut silencieux (title-agnostic : jamais deviner).
func TestLoadManifestSansTitleSlug(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "m.toml")
	ecrireFixtureTOML(t, path, `
[meta]
schema_version = 1

[[temoin]]
id = "aaaaaaaa"
famille = "test"
raison = "x"
`)
	if _, err := LoadManifest(path); err == nil {
		t.Fatal("un manifeste sans title_slug doit etre refuse")
	}
}

// TestLoadManifestSansTemoin — un manifeste vide n'a rien a comparer : erreur franche.
func TestLoadManifestSansTemoin(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "m.toml")
	ecrireFixtureTOML(t, path, `
[meta]
title_slug = "halo_infinite"
schema_version = 1
`)
	if _, err := LoadManifest(path); err == nil {
		t.Fatal("un manifeste sans temoin doit etre refuse")
	}
}

// TestLoadManifestTemoinSansID — un temoin sans identite ne peut rien designer au parc.
func TestLoadManifestTemoinSansID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "m.toml")
	ecrireFixtureTOML(t, path, `
[meta]
title_slug = "halo_infinite"
schema_version = 1

[[temoin]]
famille = "test"
raison = "x"
`)
	if _, err := LoadManifest(path); err == nil {
		t.Fatal("un temoin sans id doit etre refuse")
	}
}

// TestLoadManifestFichierAbsent — un chemin illisible ne doit jamais rendre un manifeste vide
// silencieux.
func TestLoadManifestFichierAbsent(t *testing.T) {
	if _, err := LoadManifest(filepath.Join(t.TempDir(), "absent.toml")); err == nil {
		t.Fatal("un manifeste absent doit etre une erreur")
	}
}

func ecrireFixtureTOML(t *testing.T, path, texte string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(texte), 0o600); err != nil {
		t.Fatalf("ecriture de la fixture : %v", err)
	}
}
