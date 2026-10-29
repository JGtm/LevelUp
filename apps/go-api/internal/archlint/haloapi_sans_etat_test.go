// haloapi_sans_etat_test.go — deux invariants du contrat de la brique « API Halo »
// (`internal/haloapi/doc.go`, `internal/haloapi/xblauth/doc.go`), tenus sur l'AST :
//
//   - `xblauth` est SANS ÉTAT : aucune variable de paquet en production. Deux formes restent
//     permises, parce qu'elles ne portent aucun état : l'assertion d'interface `var _ I = ...` et
//     l'erreur sentinelle `var ErrX = errors.New(...)` (immuable par convention : personne ne
//     réaffecte une sentinelle, et `errors.Is` en dépend) ;
//   - la brique ne lit NI l'environnement NI un fichier : aucun appel `os.Getenv`, `os.LookupEnv`,
//     `os.Environ`, `os.ReadFile`, `os.ReadDir`, `os.Open`, `os.OpenFile`, `os.Create`,
//     `os.WriteFile` dans sa production. Ce dont elle a besoin (client Azure, adresses, langue) lui
//     est PASSÉ.
package archlint

import (
	"go/ast"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// racineXblauthSurDisque : la chaîne d'authentification de la brique, relative à `apps/go-api`.
const racineXblauthSurDisque = racineHaloapiSurDisque + "/xblauth"

// fonctionsOSInterditesBrique : les fonctions de `os` qui lisent l'environnement ou touchent un
// fichier.
var fonctionsOSInterditesBrique = map[string]bool{
	"Getenv": true, "LookupEnv": true, "Environ": true,
	"ReadFile": true, "ReadDir": true, "Open": true, "OpenFile": true, "Create": true, "WriteFile": true,
}

// estSentinelle dit si `v` est un appel `<errors>.New(...)`.
func estSentinelle(v ast.Expr, pkgErrors string) bool {
	call, ok := v.(*ast.CallExpr)
	return ok && pkgErrors != "" && estSelecteur(call.Fun, pkgErrors, "New")
}

// variablesDePaquet rend, triées, les variables de paquet de `src` qui portent un état : tout `var`
// de premier niveau, hors assertion d'interface (`var _ I = ...`) et erreur sentinelle
// (`var ErrX = errors.New(...)`). Fonction pure.
func variablesDePaquet(src goSource) []string {
	pkgErrors := nomLocalImport(src.File, "errors")
	var out []string
	for _, decl := range src.File.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			continue
		}
		for _, spec := range gd.Specs {
			vs := spec.(*ast.ValueSpec)
			for i, nom := range vs.Names {
				if nom.Name == "_" {
					continue
				}
				if i < len(vs.Values) && estSentinelle(vs.Values[i], pkgErrors) {
					continue
				}
				out = append(out, src.Rel+":"+strconv.Itoa(src.Fset.Position(nom.Pos()).Line)+" : var "+nom.Name)
			}
		}
	}
	sort.Strings(out)
	return out
}

// lecturesEnvOuFichier rend, triés, les appels de `src` aux fonctions de `os` qui lisent
// l'environnement ou touchent un fichier. Fonction pure.
func lecturesEnvOuFichier(src goSource) []string {
	pkg := nomLocalImport(src.File, "os")
	if pkg == "" || pkg == "_" {
		return nil
	}
	var out []string
	ast.Inspect(src.File, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if ok && fonctionsOSInterditesBrique[sel.Sel.Name] && estSelecteur(sel, pkg, sel.Sel.Name) {
			out = append(out, src.Rel+":"+strconv.Itoa(src.Fset.Position(sel.Pos()).Line)+" : os."+sel.Sel.Name)
		}
		return true
	})
	sort.Strings(out)
	return out
}

func TestXblauthSansVariableDePaquet(t *testing.T) {
	var violations []string
	for _, src := range goSourcesUnder(t, racineXblauthSurDisque) {
		violations = append(violations, variablesDePaquet(src)...)
	}
	if len(violations) > 0 {
		t.Errorf("variable de paquet dans xblauth (contrat SANS ÉTAT de xblauth/doc.go : ce qui retient un "+
			"état appartient à l'appelant, qui le passe par `Chaine`) :\n  %s", strings.Join(violations, "\n  "))
	}
}

