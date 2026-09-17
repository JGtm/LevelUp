package grammar

// grenade_production_research_test.go — CE QUE LE CHEMIN DE PRODUCTION LIT, FILM PAR FILM.
//
// POURQUOI CET INSTRUMENT EXISTE, ET CE QU IL NE DOUBLE PAS. L instrument de recherche du volet
// 3.3r (`film/research/grenadeids`) mesure la grammaire AVEC SON PROPRE balayage : il prouve ce
// que les octets portent, jamais ce que la production en lit. `version_grenade_tags_research_test.go`
// (instrument H.2) balaye sous la grammaire de REFERENCE, donc il ne voit pas non plus les
// profils par clef. Celui-ci appelle [ScanFilmGrenadeThrows], c est-a-dire EXACTEMENT la fonction
// que la cuisson appelle, et publie le compte par RANG.
//
// C est la verification de la prediction du lot 3.3.2 SANS cuire un artefact ni ouvrir la base :
// lecture seule, un film a la fois, dans l ordre donne.
//
// USAGE :
//
//	GRENPROD_ROOT=<parc>/data/cache/film_chunks GRENPROD_IDS=bcb6d393,e5adf7b2 \
//	  go test ./internal/games/halo_infinite/film/internal/grammar -run TestGrenadesDeProduction -v
//
// Sans les deux variables, il se saute proprement.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	grenProdRootEnv = "GRENPROD_ROOT"
	grenProdIDsEnv  = "GRENPROD_IDS"
)

// TestGrenadesDeProduction publie, par film, ce que le chemin de cuisson lit vraiment.
func TestGrenadesDeProduction(t *testing.T) {
	racine, ids := os.Getenv(grenProdRootEnv), os.Getenv(grenProdIDsEnv)
	if racine == "" || ids == "" {
		t.Skipf("instrument de mesure : definir %s et %s", grenProdRootEnv, grenProdIDsEnv)
	}
	for _, id := range strings.Split(ids, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		throws, err := ScanFilmGrenadeThrows(filepath.Join(racine, id))
		if err != nil {
			t.Errorf("%s : %v", id, err)
			continue
		}
		var parRang [len(GrenadeTypeIDsByRank)]int
		indexMax, sansRang := -1, 0
		for _, g := range throws {
			rang, connu := g.Rank()
			if !connu {
				sansRang++
				continue
			}
			parRang[rang]++
			if g.FilmIndex > indexMax {
				indexMax = g.FilmIndex
			}
		}
		t.Logf("%s : total=%d parRang=%v sansRang=%d indexAuteurMax=%d",
			id, len(throws), parRang, sansRang, indexMax)
	}
}
