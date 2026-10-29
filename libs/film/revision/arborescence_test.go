package revision_test

// arborescence_test.go — OU VIT LA BRIQUE, pour les tests qui lisent ses artefacts versionnes.

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/JGtm/LevelUp/libs/film/revision"
)

// moduleDeLaBrique rend le module qui porte ce paquet, trouve en remontant au premier `go.mod`
// depuis le dossier de ce fichier (resolu par `runtime.Caller`, pas par le repertoire courant).
//
// Les tests y resolvent les dossiers des couches par [revision.Module.SurDisque], depuis leur nom
// HISTORIQUE : le chemin sur le disque suit le module courant, la declaration ne bouge pas. Un
// dossier que la resolution ne trouve plus fait echouer la lecture de son golden, bruyamment,
// plutot que de hacher un dossier vide (meme raison que `racinesGrammaire`, dans le gate que ces
// tests doublent).
func moduleDeLaBrique(t *testing.T) revision.Module {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	m, err := revision.ModuleDe(filepath.Dir(ici))
	if err != nil {
		t.Fatalf("module : %v", err)
	}
	return m
}

// dossierHistoriqueDesCouches : la racine des couches, en nom historique (celui des declarations
// de `revision.CouchesRevisees` et des goldens de perimetre).
const dossierHistoriqueDesCouches = "internal/games/halo_infinite/film/internal/"
