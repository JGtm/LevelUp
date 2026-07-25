package prestige

// no_cross_title_aggregation_test.go — GARDE-RAIL « indépendance stricte par titre ».
//
// Décision produit gouvernante (2026-07-18, Option A confirmée par l'utilisateur ;
// exécutée le 2026-07-25, chantier v7.2.1 item V721-09) :
//
//	Les arcs, les défis, les valeurs et les points de progression (PP) sont
//	STRICTEMENT INDÉPENDANTS PAR TITRE. Un même joueur a des arcs en Halo 5 ET des
//	arcs en Halo Infinite : deux ensembles DISJOINTS, chacun mesuré sur son titre.
//	La juxtaposition d'affichage est autorisée ; la FUSION ne l'est jamais.
//
// Interdits verrouillés ici, dans internal/prestige/ et internal/progression/ :
//   - itérer sur une COLLECTION de title_slug pour un même calcul de valeur/PP ;
//   - exposer une fonction qui produit une LISTE de titres (seam multi-titres) ;
//   - appeler/déclarer un symbole « CrossTitle » (agrégation tous titres confondus) ;
//   - requêter une table per-titre sans filtre `title_slug`.
//
// Modèle : internal/sync/no_art_patterns_test.go (scan de source + allowlist explicite
// et justifiée + test anti-pourrissement de l'allowlist). Différence : le scan est ici
// AST (go/parser) et non textuel — les commentaires ne sont donc jamais matchés (mode
// de parsing 0), ce qui évite la classe de faux positifs du scan textuel.
//
// PORTÉE — ce que ce garde-rail NE couvre PAS (pour ne pas donner de fausse assurance) :
// la couche d'infrastructure `internal/platform/duckdb/prestige/` est HORS scan (le
// périmètre arbitré du plan est prestige/ + progression/). UNE agrégation cross-titre y
// vit encore : `PrestigeSocialRepo.GetUserPrestigeCrossTitle`, qui somme `total_pp` tous
// titres confondus. Son POINT D'ENTRÉE côté service est, lui, dans le scan et allowlisté
// ci-dessous : c'est la trace exécutable de cette dette.
// La seconde (branche `titleSlug == nil` de `GetLeaderboard`, SUM … GROUP BY user_id) a
// été SUPPRIMÉE le 2026-07-25 (V721-14a / D-08) : sans appelant de production, le
// paramètre est passé de `*string` à `string` — la voie cross-titre n'est plus
// exprimable à la compilation. Morsure runtime :
// TestPrestigeSocialRepo_Leaderboard_RejectsEmptyTitle (platform/duckdb/prestige).
//
// MORSURE : TestCrossTitleDetectorsBite rejoue le code EXACT retiré le 2026-07-25
// (`creditTitlesFor` + sa boucle appelante) et exige que chaque détecteur le voie ;
// TestCrossTitleDetectorsSpareMonoTitle exige le silence sur l'équivalent mono-titre.
// Un garde-rail qui ne peut pas échouer est pire qu'aucun garde-rail.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// crossTitleScanRoots : sous-arbres scannés, relatifs à la racine du module go-api.
var crossTitleScanRoots = []string{
	"internal/prestige",
	"internal/progression",
}

// Identifiants courts des détecteurs — servent de suffixe de clé d'allowlist.
const (
	kindRangeOverTitles  = "range-over-titles"
	kindMultiTitleList   = "multi-title-list-producer"
	kindCrossTitleSymbol = "cross-title-symbol"
	kindSQLWithoutTitle  = "sql-without-title-filter"
)

