package grammar

// ecs_table_scan_test.go — LE LECTEUR DE LA CHAINE DE DISPATCH, pour le controle G1 de
// `ecs_table_guard_test.go` : il part de `consumeByName`, suit la branche `default` de maillon
// en maillon, et rend chaque etiquette `case` du paquet avec son statut.
//
// Sorti de `ecs_table_guard_test.go` au lot 2.7 (2026-09-16, scission des fichiers de plus de
// 500 lignes) : le meme lot a fait passer ce scanner d une fonction a une chaine, ce qui a
// pousse le fichier d origine au-dela du seuil. Les controles G1 a G4 restent la-bas ; ici il
// n y a que la lecture de l AST, et aucune assertion.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// ecsCase decrit le cas de la chaine de dispatch qui traite un composant.
type ecsCase struct {
	Kind string // porte | partiel
	File string // maillon de la chaine (dispatch_*.go depuis le lot 2.7)
	Line int
}

// scanConsumeByNameCases rend, par nom de composant, le cas de la CHAINE de dispatch qui le
// traite : `consumeByName` puis, de proche en proche, le maillon que sa branche `default`
// appelle (lot 2.7 — le switch de 815 lignes est devenu sept maillons chaines).
// `porte` = tous les retours du cas rendent le litteral `true` ; `partiel` sinon (retour
// data-dependant ou garde par un drapeau : le traverseur peut desynchroniser proprement).
func scanConsumeByNameCases(t *testing.T) map[string]ecsCase {
	t.Helper()
	fset := token.NewFileSet()
	ents, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	consts := map[string]string{}
	var files []*ast.File
	for _, e := range ents {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(".", n), nil, 0)
		if err != nil {
			t.Fatalf("%s : %v", n, err)
		}
		files = append(files, f)
	}
	funcs := map[string]*ast.FuncDecl{}
	for _, f := range files {
		for _, d := range f.Decls {
			switch v := d.(type) {
			case *ast.GenDecl:
				if v.Tok == token.CONST {
					collectStringConsts(v, consts)
				}
			case *ast.FuncDecl:
				if v.Recv == nil {
					funcs[v.Name.Name] = v
				}
			}
		}
	}
	if funcs["consumeByName"] == nil {
		t.Fatal("consumeByName introuvable dans le paquet")
	}
	out := map[string]ecsCase{}
	seen := map[string]bool{}
	links := 0
	for name := "consumeByName"; name != ""; name = defaultChainTarget(funcs[name], funcs) {
		if seen[name] {
			t.Fatalf("chaine de dispatch cyclique sur %s", name)
		}
		seen[name] = true
		links++
		for k, v := range caseKinds(funcs[name], fset, consts) {
			if prev, dup := out[k]; dup {
				t.Errorf("composant %q traite DEUX fois dans la chaine (lignes %d et %d) : "+
					"le premier maillon gagne et le second est mort — un nom de composant "+
					"n appartient qu a un seul maillon", k, prev.Line, v.Line)
			}
			out[k] = v
		}
	}
	if links < 2 {
		t.Fatalf("chaine de dispatch reduite a %d maillon(s) : le suiveur de `default` ne "+
			"trouve plus rien, il ne garde donc plus qu une partie du switch", links)
	}
	return out
}

// defaultChainTarget rend le nom du maillon suivant de la chaine de dispatch : la fonction
// appelee par la branche `default` du switch de `fn`, quand cette branche est un simple
// `return consumeXxx(...)` vers une fonction du paquet. Rend "" au dernier maillon (dont le
// `default` rend `ported=false`).
//
// POURQUOI SUIVRE LA CHAINE PLUTOT QUE LISTER LES MAILLONS. Le lot 2.7 a coupe le switch de
// 815 lignes de `consumeByName` en sept maillons chaines par leur `default`. Une LISTE de noms
// dans ce test se serait perimee au premier maillon ajoute, et le garde-rail aurait alors
// couvert moins de composants sans rien dire. Le suiveur, lui, voit tout ce que la production
// voit, par construction.
func defaultChainTarget(fn *ast.FuncDecl, funcs map[string]*ast.FuncDecl) string {
	if fn == nil {
		return ""
	}
	if fn.Body == nil {
		return ""
	}
	// Le switch de TETE du maillon, et lui seul : un `default` imbrique dans le corps d un arm
	// ne chaine rien.
	var sw *ast.SwitchStmt
	for _, s := range fn.Body.List {
		if v, ok := s.(*ast.SwitchStmt); ok {
			sw = v
			break
		}
	}
	if sw == nil {
		return ""
	}
	for _, s := range sw.Body.List {
		cc, ok := s.(*ast.CaseClause)
		if !ok || cc.List != nil { // seule la branche `default` a une List nil
			continue
		}
		for _, st := range cc.Body {
			rs, ok := st.(*ast.ReturnStmt)
			if !ok || len(rs.Results) != 1 {
				continue
			}
			call, ok := rs.Results[0].(*ast.CallExpr)
			if !ok {
				continue
			}
			if id, ok := call.Fun.(*ast.Ident); ok && funcs[id.Name] != nil {
				return id.Name
			}
		}
	}
	return ""
}

func collectStringConsts(gd *ast.GenDecl, out map[string]string) {
	for _, s := range gd.Specs {
		vs, ok := s.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for i, nm := range vs.Names {
			if i >= len(vs.Values) {
				continue
			}
			if bl, ok := vs.Values[i].(*ast.BasicLit); ok && bl.Kind == token.STRING {
				if v, err := strconv.Unquote(bl.Value); err == nil {
					out[nm.Name] = v
				}
			}
		}
	}
}

func caseKinds(fn *ast.FuncDecl, fset *token.FileSet, consts map[string]string) map[string]ecsCase {
	out := map[string]ecsCase{}
	ast.Inspect(fn, func(n ast.Node) bool {
		cc, ok := n.(*ast.CaseClause)
		if !ok || cc.List == nil { // `default` a une List nil
			return true
		}
		kind := "porte"
		ast.Inspect(cc, func(m ast.Node) bool {
			rs, ok := m.(*ast.ReturnStmt)
			if !ok || len(rs.Results) != 3 {
				return true
			}
			if id, ok := rs.Results[2].(*ast.Ident); !ok || id.Name != "true" {
				kind = "partiel"
			}
			return true
		})
		pos := fset.Position(cc.Pos())
		line, file := pos.Line, filepath.Base(pos.Filename)
		for _, e := range cc.List {
			var name string
			switch v := e.(type) {
			case *ast.BasicLit:
				name, _ = strconv.Unquote(v.Value)
			case *ast.Ident:
				name = consts[v.Name]
			}
			if name != "" {
				out[name] = ecsCase{Kind: kind, File: file, Line: line}
			}
		}
		return true
	})
	return out
}
