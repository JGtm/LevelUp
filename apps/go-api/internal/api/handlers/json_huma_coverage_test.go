package handlers_test

// json_huma_coverage_test.go — garde-fou : AUCUNE route JSON ne contourne Huma.
//
// Invariant de la migration Phase 3b (toutes les routes JSON sont sur Huma) :
// un handler JSON émet sa réponse via une struct Output Huma (sérialisée par
// humacore.JSONFormat), JAMAIS via writeJSON/writeJSONCached. Ces deux helpers ne
// subsistent que pour (a) leur propre définition + l'usage interne de writeError,
// et (b) les 2 routes à ENTRÉE multipart laissées en chi par conception (l'upload
// de fichier est leur caractéristique définissante ; la réponse JSON est incidente).
//
// Ce test échoue dès qu'un nouvel appel writeJSON/writeJSONCached apparaît dans le
// package handlers hors allowlist → il force soit la migration du handler vers Huma
// (Output struct), soit l'ajout explicite à l'allowlist AVEC justification (route à
// entrée multipart/binaire). Il empêche donc toute régression « partielle » vers du
// JSON servi en chi. writeJSON étant non-exporté, seul le package handlers peut
// l'appeler : scanner ce répertoire suffit.

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// jsonBypassAllowlist : fichiers du package handlers où writeJSON/writeJSONCached
// est toléré, chacun avec sa raison. Toute autre occurrence fait échouer le test.
var jsonBypassAllowlist = map[string]string{
	"helpers.go":            "définitions de writeJSON/writeJSONCached + usage interne de writeError (modèle d'erreur, pas une route)",
	"media_upload.go":       "POST /media/upload — entrée multipart/form-data (reste chi), réponse JSON incidente",
	"openspartan_import.go": "POST /import/openspartan — entrée multipart (reste chi), réponse JSON incidente",
	"groups.go":             "endpoints /groups (gestion familles, feat multititre) montés chi-natif sous RequireAuth avec path params {id}/{xuid} + flux d'invitation Xbox SSO ; livrés chi-style avec leur propre suite (groups_test.go). Réponses JSON incidentes. TODO(expiry:2026-10-01) : migration Huma non triviale (path params + flux invitation) — planifier ou re-dater à échéance.",
}

var writeJSONCallRe = regexp.MustCompile(`\bwriteJSON(Cached)?\(`)

func TestNoJSONRouteBypassesHuma(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(thisFile) // internal/api/handlers/

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("lecture du répertoire handlers : %v", err)
	}

	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("lecture %s : %v", name, err)
		}
		_, allowed := jsonBypassAllowlist[name]
		for i, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
				continue // commentaire
			}
			if strings.HasPrefix(trimmed, "func writeJSON") {
				continue // définition du helper
			}
			if writeJSONCallRe.MatchString(line) && !allowed {
				t.Errorf("%s:%d appelle writeJSON/writeJSONCached — toute route JSON DOIT passer par Huma "+
					"(struct Output, sérialisée par humacore.JSONFormat), jamais writeJSON.\n"+
					"  → migrer le handler vers Huma, OU (si entrée multipart/binaire) ajouter %q à "+
					"jsonBypassAllowlist avec justification.\n  %s", name, i+1, name, trimmed)
			}
		}
	}
}
