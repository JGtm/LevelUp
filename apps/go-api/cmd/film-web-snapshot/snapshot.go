package main

// snapshot.go — LES FICHIERS DE LA BRIQUE DU DECODEUR DE FILM QUE LES TESTS WEB RELISENT.
//
// Quatre tests web (`deltaLayersContract`, `replaySchemaStatus`, `vehicleCycleTime`,
// `replaySoundAssets`) confrontent des valeurs du client a leur source Go : balises `json:` des
// calques du document de rejeu, constantes, version de schema declaree par la chronique, regles des
// icones de mort. Cette source vit dans la brique `libs/film`, dont le dossier est celui que l'outil
// Go designe pour le module (espace de travail, puis cache de modules) : un test web ne sait pas le
// trouver, et la CI web ne lance pas Go. Ces fichiers sont donc recopies ICI, par cet outil, dans un
// instantane versionne (`dossierDuWeb`) que les tests web lisent ; le test Go de ce paquet
// (`TestInstantaneEstFrais`) tient l'instantane egal a la brique, et sa regeneration est la commande
// `go run ./cmd/film-web-snapshot`.
//
// Les copies sont textuelles (fins de ligne LF) et portent le suffixe `.txt` quand ce sont des sources Go :
// un outil qui parcourt le depot ne les prend ni pour du code compilable ni pour un paquet. La chronique
// (300 Ko) n'est reduite qu'a ses lignes d'en-tete de version, les seules que le test web lit.

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"levelup/go-api/internal/briques"
)

// dossierDuWeb : le dossier de l'instantane, relatif a la racine du depot.
const dossierDuWeb = "apps/web/src/features/match-replay/test/film-source"

// nomManifeste : le fichier qui dit d'ou vient chaque copie.
const nomManifeste = "SOURCES.txt"

// source : un fichier de la brique recopie dans l'instantane.
type source struct {
	chemin  string              // dans la brique, separateurs "/"
	sortie  string              // dans l'instantane, separateurs "/"
	extrait func([]byte) []byte // nil : copie telle quelle
}

// sources : exactement ce que lisent les quatre tests web. Ajouter une lecture a un test web, c'est
// ajouter une ligne ici puis regenerer.
var sources = []source{
	{chemin: "replay/document.go", sortie: "replay/document.go.txt"},
	{chemin: "replay/document_equipment_changes.go", sortie: "replay/document_equipment_changes.go.txt"},
	{chemin: "replay/document_ground_weapon_items.go", sortie: "replay/document_ground_weapon_items.go.txt"},
	{chemin: "replay/document_weapon_changes.go", sortie: "replay/document_weapon_changes.go.txt"},
	{chemin: "replay/vehicle_cycles.go", sortie: "replay/vehicle_cycles.go.txt"},
	{chemin: "internal/grammar/components_biped_ability.go", sortie: "internal/grammar/components_biped_ability.go.txt"},
	{chemin: "killicon/data/rules.tsv", sortie: "killicon/data/rules.tsv"},
	{chemin: "replay/document_chronicle.go", sortie: "replay/document_chronicle.entetes.txt", extrait: entetesDeChronique},
}

// motifsEntetesChronique : les trois formes d'en-tete de version attestees dans `document_chronicle.go`
// (le test web `replaySchemaStatusLogic.test.ts` lit les memes, ligne a ligne).
var motifsEntetesChronique = []*regexp.Regexp{
	regexp.MustCompile(`^// v\d+ \(`),
	regexp.MustCompile(`^// SCHEMA \d+ `),
	regexp.MustCompile(`^// CE QUE (?:LA VERSION|LE SCHEMA) \d+ `),
}

// entetesDeChronique ne garde de la chronique que les lignes d'en-tete de version, dans l'ordre du fichier.
func entetesDeChronique(brut []byte) []byte {
	var out bytes.Buffer
	for _, ligne := range strings.Split(string(brut), "\n") {
		for _, re := range motifsEntetesChronique {
			if re.MatchString(ligne) {
				out.WriteString(ligne)
				out.WriteByte('\n')
				break
			}
		}
	}
	return out.Bytes()
}

