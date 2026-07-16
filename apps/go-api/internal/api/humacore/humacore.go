// Package humacore — socle partagé de la migration Huma (Phase 3b).
//
// Contient l'infrastructure réutilisable par TOUTES les routes migrées, quel que
// soit leur package d'enregistrement :
//   - le package api (routes inline de server.go) ;
//   - le package handlers (handlers qui s'auto-enregistrent, ex. ProgressionHandler.Mount).
//
// humacore n'importe AUCUN package du projet (seulement huma/humachi/chi + stdlib)
// pour rester sans cycle : handlers et api l'importent tous les deux.
//
// Garanties contractuelles (identiques à handlers.writeJSON / writeError) :
//   - corps JSON byte-identique à writeJSON : SanitizeFloatsForJSON (NaN/Inf
//     neutralisés) → json.Marshal (HTML-escaping) → trailing "\n" ;
//   - format d'erreur {code, message, retryable} avec « internal error » générique
//     sur status >= 500 (pas de fuite d'info interne) ;
//   - PAS de champ $schema injecté (CreateHooks nil) ;
//   - aucune route OpenAPI/docs/schemas auto-enregistrée sur chi (paths vides) tant
//     que l'openapi.yaml manuel reste la source de vérité contractuelle.
package humacore

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"reflect"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

// ---------------------------------------------------------------------------
// Sanitisation NaN/Inf (déplacée depuis handlers.sanitizeFloatsForJSON).
// ---------------------------------------------------------------------------

// SanitizeFloatsForJSON parcourt v via reflect et neutralise toute valeur
// float64/float32 NaN ou +/-Inf (non représentable en JSON).
//
// Pour *float64/*float32 NaN/Inf, le pointeur est mis à nil (omitempty disparaît).
// Pour float64/float32 NaN/Inf direct, la valeur est mise à 0.
// Retourne (sanitized, paths) — paths = chemins des champs neutralisés (log).
func SanitizeFloatsForJSON(v interface{}) (interface{}, []string) {
	if v == nil {
		return nil, nil
	}
	rv := reflect.ValueOf(v)
	var paths []string
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return v, nil
		}
		walkSanitize(rv, "", &paths)
		return v, paths
	}
	ptr := reflect.New(rv.Type())
	ptr.Elem().Set(rv)
	walkSanitize(ptr, "", &paths)
	return ptr.Elem().Interface(), paths
}

func walkSanitize(v reflect.Value, path string, paths *[]string) {
	if !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return
		}
		elem := v.Elem()
		if elem.Kind() == reflect.Float64 || elem.Kind() == reflect.Float32 {
			if math.IsNaN(elem.Float()) || math.IsInf(elem.Float(), 0) {
				if v.CanSet() {
					v.Set(reflect.Zero(v.Type())) // pointer → nil
					*paths = append(*paths, path+"*")
				}
			}
			return
		}
		walkSanitize(elem, path, paths)
	case reflect.Interface:
		if v.IsNil() {
			return
		}
		walkSanitize(v.Elem(), path, paths)
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if !f.CanSet() {
				continue // champ non-exporté → ignorer
			}
			walkSanitize(f, path+"."+t.Field(i).Name, paths)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			walkSanitize(v.Index(i), path+"["+strconv.Itoa(i)+"]", paths)
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			val := v.MapIndex(k)
			keyStr := fmt.Sprintf("%v", k.Interface())
			subPath := path + "[" + keyStr + "]"
			if val.Kind() == reflect.Float64 || val.Kind() == reflect.Float32 {
				if math.IsNaN(val.Float()) || math.IsInf(val.Float(), 0) {
					v.SetMapIndex(k, reflect.Zero(val.Type()))
					*paths = append(*paths, subPath)
				}
				continue
			}
			ptr := reflect.New(val.Type())
			ptr.Elem().Set(val)
			walkSanitize(ptr, subPath, paths)
			v.SetMapIndex(k, ptr.Elem())
		}
	case reflect.Float64, reflect.Float32:
		if v.CanSet() && (math.IsNaN(v.Float()) || math.IsInf(v.Float(), 0)) {
			v.SetFloat(0)
			*paths = append(*paths, path)
		}
	}
}

// ---------------------------------------------------------------------------
// Format JSON byte-identique à handlers.writeJSON.
// ---------------------------------------------------------------------------

