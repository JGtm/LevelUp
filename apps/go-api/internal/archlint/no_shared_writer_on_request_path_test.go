// Package archlint — no_shared_writer_on_request_path_test.go : garde-rail « pas de writer
// partagé sur le chemin d'une requête HTTP » (plan sécurité du 2026-10-15, lot L5, C3-3).
//
// POURQUOI. Prendre le writer de la base partagée la bascule en écriture : les lecteurs de
// tout le site répondent 503 après 3 s. Une route /players qu'un compte peut rejouer en
// boucle (l'exclusion de match recalculait tout l'historique sous ce writer) devient un
// déni de service. Les écritures per-match passent par la file persist ; un calcul qui lit
// la base partagée la lit (Provider.Get / OpenReadForQuery) et n'écrit qu'en base joueur.
//
// CE QUI EST INTERDIT dans les paquets du chemin de requête (internal/api, internal/service)
// et dans les fichiers nommés de internal/sync qu'ils appellent : un appel direct à
// AcquireWriter, AcquireSharedWriterStandalone, AcquireMetadataWriterStandalone, ou aux
// aides qui les prennent pour le compte de l'appelant (RecomputeIsWithFriends,
// RecalculatePlayerSessions), HORS de l'allowlist ci-dessous. L'analyse suit les appels
// directs ; un appel indirect par un autre paquet n'est pas vu.
package archlint

import (
	"go/ast"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// writerAcquiringCalls — noms d'appels qui prennent un writer partagé (ou de metadata).
var writerAcquiringCalls = map[string]bool{
	"AcquireWriter":                   true,
	"AcquireSharedWriterStandalone":   true,
	"AcquireMetadataWriterStandalone": true,
	"RecomputeIsWithFriends":          true,
	"RecalculatePlayerSessions":       true,
}

// requestPathRoots — paquets dont le code tourne sous une requête HTTP.
var requestPathRoots = []string{"internal/api", "internal/service"}

// requestPathExtraFiles — fichiers de internal/sync appelés directement par une requête.
var requestPathExtraFiles = []string{"internal/sync/match_recomputer.go"}

// requestPathWriterAllowlist — fichier → nombre d'appels autorisés, avec la raison et la
// date. DÉCROISSANTE : une entrée dont le compte baisse doit être abaissée dans le même
// commit ; n'en ajouter qu'avec une justification datée.
var requestPathWriterAllowlist = map[string]struct {
	n      int
	reason string
}{
	// 2026-10-16 : routes d'administration (RequireAdmin) ou de protocole ouvrier (jeton
	// dédié), jamais /players ni anonymes.
	"internal/api/wire/registry_actions.go":           {1, "reset des lying bits : action admin (RequireAdmin), single-flight"},
	"internal/api/wire/registry_asset_names_sweep.go": {1, "balayage des noms d'assets : action admin, sérialisée"},
	"internal/api/wire/registry_build_queue.go":       {1, "protocole ouvrier de rejeu (jeton dédié RequireWorkerToken)"},
	"internal/api/handlers/settings.go":               {1, "PATCH /settings : recalcul des sessions, réglage d'administration"},
	// 2026-10-16 : import OpenSpartan, RequireAuth, un import à la fois par compte, fichier
	// validé avant la prise du writer (L5-4).
	"internal/service/openspartan_import_service.go":      {1, "import OpenSpartan : writer pris après validation du fichier"},
	"internal/service/openspartan_post_import_service.go": {6, "suite de l'import OpenSpartan : un import à la fois par compte"},
	// 2026-10-16 : DÉCOUVERTE D-L5-1, NON TRAITÉE ici (hors périmètre L5) : la liste d'amis
	// d'un profil déclenche le recalcul « avec amis » sous le writer partagé, route
	// /players/{slug}/friends réservée au propriétaire.
	"internal/service/friends_orchestrator_service.go": {2, "D-L5-1 : /players/{slug}/friends -> RecomputeIsWithFriends sous writer partagé (à traiter comme L5-3)"},
}

// writerCallSites rend, par fichier analysé, le nombre d'appels à une fonction de prise de writer.
func writerCallSites(src goSource) int {
	n := 0
	ast.Inspect(src.File, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			if writerAcquiringCalls[fn.Name] {
				n++
			}
		case *ast.SelectorExpr:
			if writerAcquiringCalls[fn.Sel.Name] {
				n++
			}
		}
		return true
	})
	return n
}

func requestPathWriterProblems(sources []goSource, allow map[string]struct {
	n      int
	reason string
}) []string {
	counts := map[string]int{}
	for _, src := range sources {
		if n := writerCallSites(src); n > 0 {
			counts[src.Rel] = n
		}
	}
	var out []string
	for rel, n := range counts {
		if limit := allow[rel].n; n > limit {
			out = append(out, rel+" : "+strconv.Itoa(n)+" prise(s) de writer, allowlist "+strconv.Itoa(limit))
		} else if n < limit {
			out = append(out, rel+" : "+strconv.Itoa(n)+" prise(s) < allowlist "+strconv.Itoa(limit)+" — ABAISSER l'entrée")
		}
	}
	for rel := range allow {
		if _, ok := counts[rel]; !ok {
			out = append(out, rel+" : plus aucune prise de writer — RETIRER l'entrée de l'allowlist")
		}
	}
	sort.Strings(out)
	return out
}

func TestNoSharedWriterOnRequestPath(t *testing.T) {
	sources := goSourcesUnder(t, requestPathRoots...)
	for _, f := range requestPathExtraFiles {
		dir := f[:strings.LastIndex(f, "/")]
		for _, src := range goSourcesUnder(t, dir) {
			if src.Rel == f {
				sources = append(sources, src)
			}
		}
	}
	if problems := requestPathWriterProblems(sources, requestPathWriterAllowlist); len(problems) > 0 {
		t.Errorf("prise du writer partagé sur le chemin d'une requête HTTP hors allowlist (bascule en écriture "+
			"de toute la base = 503 pour les lecteurs) : lire par Provider.Get / OpenReadForQuery et écrire en base "+
			"joueur, ou justifier l'entrée (date + raison) :\n  %s", strings.Join(problems, "\n  "))
	}
}

func TestRequestPathWriterGuardDetectsDefects(t *testing.T) {
	bad := parseGoSnippet(t, "internal/service/x.go", `package service
func f() { _, _, _ = AcquireSharedWriterStandalone(nil, nil, ""); p.AcquireWriter(nil) }`)
	if got := requestPathWriterProblems([]goSource{bad}, nil); len(got) != 1 {
		t.Fatalf("deux prises de writer non signalées : %v", got)
	}
	clean := parseGoSnippet(t, "internal/service/y.go", `package service
func f() { _, _, _ = acquireSharedReaderStandalone(nil, nil, "") }`)
	if got := requestPathWriterProblems([]goSource{clean}, nil); len(got) != 0 {
		t.Fatalf("faux positif : %v", got)
	}
}
