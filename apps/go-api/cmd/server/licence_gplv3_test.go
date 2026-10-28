package main

// LE GARDE-RAIL DE LICENCE — le serveur ne doit JAMAIS linker `libs/gamefiles/ooz`.
//
// `libs/gamefiles/ooz` vendorise un décodeur Oodle clean-room dérivé de powzix/ooz, **GPLv3**
// (cf. libs/gamefiles/ooz/NOTICE.md). Il ne sert qu'à l'outillage HORS LIGNE : extraction de
// géométrie de cartes, production d'assets (cmd/mapstruct-build, cmd/mapfond-build,
// cmd/mapquant-build). L'application distribuée LIT les assets figés, elle ne les fabrique
// pas — la géométrie extraite est une DONNÉE, pas du code, et se bake sans contrainte.
//
// POURQUOI UN TEST ET PAS UNE NOTE. La contrainte est invisible à la compilation : il suffit
// d'un `import` ajouté trois paquets plus bas pour que le binaire distribué devienne dérivé
// d'une œuvre GPLv3, sans qu'aucun outil ne le signale. Une note dans un NOTICE ne garde rien.
//
// La vérification est PUREMENT SOURCE (pas d'appel à `go list`) : elle parcourt le graphe des
// imports du module `levelup/go-api` ET de TOUS les modules de `libs/` (ADR 0041) depuis cmd/server.
// Sans cela, le déplacement de `ooz` hors de `internal/` aurait rendu le garde-rail aveugle : un
// import hors du module n'est pas suivi, et `TestLeParcoursFranchitLeModuleGamefiles` l'empêche.
// La chaîne `himap -> himodule -> ooz` passe, elle, par DEUX modules de `libs/` :
// `TestLaChaineDeHimapAtteintOoz` prouve que le parcours la voit, le jour où le serveur importerait
// `himap`. Le garde-rail tourne donc partout, sans toolchain ni cgo.
//
// MUTATION JOUÉE (2026-08-10) : déclarer interdit `internal/domain/title`, que le serveur
// utilise réellement, fait rougir le test avec la chaîne `cmd/server -> internal/domain/title`.
// Mesure au vert : voir le journal du test (paquets atteints depuis cmd/server, aucun interdit).

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// modulePath est le chemin du module Go de l'API ; libsPrefix le préfixe d'import des briques
// (dossier libs/ de la racine du dépôt : le chemin du module EST le chemin du dossier).
const (
	modulePath      = "levelup/go-api"
	libsPrefix      = "github.com/JGtm/LevelUp/libs/"
	gamefilesModule = libsPrefix + "gamefiles"
)

// paquetsInterdits : les paquets dont la licence contamine le binaire distribué.
var paquetsInterdits = map[string]string{
	gamefilesModule + "/ooz": "GPLv3 (powzix/ooz, cf. libs/gamefiles/ooz/NOTICE.md)",
}

const departServeur = modulePath + "/cmd/server"

func TestServeurNeLinkePasDeCodeGPL(t *testing.T) {
	racine, err := racineModule()
	if err != nil {
		t.Fatal(err)
	}
	chaines, err := chainesDepuis(racine, departServeur)
	if err != nil {
		t.Fatal(err)
	}
	for p, chaine := range chaines {
		if raison, interdit := paquetsInterdits[p]; interdit {
			t.Fatalf("cmd/server dépend de %s — %s\nchaîne : %s", p, raison, chaine)
		}
	}
	t.Logf("%d paquets internes atteints depuis cmd/server, aucun paquet interdit", len(chaines))
}

// chainesDepuis rend, pour chaque paquet atteignable depuis `depart` par les imports (modules
// `levelup/go-api` et `libs/*`, tests exclus), la chaîne d'imports qui y mène. Un seul parcours en
// largeur pour le garde-rail et pour ses témoins.
func chainesDepuis(racineModule, depart string) (map[string]string, error) {
	chaines := map[string]string{depart: strings.TrimPrefix(depart, modulePath+"/")}
	file := []string{depart}
	for len(file) > 0 {
		p := file[0]
		file = file[1:]
		imports, err := importsDuPaquet(racineModule, p)
		if err != nil {
			return nil, fmt.Errorf("%s : %w", p, err)
		}
		for _, imp := range imports {
			if _, vu := chaines[imp]; vu {
				continue
			}
			chaines[imp] = chaines[p] + " -> " + strings.TrimPrefix(imp, modulePath+"/")
			file = append(file, imp)
		}
	}
	return chaines, nil
}

