// Package archlint — session_options_single_source_test.go : garde-rail de la RÈGLE DE
// DÉCOUPAGE EN SESSIONS (CLAUDE.md règle 6).
//
// session_id / session_label s'écrivent par plusieurs chemins (synchro, recalcul complet admin,
// backfill d'enrichissement, import OpenSpartan) et se relisent à la volée (GET /pages/sessions).
// Une seule source construit leurs options : settings.SessionOptions
// (internal/platform/settings/session_options.go), qui part d'analysis.DefaultSessionOptions et
// applique le réglage résolu du titre. Un chemin qui repartirait des défauts, ou qui écrirait ses
// options à la main, ignorerait le réglage utilisateur et découperait les mêmes matchs autrement.
//
// Contrôles, sur le code hors `_test.go` de internal/ et cmd/ :
//  1. aucun appel à DefaultSessionOptions ni littéral SessionComputeOptions{…} hors du paquet
//     analysis et du helper ;
//  2. tout fichier qui construit un SyncEngine (NewSyncEngine / NewSyncEngineForTitle) lui
//     branche la règle (WithSessionOptionsLoader ou WithSessionOptionsFromConfig), sauf les
//     entrées datées de moteursSansRegleAutorises ; jamais avec l'argument nil ;
//  3. WithSessionOptionsLoader n'est appelé que par les sites qui tiennent le Store de réglages
//     du serveur (sitesAvecStoreDeReglages) ; ailleurs, WithSessionOptionsFromConfig, qui prend
//     le titre du moteur lui-même.
package archlint

