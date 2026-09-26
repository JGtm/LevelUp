package main

// main_test.go — TEST DE FUMEE DU CLI HISTORIQUE (CORPUS-R1 C12, 2026-09-07).
//
// Le journal .ai/V7.5/v2/CORPUS_TEMOIN_2026-09-06.md annoncait `go test ./cmd/replay-diff/...`
// comme un gate JOUE — le paquet n'avait pourtant AUCUN fichier de test
// (`? levelup/go-api/cmd/replay-diff [no test files]`). La promesse « comportement externe
// inchange » qui a motive l'extraction de la logique vers internal/replaydiff (memes flags,
// meme sortie, cf. l'en-tete de main.go) n'etait alors gardee par RIEN apres un futur merge —
// ce fichier la garde desormais, sur `executer`, le point d'entree que main() appelle.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// artefactMinimal ecrit un artefact de rejeu minimal mais valide (schema + un calque simple) —
// suffisant pour LireDocument/Empreindre/Comparer sans dependre d'un vrai fichier cuit.
func artefactMinimal(t *testing.T, path string, schema int, valeur float64) {
	t.Helper()
	doc := map[string]any{
		"schemaVersion": schema,
		"matchId":       "aaaa1111-test",
		"objectifs": map[string]any{
			"parStat": map[string]any{"flag_captures": valeur},
		},
	}
	blob, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal fixture : %v", err)
	}
	if err := os.WriteFile(path, blob, 0o600); err != nil {
		t.Fatalf("ecriture fixture : %v", err)
	}
}

// TestExecuterCompareDeuxArtefactsSansErreur — le chemin nominal du CLI historique : deux
// artefacts valides, aucune erreur, tableau imprime (silencieux ici via -quiet pour ne pas
// polluer la sortie du test).
func TestExecuterCompareDeuxArtefactsSansErreur(t *testing.T) {
	dir := t.TempDir()
	ancien := filepath.Join(dir, "ancien.json")
	nouveau := filepath.Join(dir, "nouveau.json")
	artefactMinimal(t, ancien, 40, 3)
	artefactMinimal(t, nouveau, 41, 3)

	if err := executer(ancien, nouveau, "", true, true); err != nil {
		t.Fatalf("executer (-tout -quiet) : %v", err)
	}
}

// TestExecuterEcritLeRapportJSON — le flag -json doit produire un fichier portant les deux
// schemas compares, sous les noms de champ attendus par internal/replaydiff.Rapport.
func TestExecuterEcritLeRapportJSON(t *testing.T) {
	dir := t.TempDir()
	ancien := filepath.Join(dir, "ancien.json")
	nouveau := filepath.Join(dir, "nouveau.json")
	sortie := filepath.Join(dir, "rapport.json")
	artefactMinimal(t, ancien, 40, 3)
	artefactMinimal(t, nouveau, 41, 5) // GAIN attendu (valeur montee, jamais une perte)

	if err := executer(ancien, nouveau, sortie, false, true); err != nil {
		t.Fatalf("executer : %v", err)
	}
	blob, err := os.ReadFile(sortie)
	if err != nil {
		t.Fatalf("lecture du rapport JSON : %v", err)
	}
	out := string(blob)
	for _, attendu := range []string{`"schemaAncien": 40`, `"schemaNouveau": 41`} {
		if !strings.Contains(out, attendu) {
			t.Errorf("rapport JSON attendu avec %q, obtenu :\n%s", attendu, out)
		}
	}
}

// TestExecuterArtefactAncienIntrouvableEstUneErreurNommee — un fichier absent doit produire
// une erreur qui NOMME le cote fautif ("artefact ancien"), pas un message opaque — c'est ce
// message que main() relaie tel quel via slog.Error avant os.Exit(1).
func TestExecuterArtefactAncienIntrouvableEstUneErreurNommee(t *testing.T) {
	dir := t.TempDir()
	nouveau := filepath.Join(dir, "nouveau.json")
	artefactMinimal(t, nouveau, 41, 3)

	err := executer(filepath.Join(dir, "absent.json"), nouveau, "", false, true)
	if err == nil {
		t.Fatal("attendu une erreur (artefact ancien introuvable)")
	}
	if !strings.Contains(err.Error(), "artefact ancien") {
		t.Fatalf("l'erreur doit nommer le cote fautif, obtenu : %v", err)
	}
}
