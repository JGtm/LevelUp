// Package archlint — title_slug_literal_test.go : pas de littéral de slug de titre dans `analysis/` ni
// `platform/duckdb/` (plan .ai/V8/PLAN_MODES_PAR_TITRE_2026-10-26.md, D4).
//
// no_slug_comparison_test.go voit `slug == "halo_infinite"` mais pas une clé de map ni un argument
// (`ModeSourceFor("halo_5")`, `map[string]X{"halo_5": ...}`) : une donnée PAR TITRE écrite en Go. Elle
// appartient au manifeste du titre (config/titles/{slug}/title.toml, chargé dans TitleDescriptor par
// domain/title/config_loader.go) et se transmet par injection depuis `config`/`api/wire` (modèle :
// `ModeSource`, `CSRSeasonID`). Le garde-rail interdit tout littéral `"halo_5"` / `"halo_infinite"` de ces
// deux arbres hors tests ; les sites existants sont allowlistés (fichier -> nombre de littéraux, raison, date)
// et ne grossissent pas : un littéral de plus, ou un de moins, échoue.
package archlint

import (
	"go/ast"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// titleSlugLiterals : les slugs de titre dont le littéral est interdit dans les arbres gardés.
var titleSlugLiterals = map[string]bool{"halo_5": true, "halo_infinite": true}

// titleSlugLiteralRoots : arbres gardés (relatifs à apps/go-api).
var titleSlugLiteralRoots = []string{"internal/analysis", "internal/platform/duckdb"}

// titleSlugLiteralAllowlist : sites existants au 2026-10-26, non migrés par ce plan.
var titleSlugLiteralAllowlist = map[string]struct {
	literals int
	reason   string
}{
	"internal/analysis/campaign_exclusion.go":                      {1, "table des variantes de Campagne masquées par titre (campaignExcludedVariantIDs) : donnée par titre en Go, à migrer vers le manifeste (plan, Découvertes)"},
	"internal/analysis/perfect_kills.go":                           {2, "table des médailles de frag parfait par titre (perfectKillMedalIDs) : même nature, à migrer vers le manifeste"},
	"internal/analysis/world_stats.go":                             {1, "slug par défaut d'un agrégat monde (analysis pur, sans dépendance au package title) : à remplacer par title.DefaultSlug"},
	"internal/platform/duckdb/media_repo_translations.go":          {1, "constante mediaStaticTitleSlug : titre par défaut des traductions média statiques"},
	"internal/platform/duckdb/world_player_stats_repo.go":          {1, "constante defaultLeaderboardTitleSlug : titre par défaut du classement monde"},
	"internal/platform/duckdb/halo5/halo5_match_history_source.go": {1, "masquage Campagne de la source Halo 5 (SQLExcludeCampaignVariants(\"halo_5\", ...)) : lié à campaign_exclusion.go"},
}

// littérauxDeSlug rend la position de chaque littéral de slug de titre d'une source.
func littérauxDeSlug(src goSource) []string {
	var out []string
	ast.Inspect(src.File, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		if v, err := strconv.Unquote(lit.Value); err == nil && titleSlugLiterals[v] {
			out = append(out, src.Fset.Position(lit.Pos()).String())
		}
		return true
	})
	return out
}

func TestTitleSlugLiteral_ReconnaitLesFormes(t *testing.T) {
	mal := parseGoSnippet(t, "internal/analysis/x.go", `package x
var parTitre = map[string]int{"halo_5": 1}
func f() int { return g("halo_infinite") }`)
	if got := littérauxDeSlug(mal); len(got) != 2 {
		t.Errorf("clé de map et argument non reconnus : %v", got)
	}
	sain := parseGoSnippet(t, "internal/analysis/y.go", `package y
const nom = "halo_5_extra"
var s = "halo"`)
	if got := littérauxDeSlug(sain); len(got) != 0 {
		t.Errorf("faux positif : %v", got)
	}
}

func TestTitleSlugLiteral_LaDonneeParTitreVientDuManifeste(t *testing.T) {
	var violations []string
	seen := map[string]int{}
	for _, src := range goSourcesUnder(t, titleSlugLiteralRoots...) {
		lits := littérauxDeSlug(src)
		if len(lits) == 0 {
			continue
		}
		seen[src.Rel] = len(lits)
		if _, ok := titleSlugLiteralAllowlist[src.Rel]; !ok {
			violations = append(violations, lits...)
		}
	}
	if len(violations) > 0 {
		sort.Strings(violations)
		t.Errorf("littéral de slug de titre dans analysis/ ou platform/duckdb/ : une donnée par titre se déclare au "+
			"manifeste (config/titles/{slug}/title.toml -> TitleDescriptor) et se transmet par injection :\n  %s",
			strings.Join(violations, "\n  "))
	}
	for rel, entry := range titleSlugLiteralAllowlist {
		if got := seen[rel]; got != entry.literals {
			t.Errorf("allowlist périmée : %s porte %d littéral(aux) de slug, l'entrée en déclare %d (%s)",
				rel, got, entry.literals, entry.reason)
		}
	}
}
