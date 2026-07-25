// Package api — huma_setup.go : intégration Huma coexistante avec chi (Phase 3b).
//
// Huma génère l'OpenAPI depuis les types Input/Output des handlers. La migration
// des handlers chi vers Huma est PROGRESSIVE : humachi enveloppe le *chi.Mux
// EXISTANT, donc les routes chi non migrées et les routes Huma cohabitent sur le
// même routeur (humachi enregistre via chiMux.MethodFunc → routes visibles à
// chi.Walk, donc internal/api/contract_test.go reste valide pour toutes les
// routes, migrées ou non). Cf. .ai/PLAN_TITLE_AGNOSTIC_REFACTORING.md §Phase 3b.
//
// Le socle réutilisable (factory, modèle d'erreur, format byte-identique writeJSON,
// sanitisation NaN) vit dans internal/api/humacore — partagé avec le package
// handlers (handlers qui s'auto-enregistrent). Ici on ne garde que des alias
// minces pour les routes inline de ce package (api) et la rétro-compat des tests.
package api

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
)

// newHumaError : alias de humacore.NewError (contrat writeError {code, message,
// retryable}, « internal error » sur 5xx). Conservé pour les wrappers du package.
func newHumaError(status int, code, message string) huma.StatusError {
	return humacore.NewError(status, code, message)
}

// newHumaAPI : alias de humacore.NewAPI (API Huma coexistante sur le routeur chi
// `r`, racine ou sous-routeur). Les options (WithSharedDoc) sont transmises telles
// quelles pour partager le document OpenAPI. Voir humacore.NewAPI pour les garanties.
func newHumaAPI(r chi.Router, opts ...humacore.MountOption) huma.API {
	return humacore.NewAPI(r, opts...)
}
