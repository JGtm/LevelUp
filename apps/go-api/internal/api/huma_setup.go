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
	"encoding/json"
	"io"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
)

// humaAPIError reproduit le contrat d'erreur de handlers.writeError pour les
// routes migrées vers Huma : corps JSON {code, message, retryable}, message
// générique « internal error » sur 5xx (pas de fuite d'info interne). Implémente
// huma.StatusError → Huma le sérialise tel quel, pour les erreurs retournées par
// les handlers ET les erreurs de validation Huma (via huma.NewError override).
type humaAPIError struct {
	status    int
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

func (e *humaAPIError) Error() string  { return e.Message }
func (e *humaAPIError) GetStatus() int { return e.status }

// newHumaError construit une erreur Huma au contrat writeError avec un code
// explicite (équivalent de writeError(ctx, w, status, code, message)).
func newHumaError(status int, code, message string) huma.StatusError {
	clientMessage := message
	if status >= http.StatusInternalServerError {
		clientMessage = "internal error"
	}
	return &humaAPIError{status: status, Code: code, Message: clientMessage, Retryable: status >= http.StatusInternalServerError}
}

// errorCodeForStatus mappe un status HTTP vers un code stable pour les erreurs
// générées par Huma lui-même (validation d'input). Les handlers passent leur
// propre code via newHumaError.
func errorCodeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusUnprocessableEntity:
		return "validation_error"
	default:
		if status >= http.StatusInternalServerError {
			return "internal_error"
		}
		return "error"
	}
}

// humaJSONFormat reproduit EXACTEMENT la sérialisation de handlers.writeJSON pour
// que le corps des routes migrées soit byte-identique à celui des routes chi :
//   - sanitizeFloatsForJSON en amont (NaN/Inf → 0 ou pointeur nil) : sinon
//     json.Marshal échoue sur un float NaN et Huma renvoie 500 là où writeJSON
//     renvoyait 200 sanitisé ;
//   - json.Marshal (HTML-escaping activé, comme writeJSON) — PAS json.Encoder
//     qui désactive l'escaping dans le format Huma par défaut ;
//   - trailing "\n" (writeJSON l'ajoute explicitement).
//
// Couvre TOUTES les routes migrées en un point unique (les ~100 routes à floats
// n'ont donc aucune sanitisation par-route à recâbler).
var humaJSONFormat = huma.Format{
	Marshal: func(w io.Writer, v any) error {
		sanitized, _ := handlers.SanitizeFloatsForJSON(v)
		body, err := json.Marshal(sanitized)
		if err != nil {
			return err
		}
		if _, err := w.Write(body); err != nil {
			return err
		}
		_, err = w.Write([]byte("\n"))
		return err
	},
	Unmarshal: json.Unmarshal,
}

// newHumaAPI crée l'API Huma adossée au routeur chi existant. Les routes
// enregistrées via huma.Register / huma.Get apparaissent sur le même routeur chi
// (humachi → chiMux.MethodFunc), donc visibles à chi.Walk et au contract_test.
//
// OpenAPIPath neutre (/_internal/openapi-huma) et DocsPath vide : on ne sert ni
// /openapi.yaml ni /docs côté Huma tant que le YAML manuel est la source de vérité.
func newHumaAPI(r chi.Router) huma.API {
	config := huma.DefaultConfig("LevelUp API", "1.0.0")
	// Sérialisation byte-identique à writeJSON (NaN-safe + HTML-escaping + \n) sur
	// les deux clés de format JSON utilisées par Huma (content-negotiation).
	config.Formats = map[string]huma.Format{
		"application/json": humaJSONFormat,
		"json":             humaJSONFormat,
	}
	// Huma n'enregistre AUCUNE route auto (spec OpenAPI / docs / schemas) sur le
	// routeur chi : ces paths vides désactivent leur enregistrement (huma api.go
	// 507/557/561). Sinon `contract_test` (route chi ⟷ openapi.yaml) verrait ces
	// routes internes comme « non documentées ». La source de vérité contractuelle
	// reste l'openapi.yaml MANUEL tant que la migration n'est pas finie.
	config.OpenAPIPath = ""
	config.DocsPath = ""
	config.SchemasPath = ""
	// Pas de $schema ni de header Link dans les réponses : Huma injecte par défaut
	// un champ `$schema` (via le SchemaLinkTransformer monté par CreateHooks). On le
	// désactive pour que le corps JSON des routes migrées reste IDENTIQUE à writeJSON
	// (le contrat front + l'openapi.yaml manuel ne connaissent pas `$schema`).
	config.CreateHooks = nil
	// Contrat d'erreur identique à writeError : les erreurs de validation générées
	// par Huma (input) sortent en {code, message, retryable}. huma.NewError est un
	// var package-level (une seule API dans ce process) ; override idempotent.
	huma.NewError = func(status int, msg string, _ ...error) huma.StatusError {
		return newHumaError(status, errorCodeForStatus(status), msg)
	}
	return humachi.New(r, config)
}
