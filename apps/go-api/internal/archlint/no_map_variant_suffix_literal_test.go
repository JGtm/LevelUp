// Package archlint — no_map_variant_suffix_literal_test.go : une seule définition de « variante de
// carte ». Les suffixes qui font d'une carte la variante d'une autre (« - Ranked », « Heavies »)
// sont déclarés dans config/titles/{slug}/mappings/map_variants.toml et lus par
// analysis.BaseMapName ; aucun autre code non-test ne les écrit en dur.
//
// POURQUOI. Une liste de suffixes recopiée diverge : trois copies existaient quand les statistiques
// par carte ont été regroupées, et chacune rattachait un ensemble de cartes différent. Une
// factorisation sans garde-rail re-diverge (CLAUDE.md, règle n°6).
//
// CE QUI EST INTERDIT, dans les fichiers .go non-test de internal/ et cmd/ :
//
//	(a) un appel strings.HasSuffix / CutSuffix / TrimSuffix dont le suffixe est un littéral qui
//	    désigne une variante (« - Ranked », « Heavies », « _-_ranked » : minuscules, espaces et
//	    soulignés ignorés) ;
//	(b) le même appel avec une constante du paquet qui porte un tel littéral ;
//	(c) une variable ou constante de paquet []string qui liste de tels suffixes (la boucle
//	    `for _, s := range liste { HasSuffix(nom, s) }` échappe à (a) et (b)).
//
// EXCEPTIONS : par fichier, datées et justifiées. Le test échoue aussi sur une exception qui ne
// désigne plus aucune occurrence (exception périmée à retirer).
package archlint

import (
	"go/ast"
	"go/token"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// mapVariantSuffixRE : un suffixe de variante de carte, une fois normalisé (minuscules, `_` -> espace,
// espaces réduits et rognés).
var mapVariantSuffixRE = regexp.MustCompile(`^(- )?(ranked|heavies)$`)

// mapVariantSuffixExceptions : fichier (depuis apps/go-api) -> justification datée.
var mapVariantSuffixExceptions = map[string]string{
	"internal/games/halo_infinite/adapter_asset_urls.go": "2026-10-26 — `mapVariantSuffixes` est la liste " +
		"d'EXCLUSION du tiroir d'assets (modes spécifiques : Heavies, Sentry Defense, Firefight), pas un " +
		"rattachement à une carte de base ; le repli de miniature d'une variante lit les suffixes du TOML.",
	"internal/games/halo_infinite/film/internal/profile/map_bounds.go": "2026-10-26 — couche `profile` du " +
		"décodeur de film, fermée par le compilateur (film_layers_deps_test) : elle ne peut pas importer " +
		"analysis. Rabote le nom d'une carte pour les bornes de déquantification, sur la preuve du level_id " +
		"(même niveau moteur). Convergence vers la liste TOML à statuer avec le décodeur.",
	"internal/games/halo_infinite/film/replay/map_background_index.go": "2026-10-26 — index des fonds de " +
		"carte : identités normalisées (`_-_ranked`) et suffixe `_firefight` propre aux sidecars de fond ; " +
		"rabotage en dernier recours après l'identité exacte. Convergence vers la liste TOML à statuer avec " +
		"le décodeur.",
}

// normaliseSuffixeDeVariante ramène un littéral à la forme comparée par mapVariantSuffixRE.
func normaliseSuffixeDeVariante(s string) string {
	s = strings.ToLower(strings.ReplaceAll(s, "_", " "))
	return strings.Join(strings.Fields(s), " ")
}

func estSuffixeDeVariante(s string) bool {
	return mapVariantSuffixRE.MatchString(normaliseSuffixeDeVariante(s))
}

// constantesChaine rend, par répertoire de paquet, les constantes de paquet de type chaîne
// (nom -> valeur) de `sources`.
func constantesChaine(sources []goSource) map[string]map[string]string {
	out := map[string]map[string]string{}
	for _, src := range sources {
		dir := path.Dir(src.Rel)
		for _, decl := range src.File.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.CONST {
				continue
			}
			for _, spec := range gd.Specs {
				vs := spec.(*ast.ValueSpec)
				for i, name := range vs.Names {
					if i >= len(vs.Values) {
						continue
					}
					if lit, ok := vs.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
						if v, err := strconv.Unquote(lit.Value); err == nil {
							if out[dir] == nil {
								out[dir] = map[string]string{}
							}
							out[dir][name.Name] = v
						}
					}
				}
			}
		}
	}
	return out
}

