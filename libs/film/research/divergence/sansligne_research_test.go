//go:build research

package divergence

// sansligne_research_test.go : OU SE LISENT LES INDEX DE REGION SANS LIGNE (plan marche complete, A2.P) :
// pour chaque paquet dont la lecture compte au moins un index sans ligne dans la table de la carte, une
// ligne TSV (chunk, index du paquet, nombre de lectures, nombre de records, bits).
//
//	SL_RACINE=<film_chunks> SL_FILM=<id> SL_NIVEAU=<module> SL_CARTE=<catalogue.json> SL_SORTIE=<fichier.tsv> \
//	  go test -tags=research ./research/divergence/ -run TestSansLigne

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

func TestSansLigne(t *testing.T) {
	racine, id, niveau, chemin, sortie := os.Getenv("SL_RACINE"), os.Getenv("SL_FILM"), os.Getenv("SL_NIVEAU"),
		os.Getenv("SL_CARTE"), os.Getenv("SL_SORTIE")
	if racine == "" || id == "" || niveau == "" || chemin == "" || sortie == "" {
		t.Skip("SL_RACINE, SL_FILM, SL_NIVEAU, SL_CARTE et SL_SORTIE requis")
	}
	refuserSortieSousData(t, sortie)
	cat, err := profile.LoadMapQuantCatalog(chemin)
	if err != nil {
		t.Fatal(err)
	}
	fc, err := ouvrirFilm(racine, id, niveau, cat)
	if err != nil {
		t.Fatal(err)
	}
	obs := grammar.NouvelleObservation()
	total := func() (n int) {
		for _, v := range obs.IndexSansLigne {
			n += v
		}
		return n
	}
	var b strings.Builder
	b.WriteString("chunk\tindex\tlectures\trecords\tbits\n")
	avant := 0
	for p, err := range fc.Trames(obs) {
		if err != nil {
			t.Fatal(err)
		}
		if n := total(); n > avant {
			fmt.Fprintf(&b, "%d\t%d\t%d\t%d\t%d\n", p.Chunk, p.Index, n-avant, len(p.Records), 8*len(p.Payload))
			avant = n
		}
	}
	fmt.Fprintf(&b, "# total %d, sans ligne %v, absolus %v\n", total(), obs.IndexSansLigne, obs.IndexAbsolus)
	if err := os.WriteFile(sortie, []byte(b.String()), 0o644); err != nil { //nolint:gosec // sortie de l operateur
		t.Fatal(err)
	}
}