// allowlistCrossTitle : violations TOLÉRÉES, clé `chemin/relatif.go#kind`, valeur =
// justification datée. Ne JAMAIS ajouter d'entrée sans justification ni date : une
// entrée non justifiée transforme le ratchet en décoration.
// TestCrossTitleAllowlistNotStale supprime le risque d'entrée morte.
var allowlistCrossTitle = map[string]string{
	"internal/prestige/repository.go#cross-title-symbol": "DETTE (relevée le 2026-07-25, V721-09) — " +
		"`PrestigeRepo.GetUserPrestigeCrossTitle` somme les PP de TOUS les titres d'un joueur " +
		"(`SELECT SUM(total_pp) FROM user_prestige_latest WHERE user_id = ?`). Contredit la " +
		"décision produit du 2026-07-18. Livré avant elle et servi en prod par " +
		"GET /prestige/me sans `title_slug` → le retirer change une réponse d'API : DÉCISION " +
		"PRODUIT REQUISE (rendre `title_slug` obligatoire, ou juxtaposer par titre). Hors " +
		"périmètre de V721-09, tracé ici pour ne pas être perdu.",
	"internal/prestige/service.go#cross-title-symbol": "DETTE (relevée le 2026-07-25, V721-09) — " +
		"`service.GetUserPrestige` bascule sur la voie cross-titre quand `titleSlug == \"\"` " +
		"(service.go, branche `if titleSlug == \"\"`). Même arbitrage produit que l'entrée " +
		"`repository.go#cross-title-symbol` ci-dessus ; les deux tombent ensemble.",
}

// crossTitlePerTitleTables : tables portant une colonne `title_slug`. Les requêter sans
// filtrer dessus = lire « tous titres confondus ». (Les tables physiquement isolées par
// chemin — match_registry, match_skill_rank_latest… — n'ont PAS de title_slug : ADR 0008,
// l'isolation est le répertoire ; elles ne sont donc pas listées ici.)
var crossTitlePerTitleTables = []string{
	"arc",
	"challenge",
	"prestige_events",
	"user_prestige_latest",
	"user_prestige_history",
	"improvement_campaign",
	"streak",
	"record_history",
	"milestone_earned",
	"baseline_state",
}

var (
	// Variable de boucle désignant un slug (motif exact du seam retiré :
	// `for _, slug := range creditTitlesFor(c)`).
	reTitleLoopVar = regexp.MustCompile(`(?i)^(slug|slugs|titleslug|titleslugs)$`)
	// Découpe un identifiant Go en mots sur les frontières camelCase/PascalCase,
	// y compris les acronymes collés (`titleIDs` -> title, IDs). Sert à décider
	// « pluriel » sur un MOT entier plutôt que sur une sous-chaîne — cf.
	// isTitleCollectionName.
	reIdentWordSplit = regexp.MustCompile(`[A-Z]+(?:[a-z0-9]*)|[a-z0-9]+`)
	// Symbole exposant explicitement une voie « tous titres confondus ».
	reCrossTitleIdent = regexp.MustCompile(`(?i)crosstitle`)
)

// isTitleCollectionName dit si un identifiant désigne une COLLECTION de titres.
//
// La décision se prend sur un MOT entier du nom, pas sur une sous-chaîne : une
// regex `(titles|slugs)` appliquée au nom entier confond le pluriel `creditTitlesFor`
// avec le singulier `titleSlug` (« title|slug » collés donnent la sous-chaîne
// « titlesl… »). La première rédaction de ce détecteur tentait de s'en sortir avec
// un suffixe `(?:[^a-z]|$)` : il écartait bien `titleSlug`, mais il écartait AUSSI
// `creditTitlesFor` (« Titles » y est suivi de « For »), c'est-à-dire précisément
// le seam que ce ratchet doit interdire. Le découpage en mots tranche les deux cas
// sans compromis.
func isTitleCollectionName(name string) bool {
	for _, w := range reIdentWordSplit.FindAllString(name, -1) {
		switch strings.ToLower(w) {
		case "titles", "slugs":
			return true
		}
	}
	return false
}

// crossTitleTableRes : `FROM|JOIN|INTO|UPDATE <table>` ancré au nom exact (`\b` ne
// franchit pas le `_`, donc `arc` ne matche jamais `arc_titles`).
var crossTitleTableRes = func() map[string]*regexp.Regexp {
	m := make(map[string]*regexp.Regexp, len(crossTitlePerTitleTables))
	for _, tbl := range crossTitlePerTitleTables {
		m[tbl] = regexp.MustCompile(`(?i)\b(?:FROM|JOIN|INTO|UPDATE)\s+` + regexp.QuoteMeta(tbl) + `\b`)
	}
	return m
}()

