package main

// main_test.go — LE CONTROLE DE `-update` SUR LA BRIQUE DU DECODEUR.

import (
	"errors"
	"strings"
	"testing"
)

// TestVerifierMiseAJour : `-update` passe quand la brique du decodeur est modifiable (un clone relie par
// GOWORK, cf. internal/briques) et sinon porte la raison du resolveur (cache de modules, script d'espace de
// travail), sans la masquer.
func TestVerifierMiseAJour(t *testing.T) {
	var vue string
	if err := verifierMiseAJour(func(brique string) error { vue = brique; return nil }); err != nil {
		t.Fatalf("brique modifiable refusee : %v", err)
	}
	if vue != "film" {
		t.Errorf("brique jugee = %q, attendu film", vue)
	}
	raison := errors.New("le dossier de la brique film est dans le cache de modules ; bash scripts/briques-workspace.sh <clone>")
	err := verifierMiseAJour(func(string) error { return raison })
	if err == nil {
		t.Fatal("brique du cache de modules acceptee")
	}
	for _, attendu := range []string{"-update refuse", "cache de modules", "scripts/briques-workspace.sh"} {
		if !strings.Contains(err.Error(), attendu) {
			t.Errorf("message %q ne contient pas %q", err, attendu)
		}
	}
	if !errors.Is(err, raison) {
		t.Error("l'erreur du resolveur n'est plus chainee")
	}
}
