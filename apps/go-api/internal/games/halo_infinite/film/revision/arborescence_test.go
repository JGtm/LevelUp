package revision_test

// arborescence_test.go — OU VIT `apps/go-api`, pour les tests qui lisent les artefacts du depot.

import (
	"path/filepath"
	"runtime"
	"testing"
)

// racineAPI rend `apps/go-api`, resolu par `runtime.Caller`.
//
// PAS un chemin relatif au repertoire courant : le jour ou ce paquet demenage, les tests qui
// s en servent doivent echouer bruyamment plutot que hacher un dossier vide (meme raison que
// `racinesGrammaire`, dans le gate qu ils doublent).
func racineAPI(t *testing.T) string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	// .../internal/games/halo_infinite/film/revision -> .../apps/go-api
	dir := filepath.Dir(ici)
	for i := 0; i < 5; i++ {
		dir = filepath.Dir(dir)
	}
	return dir
}
