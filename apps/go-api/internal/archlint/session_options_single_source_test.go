// Package archlint — session_options_single_source_test.go : garde-rail de la RÈGLE DE
// DÉCOUPAGE EN SESSIONS (CLAUDE.md règle 6).
//
// session_id / session_label s'écrivent par plusieurs chemins (synchro, recalcul complet admin,
// backfill d'enrichissement, import OpenSpartan) et se relisent à la volée (GET /pages/sessions).
// Une seule source construit leurs options : settings.SessionOptions
// (internal/platform/settings/session_options.go), qui part d'analysis.DefaultSessionOptions et
// applique le réglage résolu du titre. Un chemin qui repartirait des défauts ignorerait le
// réglage utilisateur et découperait les mêmes matchs autrement que les autres.
//
// Deux contrôles, sur le code hors `_test.go` de internal/ et cmd/ :
//  1. aucun appel à DefaultSessionOptions hors du paquet analysis et du helper ;
//  2. tout fichier qui construit un SyncEngine (NewSyncEngine / NewSyncEngineForTitle) lui
//     branche la règle (WithSessionOptionsLoader), sauf les entrées datées de
//     moteursSansRegleAutorises.
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

// engineConstructorsFile : la définition des constructeurs du moteur.
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

func TestSessionOptionsSingleSource(t *testing.T) {
	sources := goSourcesUnder(t, "internal", "cmd")

	var appelsDefauts []string
	for _, src := range sources {
		if src.Rel == sessionOptionsHelper || strings.HasPrefix(src.Rel, sessionOptionsAnalysisPkg) {
			continue
		}
		appelsDefauts = append(appelsDefauts, appelsNommes(src, "DefaultSessionOptions")...)
	}
	sort.Strings(appelsDefauts)
	if len(appelsDefauts) > 0 {
		t.Errorf("options de session construites hors de %s — appeler settings.SessionOptions "+
			"(ou SessionOptionsLoader) :\n  %s", sessionOptionsHelper, strings.Join(appelsDefauts, "\n  "))
	}

	trouvees := map[string]bool{}
	var nonCables []string
	for _, src := range sources {
		if src.Rel == engineConstructorsFile {
			continue
		}
		construit := len(appelsNommes(src, "NewSyncEngine")) + len(appelsNommes(src, "NewSyncEngineForTitle"))
		if construit == 0 {
			continue
		}
		if _, autorise := moteursSansRegleAutorises[src.Rel]; autorise {
			trouvees[src.Rel] = true
			continue
		}
		if len(appelsNommes(src, "WithSessionOptionsLoader")) == 0 {
			nonCables = append(nonCables, src.Rel)
		}
	}
	sort.Strings(nonCables)
	if len(nonCables) > 0 {
		t.Errorf("SyncEngine construit sans règle de session (WithSessionOptionsLoader) — l'étape "+
			"sessions ignorerait le réglage :\n  %s", strings.Join(nonCables, "\n  "))
	}
	for rel := range moteursSansRegleAutorises {
		if !trouvees[rel] {
			t.Errorf("entrée autorisée sans construction de moteur : %s — la retirer de moteursSansRegleAutorises", rel)
		}
	}
}

// TestSessionOptionsSingleSource_Detecte : auto-preuve — un appel qualifié ou nu est vu.
func TestSessionOptionsSingleSource_Detecte(t *testing.T) {
	src := parseGoSnippet(t, "internal/sync/x.go", `package sync
func f() { _ = analysis.DefaultSessionOptions(); _ = DefaultSessionOptions() }`)
	if got := appelsNommes(src, "DefaultSessionOptions"); len(got) != 2 {
		t.Fatalf("appels détectés = %v, attendu 2", got)
	}
}

// appelsNommes rend « fichier:ligne » de chaque appel à une fonction de ce nom, qualifiée
// (pkg.Nom(...), recv.Nom(...)) ou nue (Nom(...)).
func appelsNommes(src goSource, nom string) []string {
	var out []string
	ast.Inspect(src.File, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		var appele string
		switch fn := call.Fun.(type) {
		case *ast.SelectorExpr:
			appele = fn.Sel.Name
		case *ast.Ident:
			appele = fn.Name
		}
		if appele == nom {
			out = append(out, src.Rel+":"+strconv.Itoa(src.Fset.Position(call.Pos()).Line))
		}
		return true
	})
	return out
}
