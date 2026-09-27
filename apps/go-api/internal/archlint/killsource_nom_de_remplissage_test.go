package archlint

// killsource_nom_de_remplissage_test.go — UN SEUL PREDICAT DU NOM DE REMPLISSAGE DANS `killsource`
// (lot J7.1 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, constat FK-2, decision DT-6).
//
// # LE DEFAUT QUE CE RATCHET FERME
//
// `killsource` fabrique des noms `?N` (roster.go) et rend `?` hors bijection. Deux sites les
// comparaient chacun a sa facon : la provenance tenait le prefixe `?` pour un silence, l assistant
// ne rejetait que la chaine `?` exacte — et publiait `?10` comme assistant NOMME (`b1ad85eb`, ecrit
// en base, affiche sur la page du match). Le predicat unique est `estNomDeRemplissage`, dans
// `killsource/roster.go`, la ou le nom est fabrique.
//
// # CE QU IL EXIGE, PAR L AST DES FICHIERS DE PRODUCTION DE `killsource`
//
//   - `estNomDeRemplissage` est declaree dans `roster.go` ;
//   - hors de ce predicat, AUCUNE comparaison (`==`, `!=`, `case`) a une expression de
//     remplissage, et aucun appel d une fonction de `strings` ou de `bytes` qui en recoit une. Une
//     expression de remplissage est un litteral de chaine qui commence par `?`, le litteral rune
//     `'?'`, ou une constante / variable (du paquet ou locale) qui vaut l un des deux — les trois
//     formes qui contournaient la premiere version (revue adverse du lot J7). La variable compte
//     qu elle soit DECLAREE (`var q = "?"`) ou AFFECTEE (`q := "?"`, `q = '?'`, revue ronde 2).
//
// Fabriquer le nom (`fmt.Sprintf("?%d", ...)`, `return "?"`) n est pas le COMPARER : ces deux
// sites restent, ils sont les ancres du registre des replis.
//
// # MUTATIONS QUI DOIVENT LE FAIRE ROUGIR
//
//   - remettre `c.roster.nameOf(f.assist) == "?"` dans `killsource/assist.go` ;
//   - ecrire `strings.HasPrefix(nom, "?")` hors du predicat ;
//   - renommer ou deplacer le predicat hors de `roster.go`.

import (
	"go/ast"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const (
	// paquetKillsourceRemplissage : le paquet garde, relatif a `apps/go-api`.
	paquetKillsourceRemplissage = "internal/games/halo_infinite/film/internal/facts/killsource"
	// predicatDeRemplissage : le seul endroit ou le paquet compare un nom a `?`.
	predicatDeRemplissage = "estNomDeRemplissage"
	// fichierDuPredicatDeRemplissage : la ou le nom de remplissage est fabrique (DT-6).
	fichierDuPredicatDeRemplissage = "roster.go"
)

// TestKillsourceUnSeulPredicatDuNomDeRemplissage — LE RATCHET.
func TestKillsourceUnSeulPredicatDuNomDeRemplissage(t *testing.T) {
	dir := filepath.Join(apiRootDepuisIci(t), filepath.FromSlash(paquetKillsourceRemplissage))
	violations, predicatVu := violationsDeRemplissage(fichiersGoNonTest(t, dir))
	if !predicatVu {
		violations = append(violations, fichierDuPredicatDeRemplissage+" : `"+predicatDeRemplissage+
			"` introuvable (DT-6 : le predicat vit la ou le nom est fabrique)")
	}
	if len(violations) == 0 {
		return
	}
	sort.Strings(violations)
	t.Fatalf("comparaison au nom de remplissage hors du predicat unique :\n  %s\n"+
		"Appeler `%s(nom)` (killsource/roster.go). Deux regles pour un meme nom ont publie `?10` "+
		"comme assistant nomme (constat FK-2).", strings.Join(violations, "\n  "), predicatDeRemplissage)
}

// violationsDeRemplissage rend, fichier par fichier, les comparaisons au nom de remplissage hors
// du predicat, et si le predicat est declare dans `roster.go`. Les CONSTANTES et VARIABLES dont la
// valeur est un litteral `?` sont collectees d abord, sur tout le paquet (et dans les fonctions) :
// les comparer vaut comparer le litteral.
func violationsDeRemplissage(fichiers map[string]*ast.File) (violations []string, predicatVu bool) {
	alias := aliasDeRemplissage(fichiers)
	for nom, f := range fichiers {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Recv == nil && fn.Name.Name == predicatDeRemplissage {
				if nom == fichierDuPredicatDeRemplissage {
					predicatVu = true
				}
				continue // le predicat est le seul lieu autorise
			}
			for _, v := range comparaisonsAuRemplissage(decl, alias) {
				violations = append(violations, nom+" : "+v)
			}
		}
	}
	return violations, predicatVu
}

