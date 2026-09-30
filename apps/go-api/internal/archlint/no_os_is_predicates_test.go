package archlint

// no_os_is_predicates_test.go — `os.IsNotExist`, `os.IsExist` ET `os.IsPermission` SONT BANNIS
// DU PERIMETRE DU DECODEUR DE FILM (J12.2 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25,
// faiblesse 10).
//
// # POURQUOI
//
// Ces trois predicats ne regardent PAS la chaine d erreurs : une erreur enveloppee par
// `fmt.Errorf("...: %w", err)` leur echappe (`os.IsNotExist` rend false), alors que
// `errors.Is(err, fs.ErrNotExist)` la traverse. Le perimetre enveloppe partout ses erreurs ; un
// predicat qui ne voit que l erreur nue est une regression silencieuse au premier `%w` ajoute.
//
// # LA REGLE, EXACTEMENT
//
// Aucun fichier Go (production ou test) du perimetre ci-dessous n appelle `os.IsNotExist`,
// `os.IsExist` ni `os.IsPermission`. ALLOWLIST VIDE. Le balayage se fait sur l AST (appels
// seuls) : un commentaire qui nomme le predicat n est pas une faute.
//
// # PERIMETRE
//
// `internal/games/halo_infinite/film/`, `internal/replaybuild/`, `internal/sync/killcollector/`,
// `cmd/replay-*`, instruments de recherche compris (`film/research/` et `*_research_test.go`, compiles
// sous `-tags=research`) : le balayage lit l AST sans compiler, le tag de build est sans effet. Aucune
// exclusion.
//
// # LA MUTATION QUI DOIT ROUGIR
//
// Ajouter `os.IsNotExist(err)` dans n importe quel fichier du perimetre (par exemple
// `film/filmcache/filmcache.go`) : le test nomme le fichier et la ligne.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// perimetreSansPredicatsOS : les racines (relatives a apps/go-api) du perimetre. `cmd/replay-*`
// est resolu par glob.
var perimetreSansPredicatsOS = []string{
	"internal/games/halo_infinite/film",
	"internal/replaybuild",
	"internal/sync/killcollector",
}

var predicatsOSInterdits = map[string]bool{"IsNotExist": true, "IsExist": true, "IsPermission": true}

func TestAucunPredicatOSIsDansLePerimetreDuFilm(t *testing.T) {
	racine := apiRootDepuisIci(t)
	racines := append([]string{}, perimetreSansPredicatsOS...)
	cmds, err := filepath.Glob(filepath.Join(racine, "cmd", "replay-*"))
	if err != nil || len(cmds) == 0 {
		t.Fatalf("aucun cmd/replay-* trouve (%v) : le perimetre du ratchet a bouge", err)
	}
	for _, c := range cmds {
		rel, _ := filepath.Rel(racine, c)
		racines = append(racines, filepath.ToSlash(rel))
	}
	var fautes []string
	fichiers := 0
	for _, r := range racines {
		err := filepath.WalkDir(filepath.Join(racine, filepath.FromSlash(r)), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(racine, p)
			rel = filepath.ToSlash(rel)
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(d.Name(), ".go") {
				return nil
			}
			fichiers++
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
			if err != nil {
				return err
			}
			ast.Inspect(f, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok || !predicatsOSInterdits[sel.Sel.Name] {
					return true
				}
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == "os" {
					fautes = append(fautes, rel+":"+strconv.Itoa(fset.Position(sel.Pos()).Line)+" os."+sel.Sel.Name)
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatalf("balayage de %s : %v", r, err)
		}
	}
	if fichiers == 0 {
		t.Fatal("aucun fichier balaye : le ratchet ne mesure plus rien")
	}
	if len(fautes) > 0 {
		t.Fatalf("%d appel(s) a un predicat os.Is* dans le perimetre du decodeur de film :\n  %s\n\n"+
			"Utiliser `errors.Is(err, fs.ErrNotExist)` (ou `fs.ErrExist` / `fs.ErrPermission`) : "+
			"seul errors.Is traverse les erreurs enveloppees par `%%w`.",
			len(fautes), strings.Join(fautes, "\n  "))
	}
}