// crossTitleViolation localise un motif interdit.
type crossTitleViolation struct {
	File   string // relatif au module, séparateurs slash
	Kind   string
	Line   int
	Detail string
}

func (v crossTitleViolation) key() string { return v.File + "#" + v.Kind }

func (v crossTitleViolation) String() string {
	return fmt.Sprintf("%s:%d [%s] %s", v.File, v.Line, v.Kind, v.Detail)
}

// scanCrossTitleSource analyse UNE source Go et retourne ses violations. Extrait en
// fonction pure (source → violations) précisément pour que la preuve de morsure puisse
// lui soumettre du code SYNTHÉTIQUE : c'est le même code qui garde le dépôt et qui est
// prouvé mordant. Parsing en mode 0 → les commentaires ne sont pas dans l'AST.
func scanCrossTitleSource(rel string, src []byte) ([]crossTitleViolation, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rel, src, 0)
	if err != nil {
		return nil, err
	}
	var out []crossTitleViolation
	add := func(pos token.Pos, kind, detail string) {
		out = append(out, crossTitleViolation{
			File: rel, Kind: kind, Line: fset.Position(pos).Line, Detail: detail,
		})
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.RangeStmt:
			if detail, bad := crossTitleRangeOffender(node); bad {
				add(node.Pos(), kindRangeOverTitles, detail)
			}
		case *ast.FuncDecl:
			if isTitleCollectionName(node.Name.Name) && funcReturnsStringSlice(node.Type) {
				add(node.Pos(), kindMultiTitleList, "func "+node.Name.Name+" retourne []string")
			}
		case *ast.Ident:
			if reCrossTitleIdent.MatchString(node.Name) {
				add(node.Pos(), kindCrossTitleSymbol, node.Name)
			}
		case *ast.BasicLit:
			if node.Kind == token.STRING {
				if tbl, bad := sqlWithoutTitleFilter(node.Value); bad {
					add(node.Pos(), kindSQLWithoutTitle, "table per-titre "+tbl+" sans filtre title_slug")
				}
			}
		}
		return true
	})
	return out, nil
}

// crossTitleRangeOffender : la boucle itère-t-elle sur des titres ?
func crossTitleRangeOffender(r *ast.RangeStmt) (string, bool) {
	for _, e := range []ast.Expr{r.Key, r.Value} {
		if id, ok := e.(*ast.Ident); ok && reTitleLoopVar.MatchString(id.Name) {
			return "boucle sur la variable " + id.Name, true
		}
	}
	if name := rootIdentName(r.X); name != "" && isTitleCollectionName(name) {
		return "boucle sur l'expression " + name, true
	}
	return "", false
}

// rootIdentName extrait le nom « porteur » d'une expression itérée
// (`creditTitlesFor(c)` → creditTitlesFor ; `reg.AllTitles()` → AllTitles).
func rootIdentName(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.CallExpr:
		return rootIdentName(x.Fun)
	case *ast.SelectorExpr:
		return x.Sel.Name
	case *ast.IndexExpr:
		return rootIdentName(x.X)
	}
	return ""
}

// funcReturnsStringSlice indique si la signature renvoie au moins un []string —
// la forme canonique d'un producteur de liste de titres.
func funcReturnsStringSlice(ft *ast.FuncType) bool {
	if ft == nil || ft.Results == nil {
		return false
	}
	for _, res := range ft.Results.List {
		arr, ok := res.Type.(*ast.ArrayType)
		if !ok || arr.Len != nil {
			continue
		}
		if id, ok := arr.Elt.(*ast.Ident); ok && id.Name == "string" {
			return true
		}
	}
	return false
}

// sqlWithoutTitleFilter : littéral SQL touchant une table per-titre sans `title_slug`.
func sqlWithoutTitleFilter(lit string) (string, bool) {
	if strings.Contains(strings.ToLower(lit), "title_slug") {
		return "", false
	}
	for _, tbl := range crossTitlePerTitleTables {
		if crossTitleTableRes[tbl].MatchString(lit) {
			return tbl, true
		}
	}
	return "", false
}

