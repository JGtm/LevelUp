//go:build cgo

// body_limit_guard_test.go — garde-rail du plafond de corps des opérations Huma.
//
// Parcourt le document OpenAPI PARTAGÉ du routeur assemblé (la même surface que
// shared_openapi_doc_test) et exige, pour CHAQUE opération :
//   - MaxBodyBytes > 0 : aucune lecture de corps sans limite (une entrée RawBody
//     n'a aucun défaut côté Huma ; humacore.DefaultMaxBodyBytes le comble) ;
//   - MaxBodyBytes <= humacore.DefaultMaxBodyBytes, sauf exception nommée dans
//     bodyLimitExceptions, chacune justifiée et chiffrée.

package api_test

import (
	"sort"
	"testing"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
)

// bodyLimitExceptions : opérations dont le plafond dépasse le défaut, par
// operationId. Ajouter une entrée exige de citer la constante du domaine qui fonde
// le plafond et la route nginx qui le laisse passer (packaging/nginx/levelup.conf).
var bodyLimitExceptions = map[string]int64{
	// Dépôt d'artefact de rejeu par l'ouvrier (jeton d'ouvrier requis) ; nginx :
	// location /api/v1/internal/ à 20m.
	"postBuildQueueArtifact": domain.MaxBuildArtifactBytes,
}

func TestEveryOperationHasBodyLimit(t *testing.T) {
	t.Setenv("LEVELUP_DEMO_MODE", "true")
	t.Setenv("PRESTIGE_ENABLED", "true")

	_, doc := buildTestRouterWithDoc(t)
	if doc == nil {
		t.Fatal("document OpenAPI partagé nil")
	}

	paths := make([]string, 0, len(doc.Paths))
	for p := range doc.Paths {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	seen := 0
	exceptionsSeen := map[string]bool{}
	for _, p := range paths {
		ops := pathItemOperations(doc.Paths[p])
		for method, op := range ops {
			seen++
			where := method + " " + p + " (" + op.OperationID + ")"
			if op.MaxBodyBytes <= 0 {
				t.Errorf("%s : MaxBodyBytes = %d — plafond de corps absent (enregistrement hors des fabriques humacore ?)", where, op.MaxBodyBytes)
				continue
			}
			if op.MaxBodyBytes <= humacore.DefaultMaxBodyBytes {
				continue
			}
			want, listed := bodyLimitExceptions[op.OperationID]
			switch {
			case !listed:
				t.Errorf("%s : plafond %d > défaut %d sans exception nommée dans bodyLimitExceptions", where, op.MaxBodyBytes, humacore.DefaultMaxBodyBytes)
			case op.MaxBodyBytes != want:
				t.Errorf("%s : plafond %d, l'exception déclare %d", where, op.MaxBodyBytes, want)
			default:
				exceptionsSeen[op.OperationID] = true
			}
		}
	}
	if seen < 100 {
		t.Fatalf("opérations parcourues = %d (attendu >= 100) — document partiel ?", seen)
	}
	for id := range bodyLimitExceptions {
		if !exceptionsSeen[id] {
			t.Errorf("exception %q déclarée mais absente du document (route retirée ou plafond abaissé : retirer l'entrée)", id)
		}
	}
}
