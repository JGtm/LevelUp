//go:build cgo

// shared_openapi_doc_test.go — garde-rail de FIDÉLITÉ du document OpenAPI PARTAGÉ
// (chantier V72-01 / H1).
//
// Prouve que le document fusionné (un SEUL *huma.OpenAPI, cf. NewSharedConfig)
// contient un path pour CHAQUE route Huma enregistrée, avec son CHEMIN ABSOLU
// (préfixe de montage chi inclus), et que ce document colle au routeur chi RÉEL
// (chi.Walk = vérité terrain). C'est ce test qui remplace la crainte du « document
// partiel » : une route Huma qui oublierait WithSharedDoc apparaîtrait dans chi
// mais PAS dans le document → détectée en (4).
//
// Trois familles de routes chi coexistent :
//   - routes Huma PARTAGÉES (le sous-ensemble vérifié ici) ;
//   - routes chi-brut connues, INTENTIONNELLEMENT hors document (assets binaires,
//     média multipart, export CSV, redirects OAuth, import OpenSpartan,
//     /static, /debug/vars, SPA catch-all) — exclusions documentées en (4).
//     NB : groups.go est MIGRÉ vers Huma (V72-01 / H5) → ses 7 routes sont
//     désormais dans le document partagé, plus dans les exclusions chi-brut ;
//   - artefacts du mux (CONNECT/TRACE/OPTIONS/HEAD) — ignorés.

package api_test

import (
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
)

// docOp identifie une opération par (méthode HTTP majuscule, chemin absolu brut).
type docOp struct {
	method string
	path   string
}

// chiRoutesRawSet parcourt le routeur chi assemblé et retourne l'ensemble des
// routes avec leur chemin ABSOLU BRUT (paramètres conservés, ex. {player_slug}) —
// contrairement à chiRoutes (contract_test) qui normalise les params en {*}.
func chiRoutesRawSet(h http.Handler) map[docOp]bool {
	set := map[docOp]bool{}
	r, ok := h.(chi.Router)
	if !ok {
		return set
	}
	_ = chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		set[docOp{strings.ToUpper(method), route}] = true
		return nil
	})
	return set
}

// isKnownChiBrut : vrai si le chemin absolu désigne une route chi NON migrée vers
// Huma, intentionnellement hors du document OpenAPI partagé. Toute AUTRE route chi
// absente du document trahirait une route Huma ayant oublié WithSharedDoc.
func isKnownChiBrut(path string) bool {
	brutPrefixes := []string{
		"/static",                    // FileServer statiques (images, médailles…)
		"/debug/vars",                // expvar stdlib (admin-only, hors contrat REST)
		"/auth/xbox/",                // redirects OAuth racine
		"/api/v1/auth/xbox/",         // alias OAuth /api/v1
		"/api/v1/import/openspartan", // import multipart .db OpenSpartan
		"/api/v1/assets/",            // images binaires (medals/maps/battlepass/challenge-badge/spartan)
	}
	for _, p := range brutPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	// Routes chi-brut sous /players/{player_slug} (média multipart + export CSV).
	brutSuffixes := []string{"/media/upload", "/media/files/*", "/pages/match-history/export"}
	for _, s := range brutSuffixes {
		if strings.HasSuffix(path, s) {
			return true
		}
	}
	return path == "/*" // SPA catch-all
}

// TestSharedOpenAPIDocCoversAllHumaRoutes est le garde-rail central de H1.
func TestSharedOpenAPIDocCoversAllHumaRoutes(t *testing.T) {
	// Flags ON pour exposer le MAXIMUM de routes (multi-titre + prestige) — même
	// surface que contract_test et que le contrat publié (V721-04 : en démo, les
	// routes prestige / catalog / diag auto-sync sont montées avec des dépendances
	// de repli au lieu de disparaître).
	t.Setenv("LEVELUP_DEMO_MODE", "true")
	t.Setenv("PRESTIGE_ENABLED", "true")

	// Capture chaque enregistrement Huma (méthode, chemin ABSOLU) via le hook.
	humaOps := map[docOp]bool{}
	humacore.OnOperationRegistered = func(method, path string) {
		humaOps[docOp{method, path}] = true
	}
	defer func() { humacore.OnOperationRegistered = nil }()

	router, doc := buildTestRouterWithDoc(t)
	if doc == nil {
		t.Fatal("document OpenAPI partagé nil — NewRouter n'a pas retourné le doc")
	}
	if len(humaOps) < 100 {
		t.Fatalf("routes Huma capturées = %d (attendu >= 100) — hook ou partage du document cassés ?", len(humaOps))
	}

	// (1) FIDÉLITÉ : chaque opération Huma figure dans le document PARTAGÉ avec son
	// chemin ABSOLU (préfixe de montage inclus).
	for op := range humaOps {
		pi := doc.Paths[op.path]
		if pi == nil {
			t.Errorf("document : chemin ABSENT %s (op %s) — préfixe non appliqué au document ?", op.path, op.method)
			continue
		}
		if pathItemOperations(pi)[op.method] == nil {
			t.Errorf("document : chemin %s présent mais méthode %s absente", op.path, op.method)
		}
	}

	// (2) ROUTAGE : chaque opération Huma apparaît dans le routeur chi RÉEL avec le
	// même chemin absolu (le document colle au routage — chi.Walk = vérité terrain).
	chiSet := chiRoutesRawSet(router)
	for op := range humaOps {
		if !chiSet[op] {
			t.Errorf("op Huma %s %s absente de chi.Walk — chemin documenté ≠ chemin routé", op.method, op.path)
		}
	}

	// (3) PAS DE FANTÔME : chaque (chemin, méthode) du document correspond à une op
	// Huma réellement enregistrée (le document ne contient QUE des routes montées).
	for path, pi := range doc.Paths {
		for method := range pathItemOperations(pi) {
			if !humaOps[docOp{method, path}] {
				t.Errorf("document : opération FANTÔME %s %s (aucun enregistrement Huma correspondant)", method, path)
			}
		}
	}

	// (4) EXCLUSIONS DOCUMENTÉES : toute route chi qui n'est PAS une op Huma
	// partagée DOIT être une route chi-brut connue. Une route inconnue = une route
	// Huma ayant oublié WithSharedDoc (document partiel) → régression.
	var unexpected []string
	for op := range chiSet {
		switch op.method {
		case "CONNECT", "TRACE", "OPTIONS", "HEAD":
			continue // artefacts du mux (expvar, FileServer)
		}
		if humaOps[op] || isKnownChiBrut(op.path) {
			continue
		}
		unexpected = append(unexpected, op.method+" "+op.path)
	}
	if len(unexpected) > 0 {
		sort.Strings(unexpected)
		t.Errorf("%d route(s) chi ni documentées (Huma partagé) ni chi-brut connues — "+
			"probable oubli de WithSharedDoc (document partiel) :\n  %s",
			len(unexpected), strings.Join(unexpected, "\n  "))
	}

	t.Logf("document partagé : %d ops Huma fusionnées, %d paths ; chi total %d routes",
		len(humaOps), len(doc.Paths), len(chiSet))
}
