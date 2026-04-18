// Package handlers contient les handlers HTTP des endpoints P0.
package handlers

import (
	"encoding/json"
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

// extractKillsFromLabel extrait un compteur de kills depuis un label textuel.
// Placeholder — retourne toujours 0 en attendant le format de label stabilisé.
func extractKillsFromLabel(_ string) int {
	return 0
}

// writeError écrit une réponse d'erreur JSON standardisée.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]interface{}{
		"code":      code,
		"message":   message,
		"retryable": status >= 500,
	})
}
