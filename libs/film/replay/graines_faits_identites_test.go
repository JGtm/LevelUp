package replay

// graines_faits_identites_test.go — LES GRAINES DE FUZZ DU FICHIER DE FAITS : AUCUN XUID REEL.
//
// Une graine de `testdata/fuzz/FuzzDecodeFilmFactsFile/` est un fichier de faits ecrit par
// l encodeur de production : ses xuids sont des ENTIERS VARIABLES du codec, ni du texte ni un motif
// binaire que le balayage des fixtures de `cmd/anonymise-gamertags` puisse voir. Elle se
// regenere donc depuis la fixture fictive (`TestFuzzFilmFactsSeedsRegenerate`), et ce test la
// relit : chaque xuid decode est dans la plage des xuids fictifs ET au registre
// `testdata/IDENTITES.tsv`.
//
// Une graine que le decodeur refuse (une troncature) n a pas d identite a lire : elle doit etre un
// DEBUT d une graine lisible, donc ne porter que des octets deja controles.

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// Bornes de la plage des xuids fictifs (D-L1-8) : [debut, fin[. Le registre en est un sous-ensemble.
const (
	xuidFictifDebut uint64 = 0x0009000000000000
	xuidFictifFin   uint64 = 0x000A000000000000
)

// lireGraineGo rend les octets d une graine au format de corpus natif de Go
// (`go test fuzz v1` puis `[]byte("...")`).
func lireGraineGo(t *testing.T, chemin string) []byte {
	t.Helper()
	brut, err := os.ReadFile(chemin) //nolint:gosec // chemin issu du dossier des graines
	if err != nil {
		t.Fatalf("graine %s : %v", chemin, err)
	}
	lignes := strings.Split(strings.TrimSpace(string(brut)), "\n")
	if len(lignes) != 2 || lignes[0] != "go test fuzz v1" ||
		!strings.HasPrefix(lignes[1], "[]byte(") || !strings.HasSuffix(lignes[1], ")") {
		t.Fatalf("graine %s : format de corpus Go inattendu", chemin)
	}
	s, err := strconv.Unquote(strings.TrimSuffix(strings.TrimPrefix(lignes[1], "[]byte("), ")"))
	if err != nil {
		t.Fatalf("graine %s : %v", chemin, err)
	}
	return []byte(s)
}

// xuidsHorsFictifs rend les xuids du fichier de faits `octets` qui ne sont pas dans la plage des
// xuids fictifs ou pas au registre. Un fichier que le decodeur refuse rend l erreur.
func xuidsHorsFictifs(octets []byte, entry profile.MapQuantEntry, registre map[uint64]bool) (hors []uint64, lus int, err error) {
	fichier, err := DecodeFilmFactsFile(octets, entry)
	if err != nil {
		return nil, 0, err
	}
	xuids := xuidsDesEntrees(&fichier.Facts)
	for _, x := range xuids {
		if x < xuidFictifDebut || x >= xuidFictifFin || !registre[x] {
			hors = append(hors, x)
		}
	}
	return hors, len(xuids), nil
}

// TestGrainesDeFaitsPortentDesXuidsFictifs : chaque xuid d une graine lisible est fictif et au
// registre ; chaque graine illisible est le debut d une graine lisible (une graine qui ne se relit
// pas ne franchit pas l en-tete : elle n exercerait rien, et ses xuids ne se verifieraient pas).
func TestGrainesDeFaitsPortentDesXuidsFictifs(t *testing.T) {
	chemins, err := filepath.Glob(filepath.Join(grainesFaitsDir, "seed_*"))
	if err != nil || len(chemins) == 0 {
		t.Fatalf("aucune graine sous %s (%v)", grainesFaitsDir, err)
	}
	sort.Strings(chemins)
	registre := registreIdentitesFictives(t)
	entry := goldenEntryPourTest(t)
	var lisibles [][]byte
	corps := map[string][]byte{}
	erreurs := map[string]error{}
	xuidsVus := 0
	for _, chemin := range chemins {
		octets := lireGraineGo(t, chemin)
		corps[chemin] = octets
		hors, lus, err := xuidsHorsFictifs(octets, entry, registre)
		if err != nil {
			erreurs[chemin] = err
			continue
		}
		lisibles = append(lisibles, octets)
		xuidsVus += lus
		if len(hors) > 0 {
			t.Errorf("%s : %d xuid(s) hors de la plage fictive 0x%016x..0x%016x ou du registre "+
				"IDENTITES.tsv : %v ; regenerer la graine (go test ./replay/ -run "+
				"TestFuzzFilmFactsSeedsRegenerate -update-graines-faits)",
				chemin, len(hors), xuidFictifDebut, xuidFictifFin, hors)
		}
	}
	for chemin, err := range erreurs {
		if !estDebutDUneGraine(corps[chemin], lisibles) {
			t.Errorf("%s : graine illisible (%v) qui n'est le debut d'aucune graine lisible : ses "+
				"xuids ne peuvent pas etre verifies ; regenerer la graine (-update-graines-faits)", chemin, err)
		}
	}
	if xuidsVus == 0 {
		t.Fatal("aucune graine lisible ne porte d'identite : le garde-rail ne lit rien")
	}
}

// TestLeGardeRailDesGrainesVoitUnXuidReel : un fichier de faits qui porte un xuid reel (hors
// registre, dans la plage des xuids de Xbox) est refuse, un fictif du registre passe.
func TestLeGardeRailDesGrainesVoitUnXuidReel(t *testing.T) {
	registre := registreIdentitesFictives(t)
	entry := goldenEntryPourTest(t)
	var fictif uint64
	for x := range registre {
		fictif = x
		break
	}
	reel := xuidFictifDebut + 0x7a5c3 // dans la plage, absent du registre (verifie ci-dessous)
	if registre[reel] {
		t.Fatalf("%d est au registre : choisir un autre xuid de test", reel)
	}
	g := faitsVidesDeLaCarte(entry)
	g.Deaths = append(g.Deaths, types.Death{XUID: fictif, Gamertag: "A", TimeMS: 1},
		types.Death{XUID: reel, Gamertag: "B", TimeMS: 2})
	hors, lus, err := xuidsHorsFictifs(encoderFichierDeFaits(t, g), entry, registre)
	if err != nil || lus != 2 || len(hors) != 1 || hors[0] != reel {
		t.Fatalf("attendu le seul xuid reel %d sur 2 lus, lu %v sur %d (%v)", reel, hors, lus, err)
	}
}

// estDebutDUneGraine : `octets` est un prefixe d une des graines `lisibles`.
func estDebutDUneGraine(octets []byte, lisibles [][]byte) bool {
	for _, g := range lisibles {
		if len(octets) <= len(g) && bytes.Equal(octets, g[:len(octets)]) {
			return true
		}
	}
	return false
}