// scanCrossTitleRoots scanne les sous-arbres surveillés (hors _test.go et testdata/).
func scanCrossTitleRoots(t *testing.T) []crossTitleViolation {
	t.Helper()
	root := findGoAPIModuleRoot(t)
	var all []crossTitleViolation
	for _, sub := range crossTitleScanRoots {
		dir := filepath.Join(root, filepath.FromSlash(sub))
		if _, err := os.Stat(dir); err != nil {
			t.Fatalf("racine de scan introuvable %q: %v", dir, err)
		}
		err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() {
				if info.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			src, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			vs, parseErr := scanCrossTitleSource(filepath.ToSlash(rel), src)
			if parseErr != nil {
				return parseErr
			}
			all = append(all, vs...)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %q: %v", sub, err)
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].File != all[j].File {
			return all[i].File < all[j].File
		}
		return all[i].Line < all[j].Line
	})
	return all
}

// findGoAPIModuleRoot remonte jusqu'au go.mod du module (apps/go-api). Les chemins
// d'allowlist sont relatifs à cette racine (`internal/...`).
func findGoAPIModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 10; i++ {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("racine de module (go.mod) introuvable")
	return ""
}

// TestNoCrossTitleAggregation — RATCHET PRINCIPAL. Toute agrégation cross-titre
// introduite dans prestige/ ou progression/ fait échouer la CI.
func TestNoCrossTitleAggregation(t *testing.T) {
	var unexpected []string
	for _, v := range scanCrossTitleRoots(t) {
		if _, allowed := allowlistCrossTitle[v.key()]; allowed {
			continue
		}
		unexpected = append(unexpected, v.String())
	}
	if len(unexpected) > 0 {
		t.Errorf("COUPLAGE CROSS-TITRE détecté (%d) — arcs, défis et points de progression sont "+
			"STRICTEMENT indépendants par titre (décision produit 2026-07-18) :\n  - %s",
			len(unexpected), strings.Join(unexpected, "\n  - "))
		t.Log("Corriger le calcul pour qu'il soit scopé à UN title_slug. Si l'agrégation est " +
			"volontaire et arbitrée, ajouter une entrée DATÉE et JUSTIFIÉE à allowlistCrossTitle.")
	}
}

// TestCrossTitleAllowlistNotStale — anti-pourrissement : une entrée d'allowlist qui ne
// correspond plus à aucune violation protège du code disparu et doit être retirée.
func TestCrossTitleAllowlistNotStale(t *testing.T) {
	seen := make(map[string]bool)
	for _, v := range scanCrossTitleRoots(t) {
		seen[v.key()] = true
	}
	for key, reason := range allowlistCrossTitle {
		if !seen[key] {
			t.Errorf("allowlistCrossTitle obsolète : %q ne correspond à aucune violation — "+
				"retirer l'entrée (raison historique : %q)", key, reason)
		}
	}
}

// ── Preuve de morsure ────────────────────────────────────────────────────────────

// crossTitleFixtureRemovedSeam reproduit LITTÉRALEMENT le code retiré le 2026-07-25 :
// `creditTitlesFor` (internal/prestige/cross_title.go) et sa boucle appelante
// (internal/prestige/service_evaluate.go, creditCompletion). Si ce motif revient, le
// ratchet DOIT le voir.
const crossTitleFixtureRemovedSeam = `package prestige

func creditTitlesFor(c Challenge) []string {
	return []string{c.TitleSlug}
}

func creditCompletion(s *service, ctx context.Context, c Challenge, pp int) {
	for _, slug := range creditTitlesFor(c) {
		ev := PrestigeEvent{UserID: c.UserID, TitleSlug: slug, PPAmount: pp}
		_ = s.deps.Prestige.EmitEvent(ctx, ev)
	}
}
`

