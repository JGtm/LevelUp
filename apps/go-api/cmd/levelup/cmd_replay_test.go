package main

// cmd_replay_test.go — `levelup replay compress` : la conversion d'un parc d'artefacts bruts en artefacts
// compressés, vue de la ligne de commande.

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/replayartifact"
)

func poserParcBrut(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	out := map[string][]byte{}
	for _, court := range []string{"a0000001", "a0000002"} {
		doc := []byte(`{"schemaVersion":1,"matchId":"` + court + `","tracks":[{"xuid":"1"}]}`)
		out[court] = doc
		if err := os.WriteFile(filepath.Join(dir, court+".json"), doc, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

func TestReplayCompress_SimulationPuisConversionPuisIdempotence(t *testing.T) {
	repo := t.TempDir()
	cfg := &config.AppConfig{RepoRoot: repo}
	dir := titlePkg.NewPathResolver(repo).ReplayArtifactsDir(titlePkg.DefaultSlug)
	docs := poserParcBrut(t, dir)

	var blanc bytes.Buffer
	if err := runReplayTo(cfg, []string{"compress", "--dry-run"}, &blanc); err != nil {
		t.Fatalf("simulation : %v", err)
	}
	if !strings.Contains(blanc.String(), "2 a convertir") || !strings.Contains(blanc.String(), "simulation") {
		t.Errorf("compte rendu de la simulation = %q", blanc.String())
	}
	for court := range docs {
		if _, err := os.Stat(filepath.Join(dir, court+".json")); err != nil {
			t.Fatalf("la simulation a touche %s : %v", court, err)
		}
	}

	var reel bytes.Buffer
	if err := runReplayTo(cfg, []string{"compress"}, &reel); err != nil {
		t.Fatalf("conversion : %v", err)
	}
	if !strings.Contains(reel.String(), "2 convertis") {
		t.Errorf("compte rendu de la conversion = %q", reel.String())
	}
	for court, doc := range docs {
		got, err := replayartifact.ReadBytes(filepath.Join(dir, court+titlePkg.ReplayArtifactExt))
		if err != nil || !bytes.Equal(got, doc) {
			t.Errorf("%s : contenu relu %q, %v", court, got, err)
		}
	}

	var encore bytes.Buffer
	if err := runReplayTo(cfg, []string{"compress"}, &encore); err != nil {
		t.Fatalf("seconde passe : %v", err)
	}
	if !strings.Contains(encore.String(), "0 convertis") || !strings.Contains(encore.String(), "2 deja compresses") {
		t.Errorf("la seconde passe doit tout trouver deja compresse : %q", encore.String())
	}
}

func TestReplayCompress_DossierExplicite(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "parc")
	poserParcBrut(t, dir)
	var out bytes.Buffer
	if err := runReplayTo(&config.AppConfig{RepoRoot: t.TempDir()}, []string{"compress", "--dir", dir}, &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a0000001.json.gz")); err != nil {
		t.Errorf("--dir doit cibler ce dossier : %v", err)
	}
}

func TestReplayCompress_UnEchecDonneUneErreur(t *testing.T) {
	repo := t.TempDir()
	dir := titlePkg.NewPathResolver(repo).ReplayArtifactsDir(titlePkg.DefaultSlug)
	poserParcBrut(t, dir)
	// Octets magiques gzip, flux tronque : illisible.
	if err := os.WriteFile(filepath.Join(dir, "b0000001.json"), []byte{0x1f, 0x8b, 0x08, 0x00}, 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := runReplayTo(&config.AppConfig{RepoRoot: repo}, []string{"compress"}, &out)
	if err == nil || !strings.Contains(out.String(), "1 en echec") {
		t.Fatalf("un echec doit etre compte et rendre une erreur : %v / %q", err, out.String())
	}
}

func TestReplayCompress_SousCommandeInconnue(t *testing.T) {
	if err := runReplayTo(&config.AppConfig{RepoRoot: t.TempDir()}, []string{"nope"}, &bytes.Buffer{}); err == nil {
		t.Fatal("sous-commande inconnue : erreur attendue")
	}
	if err := runReplayTo(&config.AppConfig{RepoRoot: t.TempDir()}, nil, &bytes.Buffer{}); err == nil {
		t.Fatal("sans sous-commande : erreur attendue")
	}
}

// TestReplayCompress_DoublonsIdentiqueEtDifferent : un doublon identique est résorbé, un doublon différent n'est
// rien supprimé ; le compte rendu le dit.
func TestReplayCompress_DoublonsIdentiqueEtDifferent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "parc")
	docs := poserParcBrut(t, dir) // a0000001 et a0000002 bruts
	// a0000001 : son .gz existe déjà, identique au .json.
	gz, err := replayartifact.Encode(docs["a0000001"])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a0000001.json.gz"), gz, 0o644); err != nil {
		t.Fatal(err)
	}
	// a0000002 : son .gz diffère du .json.
	gz2, err := replayartifact.Encode([]byte(`{"autre":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a0000002.json.gz"), gz2, 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := runReplayTo(&config.AppConfig{RepoRoot: t.TempDir()}, []string{"compress", "--dir", dir}, &out); err != nil {
		t.Fatalf("%v / %s", err, out.String())
	}
	if !strings.Contains(out.String(), "1 doublons identiques resorbes") || !strings.Contains(out.String(), "1 doublons de contenus differents") {
		t.Errorf("compte rendu = %q", out.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "a0000001.json")); !os.IsNotExist(err) {
		t.Errorf("le .json identique doit etre supprime : %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a0000002.json")); err != nil {
		t.Errorf("le .json different doit rester : %v", err)
	}
}

// poserCompresse range un artefact de test par la porte (compressé, sous le nom canonique de p) ; la signature suit
// os.WriteFile pour que les écritures des tests de ce paquet ne changent que de nom.
func poserCompresse(p string, raw []byte, _ os.FileMode) error {
	_, _, err := replayartifact.Write(context.Background(), p, raw)
	return err
}
