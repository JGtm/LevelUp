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
	{regexp.MustCompile(`^replay/testdata/equipement_origines_[0-9a-f]{8}.bin.gz$`),
		"encodeFaitsDeGarde sans champ xuid (slots, rangs, lectures de l equipement porte, projectiles) ; " +
			"inventaire cmd/anonymise-gamertags du 2026-11-09, 0 des 420 originaux"},
	{regexp.MustCompile(`^internal/grammar/testdata/fuzz/FuzzFilmRecordReaders/seed_[0-9]+$`),
		"payloads de records de la mini-bobine fictive, sans table d identite ; mesure du 2026-11-08, " +
			"0 des 420 originaux apres decodage des graines"},
	{regexp.MustCompile(`\.bin$`),
		"octets de film : cmd/anonymise-gamertags (xuids en 64 bits aux huit decalages de bit), item 4.0"},
}

// binairesCouvertsApps : la liste fermee des fichiers suivis sous `apps/` que le balayage de texte ne lit pas.
// Etablie le 2026-11-10 (inventaire de l outil sur tout `apps/`, huit decalages de bit).
var binairesCouvertsApps = []binaireCouvert{
	{regexp.MustCompile(`\.(webp|png|wav|woff2|ico)$`),
		"images, sons, polices et icones : aucun identifiant ; inventaire cmd/anonymise-gamertags du 2026-11-10, " +
			"0 occurrence des xuids et des noms d origine"},
	{regexp.MustCompile(`^go-api/internal/api/wire/testdata/film_e2e/[0-9a-f]{8}/chunks/chunk_[0-9]+\.bin$`),
		"chunks de film zlib : xuids reecrits par cmd/anonymise-gamertags (64 bits, huit decalages) ; la table " +
			"des joueurs du film est comparee au fixture.json voisin, texte balaye par TestLesXuidsDesFixturesSontFictifs"},
	{regexp.MustCompile(`^go-api/internal/platform/duckdb/testdata/wal_orphan_fixture/shared_social\.duckdb(\.gz|\.wal)$`),
		"base DuckDB de production capturee, LAISSEE EN L ETAT : ses blocs portent une somme de controle, la " +
			"reecriture des octets la corrompt (mesure du 2026-11-10 : checksum mismatch a la lecture) ; elle porte " +
			"des identifiants reels, que le NOTICE enonce"},
	{regexp.MustCompile(`^go-api/internal/resourcepack/testdata/signed_payload\.bin(\.sig)?$`),
		"charge utile et signature de test de l updater : octets sans identifiant de joueur"},
}

// sortiesDeCINonSuivies : fichiers que le job de test de la CI ecrit dans `apps/go-api` (journal JSONL du lot,
// profil de couverture) ; git ne les suit pas, le balayage ne les lit pas.
var sortiesDeCINonSuivies = regexp.MustCompile(`^go-api/(baseline_[A-Za-z0-9_.-]+\.jsonl|coverage\.[A-Za-z0-9_.-]+\.raw)$`)

// couvertureDuBinaire rend la ligne de `liste` qui couvre `rel`, ou -1.
func couvertureDuBinaire(rel string, liste []binaireCouvert) int {
	for i, b := range liste {
		if b.motif.MatchString(rel) {
			return i
		}
	}
	return -1
}

// binairesSansCouverture parcourt `racine` et rend les fichiers ni texte (`extensions`, un `.gz` etant jauge
// sur son extension interne) ni couverts par `liste`, et le nombre de fichiers couverts par ligne de la liste.
func binairesSansCouverture(racine string, extensions map[string]bool, liste []binaireCouvert) (sans []string, couverts []int, err error) {
	couverts = make([]int, len(liste))
	err = filepath.WalkDir(racine, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if dossiersNonSuivis[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(racine, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if sortiesDeCINonSuivies.MatchString(rel) {
			return nil
		}
		if i := couvertureDuBinaire(rel, liste); i >= 0 {
			couverts[i]++
			return nil
		}
		if !extensions[extensionDuTexte(p)] {
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
	sans, couverts, err := binairesSansCouverture(dossierDeLaBrique(t), extensionsTexte, binairesCouverts)
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

// TestLesBinairesDeAppsSontCouverts : aucun fichier suivi sous `apps/` n echappe a la fois au balayage de
// texte (`.gz` lus decompresses) et a la liste fermee `binairesCouvertsApps`.
func TestLesBinairesDeAppsSontCouverts(t *testing.T) {
	sans, couverts, err := binairesSansCouverture(filepath.Join(racineDuDepot, "apps"), extensionsTexteApps, binairesCouvertsApps)
	if err != nil {
		t.Fatal(err)
	}
	if len(sans) > 0 {
		t.Errorf("%d fichier(s) ni texte ni couverts par binairesCouvertsApps : un identifiant reel pourrait "+
			"s y cacher ; ajouter la preuve qui le couvre :\n  %s", len(sans), strings.Join(sans, "\n  "))
	}
	for i, n := range couverts {
		if n == 0 {
			t.Errorf("binairesCouvertsApps[%d] (%s) ne couvre plus aucun fichier : la retirer", i, binairesCouvertsApps[i].motif)
		}
	}
}

// TestLaListeDesBinairesVoitUnBinaireNeuf : sur un dossier synthetique, un binaire neuf et un
// fichier sans extension hors liste sont signales, un fichier texte et un film liste ne le sont pas.
func TestLaListeDesBinairesVoitUnBinaireNeuf(t *testing.T) {
	dossier := t.TempDir()
	for _, f := range []string{"a.go", "film/chunk_00.bin", "neuf.dat", "sans_extension", "replay/testdata/fuzz/FuzzDecodeFilmFactsFile/seed_00", "go-api/baseline_reste.jsonl", "go-api/coverage.reste.raw"} {
		ecrire(t, filepath.Join(dossier, filepath.FromSlash(f)), "x")
	}
	sans, couverts, err := binairesSansCouverture(dossier, extensionsTexte, binairesCouverts)
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
