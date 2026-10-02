package archlint

// film_slog_contexte_test.go — LE DECODEUR JOURNALISE SOUS LE CONTEXTE DE SON APPELANT (J12.3 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, faiblesse 8).
//
// # LA REGLE
//
// Les points d entree de `replay` et `replaybuild` recoivent le `ctx` de l appelant ; toute ligne
// de journal du perimetre le porte (`slog.InfoContext(ctx, ...)`, `slog.Log(ctx, ...)`). Un
// `slog.Info(...)` sans contexte, ou un `context.Background()` fabrique en cours de route, perd ce
// que le contexte porte (identifiant de requete, cycle de synchronisation, annulation).
//
// Le test refuse, dans le perimetre :
//
//   - tout `context.Background()` / `context.TODO()`, SAUF dans la fonction `main` d un paquet
//     `main` sous `cmd/replay-*` : c est la que le contexte d un outil nait ;
//   - toute reference a `slog.Info`, `slog.Warn`, `slog.Error`, `slog.Debug` — appel OU valeur de
//     fonction (`niveau := slog.Info` contournait la regle) : on passe aux variantes `...Context`
//     ou a `slog.Log(ctx, niveau, ...)`.
//
// # PERIMETRE
//
// `internal/games/halo_infinite/film/` (hors `research/`), `internal/replaybuild/`,
// `internal/sync/killcollector/`, `cmd/replay-*`. Fichiers non-test, hors `//go:build research`.
// `grammar` et `facts` n ont en outre AUCUN journal : `film_no_slog_in_layers_test.go`.
//
// # ALLOWLIST
//
// AUCUNE, et elle doit le rester.
//
// # LA MUTATION QUI DOIT ROUGIR
//
// Ajouter `slog.Info("x")` ou `_ = context.Background()` dans un fichier non-test de `replay`,
// `replaybuild` ou `killcollector` : le test nomme le fichier et la ligne.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// slogSansContexte : les quatre fonctions de `log/slog` qui n emportent pas de contexte.
var slogSansContexte = map[string]bool{"Info": true, "Warn": true, "Error": true, "Debug": true}

// fichierDuPerimetreFilm est un fichier non-test du perimetre, deja parse.
type fichierDuPerimetreFilm struct {
	rel  string
	fset *token.FileSet
	ast  *ast.File
}

// fichiersDuPerimetreFilm rend les fichiers de production du perimetre, hors `research`. Il
// echoue sur un balayage vide : un ratchet qui ne mesure rien ne garde rien.
func fichiersDuPerimetreFilm(t *testing.T, racines ...string) []fichierDuPerimetreFilm {
	t.Helper()
	racine := apiRootDepuisIci(t)
	var out []fichierDuPerimetreFilm
	for _, r := range racines {
		err := filepath.WalkDir(filepath.Join(racine, filepath.FromSlash(r)), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(racine, p)
			rel = filepath.ToSlash(rel)
			if d.IsDir() {
				if rel == "internal/games/halo_infinite/film/research" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
				return nil
			}
			blob, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			if estSousTagResearch(blob) {
				return nil
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, p, blob, parser.SkipObjectResolution)
			if err != nil {
				return err
			}
			out = append(out, fichierDuPerimetreFilm{rel: rel, fset: fset, ast: f})
			return nil
		})
		if err != nil {
			t.Fatalf("balayage de %s : %v", r, err)
		}
	}
	if len(out) == 0 {
		t.Fatal("aucun fichier balaye : le perimetre du ratchet a bouge")
	}
	return out
}

// racinesDuPerimetreFilm : le decodeur, ses deux orchestrateurs et les outils `cmd/replay-*`.
func racinesDuPerimetreFilm(t *testing.T) []string {
	t.Helper()
	racines := []string{
		"internal/games/halo_infinite/film",
		"internal/replaybuild",
		"internal/sync/killcollector",
	}
	racine := apiRootDepuisIci(t)
	cmds, _ := filepath.Glob(filepath.Join(racine, "cmd", "replay-*"))
	if len(cmds) == 0 {
		t.Fatal("aucun cmd/replay-* trouve : le perimetre du ratchet a bouge")
	}
	for _, c := range cmds {
		rel, _ := filepath.Rel(racine, c)
		racines = append(racines, filepath.ToSlash(rel))
	}
	return racines
}

func TestLeDecodeurJournaliseSousLeContexteDeLAppelant(t *testing.T) {
	var fautes []string
	for _, f := range fichiersDuPerimetreFilm(t, racinesDuPerimetreFilm(t)...) {
		fautes = append(fautes, fautesDeContexte(f)...)
	}
	if len(fautes) > 0 {
		sort.Strings(fautes)
		t.Fatalf("journal ou contexte hors regle dans le perimetre du decodeur (J12.3) :\n  %s\n\n"+
			"Passer le `ctx` de l appelant et journaliser par `slog.XxxContext(ctx, ...)` ou "+
			"`slog.Log(ctx, niveau, ...)` ; un contexte ne nait qu a `main` d un outil `cmd/replay-*`.",
			strings.Join(fautes, "\n  "))
	}
}

// fautesDeContexte rend les references interdites d un fichier, `chemin:ligne: quoi`.
func fautesDeContexte(f fichierDuPerimetreFilm) []string {
	mainDeCmd := f.ast.Name.Name == "main" && strings.HasPrefix(f.rel, "cmd/replay-")
	var fautes []string
	for _, decl := range f.ast.Decls {
		exempte := false
		if fd, ok := decl.(*ast.FuncDecl); ok {
			exempte = mainDeCmd && fd.Recv == nil && fd.Name.Name == "main"
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			pos := f.fset.Position(sel.Pos())
			switch {
			case pkg.Name == "slog" && slogSansContexte[sel.Sel.Name]:
				fautes = append(fautes, f.rel+":"+strconv.Itoa(pos.Line)+": slog."+sel.Sel.Name+" sans contexte")
			case pkg.Name == "context" && (sel.Sel.Name == "Background" || sel.Sel.Name == "TODO") && !exempte:
				fautes = append(fautes, f.rel+":"+strconv.Itoa(pos.Line)+": context."+sel.Sel.Name+"() hors de main")
			}
			return true
		})
	}
	return fautes
}
