package archlint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// goSource est un fichier Go non-test du module, déjà analysé.
type goSource struct {
	Rel  string // chemin relatif à apps/go-api, séparateurs "/"
	Src  []byte
	Fset *token.FileSet
	File *ast.File
}

// apiModuleRoot rend la racine du module apps/go-api.
func apiModuleRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
}

// goSourcesUnder analyse les fichiers Go non-test sous les racines données (relatives à
// apps/go-api, séparateurs "/"). Échoue si une racine n'existe pas ou ne contient rien :
// un garde-rail qui ne voit aucun fichier ne garde rien.
func goSourcesUnder(t *testing.T, roots ...string) []goSource {
	t.Helper()
	base := apiModuleRoot(t)
	var out []goSource
	for _, root := range roots {
		dir := filepath.Join(base, filepath.FromSlash(root))
		before := len(out)
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(base, path)
			out = append(out, goSource{Rel: filepath.ToSlash(rel), Src: src, Fset: fset, File: file})
			return nil
		})
		if err != nil {
			t.Fatalf("parcours de %s : %v", root, err)
		}
		if len(out) == before {
			t.Fatalf("aucun fichier Go sous %s : le garde-rail ne couvre rien", root)
		}
	}
	return out
}

// parseGoSnippet analyse une source de test (auto-preuve des garde-rails).
func parseGoSnippet(t *testing.T, rel, src string) goSource {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, rel, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("source de test invalide : %v", err)
	}
	return goSource{Rel: rel, Src: []byte(src), Fset: fset, File: file}
}

// racinesEspaceDeTravail : les racines des modules Go du dépôt (ADR 0041), relatives à la racine
// du dépôt. Une brique de `libs/` y entre le jour où son `go.mod` est créé.
var racinesEspaceDeTravail = []string{"apps/go-api", "libs/gamefiles"}

// workspaceRoot rend la racine du dépôt (celle du `go.work`).
func workspaceRoot(t *testing.T) string {
	t.Helper()
	return filepath.Dir(filepath.Dir(apiModuleRoot(t)))
}

// fichierGo est un fichier `.go` de l'espace de travail, localisé sans être lu.
type fichierGo struct {
	Rel    string // relatif à la racine du dépôt, séparateurs "/" (ex. apps/go-api/internal/x/y.go)
	Chemin string // chemin absolu
}

// workspaceGoSources liste TOUS les fichiers `.go` (tests compris) sous les racines données,
// relatives à la racine du dépôt (`racinesEspaceDeTravail` pour « tout le Go du dépôt »). C'est le
// parcours des garde-rails à règle GÉNÉRALE : leur racine est l'espace de travail, pas un module,
// sinon une brique sous `libs/` sortirait de leur vue. Les garde-rails propres à `go-api` restent
// sur `goSourcesUnder` / `apiModuleRoot`.
//
// Les répertoires que l'outil Go n'ouvre jamais sont sautés (`dossiersInvisiblesAuGo`, `.foo`,
// `_foo`) : un fichier posé là n'est ni compilé ni testé. Échoue si une racine n'existe pas ou ne
// porte aucun fichier Go : un garde-rail qui ne voit rien ne garde rien.
func workspaceGoSources(t *testing.T, roots ...string) []fichierGo {
	t.Helper()
	depot := workspaceRoot(t)
	var out []fichierGo
	for _, root := range roots {
		base := filepath.Join(depot, filepath.FromSlash(root))
		avant := len(out)
		err := filepath.WalkDir(base, func(chemin string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				nom := d.Name()
				if chemin != base && (dossiersInvisiblesAuGo[nom] ||
					strings.HasPrefix(nom, ".") || strings.HasPrefix(nom, "_")) {
					return fs.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(chemin, ".go") {
				return nil
			}
			rel, _ := filepath.Rel(depot, chemin)
			out = append(out, fichierGo{Rel: filepath.ToSlash(rel), Chemin: chemin})
			return nil
		})
		if err != nil {
			t.Fatalf("parcours de %s : %v", root, err)
		}
		if len(out) == avant {
			t.Fatalf("aucun fichier Go sous %s : le garde-rail ne couvre rien", root)
		}
	}
	return out
}
