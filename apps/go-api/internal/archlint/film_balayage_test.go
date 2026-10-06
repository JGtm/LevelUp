package archlint

// film_balayage_test.go — LE BALAYAGE DE LA PRODUCTION DU DECODEUR, COMMUN A SES GARDE-RAILS (regle
// des deux copies, CLAUDE.md n. 6) : le localisateur unique, la lecture unique de la vue A, le
// cliquet des tris non totaux et la garde de tampon de la section d identification parcourent la
// meme production par ces deux fonctions.
//
// Le parcours ecarte les repertoires caches, `_x`, `testdata` et les sous-arbres de recherche
// ([repertoireExcluDuTriTotal]), et les fichiers `_test.go`. Une erreur du parcours ou du
// visiteur est fatale.

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// balayerLaProduction rend a `visiter` chaque source Go de production sous les `racines` (relatives
// a apps/go-api) : son chemin relatif a apps/go-api, a barres obliques, et son chemin sur le disque.
func balayerLaProduction(t *testing.T, racines []string, visiter func(rel, chemin string) error) {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(ici))) // .../apps/go-api
	for _, racine := range racines {
		base := filepath.Join(goAPIRoot, filepath.FromSlash(racine))
		err := filepath.WalkDir(base, func(chemin string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(goAPIRoot, chemin)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if d.IsDir() {
				if chemin != base && repertoireExcluDuTriTotal(d.Name(), rel) {
					return fs.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(chemin, ".go") || strings.HasSuffix(chemin, "_test.go") {
				return nil
			}
			return visiter(rel, chemin)
		})
		if err != nil {
			t.Fatalf("balayage de %s : %v", base, err)
		}
	}
}

// balayerLaProductionHorsResearch est [balayerLaProduction] qui lit chaque source, ecarte les
// fichiers `//go:build research` ([estSousTagResearch]) et rend le contenu des autres a `visiter`.
// Elle rend le nombre de fichiers visites, pour le plancher contre un balayage muet de l appelant.
func balayerLaProductionHorsResearch(t *testing.T, racines []string, visiter func(rel string, blob []byte)) int {
	t.Helper()
	fichiers := 0
	balayerLaProduction(t, racines, func(rel, chemin string) error {
		blob, err := os.ReadFile(chemin) //nolint:gosec // chemin derive du perimetre
		if err != nil || estSousTagResearch(blob) {
			return err
		}
		fichiers++
		visiter(rel, blob)
		return nil
	})
	return fichiers
}
