package main

import (
	"os"
	"path/filepath"
	"strings"
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

// manifesteTrois fabrique un manifeste minimal de trois temoins, pour les tests de --temoins.
func manifesteTrois() Manifest {
	var m Manifest
	m.Meta.TitleSlug = "halo_infinite"
	m.Temoins = []Temoin{
		{ID: "c75f33b8", Famille: "assaut"},
		{ID: "111fa685", Famille: "btb"},
		{ID: "bcb6d393", Famille: "ctf_mono_manche"},
	}
	return m
}

// TestFiltrerTemoinsGardeLesNommesDansLOrdreDemande — D2 (cloture M1) : rejouer les seuls
// temoins perdus par l'alea de l'export, sans ecrire un manifeste reduit a la main. L'ordre est
// celui de l'operateur, pas celui du manifeste.
func TestFiltrerTemoinsGardeLesNommesDansLOrdreDemande(t *testing.T) {
	m, err := FiltrerTemoins(manifesteTrois(), NomsDemandes("111fa685, c75f33b8"))
	if err != nil {
		t.Fatalf("filtre sur deux ids connus : %v", err)
	}
	if len(m.Temoins) != 2 {
		t.Fatalf("%d temoin(s) gardes, 2 attendus : %+v", len(m.Temoins), m.Temoins)
	}
	if m.Temoins[0].ID != "111fa685" || m.Temoins[1].ID != "c75f33b8" {
		t.Errorf("ordre = %s, %s — celui de --temoins attendu", m.Temoins[0].ID, m.Temoins[1].ID)
	}
	if m.Temoins[0].Famille != "btb" {
		t.Errorf("le temoin garde a perdu sa famille : %+v", m.Temoins[0])
	}
}

// TestFiltrerTemoinsVideRendLeManifesteEntier — sans --temoins, le corpus complet : le chemin
// par defaut ne doit rien filtrer du tout.
func TestFiltrerTemoinsVideRendLeManifesteEntier(t *testing.T) {
	m, err := FiltrerTemoins(manifesteTrois(), NomsDemandes(""))
	if err != nil {
		t.Fatalf("filtre vide : %v", err)
	}
	if len(m.Temoins) != 3 {
		t.Fatalf("%d temoin(s), 3 attendus — un filtre vide ne filtre pas", len(m.Temoins))
	}
}

// TestFiltrerTemoinsRefuseUnIdInconnu — UNE FAUTE DE FRAPPE N'EST PAS UN CORPUS REDUIT. Un id
// inconnu doit rendre une erreur nommant l'intrus ET les ids connus, jamais un manifeste
// ampute en silence (le meme silence que verifierCouverture interdit).
func TestFiltrerTemoinsRefuseUnIdInconnu(t *testing.T) {
	_, err := FiltrerTemoins(manifesteTrois(), NomsDemandes("c75f33b8,c75f33b9"))
	if err == nil {
		t.Fatal("un id absent du manifeste doit etre une erreur, pas un filtre silencieux")
	}
	for _, attendu := range []string{"c75f33b9", "bcb6d393"} {
		if !strings.Contains(err.Error(), attendu) {
			t.Errorf("le message doit nommer l'intrus et les ids connus, %q manquant : %v", attendu, err)
		}
	}
}

// TestFiltrerTemoinsDedoublonne — `--temoins a,a` est une maladresse, pas une demande de cuire
// deux fois le meme film (chaque cuisson coute 1 a 3 min).
func TestFiltrerTemoinsDedoublonne(t *testing.T) {
	m, err := FiltrerTemoins(manifesteTrois(), NomsDemandes("c75f33b8,c75f33b8"))
	if err != nil {
		t.Fatalf("filtre avec doublon : %v", err)
	}
	if len(m.Temoins) != 1 {
		t.Fatalf("%d temoin(s), 1 attendu : un id repete ne cuit pas deux fois", len(m.Temoins))
	}
}

// TestNomsDemandesTolereEspacesEtVides — la valeur est tapee a la main sur une ligne de
// commande : « a, b ,, c » vaut trois ids.
func TestNomsDemandesTolereEspacesEtVides(t *testing.T) {
	got := NomsDemandes(" a, b ,, c ")
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("NomsDemandes = %q, [a b c] attendu", got)
	}
}
