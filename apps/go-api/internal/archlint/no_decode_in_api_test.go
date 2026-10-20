// Package archlint — no_decode_in_api_test.go : le serveur HTTP ne decode aucun film (lot J2.13,
// constat OPS-2, 2026-09-26).
//
// L'action admin « construire le rejeu » appelait `replaybuild.NewBuilder` puis `BuildMatch` DANS
// le processus serveur : hors du verrou solo, sans sentinelle memoire — un septieme point
// d'entree du decodage, que l'ADR 0034 ne comptait pas. Elle passe desormais par
// `replayartifacts.ConstruireEtRanger` et l'enfant borne. Ce ratchet interdit le retour d'une
// construction in-process sous `internal/api/`.
package archlint

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// constructionInProcessRE : les trois portes du decodage d'un film par `replaybuild`.
var constructionInProcessRE = regexp.MustCompile(`replaybuild\.NewBuilder\(|\.BuildMatch\(|\.BuildBytes\(`)

func TestAucunDecodageDeFilmSousInternalAPI(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	racine := filepath.Join(filepath.Dir(filepath.Dir(thisFile)), "api")
	lus := 0
	err := filepath.WalkDir(racine, func(chemin string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() || !strings.HasSuffix(chemin, ".go") || strings.HasSuffix(chemin, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(chemin) //nolint:gosec // fichier source du module
		if err != nil {
			return err
		}
		lus++
		for i, ligne := range strings.Split(string(src), "\n") {
			if strings.HasPrefix(strings.TrimSpace(ligne), "//") {
				continue // un commentaire peut nommer l'ancien chemin
			}
			if constructionInProcessRE.MatchString(ligne) {
				t.Errorf("%s:%d : construction d'un rejeu DANS le serveur — passer par "+
					"replayartifacts.ConstruireEtRanger (enfant borne, verrou solo)",
					filepath.ToSlash(chemin), i+1)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de %s : %v", racine, err)
	}
	if lus < 50 {
		t.Fatalf("ratchet vacant : %d fichier(s) lu(s) sous %s", lus, racine)
	}
}
