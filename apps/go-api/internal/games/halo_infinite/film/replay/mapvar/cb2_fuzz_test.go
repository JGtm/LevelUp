package mapvar

// cb2_fuzz_test.go — LE HARNAIS DE FUZZ DU LECTEUR .mvar (lot J2.8, constat RB1-4, 2026-09-26),
// sur le modele de `film/internal/grammar/fuzz_records_test.go`.
//
// CE QUE CE HARNAIS GARANTIT, ET RIEN DE PLUS : [Parse] ne panique sur AUCUNE entree.
//
// DEUX SOURCES DE GRAINES, ET C EST VOULU :
//   - les `.mvar` VERSIONNES (.ai/V7.5/dumps/mapvar/) sont ajoutes par `f.Add` a chaque
//     execution, lus la ou ils vivent : ce sont des actifs proprietaires 343/Microsoft que
//     `mapvar_test.go` refuse de dupliquer dans testdata/ — le corpus de fuzz ne les copie pas
//     davantage ;
//   - testdata/fuzz/FuzzMapvarParse/ porte des documents SYNTHETIQUES au format de corpus natif
//     de Go : une racine valide minimale et les cinq formes de compte du format a 2^40
//     (liste, ensemble, map, chaine, chaine large) — les entrees du defaut RB1-4.
//
// CAMPAGNE (a la main, jamais en CI) :
//
//	go test ./internal/games/halo_infinite/film/replay/mapvar/ -run '^$' -fuzz FuzzMapvarParse -fuzztime 30s -fuzzminimizetime 0s
//
// REGENERATION DES GRAINES SYNTHETIQUES (porte dediee) :
//
//	go test ./internal/games/halo_infinite/film/replay/mapvar/ -run TestFuzzMapvarSeedsRegenerate -update-graines-mvar

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
)

var updateGrainesMvar = flag.Bool("update-graines-mvar", false,
	"reecrire le corpus de graines synthetiques de FuzzMapvarParse (testdata/fuzz/)")

const grainesMvarDir = "testdata/fuzz/FuzzMapvarParse"

// mvarVersionnes : les `.mvar` commis au depot (cf. `fixture`).
var mvarVersionnes = []string{
	"vagabond_fo08_wetland.mvar", "cliffhanger_map.mvar", "cliffhanger_ridgeline.mvar",
}

// FuzzMapvarParse : le lecteur .mvar ne panique sur AUCUNE entree.
func FuzzMapvarParse(f *testing.F) {
	for _, nom := range mvarVersionnes {
		path := filepath.Join("..", "..", "..", "..", "..", "..", "..", "..", ".ai", "V7.5", "dumps", "mapvar", nom)
		buf, err := os.ReadFile(path) //nolint:gosec // chemin fige dans le code
		if err != nil {
			f.Logf("graine %s absente (%v) : non ajoutee", nom, err)
			continue
		}
		f.Add(buf)
	}
	f.Fuzz(func(t *testing.T, doc []byte) {
		_, _ = Parse(doc)
	})
}

// TestFuzzMapvarSeedsRegenerate : LA SEULE PORTE D ECRITURE DU CORPUS SYNTHETIQUE.
func TestFuzzMapvarSeedsRegenerate(t *testing.T) {
	if !*updateGrainesMvar {
		t.Skip("regeneration du corpus de graines : passer -update-graines-mvar")
	}
	if err := os.MkdirAll(grainesMvarDir, 0o750); err != nil {
		t.Fatalf("creation de %s : %v", grainesMvarDir, err)
	}
	for i, g := range grainesSynthetiques() {
		path := filepath.Join(grainesMvarDir, fmt.Sprintf("seed_%02d", i))
		body := "go test fuzz v1\n[]byte(" + strconv.Quote(string(g)) + ")\n"
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatalf("ecriture de %s : %v", path, err)
		}
	}
}

// grainesSynthetiques : une racine valide (liste de huit uint32), puis les cinq formes de
// compte a `compteImpossible`, dans l ordre des noms pour que la regeneration soit stable.
func grainesSynthetiques() [][]byte {
	out := [][]byte{docAvecChamp3(btList, []byte{btUint32, 8, 1, 2, 3, 4, 5, 6, 7, 8, btStop})}
	docs := documentsAuCompte(compteImpossible)
	for _, forme := range slices.Sorted(maps.Keys(docs)) {
		out = append(out, docs[forme])
	}
	return out
}