// JSONFormat reproduit EXACTEMENT la sérialisation de handlers.writeJSON :
// sanitisation NaN/Inf en amont, json.Marshal (HTML-escaping activé, comme
// writeJSON — pas json.Encoder qui le désactive), trailing "\n".
var JSONFormat = huma.Format{
	Marshal: func(w io.Writer, v any) error {
		sanitized, _ := SanitizeFloatsForJSON(v)
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

// ---------------------------------------------------------------------------
// Modèle d'erreur (identique à handlers.writeError).
// ---------------------------------------------------------------------------

// apiError reproduit le contrat d'erreur de handlers.writeError : corps JSON
// {code, message, retryable}, message générique « internal error » sur 5xx.
// Implémente huma.StatusError → Huma le sérialise tel quel (erreurs handler ET
// erreurs de validation Huma via NewError override).
type apiError struct {
	status    int
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

func (e *apiError) Error() string  { return e.Message }
func (e *apiError) GetStatus() int { return e.status }

// NewError construit une erreur Huma au contrat writeError (équivalent de
// writeError(ctx, w, status, code, message)).
func NewError(status int, code, message string) huma.StatusError {
	clientMessage := message
	if status >= http.StatusInternalServerError {
		clientMessage = "internal error"
	}
	return &apiError{status: status, Code: code, Message: clientMessage, Retryable: status >= http.StatusInternalServerError}
}

// ErrorCodeForStatus mappe un status HTTP vers un code stable pour les erreurs
// générées par Huma lui-même (validation d'input). Les handlers passent leur
// propre code via NewError.
func ErrorCodeForStatus(status int) string {
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

// ---------------------------------------------------------------------------
// Factory d'API Huma coexistante avec chi.
// ---------------------------------------------------------------------------

// MarkRequestBodyOptional met RequestBody.Required = false sur l'opération
// (method, path) déjà enregistrée. À utiliser pour les routes à body OPTIONNEL
// déclarées avec un champ Input `RawBody []byte` — que Huma rend REQUIS par défaut
// (400 "request body is required" si corps absent). Cela préserve le 200 sur
// corps absent TOUT EN gardant le décodage maison (400 sur JSON malformé), contrat
// que ni RawBody seul (corps requis) ni un Body typé pointeur (422 sur malformé)
// ne reproduisent. Le pointeur d'opération est partagé entre l'OpenAPI et le
// handler runtime (huma AddOperation/Handle), donc la mutation prend effet au runtime.
func MarkRequestBodyOptional(api huma.API, method, path string) {
	pi, ok := api.OpenAPI().Paths[path]
	if !ok || pi == nil {
		return
	}
	var op *huma.Operation
	switch method {
	case http.MethodGet:
		op = pi.Get
	case http.MethodPost:
		op = pi.Post
	case http.MethodPut:
		op = pi.Put
	case http.MethodPatch:
		op = pi.Patch
	case http.MethodDelete:
		op = pi.Delete
	}
	if op != nil && op.RequestBody != nil {
		op.RequestBody.Required = false
	}
}

// OnAPICreated, s'il est non-nil, est invoqué avec chaque huma.API créée par
// NewAPI. Nil par défaut → ZÉRO impact prod (le serveur ne le branche jamais).
// SEUL l'outil de drift-detection des schémas (test cgo, openapi_schema_drift_test)
// le branche temporairement pour capturer les instances Huma : leurs registres de
// schémas auto-dérivés (api.OpenAPI().Components.Schemas) sont autrement créés
// localement dans chaque Mount() puis jetés. Le brancher en prod retiendrait des
// références API → garder nil hors test.
var OnAPICreated func(huma.API)

// OnAPICreatedRouter, s'il est non-nil, est invoqué avec le routeur chi ET la
// huma.API de CHAQUE NewAPI. Nil par défaut → ZÉRO impact prod (le serveur ne le
// branche jamais).
//
// SEUL le garde-rail anti-collision de routes (route_collision_test) le branche
// temporairement. Il groupe les opérations Huma par IDENTITÉ DE ROUTEUR (pointeur
// du *chi.Mux passé à NewAPI) : deux enregistrements du même (méthode, chemin) sur
// le même routeur = collision par écrasement silencieux de chi. Un chi.Walk de
// l'arbre final NE PEUT PAS détecter ce cas (chi remplace l'endpoint, une seule
// visite au walk) — d'où la capture au moment de l'enregistrement plutôt qu'au
// parcours.
var OnAPICreatedRouter func(chi.Router, huma.API)

// NewAPI crée une API Huma adossée au routeur chi `r` (qui peut être le routeur
// racine OU un sous-routeur d'un r.Route/r.Group — les routes Huma héritent alors
// du middleware du sous-groupe et lisent les path params parents, cf.
// TestHumaNestedSubrouterProbe).
//
// Config : format byte-identique writeJSON, pas de $schema, aucune route auto
// (OpenAPI/docs/schemas) — l'openapi.yaml MANUEL reste source de vérité.
func NewAPI(r chi.Router) huma.API {
	config := huma.DefaultConfig("LevelUp API", "1.0.0")
	config.Formats = map[string]huma.Format{
		"application/json": JSONFormat,
		"json":             JSONFormat,
	}
	config.OpenAPIPath = ""
	config.DocsPath = ""
	config.SchemasPath = ""
	config.CreateHooks = nil
	// huma.NewError est un var package-level (une seule instance par process) ;
	// override idempotent (même valeur quelle que soit l'API).
	huma.NewError = func(status int, msg string, _ ...error) huma.StatusError {
		return NewError(status, ErrorCodeForStatus(status), msg)
	}
	api := humachi.New(r, config)
	if OnAPICreated != nil {
		OnAPICreated(api)
	}
	if OnAPICreatedRouter != nil {
		OnAPICreatedRouter(r, api)
	}
	return api
}
