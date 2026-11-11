//go:build research

package divergence

// paquets_diff_research_test.go : le DETAIL de chaque paquet delta d un film sous un catalogue de carte
// (plan marche complete, A2.P, expliquer les ecarts du gate) : a lancer sur l etat d AVANT et sur celui
// d APRES, puis a joindre par (chunk, index).
//
//	PAQUETS_RACINE=<film_chunks> PAQUETS_FILM=<id> PAQUETS_NIVEAU=<module> PAQUETS_CARTE=<json> \
//	  PAQUETS_SORTIE=<fichier.tsv> go test -tags=research ./research/divergence/ -run TestDetailDesPaquets

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

func TestDetailDesPaquets(t *testing.T) {
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
	b.WriteString("chunk\tindex\tts\tfermee\tcause\tbits\tcurseur\trecords\tfinB\tsortie\tinvariants\tdeborde\tdernier\tneufs\tdesync\tlocalisee\tlue\n")
	_, err = grammar.FrameClosureDetaillee(fc, grammar.UsagesProduit{}, func(d grammar.PaquetDeCarte) {
		fmt.Fprintf(&b, "%d\t%d\t%d\t%t\t%s\t%d\t%d\t%d\t%d\t%v\t%d\t%t\t%s\t%v\t%v\t%t\t%t\n", d.Chunk, d.Index, d.TimestampUS, d.Fermee, d.Cause,
			d.Bits, d.Curseur, d.RecordsLus, d.FinVueB, d.Sortie, d.Invariants, d.Deborde, d.DernierLu, d.NeufsLus, d.NeufsDesynchronises, d.ListeLocalisee, d.ListeLue)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sortie, []byte(b.String()), 0o644); err != nil { //nolint:gosec // sortie de l operateur
		t.Fatal(err)
	}
}
