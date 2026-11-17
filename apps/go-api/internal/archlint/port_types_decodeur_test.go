// Package archlint — port_types_decodeur_test.go : l'ensemble des paquets du décodeur de film que
// `internal/port` importe est gelé, et il ne fait que rétrécir.
//
// POURQUOI. Un contrat de port qui nomme un type du décodeur (`replay`, `objectiveevent`,
// `playerposition`) lie chaque consommateur du port, handlers compris, au décodeur : un changement de forme
// côté décodeur se propage à toute la couche. L'état courant est une dette connue (trois paquets, consommés
// par plusieurs fichiers) ; ce garde-rail l'empêche de s'étendre, il ne la retire pas.
//
// RÈGLE. Les imports non-test de `internal/port` (lus dans l'AST, chemin exact) qui sont des paquets du
// module `github.com/JGtm/halo-infinite-film-decoder` forment EXACTEMENT [paquetsDuDecodeurDansPort] :
//   - un paquet de plus est une violation (la liste ne grandit jamais) ;
//   - un paquet de la liste qui n'est plus importé est une violation aussi : on le retire de la liste avec
//     le dernier import, de sorte que le plafond ne puisse que descendre (CLAUDE.md règle 7).
package archlint

import (
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// moduleDuDecodeur : le module de la brique du décodeur de film.
const moduleDuDecodeur = "github.com/JGtm/halo-infinite-film-decoder"

// paquetsDuDecodeurDansPort : les paquets du décodeur que `internal/port` importe. Ne fait que rétrécir.
var paquetsDuDecodeurDansPort = []string{
	moduleDuDecodeur + "/objectiveevent",
	moduleDuDecodeur + "/playerposition",
	moduleDuDecodeur + "/replay",
}

// importsDuDecodeur rend, triés et sans doublon, les paquets du décodeur importés par les sources données.
func importsDuDecodeur(srcs []goSource) []string {
	vus := map[string]bool{}
	for _, src := range srcs {
		for _, imp := range src.File.Imports {
			chemin, err := strconv.Unquote(imp.Path.Value)
			if err == nil && (chemin == moduleDuDecodeur || strings.HasPrefix(chemin, moduleDuDecodeur+"/")) {
				vus[chemin] = true
			}
		}
	}
	out := make([]string, 0, len(vus))
	for chemin := range vus {
		out = append(out, chemin)
	}
	sort.Strings(out)
	return out
}

// TestOutilImportsDuDecodeur : auto-preuve de l'outil, sur des sources de test.
func TestOutilImportsDuDecodeur(t *testing.T) {
	a := parseGoSnippet(t, "internal/port/a.go", `package port

import (
	"fmt"
	"github.com/JGtm/halo-infinite-film-decoder/replay"
	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// "github.com/JGtm/halo-infinite-film-decoder/film" cité en commentaire
var _ = fmt.Sprint("github.com/JGtm/halo-infinite-film-decoder/film")
`)
	b := parseGoSnippet(t, "internal/port/b.go", `package port

import (
	dec "github.com/JGtm/halo-infinite-film-decoder/playerposition"
	"github.com/JGtm/halo-infinite-files-reader/gamefiles"
)
`)
	want := []string{moduleDuDecodeur + "/playerposition", moduleDuDecodeur + "/replay"}
	if got := importsDuDecodeur([]goSource{a, b}); !slices.Equal(got, want) {
		t.Fatalf("attendu %v, obtenu %v", want, got)
	}
}

func TestPortImporteUnEnsembleGeleDePaquetsDuDecodeur(t *testing.T) {
	got := importsDuDecodeur(goSourcesUnder(t, "internal/port"))
	for _, p := range got {
		if !slices.Contains(paquetsDuDecodeurDansPort, p) {
			t.Errorf("internal/port importe %s, absent de la liste gelée : un contrat de port ne nomme pas un type "+
				"du décodeur de plus (la liste ne grandit jamais)", p)
		}
	}
	for _, p := range paquetsDuDecodeurDansPort {
		if !slices.Contains(got, p) {
			t.Errorf("internal/port n'importe plus %s : le retirer de paquetsDuDecodeurDansPort, le plafond "+
				"ne fait que descendre", p)
		}
	}
}
