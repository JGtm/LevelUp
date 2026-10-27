// Package archlint — mode_gamevariant_fallback_test.go : garde-rail « le mode d'un match est fourni
// par la source du titre » (plan .ai/V8/PLAN_MODES_PAR_TITRE_2026-10-26.md).
//
// Le mode d'un match est `Summary.PairMode`, rempli UNE fois par la source de lecture selon la règle
// du titre (`TitleDescriptor.ModeSource`, déclaré au manifeste du titre ; platform/duckdb.ModeAssetReference). Une page qui lit
// `Summary.GameVariant` pour décider d'un libellé de mode refait le repli « partie sinon type de jeu »
// que la source rend déjà, et diverge de la source au premier titre qui change de règle.
//
// Le garde-rail interdit tout accès `<expr>.Summary.GameVariant` dans le code de production hors de
// l'allowlist ci-dessous (fichier -> nombre d'accès tolérés, avec sa raison). Une entrée dont le
// compte change échoue dans les deux sens : un accès de plus doit être justifié ; un accès de moins
// retire l'entrée.
package archlint

import (
	"go/ast"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// gameVariantReadsAllowlist : accès légitimes à Summary.GameVariant, datés du 2026-10-26.
var gameVariantReadsAllowlist = map[string]struct {
	reads  int
	reason string
}{
	"internal/analysis/home_canonical_converters.go":               {5, "expose l'identifiant et le nom du type de jeu à l'API : donnée du match, pas un libellé de mode"},
	"internal/analysis/home_canonical_recent.go":                   {2, "variantNameOf : le nom du type de jeu est la clé de [rounds_decide] (ADR 0032), pas un libellé de mode"},
	"internal/analysis/stats_canonical.go":                         {3, "recopie le type de jeu dans la ligne brute (GameVariantName/FR) ; le repli de la famille ligne brute est analysis.ResolveModeUIWithVariant, hors du périmètre canonique (plan, Découvertes)"},
	"internal/platform/duckdb/home_repo_translations_canonical.go": {2, "traduit le type de jeu lui-même (dictionnaire game_variant)"},
	"internal/service/squad_service_v2_history.go":                 {2, "libellé de mode de l'historique Escouade lu sur le type de jeu pour les deux titres : lecture propre, à trancher hors de ce plan (plan, Découvertes)"},
	"internal/service/teammates/teammates_squad_expected_fda.go":   {2, "le nom du type de jeu est la clé du modèle d'écart attendu, pas un libellé de mode"},
}

// lecturesDeGameVariant rend la position de chaque accès `<expr>.Summary.GameVariant` d'une source.
func lecturesDeGameVariant(src goSource) []string {
	var out []string
	ast.Inspect(src.File, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "GameVariant" {
			return true
		}
		inner, ok := sel.X.(*ast.SelectorExpr)
		if !ok || inner.Sel.Name != "Summary" {
			return true
		}
		out = append(out, src.Fset.Position(sel.Pos()).String())
		return true
	})
	return out
}

func TestModeGameVariantFallback_ReconnaitLaLecture(t *testing.T) {
	mal := parseGoSnippet(t, "internal/service/x.go", `package x
func f(r Row) string {
	if r.Summary.PairMode == nil {
		return r.Summary.GameVariant.DefaultLabel
	}
	return ""
}`)
	if got := lecturesDeGameVariant(mal); len(got) != 1 {
		t.Errorf("le repli sur Summary.GameVariant n'est pas reconnu : %v", got)
	}
	sain := parseGoSnippet(t, "internal/service/y.go", `package y
func f(r Row, s Summary) string {
	_ = s.GameVariant
	return r.Summary.PairMode.DefaultLabel
}`)
	if got := lecturesDeGameVariant(sain); len(got) != 0 {
		t.Errorf("faux positif : %v", got)
	}
}

func TestModeGameVariantFallback_LeModeVientDeLaSource(t *testing.T) {
	var violations []string
	seen := map[string]int{}
	for _, src := range goSourcesUnder(t, "internal") {
		reads := lecturesDeGameVariant(src)
		if len(reads) == 0 {
			continue
		}
		seen[src.Rel] = len(reads)
		if _, ok := gameVariantReadsAllowlist[src.Rel]; !ok {
			violations = append(violations, reads...)
		}
	}
	if len(violations) > 0 {
		sort.Strings(violations)
		t.Errorf("Summary.GameVariant lu hors allowlist : le mode d'un match est Summary.PairMode, fourni par la "+
			"source du titre (mode_source du manifeste). Lire PairMode, ou allowlister avec une raison datée :\n  %s",
			strings.Join(violations, "\n  "))
	}
	for rel, entry := range gameVariantReadsAllowlist {
		if got := seen[rel]; got != entry.reads {
			t.Errorf("allowlist périmée : %s lit Summary.GameVariant %s fois, l'entrée en déclare %d (%s)",
				rel, strconv.Itoa(got), entry.reads, entry.reason)
		}
	}
}
