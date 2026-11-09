package archlint

// deployment_single_resolver_test.go — garde-rail du profil de déploiement (ADR 0044 D-4 et D-6,
// lot L10.4).
//
// POURQUOI. Trois réglages décrivaient « quelle instance tourne » et chaque lecteur en
// redérivait un sens : `AppConfig.DemoMode`, `AppConfig.AuthMode` et `LEVELUP_ENV`. Le profil
// de déploiement (internal/config/deployment.go) les résout UNE fois et publie des CAPACITÉS ;
// un consommateur branche sur une capacité, jamais sur le nom du profil (même règle que
// `slug == "..."` pour les titres, no_slug_comparison_test.go).
//
// DEUX RÈGLES.
//
//  1. Aucune NOUVELLE lecture de `.DemoMode`, `.AuthMode` ni du nom `LEVELUP_ENV` hors du
//     résolveur. Les lectures existantes sont comptées par fichier dans [lecturesProfilAdmises]
//     (relevé du 2026-11-01) : un fichier au-dessus de son compte est une nouvelle lecture
//     (échec) ; un fichier au-dessous, ou absent, est une entrée périmée (échec aussi) — le compte
//     est gelé et ne fait que descendre, à mesure que le code touché passe aux capacités.
//  2. Aucun code hors du résolveur ne COMPARE une valeur `config.Deployment` : ni `==`, ni `!=`,
//     ni `switch`/`case` sur le profil ou sur l'une des constantes `config.Deployment*`. Indexer
//     une table par profil (taskPolicies) n'est pas une comparaison.
//
// MAINTENANCE. Une lecture qui entre dans le résolveur sort de la liste (abaisser le compte).
// Ne jamais relever un compte : lire une capacité de AppConfig.Capabilities, ou en ajouter une.

