// Package middleware fournit les middlewares HTTP transverses.
package middleware

import (
	"net/http"

	"github.com/google/uuid"
)

const headerRequestID = "X-Request-ID"

// RequestID injecte un X-Request-ID unique dans chaque requête et réponse.
// Si le client envoie déjà un X-Request-ID, celui-ci est préservé.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(headerRequestID)
		if id == "" {
			id = uuid.New().String()
		}
		w.Header().Set(headerRequestID, id)
		next.ServeHTTP(w, r)
	})
}
