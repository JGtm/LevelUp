// Package handlers contient les handlers HTTP des endpoints P0.
package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"reflect"
	"strconv"
)

// Clés JSON partagées entre handlers (params actor logs, response error envelopes).
// Externalisées pour éviter les doublons goconst dans les helpers et middlewares.
const (
	jsonKeyStatus    = "status"
	jsonKeyCount     = "count"
	jsonKeyCode      = "code"
	jsonKeyMessage   = "message"
	jsonKeyRetryable = "retryable"

	// Valeurs courantes
	jsonBoolTrueStr = "true"

	// Codes d'erreur partagés (MSAL device-code flow).
	errCodeMSALAcquire = "msal_acquire_error"

	// Modes auth.
	authModeXbox = "xbox"
)

// sanitizeFloatsForJSON parcourt v via reflect et neutralise toute valeur
// float64/float32 NaN ou +/-Inf (non représentable en JSON).
//
// Retourne (sanitized, paths) :
//   - sanitized = la valeur nettoyée (peut être différente de v si v était une
//     struct passée par valeur — voir note ci-dessous).
//   - paths = liste des chemins de champs neutralisés (ex. ".Radar.Axes[2]"),
//     utile pour logger la source des NaN sans patcher chaque builder.
//
// Note addressability : si v est une struct/slice/array passée par valeur,
// reflect.ValueOf(v) est non-addressable → CanSet()=false → le walk ne peut
// rien modifier en place. On copie alors dans un *T addressable via
// reflect.New et on retourne ptr.Elem().Interface() (struct nettoyée).
// Pour un pointeur, on modifie en place et on retourne v inchangé.
//
// Pour *float64/*float32 NaN/Inf, le pointeur est mis à nil (omitempty disparait).
// Pour float64/float32 NaN/Inf direct, la valeur est mise à 0.
//
// Sécurité défensive : on a déjà nettoyé en aval (cf. sanitizeF64 dans
// match_view_repo.go) mais cette passe finale couvre tous les chemins futurs
// (Q17, Q22, Q18, agrégats, narrative, etc.) sans avoir à patcher chaque scan.
func sanitizeFloatsForJSON(v interface{}) (interface{}, []string) {
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
		// Cas spécial : *float64 / *float32 → nil le pointeur si NaN/Inf.
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
		// Les valeurs de map ne sont pas addressables. On gère deux cas :
		//   1. map[K]float : on remplace directement la valeur via SetMapIndex.
		//   2. map[K]Struct/Slice/Map : on recopie la valeur dans un *T
		//      addressable, on walk dedans, et on remet la copie sanitisée.
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

// writeJSON sérialise v en JSON et l'écrit dans w avec le code HTTP donné.
// Utilise json.Marshal (buffer complet) avant d'écrire les headers : si la
// sérialisation échoue (ex. float64 NaN/Inf), renvoie 500 avec un corps JSON
// valide au lieu d'un 200 avec corps vide (ancien comportement silencieux).
// Passe sanitizeFloatsForJSON en amont pour neutraliser les NaN/Inf ; chaque
// neutralisation est loggée avec son path de struct pour identifier la source.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	sanitized, nanPaths := sanitizeFloatsForJSON(v)
	if len(nanPaths) > 0 {
		slog.Warn("writeJSON: NaN/Inf neutralized", "paths", nanPaths, "count", len(nanPaths))
	}
	body, err := json.Marshal(sanitized)
	if err != nil {
		slog.Error("writeJSON: marshal failed", "err", err)
		// Réponse d'erreur inline — pas via writeError pour éviter la récursion.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code":"encode_error","message":"erreur de sérialisation","retryable":true}` + "\n"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
	_, _ = w.Write([]byte("\n"))
}

// writeJSONCached sérialise v, pose un ETag SHA-256 et retourne 304 si le client est à jour.
// À utiliser sur les endpoints GET dont les données changent peu entre deux syncs.
//
// tous les callers passent 200/OK mais la signature reste configurable.
//
//nolint:unparam // status paramétré pour cohérence avec writeJSON ; aujourd'hui
func writeJSONCached(w http.ResponseWriter, r *http.Request, status int, v interface{}) {
	sanitized, nanPaths := sanitizeFloatsForJSON(v)
	if len(nanPaths) > 0 {
		slog.WarnContext(r.Context(), "writeJSONCached: NaN/Inf neutralized", "paths", nanPaths, "count", len(nanPaths))
	}
	body, err := json.Marshal(sanitized)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "encode_error", "erreur de sérialisation")
		return
	}
	sum := sha256.Sum256(body)
	etag := fmt.Sprintf(`"%x"`, sum[:8])
	w.Header().Set("ETag", etag)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// writeError écrit une réponse d'erreur JSON standardisée et **logge l'erreur
// côté serveur** selon le statut HTTP :
//   - 5xx → slog.ErrorContext (erreur serveur, anormal — toujours visible)
//   - 401/403/422 → slog.WarnContext (auth/validation refusée — tracé pour audit)
//   - autres 4xx → slog.DebugContext (404 / bad request — volume, log à la demande)
//
// L'objectif est qu'aucune erreur renvoyée au client n'échappe à un trace serveur.
// Le ctx fait remonter event_id / request_id auto-injectés par ContextHandler.
func writeError(ctx context.Context, w http.ResponseWriter, status int, code, message string) {
	logErrorResponse(ctx, status, code, message, nil)
	writeJSON(w, status, map[string]interface{}{
		"code":      code,
		"message":   message,
		"retryable": status >= 500,
	})
}

// httpError est un wrapper de net/http.Error qui logge l'erreur côté serveur
// (mêmes règles de niveau que writeError) avant de répondre en text/plain.
//
// À utiliser pour les endpoints qui servent du contenu non-JSON (assets, fichiers
// statiques, redirects) — sinon préférer writeError qui renvoie un body JSON
// standardisé.
func httpError(ctx context.Context, w http.ResponseWriter, message string, status int) {
	logErrorResponse(ctx, status, "", message, nil)
	http.Error(w, message, status)
}

// logErrorResponse centralise la décision de niveau de log selon le statut HTTP.
// Séparé de writeError pour permettre la réutilisation (writeServerError, et
// futurs helpers d'erreur).
func logErrorResponse(ctx context.Context, status int, code, message string, err error) {
	attrs := []any{jsonKeyStatus, status, jsonKeyCode, code, jsonKeyMessage, message}
	if err != nil {
		attrs = append(attrs, "err", err)
	}
	switch {
	case status >= 500:
		slog.ErrorContext(ctx, "http: erreur réponse", attrs...)
	case status == http.StatusUnauthorized, status == http.StatusForbidden, status == http.StatusUnprocessableEntity:
		slog.WarnContext(ctx, "http: refus", attrs...)
	case status >= 400:
		slog.DebugContext(ctx, "http: client error", attrs...)
	}
}
