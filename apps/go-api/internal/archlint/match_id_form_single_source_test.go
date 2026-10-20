// Package archlint — match_id_form_single_source_test.go : garde-rail « une seule forme
// d'identifiant de match » (CLAUDE.md règle 6).
//
// La forme d'un identifiant de match Halo (hexadécimal et tirets, longueur bornée) décide
// si un identifiant venu d'une requête ou d'un fichier peut devenir un composant de chemin
// ou une valeur SQL. Sa définition unique est `title.IsValidMatchID`
// (internal/domain/title/film_id.go), qui garde aussi `title.FilmShortMatchID`, la clé de
// tous les chemins dérivés d'un film. Une copie ailleurs (fonction de validation, motif
// regexp) diverge en silence : trois copies de largeur différente ont coexisté (hexadécimal
// pour l'import et le sync, alphanumérique pour la séquence tactique).
package archlint

import (
	"go/ast"
	"regexp"
	"strings"
	"testing"
)

const matchIDFormOwner = "internal/domain/title/film_id.go"

// matchIDValidatorName : nom d'une fonction qui dit si un identifiant de match est valide.
var matchIDValidatorName = regexp.MustCompile(`(?i)(valid\w*match_?id|match_?id\w*valid)`)

// matchIDRegexpVarName : nom d'une variable regexp dédiée aux identifiants de match.
var matchIDRegexpVarName = regexp.MustCompile(`(?i)(motif\w*match|match\w*(re|rx|regexp|motif|pattern)$)`)

// matchIDFormCopies rend les copies de la forme d'un identifiant de match hors du
// propriétaire, et le nombre de définitions de IsValidMatchID chez lui.
func matchIDFormCopies(sources []goSource) (copies []string, ownerDefs int) {
	for _, s := range sources {
		for _, decl := range s.File.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if !matchIDValidatorName.MatchString(d.Name.Name) {
					continue
				}
				if s.Rel == matchIDFormOwner && d.Name.Name == "IsValidMatchID" && d.Recv == nil {
					ownerDefs++
					continue
				}
				copies = append(copies, s.Rel+" : func "+d.Name.Name)
			case *ast.GenDecl:
				copies = append(copies, matchIDRegexpVars(s, d)...)
			}
		}
	}
	return copies, ownerDefs
}

func matchIDRegexpVars(s goSource, d *ast.GenDecl) []string {
	var out []string
	for _, spec := range d.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for i, name := range vs.Names {
			if !matchIDRegexpVarName.MatchString(name.Name) || i >= len(vs.Values) {
				continue
			}
			if call, ok := vs.Values[i].(*ast.CallExpr); ok && isRegexpCompile(call) {
				out = append(out, s.Rel+" : var "+name.Name)
			}
		}
	}
	return out
}

func isRegexpCompile(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "regexp" && strings.HasPrefix(sel.Sel.Name, "MustCompile")
}

func TestMatchIDFormHasASingleDefinition(t *testing.T) {
	copies, ownerDefs := matchIDFormCopies(goSourcesUnder(t, "internal", "cmd"))
	if ownerDefs != 1 {
		t.Fatalf("%s doit définir IsValidMatchID exactement une fois (vu %d) : le garde-rail ne garde plus rien",
			matchIDFormOwner, ownerDefs)
	}
	for _, c := range copies {
		t.Errorf("forme d'identifiant de match recopiée : %s — appeler title.IsValidMatchID", c)
	}
}

// TestMatchIDFormGuardDetectsCopies : auto-preuve du détecteur.
func TestMatchIDFormGuardDetectsCopies(t *testing.T) {
	cases := map[string]string{
		"fonction privée":   "package x\nfunc isValidMatchID(id string) bool { return id != \"\" }\n",
		"fonction inverse":  "package x\nfunc MatchIDValide(id string) bool { return true }\n",
		"motif regexp":      "package x\nimport \"regexp\"\nvar motifMatchSequence = regexp.MustCompile(`^[a-z]+$`)\n",
		"regexp suffixée":   "package x\nimport \"regexp\"\nvar matchIDRe = regexp.MustCompile(`^[0-9a-f-]+$`)\n",
		"hors propriétaire": "package title\nfunc IsValidMatchID(id string) bool { return true }\n",
	}
	for name, src := range cases {
		copies, _ := matchIDFormCopies([]goSource{parseGoSnippet(t, "internal/x/x.go", src)})
		if len(copies) != 1 {
			t.Errorf("%s : %d copie(s) détectée(s), attendu 1", name, len(copies))
		}
	}
	owner := parseGoSnippet(t, matchIDFormOwner, "package title\nfunc IsValidMatchID(id string) bool { return true }\n")
	if copies, defs := matchIDFormCopies([]goSource{owner}); len(copies) != 0 || defs != 1 {
		t.Errorf("propriétaire : copies=%v définitions=%d, attendu 0 et 1", copies, defs)
	}
	neutral := parseGoSnippet(t, "internal/x/y.go",
		"package x\nimport \"regexp\"\nvar uuidRe = regexp.MustCompile(`x`)\nfunc ParseMatchID() {}\n")
	if copies, _ := matchIDFormCopies([]goSource{neutral}); len(copies) != 0 {
		t.Errorf("neutre : %v, attendu aucune copie", copies)
	}
}
