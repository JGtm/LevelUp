// Package handlers contient les handlers HTTP des endpoints P0.
package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"reflect"
)

// sanitizeFloatsForJSON parcourt v via reflect et neutralise toute valeur
// float64/float32 NaN ou +/-Inf (non représentable en JSON).
// Pour un *float64 NaN/Inf, le pointeur est mis à nil (champ omitempty disparait).
// Pour un float64 NaN/Inf direct, la valeur est mise à 0.
// Sécurité défensive : on a déjà nettoyé en aval (cf. sanitizeF64 dans
// match_view_repo.go) mais cette passe finale couvre tous les chemins futurs
// (Q17, Q22, Q18, agrégats, narrative, etc.) sans avoir à patcher chaque scan.
func sanitizeFloatsForJSON(v interface{}) {
	if v == nil {
		return
	}
	walkSanitize(reflect.ValueOf(v))
}

func walkSanitize(v reflect.Value) {
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
				}
			}
			return
		}
		walkSanitize(elem)
	case reflect.Interface:
		if v.IsNil() {
			return
		}
		walkSanitize(v.Elem())
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if !f.CanSet() {
				continue // champ non-exporté → ignorer
			}
			walkSanitize(f)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			walkSanitize(v.Index(i))
		}
	case reflect.Map:
		// Map values ne sont pas addressables. On itère et on remplace
		// uniquement les valeurs float problématiques.
		for _, k := range v.MapKeys() {
			val := v.MapIndex(k)
			if val.Kind() == reflect.Float64 || val.Kind() == reflect.Float32 {
				if math.IsNaN(val.Float()) || math.IsInf(val.Float(), 0) {
					v.SetMapIndex(k, reflect.Zero(val.Type()))
				}
			}
		}
	case reflect.Float64, reflect.Float32:
		if v.CanSet() && (math.IsNaN(v.Float()) || math.IsInf(v.Float(), 0)) {
			v.SetFloat(0)
		}
	}
}

// writeJSON sérialise v en JSON et l'écrit dans w avec le code HTTP donné.
// Utilise json.Marshal (buffer complet) avant d'écrire les headers : si la
// sérialisation échoue (ex. float64 NaN/Inf), renvoie 500 avec un corps JSON
// valide au lieu d'un 200 avec corps vide (ancien comportement silencieux).
// Passe sanitizeFloatsForJSON en amont pour neutraliser les NaN/Inf.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	sanitizeFloatsForJSON(v)
	body, err := json.Marshal(v)
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
func writeJSONCached(w http.ResponseWriter, r *http.Request, status int, v interface{}) {
	sanitizeFloatsForJSON(v)
	body, err := json.Marshal(v)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "encode_error", "erreur de sérialisation")
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

// writeError écrit une réponse d'erreur JSON standardisée.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]interface{}{
		"code":      code,
		"message":   message,
		"retryable": status >= 500,
	})
}