import (
	"go/ast"
	"go/token"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// fichierResolveurProfil : le résolveur, seul fichier exempt des deux règles.
const fichierResolveurProfil = "internal/config/deployment.go"

// racinesLecturesProfil : le code non-test d'apps/go-api que les deux règles couvrent.
var racinesLecturesProfil = []string{"internal", "cmd", "pkg", "tools", "scripts"}

// lectureProfil compte, pour un fichier, les lectures des trois réglages d'avant le profil.
type lectureProfil struct {
	demoMode, authMode, levelupEnv int
}

// lecturesProfilAdmises : relevé daté du 2026-11-01, mesuré par ce garde-rail à sa création
// (`.DemoMode` et `.AuthMode` : sélecteurs ; `LEVELUP_ENV` : littéral de chaîne ; commentaires
// ignorés : 83 + 52 + 3 lectures dans 23 fichiers, le décompte plus large de l'ADR 0044 compte
// aussi commentaires et tests). Compte gelé : il ne fait que descendre (règle 1 de l'en-tête).
var lecturesProfilAdmises = map[string]lectureProfil{
	"cmd/server/background_tasks.go":                {demoMode: 1},
	"cmd/server/boot_config.go":                     {demoMode: 3},
	"cmd/server/boot_migrations.go":                 {demoMode: 5},
	"cmd/server/boot_providers.go":                  {demoMode: 1, authMode: 2},
	"cmd/server/demo_paths.go":                      {demoMode: 1},
	"cmd/server/security_warnings.go":               {levelupEnv: 1},
	"internal/api/handlers/settings.go":             {demoMode: 2, authMode: 1},
	"internal/api/handlers/setup.go":                {demoMode: 1, authMode: 1},
	"internal/api/openapi_docs.go":                  {demoMode: 2},
	"internal/api/requester_resolver.go":            {demoMode: 1, authMode: 1},
	"internal/api/server.go":                        {demoMode: 4, authMode: 2},
	"internal/api/server_apiv1.go":                  {demoMode: 42, authMode: 38},
	"internal/api/server_player_directory.go":       {demoMode: 1},
	"internal/api/wire/og_inject.go":                {demoMode: 1},
	"internal/api/wire/registry_auth.go":            {demoMode: 2},
	"internal/api/wire/registry_pages_home.go":      {demoMode: 1},
	"internal/api/wire/registry_replay_service.go":  {demoMode: 2},
	"internal/api/wire/server_titles_additional.go": {demoMode: 1},
	"internal/config/config.go":                     {authMode: 1, levelupEnv: 2},
	"internal/config/config_auth.go":                {authMode: 1},
	"internal/config/config_demo.go":                {demoMode: 1},
	"internal/service/bootstrap_ownership.go":       {demoMode: 2, authMode: 2},
	"internal/service/bootstrap_service.go":         {demoMode: 9, authMode: 3},
}

var nomLevelupEnv = regexp.MustCompile(`\bLEVELUP_ENV\b`)

// lecturesProfilDe compte les trois lectures dans un fichier analysé.
func lecturesProfilDe(src goSource) lectureProfil {
	var n lectureProfil
	ast.Inspect(src.File, func(node ast.Node) bool {
		switch x := node.(type) {
		case *ast.SelectorExpr:
			switch x.Sel.Name {
			case "DemoMode":
				n.demoMode++
			case "AuthMode":
				n.authMode++
			}
		case *ast.BasicLit:
			if x.Kind == token.STRING && nomLevelupEnv.MatchString(x.Value) {
				n.levelupEnv++
			}
		}
		return true
	})
	return n
}

// lecturesProfilProblemes compare les comptes mesurés à l'allowlist.
func lecturesProfilProblemes(mesure map[string]lectureProfil, admis map[string]lectureProfil) []string {
	var out []string
	for rel, got := range mesure {
		want := admis[rel]
		if got.demoMode > want.demoMode || got.authMode > want.authMode || got.levelupEnv > want.levelupEnv {
			out = append(out, rel+" : nouvelle lecture de .DemoMode / .AuthMode / LEVELUP_ENV hors du résolveur "+
				"(mesuré demo="+strconv.Itoa(got.demoMode)+" auth="+strconv.Itoa(got.authMode)+" env="+strconv.Itoa(got.levelupEnv)+
				", admis demo="+strconv.Itoa(want.demoMode)+" auth="+strconv.Itoa(want.authMode)+" env="+strconv.Itoa(want.levelupEnv)+
				") — lire une capacité de AppConfig.Capabilities (ADR 0044 D-4)")
		}
	}
	for rel, want := range admis {
		got := mesure[rel]
		if got.demoMode < want.demoMode || got.authMode < want.authMode || got.levelupEnv < want.levelupEnv {
			out = append(out, rel+" : entrée périmée de l'allowlist (admis demo="+strconv.Itoa(want.demoMode)+
				" auth="+strconv.Itoa(want.authMode)+" env="+strconv.Itoa(want.levelupEnv)+", mesuré demo="+strconv.Itoa(got.demoMode)+
				" auth="+strconv.Itoa(got.authMode)+" env="+strconv.Itoa(got.levelupEnv)+") — abaisser le compte, il ne fait que descendre")
		}
	}
	sort.Strings(out)
	return out
}

func TestDeploymentSingleResolver_LecturesDesTroisReglages(t *testing.T) {
	mesure := map[string]lectureProfil{}
	for _, src := range goSourcesUnder(t, racinesLecturesProfil...) {
		if src.Rel == fichierResolveurProfil {
			continue
		}
		if n := lecturesProfilDe(src); n != (lectureProfil{}) {
			mesure[src.Rel] = n
		}
	}
	for _, p := range lecturesProfilProblemes(mesure, lecturesProfilAdmises) {
		t.Error(p)
	}
}

// profilCompare dit si le nœud compare une valeur de profil : opérande `x.Deployment` ou
// `config.Deployment<Nom>` d'un `==` / `!=`, ou d'un `case`, ou tag d'un `switch`.
func nomDeProfil(e ast.Expr) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name == "Deployment" {
		return true
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "config" && strings.HasPrefix(sel.Sel.Name, "Deployment") && sel.Sel.Name != "Deployments"
}

// comparaisonsDeProfil rend les positions des comparaisons de profil d'un fichier.
func comparaisonsDeProfil(src goSource) []string {
	var out []string
	pos := func(p token.Pos) string { return src.Rel + ":" + strconv.Itoa(src.Fset.Position(p).Line) }
	ast.Inspect(src.File, func(node ast.Node) bool {
		switch x := node.(type) {
		case *ast.BinaryExpr:
			if (x.Op == token.EQL || x.Op == token.NEQ) && (nomDeProfil(x.X) || nomDeProfil(x.Y)) {
				out = append(out, pos(x.Pos()))
			}
		case *ast.SwitchStmt:
			if x.Tag != nil && nomDeProfil(x.Tag) {
				out = append(out, pos(x.Pos()))
			}
		case *ast.CaseClause:
			for _, e := range x.List {
				if nomDeProfil(e) {
					out = append(out, pos(e.Pos()))
				}
			}
		}
		return true
	})
	return out
}

func TestDeploymentSingleResolver_AucuneComparaisonDeProfil(t *testing.T) {
	var violations []string
	for _, src := range goSourcesUnder(t, racinesLecturesProfil...) {
		if src.Rel == fichierResolveurProfil {
			continue
		}
		violations = append(violations, comparaisonsDeProfil(src)...)
	}
	for _, v := range violations {
		t.Errorf("%s : comparaison d'une valeur config.Deployment hors du résolveur — brancher sur une "+
			"capacité de AppConfig.Capabilities (ADR 0044 D-4)", v)
	}
}

// Auto-preuve : le garde-rail voit une lecture nouvelle, une entrée périmée et une comparaison.
func TestDeploymentSingleResolver_AutoPreuve(t *testing.T) {
	lecture := parseGoSnippet(t, "internal/x/x.go", `package x
func f(cfg *C) bool { _ = "LEVELUP_ENV"; return cfg.DemoMode || cfg.AuthMode == "xbox" }`)
	if got := lecturesProfilDe(lecture); got != (lectureProfil{demoMode: 1, authMode: 1, levelupEnv: 1}) {
		t.Errorf("lectures = %+v, attendu 1/1/1", got)
	}
	comme := parseGoSnippet(t, "internal/x/x.go", `package x
func f(cfg *C) bool {
	switch cfg.Deployment {
	case config.DeploymentDemo:
	}
	return cfg.Deployment == config.DeploymentDesktop || cfg.Capabilities.UpdateApp
}`)
	if got := comparaisonsDeProfil(comme); len(got) != 3 {
		t.Errorf("comparaisons = %v, attendu 3 (switch, case, ==)", got)
	}
	indexe := parseGoSnippet(t, "internal/x/x.go", `package x
var t = map[config.Deployment]int{config.DeploymentDemo: 1}
func f(cfg *C) int { return t[cfg.Deployment] }`)
	if got := comparaisonsDeProfil(indexe); len(got) != 0 {
		t.Errorf("une table indexée par profil n'est pas une comparaison : %v", got)
	}
	nouvelle := lecturesProfilProblemes(map[string]lectureProfil{"a.go": {demoMode: 2}}, map[string]lectureProfil{"a.go": {demoMode: 1}})
	perimee := lecturesProfilProblemes(map[string]lectureProfil{"a.go": {demoMode: 1}}, map[string]lectureProfil{"a.go": {demoMode: 2}})
	if len(nouvelle) != 1 || len(perimee) != 1 {
		t.Errorf("nouvelle=%v périmée=%v, attendu une entrée chacune", nouvelle, perimee)
	}
}
