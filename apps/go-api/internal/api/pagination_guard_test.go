//go:build cgo

// pagination_guard_test.go — garde-rail des paramètres de taille de l'API.
//
// Parcourt le document OpenAPI PARTAGÉ et traite tout paramètre de requête dont le
// nom désigne une taille (limit, page_size, count, n, *_limit) :
//   - entier : le schéma déclare un `maximum` (Huma refuse alors en 422 au-delà) ;
//   - chaîne : le paramètre est lu avec un parsing souple (valeur invalide ignorée,
//     pas de 422), donc le plafond vit dans le code. Il figure alors dans
//     paginationStringParams avec son plafond, la date d'entrée et le test qui
//     prouve que le plafond tient.
//
// Un nouveau paramètre de taille sans l'un ni l'autre fait échouer ce test.

package api_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/notifications"
)

// paginationStringParam : un paramètre de taille en chaîne, son plafond effectif et
// le test qui le prouve (Test, dans File, relatif à internal/api).
type paginationStringParam struct {
	Ceiling int
	Since   string
	File    string
	Test    string
}

// paginationStringParams est indexé par "operationId.paramètre".
var paginationStringParams = map[string]paginationStringParam{
	"getLeaderboardPage.limit":                  {200, "2026-10-15", "handlers/leaderboard_limit_test.go", "TestLeaderboardPage_LimitIsCappedServerSide"},
	"getAdminMonitoringDataQualityIssues.limit": {500, "2026-10-15", "handlers/pagination_ceiling_test.go", "TestAdminDQ_GetIssues_LimitCeiling"},
	"getAdminMonitoringJobs.limit":              {50, "2026-10-15", "handlers/pagination_ceiling_test.go", "TestAdminMonitoring_Jobs_LimitCeiling"},
	"getAdminMonitoringLogsTail.n":              {1000, "2026-10-15", "handlers/pagination_ceiling_test.go", "TestAdminLogsTail_NCeiling"},
	"getPlayerPatterns.n":                       {200, "2026-10-15", "handlers/pagination_ceiling_test.go", "TestPatterns_NCeiling"},
	"suggestPrestigeTemplates.count":            {10, "2026-10-15", "handlers/pagination_ceiling_test.go", "TestPrestige_SuggestTemplates_CountCeiling"},
	"getAdminMonitoringDetections.limit":        {500, "2026-10-15", "../ops/pagination_ceiling_cgo_test.go", "TestMonitoringStore_DetectionsLimitCeiling"},
	"getAdminMonitoringBuildQueue.limit":        {200, "2026-10-15", "../ops/pagination_ceiling_cgo_test.go", "TestMonitoringStore_BuildQueueLimitCeiling"},
}

// paginationIntegerMaxima : le `maximum` attendu des paramètres entiers connus.
var paginationIntegerMaxima = map[string]float64{
	"listNotifications.limit":              notifications.MaxListLimit,
	"listProgressionRecords.history_limit": 200,
}

func isPaginationName(name string) bool {
	switch name {
	case "limit", "page_size", "count", "n":
		return true
	}
	return strings.HasSuffix(name, "_limit")
}

func TestPaginationParamsAreBounded(t *testing.T) {
	t.Setenv("LEVELUP_DEMO_MODE", "true")
	t.Setenv("PRESTIGE_ENABLED", "true")

	_, doc := buildTestRouterWithDoc(t)
	if doc == nil {
		t.Fatal("document OpenAPI partagé nil")
	}

	seen := map[string]bool{}
	for path, pi := range doc.Paths {
		for method, op := range pathItemOperations(pi) {
			for _, prm := range op.Parameters {
				if prm.In != "query" || !isPaginationName(prm.Name) || prm.Schema == nil {
					continue
				}
				key := op.OperationID + "." + prm.Name
				seen[key] = true
				where := method + " " + path + " (" + key + ")"
				switch prm.Schema.Type {
				case "integer", "number":
					want, known := paginationIntegerMaxima[key]
					switch {
					case prm.Schema.Maximum == nil:
						t.Errorf("%s : paramètre de taille entier sans `maximum`", where)
					case known && *prm.Schema.Maximum != want:
						t.Errorf("%s : maximum = %v, attendu %v", where, *prm.Schema.Maximum, want)
					case !known:
						t.Errorf("%s : paramètre entier borné mais absent de paginationIntegerMaxima", where)
					}
				case "string":
					if _, listed := paginationStringParams[key]; !listed {
						t.Errorf("%s : paramètre de taille en chaîne absent de paginationStringParams (plafond + test à déclarer)", where)
					}
				default:
					t.Errorf("%s : type %q non pris en charge par le garde-rail", where, prm.Schema.Type)
				}
			}
		}
	}

	for key := range paginationIntegerMaxima {
		if !seen[key] {
			t.Errorf("paginationIntegerMaxima : %q n'existe plus dans le document (retirer l'entrée)", key)
		}
	}
	for key, e := range paginationStringParams {
		if !seen[key] {
			t.Errorf("paginationStringParams : %q n'existe plus dans le document (retirer l'entrée)", key)
		}
		if e.Ceiling <= 0 || e.Since == "" {
			t.Errorf("paginationStringParams[%q] : plafond et date d'entrée requis", key)
		}
		src, err := os.ReadFile(filepath.FromSlash(e.File))
		if err != nil {
			t.Errorf("paginationStringParams[%q] : fichier de test %s illisible : %v", key, e.File, err)
			continue
		}
		if !strings.Contains(string(src), "func "+e.Test+"(") {
			t.Errorf("paginationStringParams[%q] : %s ne définit pas %s", key, e.File, e.Test)
		}
	}
}