const crossTitleFixtureRegistryLoop = `package prestige

func sumOverAllTitles(reg TitleRegistry, userID string) int {
	total := 0
	for _, titleSlug := range reg.AllTitles() {
		total += ppFor(userID, titleSlug)
	}
	return total
}
`

const crossTitleFixtureCrossSymbol = `package prestige

func total(r PrestigeRepo, ctx context.Context, userID string) UserPrestige {
	up, _ := r.GetUserPrestigeCrossTitle(ctx, userID)
	return up
}
`

const crossTitleFixtureUnscopedSQL = `package prestige

const totalPP = "SELECT COALESCE(SUM(total_pp), 0) FROM user_prestige_latest WHERE user_id = ?"
`

// crossTitleFixtureMonoTitle : l'équivalent CONFORME de tous les cas ci-dessus (dont la
// forme réellement livrée par V721-09 dans creditCompletion). Doit être TOTALEMENT muet.
const crossTitleFixtureMonoTitle = `package prestige

func creditCompletion(s *service, ctx context.Context, c Challenge, pp int) {
	ev := PrestigeEvent{UserID: c.UserID, TitleSlug: c.TitleSlug, PPAmount: pp}
	_ = s.deps.Prestige.EmitEvent(ctx, ev)
}

const totalPP = "SELECT COALESCE(SUM(total_pp), 0) FROM user_prestige_latest WHERE user_id = ? AND title_slug = ?"

func listForOneTitle(r ArcRepo, ctx context.Context, userID, titleSlug string) []Arc {
	arcs, _ := r.ListByUser(ctx, userID, titleSlug)
	return arcs
}

func challengesForTitleSlug(all []Challenge, titleSlug string) []Challenge {
	var out []Challenge
	for _, c := range all {
		if c.TitleSlug == titleSlug {
			out = append(out, c)
		}
	}
	return out
}
`

// TestCrossTitleDetectorsBite — PREUVE D'ANTI-NO-OP. Chaque détecteur est confronté au
// motif interdit correspondant et DOIT le signaler. Sans ce test, un ratchet vert ne
// prouverait rien (il pourrait ne rien détecter du tout).
func TestCrossTitleDetectorsBite(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "seam creditTitlesFor retiré le 2026-07-25 (code exact)",
			src:  crossTitleFixtureRemovedSeam,
			want: []string{kindRangeOverTitles, kindMultiTitleList},
		},
		{
			name: "boucle de sommation sur le registre des titres",
			src:  crossTitleFixtureRegistryLoop,
			want: []string{kindRangeOverTitles},
		},
		{
			name: "appel d'un symbole CrossTitle",
			src:  crossTitleFixtureCrossSymbol,
			want: []string{kindCrossTitleSymbol},
		},
		{
			name: "SQL sur table per-titre sans filtre title_slug",
			src:  crossTitleFixtureUnscopedSQL,
			want: []string{kindSQLWithoutTitle},
		},
	}
	for _, tc := range cases {
		got, err := scanCrossTitleSource("fixture.go", []byte(tc.src))
		if err != nil {
			t.Fatalf("%s: parse fixture: %v", tc.name, err)
		}
		kinds := make(map[string]bool, len(got))
		for _, v := range got {
			kinds[v.Kind] = true
		}
		for _, want := range tc.want {
			if !kinds[want] {
				t.Errorf("MORSURE ABSENTE — %s : le détecteur %q n'a rien vu (violations: %v)",
					tc.name, want, got)
			}
		}
	}
}

// TestCrossTitleDetectorsSpareMonoTitle — contrôle négatif : le code mono-titre
// conforme (dont la forme livrée par V721-09) ne doit produire AUCUNE violation, sans
// quoi le ratchet serait ingérable et finirait désactivé.
func TestCrossTitleDetectorsSpareMonoTitle(t *testing.T) {
	got, err := scanCrossTitleSource("fixture_mono.go", []byte(crossTitleFixtureMonoTitle))
	if err != nil {
		t.Fatalf("parse fixture mono-titre: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("FAUX POSITIF : le code mono-titre conforme déclenche %d violation(s) : %v", len(got), got)
	}
}