// TestGardeRailDeLicenceDetecteUneViolation joue la MUTATION du garde-rail ci-dessus : si l'on
// déclare interdit un paquet que le serveur utilise réellement, la recherche doit le trouver.
//
// Sans ce témoin, un parcours qui n'explorerait rien (mauvaise racine, extraction d'imports
// cassée) passerait au vert en ne trouvant jamais rien — le piège le plus cher de ce chantier.
func TestGardeRailDeLicenceDetecteUneViolation(t *testing.T) {
	racine, err := racineModule()
	if err != nil {
		t.Fatal(err)
	}
	// `internal/domain/title` est atteint par le serveur (PathResolver, registre des titres).
	cible := modulePath + "/internal/domain/title"
	chaines, err := chainesDepuis(racine, departServeur)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := chaines[cible]; !ok {
		t.Fatalf("le parcours n'atteint pas %s : il n'explore pas le graphe, et son verdict ne vaut rien", cible)
	}
}

// dossierDuPaquet rend le dossier d'un paquet des modules suivis : `levelup/go-api/…` sous la
// racine du module, `…/libs/<brique>/…` sous `libs/<brique>` de la racine du dépôt (deux niveaux
// au-dessus de `apps/go-api`).
func dossierDuPaquet(racineModule, paquet string) string {
	if reste, ok := strings.CutPrefix(paquet, libsPrefix); ok {
		depot := filepath.Dir(filepath.Dir(racineModule))
		return filepath.Join(depot, "libs", filepath.FromSlash(reste))
	}
	return filepath.Join(racineModule, filepath.FromSlash(strings.TrimPrefix(paquet, modulePath+"/")))
}

// estSuivi dit si un import vise un paquet d'un des modules dont on parcourt le graphe : le module
// de l'application ou une brique de `libs/`.
func estSuivi(chemin string) bool {
	return strings.HasPrefix(chemin, modulePath+"/") || strings.HasPrefix(chemin, libsPrefix)
}

// importsDuPaquet rend les imports d'un paquet qui visent l'un des modules suivis, tests exclus
// (un test n'est pas linké dans le binaire distribué).
func importsDuPaquet(racineModule, paquet string) ([]string, error) {
	dir := dossierDuPaquet(racineModule, paquet)
	entrees, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	vus := map[string]bool{}
	var out []string
	for _, e := range entrees {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, nom), nil, parser.ImportsOnly)
		if err != nil {
			return nil, err
		}
		for _, imp := range f.Imports {
			chemin, err := strconv.Unquote(imp.Path.Value)
			if err != nil || !estSuivi(chemin) || vus[chemin] {
				continue
			}
			vus[chemin] = true
			out = append(out, chemin)
		}
	}
	return out, nil
}

// racineModule remonte jusqu'au dossier qui porte go.mod.
func racineModule() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for d := wd; ; {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d, nil
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", os.ErrNotExist
		}
		d = parent
	}
}

// TestLeParcoursFranchitLeModuleGamefiles : le parcours entre bien dans le module `libs/gamefiles`.
// Le serveur importe `mapvar` (via le décodeur de film) : si le parcours ne suit plus les imports
// vers ce module, `ooz` n'est plus jamais atteint ni cherché, et le garde-rail ci-dessus passerait
// au vert en ne regardant rien — le même piège que la mutation sur `internal/domain/title`.
func TestLeParcoursFranchitLeModuleGamefiles(t *testing.T) {
	racine, err := racineModule()
	if err != nil {
		t.Fatal(err)
	}
	cible := gamefilesModule + "/mapvar"
	chaines, err := chainesDepuis(racine, departServeur)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := chaines[cible]; !ok {
		t.Fatalf("le parcours n'atteint pas %s : il ne franchit pas la frontière du module gamefiles", cible)
	}
}

// TestLaChaineDeHimapAtteintOoz : la chaîne GPL de la brique `himap` (`himap -> himodule -> ooz`,
// deux modules de `libs/`) est visible du parcours. Le serveur n'importe pas `himap` aujourd'hui ;
// le jour où un de ses paquets l'importe, `TestServeurNeLinkePasDeCodeGPL` ne rougit QUE si le
// parcours suit un import de `libs/himap` puis de `libs/gamefiles`. Ce témoin le prouve en
// partant de `himap` lui-même, sans qu'il faille muter le serveur.
func TestLaChaineDeHimapAtteintOoz(t *testing.T) {
	racine, err := racineModule()
	if err != nil {
		t.Fatal(err)
	}
	chaines, err := chainesDepuis(racine, libsPrefix+"himap/himap")
	if err != nil {
		t.Fatal(err)
	}
	ooz := gamefilesModule + "/ooz"
	chaine, ok := chaines[ooz]
	if !ok {
		t.Fatalf("depuis libs/himap/himap, le parcours n'atteint pas %s : un serveur qui importerait himap "+
			"linkerait du GPLv3 sans que le garde-rail le voie", ooz)
	}
	if _, interdit := paquetsInterdits[ooz]; !interdit {
		t.Fatalf("%s atteint par %s mais absent de paquetsInterdits", ooz, chaine)
	}
}