import (
	"go/ast"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// sessionOptionsHelper : la source unique des options de session.
const sessionOptionsHelper = "internal/platform/settings/session_options.go"

// sessionOptionsAnalysisPkg : le paquet qui définit les défauts.
const sessionOptionsAnalysisPkg = "internal/analysis/"

// engineConstructorsFile : la définition des constructeurs du moteur et de ses setters.
const engineConstructorsFile = "internal/sync/engine_options.go"

// moteursSansRegleAutorises : fichiers qui construisent un SyncEngine sans lui brancher la règle
// de session, datés et justifiés. Le moteur applique alors la règle par défaut et le journalise
// (WARN) à chaque étape sessions. Une entrée devenue sans construction doit sortir.
//
//   - internal/sync/trigger.go (2026-10-21) : chemin legacy SANS fabrique de moteur, emprunté
//     seulement par des tests ; en production le Trigger reçoit scheduler.BuildEngine, câblé.
//   - cmd/backfill_all/main.go (2026-10-21) : outil de rattrapage des PSA seul, qui n'exécute
//     jamais l'étape sessions et ne charge pas de configuration.
var moteursSansRegleAutorises = map[string]string{
	"internal/sync/trigger.go": "legacy sans fabrique, tests seulement (2026-10-21)",
	"cmd/backfill_all/main.go": "rattrapage PSA seul, sans configuration (2026-10-21)",
}

// sitesAvecStoreDeReglages : les seuls fichiers qui appellent WithSessionOptionsLoader, avec le
// Store de réglages injecté au serveur (celui que modifie PATCH /settings).
var sitesAvecStoreDeReglages = map[string]bool{
	engineConstructorsFile:                    true, // WithSessionOptionsFromConfig s'y appuie
	"internal/scheduler/auto_sync_engine.go":  true,
	"cmd/server/sync_v2_wiring.go":            true,
	"internal/api/handlers/sync_handler.go":   true,
	"internal/api/wire/registry_lusr_gaps.go": true,
}

func TestSessionOptionsSingleSource(t *testing.T) {
	sources := goSourcesUnder(t, "internal", "cmd")

	var horsSource []string
	for _, src := range sources {
		if src.Rel == sessionOptionsHelper || strings.HasPrefix(src.Rel, sessionOptionsAnalysisPkg) {
			continue
		}
		horsSource = append(horsSource, appelsNommes(src, "DefaultSessionOptions")...)
		horsSource = append(horsSource, litterauxDeType(src, "SessionComputeOptions")...)
	}
	sort.Strings(horsSource)
	if len(horsSource) > 0 {
		t.Errorf("options de session construites hors de %s — appeler settings.SessionOptions "+
			"(ou SessionOptionsLoader / SessionOptionsForConfig) :\n  %s",
			sessionOptionsHelper, strings.Join(horsSource, "\n  "))
	}

	verifierCablageDesMoteurs(t, sources)
}

// verifierCablageDesMoteurs : contrôles 2 et 3.
func verifierCablageDesMoteurs(t *testing.T, sources []goSource) {
	t.Helper()
	trouvees := map[string]bool{}
	var nonCables, avecNil, horsStore []string
	for _, src := range sources {
		loaders := appelsNommes(src, "WithSessionOptionsLoader")
		fromConfig := appelsNommes(src, "WithSessionOptionsFromConfig")
		avecNil = append(avecNil, appelsAvecArgumentNil(src, "WithSessionOptionsLoader", "WithSessionOptionsFromConfig")...)
		if len(loaders) > 0 && !sitesAvecStoreDeReglages[src.Rel] {
			horsStore = append(horsStore, loaders...)
		}
		if src.Rel == engineConstructorsFile {
			continue
		}
		if len(appelsNommes(src, "NewSyncEngine"))+len(appelsNommes(src, "NewSyncEngineForTitle")) == 0 {
			continue
		}
		if _, autorise := moteursSansRegleAutorises[src.Rel]; autorise {
			trouvees[src.Rel] = true
			continue
		}
		if len(loaders)+len(fromConfig) == 0 {
			nonCables = append(nonCables, src.Rel)
		}
	}
	signaler(t, nonCables, "SyncEngine construit sans règle de session (WithSessionOptionsFromConfig) — "+
		"l'étape sessions ignorerait le réglage")
	signaler(t, avecNil, "règle de session branchée à nil — le moteur retomberait sur les défauts")
	signaler(t, horsStore, "WithSessionOptionsLoader hors d'un site qui tient le Store de réglages — "+
		"utiliser WithSessionOptionsFromConfig(cfg), qui prend le titre du moteur")
	for rel := range moteursSansRegleAutorises {
		if !trouvees[rel] {
			t.Errorf("entrée autorisée sans construction de moteur : %s — la retirer de moteursSansRegleAutorises", rel)
		}
	}
	for rel := range sitesAvecStoreDeReglages {
		if !fichierPresent(sources, rel) {
			t.Errorf("site avec Store introuvable : %s — le retirer de sitesAvecStoreDeReglages", rel)
		}
	}
}

func signaler(t *testing.T, sites []string, message string) {
	t.Helper()
	if len(sites) == 0 {
		return
	}
	sort.Strings(sites)
	t.Errorf("%s :\n  %s", message, strings.Join(sites, "\n  "))
}

func fichierPresent(sources []goSource, rel string) bool {
	for _, src := range sources {
		if src.Rel == rel {
			return true
		}
	}
	return false
}

// TestSessionOptionsSingleSource_Detecte : auto-preuve — appels qualifiés ou nus, littéraux
// qualifiés ou nus, et argument nil sont vus.
func TestSessionOptionsSingleSource_Detecte(t *testing.T) {
	src := parseGoSnippet(t, "internal/sync/x.go", `package sync
func f() {
	_ = analysis.DefaultSessionOptions(); _ = DefaultSessionOptions()
	_ = domain.SessionComputeOptions{GapMinutes: 15}; _ = SessionComputeOptions{}
	e.WithSessionOptionsLoader(nil); e.WithSessionOptionsFromConfig(nil); e.WithSessionOptionsLoader(rule)
}`)
	if got := appelsNommes(src, "DefaultSessionOptions"); len(got) != 2 {
		t.Errorf("appels détectés = %v, attendu 2", got)
	}
	if got := litterauxDeType(src, "SessionComputeOptions"); len(got) != 2 {
		t.Errorf("littéraux détectés = %v, attendu 2", got)
	}
	if got := appelsAvecArgumentNil(src, "WithSessionOptionsLoader", "WithSessionOptionsFromConfig"); len(got) != 2 {
		t.Errorf("arguments nil détectés = %v, attendu 2", got)
	}
}

// appelsNommes rend « fichier:ligne » de chaque appel à une fonction de ce nom, qualifiée
// (pkg.Nom(...), recv.Nom(...)) ou nue (Nom(...)).
func appelsNommes(src goSource, nom string) []string {
	var out []string
	ast.Inspect(src.File, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok && nomAppele(call.Fun) == nom {
			out = append(out, positionAppel(src, call))
		}
		return true
	})
	return out
}

// appelsAvecArgumentNil rend les appels à l'une de ces fonctions dont un argument est `nil`.
func appelsAvecArgumentNil(src goSource, noms ...string) []string {
	var out []string
	ast.Inspect(src.File, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		appele := nomAppele(call.Fun)
		for _, nom := range noms {
			if appele != nom {
				continue
			}
			for _, arg := range call.Args {
				if id, isIdent := arg.(*ast.Ident); isIdent && id.Name == "nil" {
					out = append(out, positionAppel(src, call))
				}
			}
		}
		return true
	})
	return out
}

// litterauxDeType rend les littéraux composites `pkg.Type{…}` ou `Type{…}` de ce nom de type.
func litterauxDeType(src goSource, typeName string) []string {
	var out []string
	ast.Inspect(src.File, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		var nom string
		switch ty := lit.Type.(type) {
		case *ast.SelectorExpr:
			nom = ty.Sel.Name
		case *ast.Ident:
			nom = ty.Name
		}
		if nom == typeName {
			out = append(out, src.Rel+":"+strconv.Itoa(src.Fset.Position(lit.Pos()).Line))
		}
		return true
	})
	return out
}

// positionAppel rend « fichier:ligne » d'un appel.
func positionAppel(src goSource, call *ast.CallExpr) string {
	return src.Rel + ":" + strconv.Itoa(src.Fset.Position(call.Pos()).Line)
}
