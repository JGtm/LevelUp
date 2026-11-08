package anonymisegamertags

// binaires_couverts_test.go — LES FICHIERS DE `libs/film` QUE LE BALAYAGE DE TEXTE NE LIT PAS.
//
// Le garde-rail du registre (`registre_test.go`) lit les fichiers TEXTE. Les autres (films, entrees
// du codec des faits, graines de fuzz) portent leurs identifiants sous une forme qu un balayage de
// texte ne voit pas : chacun est donc couvert par une autre preuve, NOMMEE ici. La liste est FERMEE :
// un fichier ni texte ni listé rougit, et un binaire neuf rejoint la liste avec la preuve qui le
// couvre ou une extension de texte. Une ligne qui ne couvre plus aucun fichier rougit aussi.

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// binaireCouvert : un motif de chemin (relatif a `libs/film`, separateurs `/`) et ce qui le couvre.
type binaireCouvert struct {
	motif      *regexp.Regexp
	couverture string
}

// binairesCouverts : liste fermee, etablie le 2026-11-08 (revue adversariale 4.7).
var binairesCouverts = []binaireCouvert{
	{regexp.MustCompile(`^replay/testdata/fuzz/FuzzDecodeFilmFactsFile/seed_[0-9]+$`),
		"libs/film/replay TestGrainesDeFaitsPortentDesXuidsFictifs (decodage par le codec, plage et registre)"},
	{regexp.MustCompile(`^replay/testdata/inputs_[0-9a-f]{8}\.bin\.gz$`),
		"libs/film/replay TestGoldenInputsPorteDesXuidsFictifs (decodage par le codec, registre)"},
	{regexp.MustCompile(`^replay/testdata/(entites|equipement)_[0-9a-f]{8}\.bin\.gz$`),
		"codec sans champ xuid (filmfacts_encode.go : morts, table des index et des sieges seuls) ; " +
			"mesure du 2026-11-08, 0 des 420 originaux"},
	{regexp.MustCompile(`^internal/grammar/testdata/fuzz/FuzzFilmRecordReaders/seed_[0-9]+$`),
		"payloads de records de la mini-bobine fictive, sans table d identite ; mesure du 2026-11-08, " +
			"0 des 420 originaux apres decodage des graines"},
	{regexp.MustCompile(`\.bin$`),
		"octets de film : cmd/anonymise-gamertags (xuids en 64 bits aux huit decalages de bit), item 4.0"},
}

// couvertureDuBinaire rend la ligne de la liste fermee qui couvre `rel`, ou -1.
func couvertureDuBinaire(rel string) int {
	for i, b := range binairesCouverts {
		if b.motif.MatchString(rel) {
			return i
		}
	}
	return -1
}

// binairesSansCouverture parcourt `racine` et rend les fichiers ni texte ni listes, et le nombre de
// fichiers couverts par ligne de la liste.
func binairesSansCouverture(racine string) (sans []string, couverts []int, err error) {
	couverts = make([]int, len(binairesCouverts))
	err = filepath.WalkDir(racine, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(racine, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if i := couvertureDuBinaire(rel); i >= 0 {
			couverts[i]++
			return nil
		}
		if !extensionsTexte[filepath.Ext(p)] {
			sans = append(sans, rel)
		}
		return nil
	})
	sort.Strings(sans)
	return sans, couverts, err
}

// TestLesBinairesDeLaBriqueSontCouverts : aucun fichier de `libs/film` n echappe a la fois au
// balayage de texte et a la liste fermee des binaires couverts.
func TestLesBinairesDeLaBriqueSontCouverts(t *testing.T) {
	sans, couverts, err := binairesSansCouverture(filepath.Join(racineDuDepot, "libs", "film"))
	if err != nil {
		t.Fatal(err)
	}
	if len(sans) > 0 {
		t.Errorf("%d fichier(s) ni texte (%s) ni couverts par binairesCouverts : un identifiant reel "+
			"pourrait s y cacher ; ajouter la preuve qui le couvre :\n  %s",
			len(sans), extensionsTexteListe(), strings.Join(sans, "\n  "))
	}
	for i, n := range couverts {
		if n == 0 {
			t.Errorf("binairesCouverts[%d] (%s) ne couvre plus aucun fichier : la retirer", i, binairesCouverts[i].motif)
		}
	}
}

// TestLaListeDesBinairesVoitUnBinaireNeuf : sur un dossier synthetique, un binaire neuf et un
// fichier sans extension hors liste sont signales, un fichier texte et un film liste ne le sont pas.
func TestLaListeDesBinairesVoitUnBinaireNeuf(t *testing.T) {
	dossier := t.TempDir()
	for _, f := range []string{"a.go", "film/chunk_00.bin", "neuf.dat", "sans_extension", "replay/testdata/fuzz/FuzzDecodeFilmFactsFile/seed_00"} {
		ecrire(t, filepath.Join(dossier, filepath.FromSlash(f)), "x")
	}
	sans, couverts, err := binairesSansCouverture(dossier)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprint(sans); got != "[neuf.dat sans_extension]" {
		t.Fatalf("attendu [neuf.dat sans_extension], lu %s", got)
	}
	if couverts[0] != 1 || couverts[len(couverts)-1] != 1 {
		t.Fatalf("couverture par ligne inattendue : %v", couverts)
	}
}

func extensionsTexteListe() string {
	var l []string
	for e := range extensionsTexte {
		l = append(l, e)
	}
	sort.Strings(l)
	return strings.Join(l, " ")
}

func ecrire(t *testing.T, chemin, contenu string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(chemin), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chemin, []byte(contenu), 0o600); err != nil {
		t.Fatal(err)
	}
}