// suffixesDeVarianteEnDur rend les positions des écritures en dur de suffixes de variante dans `src`.
func suffixesDeVarianteEnDur(src goSource, consts map[string]map[string]string) []string {
	var found []string
	at := func(p token.Pos) string {
		return src.Rel + ":" + strconv.Itoa(src.Fset.Position(p).Line)
	}
	ast.Inspect(src.File, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			sel, ok := x.Fun.(*ast.SelectorExpr)
			pkg, isIdent := ast.Expr(nil), false
			if ok {
				pkg = sel.X
				_, isIdent = pkg.(*ast.Ident)
			}
			if !ok || !isIdent || pkg.(*ast.Ident).Name != "strings" || len(x.Args) != 2 {
				return true
			}
			switch sel.Sel.Name {
			case "HasSuffix", "CutSuffix", "TrimSuffix":
			default:
				return true
			}
			switch arg := x.Args[1].(type) {
			case *ast.BasicLit:
				if v, err := strconv.Unquote(arg.Value); err == nil && arg.Kind == token.STRING && estSuffixeDeVariante(v) {
					found = append(found, at(arg.Pos())+"  "+sel.Sel.Name+" avec le littéral "+arg.Value)
				}
			case *ast.Ident:
				if v, ok := consts[path.Dir(src.Rel)][arg.Name]; ok && estSuffixeDeVariante(v) {
					found = append(found, at(arg.Pos())+"  "+sel.Sel.Name+" avec la constante "+arg.Name)
				}
			}
		case *ast.GenDecl:
			if x.Tok != token.VAR && x.Tok != token.CONST {
				return true
			}
			for _, spec := range x.Specs {
				for _, v := range spec.(*ast.ValueSpec).Values {
					found = append(found, listeDeSuffixes(src, v, at)...)
				}
			}
		}
		return true
	})
	return found
}

// listeDeSuffixes rend les éléments d'une liste de chaînes littérale qui sont des suffixes de variante.
func listeDeSuffixes(src goSource, v ast.Expr, at func(token.Pos) string) []string {
	cl, ok := v.(*ast.CompositeLit)
	if !ok {
		return nil
	}
	var found []string
	for _, el := range cl.Elts {
		lit, ok := el.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			continue
		}
		if s, err := strconv.Unquote(lit.Value); err == nil && estSuffixeDeVariante(s) {
			found = append(found, at(lit.Pos())+"  liste de suffixes de variante "+lit.Value)
		}
	}
	return found
}

func TestNoMapVariantSuffixLiteral(t *testing.T) {
	sources := goSourcesUnder(t, "internal", "cmd")
	consts := constantesChaine(sources)

	var violations []string
	vues := map[string]bool{}
	for _, src := range sources {
		for _, v := range suffixesDeVarianteEnDur(src, consts) {
			if _, exempt := mapVariantSuffixExceptions[src.Rel]; exempt {
				vues[src.Rel] = true
				continue
			}
			violations = append(violations, v)
		}
	}
	sort.Strings(violations)
	for _, v := range violations {
		t.Errorf("suffixe de variante de carte écrit en dur (le lire de map_variants.toml via analysis.BaseMapName) : %s", v)
	}
	for f := range mapVariantSuffixExceptions {
		if !vues[f] {
			t.Errorf("exception périmée : %s n'écrit plus de suffixe de variante — la retirer", f)
		}
	}
}

// Auto-preuve : le détecteur voit les trois formes interdites et ignore les noms de carte complets.
func TestNoMapVariantSuffixLiteral_AutoPreuve(t *testing.T) {
	interdit := parseGoSnippet(t, "internal/x/interdit.go", `package x
import "strings"
const classee = " - Ranked"
var liste = []string{"_-_ranked", "_heavies"}
func f(n string) {
	_ = strings.HasSuffix(n, "Heavies")
	_, _ = strings.CutSuffix(n, classee)
}`)
	if got := suffixesDeVarianteEnDur(interdit, constantesChaine([]goSource{interdit})); len(got) != 4 {
		t.Errorf("formes interdites détectées = %d (%v), attendu 4", len(got), got)
	}
	permis := parseGoSnippet(t, "internal/x/permis.go", `package x
import "strings"
var noms = []string{"Breaker Heavies", "BTB Heavies", "Ranked Arena"}
func f(n string) {
	_ = strings.HasSuffix(n, ".png")
	_ = strings.HasPrefix(n, "Ranked")
}`)
	if got := suffixesDeVarianteEnDur(permis, constantesChaine([]goSource{permis})); len(got) != 0 {
		t.Errorf("faux positifs : %v", got)
	}
}
