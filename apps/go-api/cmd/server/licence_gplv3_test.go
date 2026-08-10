package main

// LE GARDE-RAIL DE LICENCE — le serveur ne doit JAMAIS linker `internal/ooz`.
//
// `internal/ooz` vendorise un décodeur Oodle clean-room dérivé de powzix/ooz, **GPLv3**
// (cf. internal/ooz/NOTICE.md). Il ne sert qu'à l'outillage HORS LIGNE : extraction de
// géométrie de cartes, production d'assets (cmd/mapstruct-build, cmd/mapfond-build,
// cmd/mapquant-build). L'application distribuée LIT les assets figés, elle ne les fabrique
// pas — la géométrie extraite est une DONNÉE, pas du code, et se bake sans contrainte.
//
// POURQUOI UN TEST ET PAS UNE NOTE. La contrainte est invisible à la compilation : il suffit
// d'un `import` ajouté trois paquets plus bas pour que le binaire distribué devienne dérivé
// d'une œuvre GPLv3, sans qu'aucun outil ne le signale. Une note dans un NOTICE ne garde rien.
//
// La vérification est PUREMENT SOURCE (pas d'appel à `go list`) : elle parcourt le graphe des
// imports internes depuis cmd/server. Elle tourne donc partout, sans toolchain ni cgo.
//
// MUTATION JOUÉE (2026-08-10) : déclarer interdit `internal/domain/title`, que le serveur
// utilise réellement, fait rougir le test avec la chaîne `cmd/server -> internal/domain/title`.
// Mesure au vert : 108 paquets internes atteints depuis cmd/server, aucun interdit.

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// modulePath est le chemin du module Go de l'API.
const modulePath = "levelup/go-api"

// paquetsInterdits : les paquets dont la licence contamine le binaire distribué.
var paquetsInterdits = map[string]string{
	modulePath + "/internal/ooz": "GPLv3 (powzix/ooz, cf. internal/ooz/NOTICE.md)",
}

func TestServeurNeLinkePasDeCodeGPL(t *testing.T) {
	racine, err := racineModule()
	if err != nil {
		t.Fatal(err)
	}
	vus := map[string]bool{}
	chemin := map[string]string{} // paquet -> chaîne d'imports qui y mène
	file := []string{modulePath + "/cmd/server"}
	vus[file[0]] = true
	chemin[file[0]] = "cmd/server"

	for len(file) > 0 {
		p := file[0]
		file = file[1:]
		if raison, interdit := paquetsInterdits[p]; interdit {
			t.Fatalf("cmd/server dépend de %s — %s\nchaîne : %s", p, raison, chemin[p])
		}
		imports, err := importsDuPaquet(racine, p)
		if err != nil {
			t.Fatalf("%s : %v", p, err)
		}
		for _, imp := range imports {
			if vus[imp] {
				continue
			}
			vus[imp] = true
			chemin[imp] = chemin[p] + " -> " + strings.TrimPrefix(imp, modulePath+"/")
			file = append(file, imp)
		}
	}
	t.Logf("%d paquets internes atteints depuis cmd/server, aucun paquet interdit", len(vus))
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
	vus := map[string]bool{modulePath + "/cmd/server": true}
	file := []string{modulePath + "/cmd/server"}
	for len(file) > 0 {
		p := file[0]
		file = file[1:]
		if p == cible {
			return // le parcours atteint bien un paquet profond : il explore vraiment
		}
		imports, err := importsDuPaquet(racine, p)
		if err != nil {
			t.Fatalf("%s : %v", p, err)
		}
		for _, imp := range imports {
			if !vus[imp] {
				vus[imp] = true
				file = append(file, imp)
			}
		}
	}
	t.Fatalf("le parcours n'atteint pas %s : il n'explore pas le graphe, et son verdict ne vaut rien", cible)
}

// importsDuPaquet rend les imports INTERNES au module d'un paquet, tests exclus (un test n'est
// pas linké dans le binaire distribué).
func importsDuPaquet(racineModule, paquet string) ([]string, error) {
	dir := filepath.Join(racineModule, filepath.FromSlash(strings.TrimPrefix(paquet, modulePath+"/")))
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
			if err != nil || !strings.HasPrefix(chemin, modulePath+"/") || vus[chemin] {
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
