// Package handlers contient les handlers HTTP des endpoints P0.
package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
)

// writeJSON sérialise v en JSON et l'écrit dans w avec le code HTTP donné.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Le header est déjà envoyé, on ne peut pas changer le status.
		// On log silencieusement — la connexion est peut-être déjà fermée.
		_ = err
	}
}

// writeJSONCached sérialise v, pose un ETag SHA-256 et retourne 304 si le client est à jour.
// À utiliser sur les endpoints GET dont les données changent peu entre deux syncs.
func writeJSONCached(w http.ResponseWriter, r *http.Request, status int, v interface{}) {
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
