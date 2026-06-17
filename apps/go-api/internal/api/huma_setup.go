// Package api — huma_setup.go : intégration Huma coexistante avec chi (Phase 3b).
//
// Huma génère l'OpenAPI depuis les types Input/Output des handlers. La migration
// des ~79 handlers chi vers Huma est PROGRESSIVE : humachi enveloppe le *chi.Mux
// EXISTANT, donc les routes chi non migrées et les routes Huma cohabitent sur le
// même routeur (humachi enregistre via chiMux.MethodFunc → routes visibles à
// chi.Walk, donc internal/api/contract_test.go reste valide pour toutes les
// routes, migrées ou non). Cf. .ai/PLAN_TITLE_AGNOSTIC_REFACTORING.md §Phase 3b.
//
// Étape phase-3b-start : pose l'API Huma coexistante SANS migrer aucun handler.
// L'OpenAPI Huma n'est PAS exposé sur /openapi.yaml ni /docs — le YAML manuel
// (api/openapi.yaml) reste la source de vérité contractuelle tant que la
// migration n'est pas terminée (sinon collision + régénération du client front).
package api

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

// newHumaAPI crée l'API Huma adossée au routeur chi existant. Les routes
// enregistrées via huma.Register / huma.Get apparaissent sur le même routeur chi
// (humachi → chiMux.MethodFunc), donc visibles à chi.Walk et au contract_test.
//
// OpenAPIPath neutre (/_internal/openapi-huma) et DocsPath vide : on ne sert ni
// /openapi.yaml ni /docs côté Huma tant que le YAML manuel est la source de vérité.
func newHumaAPI(r chi.Router) huma.API {
	config := huma.DefaultConfig("LevelUp API", "1.0.0")
	config.OpenAPIPath = "/_internal/openapi-huma"
	config.DocsPath = ""
	return humachi.New(r, config)
}
