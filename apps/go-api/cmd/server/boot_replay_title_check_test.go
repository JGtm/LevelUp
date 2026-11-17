package main

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestReplayArtifactTitleCheck : le prédicat lit la capacité `film.replay_artifact` du titre (vrai pour un titre qui
// la déclare, faux pour un titre qui ne l'expose pas ou qui n'existe pas) et rend la même réponse au second appel.
func TestReplayArtifactTitleCheck(t *testing.T) {
	_, ce, _, _ := runtime.Caller(0)
	racine := filepath.Join(filepath.Dir(ce), "..", "..", "..", "..")
	check := replayArtifactTitleCheck(racine)
	for range 2 { // second tour : réponse mémoïsée, identique
		if !check("halo_infinite") {
			t.Error("halo_infinite déclare film.replay_artifact : attendu vrai")
		}
		if check("halo_5") {
			t.Error("halo_5 n'expose pas film.replay_artifact : attendu faux")
		}
		if check("titre_inexistant") {
			t.Error("un titre inconnu : attendu faux")
		}
	}
}