func TestHaloapiSansEnvNiFichier(t *testing.T) {
	var violations []string
	for _, src := range goSourcesUnder(t, racineHaloapiSurDisque) {
		violations = append(violations, lecturesEnvOuFichier(src)...)
	}
	if len(violations) > 0 {
		t.Errorf("la brique haloapi lit l'environnement ou un fichier (contrat SANS POLITIQUE : la valeur "+
			"lui est PASSÉE par LevelUp) :\n  %s", strings.Join(violations, "\n  "))
	}
}

// TestHaloapiSansEtat_AutoPreuve : chaque forme interdite rougit dans un extrait présenté comme un
// fichier de la brique ; les formes permises passent.
func TestHaloapiSansEtat_AutoPreuve(t *testing.T) {
	const rel = "internal/haloapi/xblauth/x.go"
	for nom, source := range map[string]string{
		"compteur":          `package xblauth; var n int`,
		"carte":             `package xblauth; var cache = map[string]string{}`,
		"bloc":              "package xblauth\nvar (\n\t_ = 1\n\tmu int\n)",
		"erreur non errors": `package xblauth; import "fmt"; var ErrX = fmt.Errorf("x")`,
		// Import aliasé : les deux sentinelles passent, seule `z` porte un état.
		"errors aliasé": `package xblauth; import e "errors"; var ErrX = e.New("x"); var ErrY = e.New("y"); var z = 3`,
	} {
		if got := variablesDePaquet(parseGoSnippet(t, rel, source)); len(got) != 1 {
			t.Errorf("%s : %d violation(s) %v, 1 attendue", nom, len(got), got)
		}
	}
	for nom, source := range map[string]string{
		"sentinelle":      `package xblauth; import "errors"; var ErrX = errors.New("x")`,
		"assertion":       `package xblauth; type T struct{}; func (T) M() {}; var _ interface{ M() } = T{}`,
		"variable locale": `package xblauth; func f() { var n int; _ = n }`,
		"constante":       `package xblauth; const n = 1`,
	} {
		if got := variablesDePaquet(parseGoSnippet(t, rel, source)); len(got) != 0 {
			t.Errorf("%s : violation(s) %v, aucune attendue", nom, got)
		}
	}

	for nom, source := range map[string]string{
		"Getenv":    `package infinite; import "os"; func f() { _ = os.Getenv("X") }`,
		"LookupEnv": `package infinite; import "os"; func f() { _, _ = os.LookupEnv("X") }`,
		"ReadFile":  `package infinite; import "os"; func f() { _, _ = os.ReadFile("x") }`,
		"Open":      `package infinite; import "os"; func f() { _, _ = os.Open("x") }`,
		"Create":    `package infinite; import "os"; func f() { _, _ = os.Create("x") }`,
		"WriteFile": `package infinite; import "os"; func f() { _ = os.WriteFile("x", nil, 0) }`,
		"valeur":    `package infinite; import "os"; var lire = os.ReadFile`,
		"aliasé":    `package infinite; import o "os"; func f() { _ = o.Getenv("X") }`,
	} {
		if got := lecturesEnvOuFichier(parseGoSnippet(t, rel, source)); len(got) != 1 {
			t.Errorf("%s : %d violation(s) %v, 1 attendue", nom, len(got), got)
		}
	}
	for nom, source := range map[string]string{
		"commentaire":    "package infinite\n// jamais os.Getenv ni os.ReadFile\nfunc f() {}",
		"sans import os": `package infinite; type o struct{}; func (o) Getenv(string) string { return "" }; func f() { var os o; _ = os.Getenv("X") }`,
		"os permis":      `package infinite; import "os"; var _ = os.ErrNotExist`,
	} {
		if got := lecturesEnvOuFichier(parseGoSnippet(t, rel, source)); len(got) != 0 {
			t.Errorf("%s : violation(s) %v, aucune attendue", nom, got)
		}
	}
}