// aliasDeRemplissage : les noms qui recoivent un litteral de remplissage, sur tout le paquet — par
// une DECLARATION (`const q = "?"`, `var q = '?'`) ou par une INSTRUCTION d affectation, definition
// courte comprise (`q := "?"`, `q = '?'`, `a, q := 1, "?1"` ; revue ronde 2 du lot J7, constat 3).
// La collecte est par NOM, sans portee : un homonyme ailleurs dans le paquet est suspect aussi.
func aliasDeRemplissage(fichiers map[string]*ast.File) map[string]bool {
	alias := map[string]bool{}
	noter := func(nom, valeur ast.Expr) {
		if id, ok := nom.(*ast.Ident); ok && id.Name != "_" && litteralDeRemplissage(valeur) {
			alias[id.Name] = true
		}
	}
	for _, f := range fichiers {
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.ValueSpec:
				for i, nom := range x.Names {
					if i < len(x.Values) {
						noter(nom, x.Values[i])
					}
				}
			case *ast.AssignStmt:
				if (x.Tok == token.DEFINE || x.Tok == token.ASSIGN) && len(x.Lhs) == len(x.Rhs) {
					for i := range x.Lhs {
						noter(x.Lhs[i], x.Rhs[i])
					}
				}
			}
			return true
		})
	}
	return alias
}

// comparaisonsAuRemplissage rend les comparaisons au nom de remplissage dans une declaration :
// `==` / `!=`, `case`, et tout appel d une fonction de `strings` ou de `bytes` qui recoit une
// expression de remplissage ([expressionDeRemplissage]).
func comparaisonsAuRemplissage(decl ast.Decl, alias map[string]bool) []string {
	var out []string
	ast.Inspect(decl, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BinaryExpr:
			if (x.Op == token.EQL || x.Op == token.NEQ) &&
				(expressionDeRemplissage(x.X, alias) || expressionDeRemplissage(x.Y, alias)) {
				out = append(out, "comparaison "+x.Op.String()+" au remplissage `?`")
			}
		case *ast.CaseClause:
			for _, e := range x.List {
				if expressionDeRemplissage(e, alias) {
					out = append(out, "case sur le remplissage `?`")
				}
			}
		case *ast.CallExpr:
			if q := paquetDeChaine(x); q != "" {
				for _, a := range x.Args {
					if expressionDeRemplissage(a, alias) {
						out = append(out, q+" sur le remplissage `?`")
					}
				}
			}
		}
		return true
	})
	return out
}

// paquetDeChaine rend `strings.X` / `bytes.X` pour un appel d une fonction de ces deux paquets,
// "" sinon. TOUTES leurs fonctions : chacune compare ou cherche.
func paquetDeChaine(call *ast.CallExpr) string {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	if id, ok := sel.X.(*ast.Ident); ok && (id.Name == "strings" || id.Name == "bytes") {
		return id.Name + "." + sel.Sel.Name
	}
	return ""
}

// expressionDeRemplissage : un litteral de chaine qui commence par `?`, le litteral rune `'?'`, ou
// l identifiant d une constante / variable qui en vaut un.
func expressionDeRemplissage(e ast.Expr, alias map[string]bool) bool {
	if id, ok := e.(*ast.Ident); ok {
		return alias[id.Name]
	}
	return litteralDeRemplissage(e)
}

// litteralDeRemplissage : `"?..."` ou `'?'`.
func litteralDeRemplissage(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	if !ok {
		return false
	}
	switch lit.Kind {
	case token.STRING:
		v, err := strconv.Unquote(lit.Value)
		return err == nil && strings.HasPrefix(v, "?")
	case token.CHAR:
		v, err := strconv.Unquote(lit.Value)
		return err == nil && v == "?"
	}
	return false
}