// versLF ramene les fins de ligne a LF : l'instantane est le meme sous Windows et sous Linux.
func versLF(brut []byte) []byte {
	return bytes.ReplaceAll(brut, []byte("\r\n"), []byte("\n"))
}

// generer lit les sources dans la brique et rend le contenu de l'instantane, par chemin relatif a son dossier.
func generer() (map[string][]byte, error) {
	out := map[string][]byte{}
	var manifeste strings.Builder
	manifeste.WriteString("Copies des fichiers de la brique du decodeur de film que les tests web relisent.\n")
	manifeste.WriteString("Generees par `go run ./cmd/film-web-snapshot` (depuis apps/go-api) ; jamais editees a la main.\n")
	manifeste.WriteString("Colonnes : fichier de l'instantane, fichier de la brique (module du decodeur de film).\n")
	for _, s := range sources {
		chemin, err := briques.Chemin("film", filepath.FromSlash(s.chemin))
		if err != nil {
			return nil, err
		}
		brut, err := os.ReadFile(chemin) //nolint:gosec // chemin rendu par le resolveur des briques
		if err != nil {
			return nil, fmt.Errorf("lecture de %s : %w", s.chemin, err)
		}
		brut = versLF(brut)
		if s.extrait != nil {
			brut = s.extrait(brut)
			if len(brut) == 0 {
				return nil, fmt.Errorf("%s : l'extrait est vide, la forme du fichier a change", s.chemin)
			}
		}
		out[s.sortie] = brut
		fmt.Fprintf(&manifeste, "%s\t%s\n", s.sortie, s.chemin)
	}
	out[nomManifeste] = []byte(manifeste.String())
	return out, nil
}

// racineInstantane rend le dossier de l'instantane sous la racine du depot.
func racineInstantane(racineDepot string) string {
	return filepath.Join(racineDepot, filepath.FromSlash(dossierDuWeb))
}

// ecarts compare l'instantane du depot a `attendu` et nomme chaque fichier absent, different ou en trop.
func ecarts(racineDepot string, attendu map[string][]byte) ([]string, error) {
	base := racineInstantane(racineDepot)
	var out []string
	for nom, contenu := range attendu {
		lu, err := os.ReadFile(filepath.Join(base, filepath.FromSlash(nom))) //nolint:gosec // chemin sous le depot
		switch {
		case os.IsNotExist(err):
			out = append(out, nom+" : absent")
		case err != nil:
			return nil, err
		case !bytes.Equal(versLF(lu), contenu):
			out = append(out, nom+" : different de la brique")
		}
	}
	err := filepath.WalkDir(base, func(chemin string, d os.DirEntry, errMarche error) error {
		if errMarche != nil {
			if os.IsNotExist(errMarche) {
				return nil
			}
			return errMarche
		}
		if d.IsDir() {
			return nil
		}
		rel, errRel := filepath.Rel(base, chemin)
		if errRel != nil {
			return errRel
		}
		if _, ok := attendu[filepath.ToSlash(rel)]; !ok {
			out = append(out, filepath.ToSlash(rel)+" : en trop")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

// ecrire remplace le contenu de l'instantane par `attendu`.
func ecrire(racineDepot string, attendu map[string][]byte) error {
	base := racineInstantane(racineDepot)
	if err := os.RemoveAll(base); err != nil {
		return err
	}
	for nom, contenu := range attendu {
		chemin := filepath.Join(base, filepath.FromSlash(nom))
		if err := os.MkdirAll(filepath.Dir(chemin), 0o750); err != nil {
			return err
		}
		if err := os.WriteFile(chemin, contenu, 0o600); err != nil {
			return err
		}
	}
	return nil
}
