//go:build research

package divergence

// fermeture_paquets_research_test.go : la fermeture PAQUET PAR PAQUET d un film sous un catalogue de
// carte (plan marche complete, A2.P) — pour expliquer un film dont le compte de paquets fermes bouge
// entre deux etats du decodeur.
//
//	PAQUETS_RACINE=<film_chunks> PAQUETS_FILM=<id> PAQUETS_NIVEAU=<module> PAQUETS_CARTE=<json> \
//	  PAQUETS_SORTIE=<fichier.tsv> go test -tags=research ./research/divergence/ -run TestFermeturePaquets
//
// Sortie : une ligne par paquet delta (chunk, index, fermee, cause d arret, bits du payload).

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

func TestFermeturePaquets(t *testing.T) {
	racine, id, niveau, chemin, sortie := os.Getenv("PAQUETS_RACINE"), os.Getenv("PAQUETS_FILM"),
		os.Getenv("PAQUETS_NIVEAU"), os.Getenv("PAQUETS_CARTE"), os.Getenv("PAQUETS_SORTIE")
	if racine == "" || id == "" || niveau == "" || chemin == "" || sortie == "" {
		t.Skip("PAQUETS_RACINE, PAQUETS_FILM, PAQUETS_NIVEAU, PAQUETS_CARTE et PAQUETS_SORTIE requis")
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
	var b strings.Builder
	b.WriteString("chunk\tindex\tfermee\tcause\tbits\tliste_non_localisee\tliste_localisee\tliste_lue\n")
	_, err = grammar.FrameClosureDetaillee(fc, grammar.UsagesProduit{}, func(d grammar.PaquetDeCarte) {
		fmt.Fprintf(&b, "%d\t%d\t%t\t%s\t%d\t%t\t%t\t%t\n", d.Chunk, d.Index, d.Fermee, d.Cause, d.Bits,
			d.ListeNonLocalisee, d.ListeLocalisee, d.ListeLue)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sortie, []byte(b.String()), 0o644); err != nil { //nolint:gosec // sortie de l operateur
		t.Fatal(err)
	}
}
