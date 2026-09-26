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
//   - hors de ce predicat, AUCUNE comparaison (`==`, `!=`, `case`) a un litteral qui commence par
//     `?`, et aucun appel `strings.HasPrefix` / `HasSuffix` / `Contains` / `EqualFold` /
//     `TrimPrefix` / `CutPrefix` / `Index` sur un tel litteral.
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

// fonctionsDeChaineSurRemplissage : les fonctions de `strings` qui comparent une chaine a un motif.
var fonctionsDeChaineSurRemplissage = map[string]bool{
	"HasPrefix": true, "HasSuffix": true, "Contains": true, "EqualFold": true,
	"TrimPrefix": true, "CutPrefix": true, "Index": true,
}

// TestKillsourceUnSeulPredicatDuNomDeRemplissage — LE RATCHET.
func TestKillsourceUnSeulPredicatDuNomDeRemplissage(t *testing.T) {
	dir := filepath.Join(apiRootDepuisIci(t), filepath.FromSlash(paquetKillsourceRemplissage))
	fichiers := fichiersGoNonTest(t, dir)
	var violations []string
	predicatVu := false
	for nom, f := range fichiers {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Recv == nil && fn.Name.Name == predicatDeRemplissage {
				if nom == fichierDuPredicatDeRemplissage {
					predicatVu = true
				}
				continue // le predicat est le seul lieu autorise
			}
			for _, v := range comparaisonsAuRemplissage(decl) {
				violations = append(violations, nom+" : "+v)
			}
		}
	}
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

// comparaisonsAuRemplissage rend les comparaisons a un litteral `?...` dans une declaration.
func comparaisonsAuRemplissage(decl ast.Decl) []string {
	var out []string
	ast.Inspect(decl, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BinaryExpr:
			if (x.Op == token.EQL || x.Op == token.NEQ) && (litteralDeRemplissage(x.X) || litteralDeRemplissage(x.Y)) {
				out = append(out, "comparaison "+x.Op.String()+" a un litteral `?`")
			}
		case *ast.CaseClause:
			for _, e := range x.List {
				if litteralDeRemplissage(e) {
					out = append(out, "case sur un litteral `?`")
				}
			}
		case *ast.CallExpr:
			sel, ok := x.Fun.(*ast.SelectorExpr)
			if !ok || !fonctionsDeChaineSurRemplissage[sel.Sel.Name] {
				return true
			}
			if id, ok := sel.X.(*ast.Ident); !ok || id.Name != "strings" {
				return true
			}
			for _, a := range x.Args {
				if litteralDeRemplissage(a) {
					out = append(out, "strings."+sel.Sel.Name+" sur un litteral `?`")
				}
			}
		}
		return true
	})
	return out
}

// litteralDeRemplissage : un litteral de chaine dont la valeur commence par `?`.
func litteralDeRemplissage(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return false
	}
	v, err := strconv.Unquote(lit.Value)
	return err == nil && strings.HasPrefix(v, "?")
}
