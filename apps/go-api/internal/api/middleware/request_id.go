// Package middleware fournit les middlewares HTTP transverses.
package middleware

import (
	"net/http"

	"github.com/google/uuid"

	"levelup/go-api/internal/ctxkeys"
)

const headerRequestID = "X-Request-ID"

// RequestID injecte un X-Request-ID unique dans chaque requête et réponse,
// ET le propage dans le ctx via ctxkeys.WithRequestID (P6.4, ADR 0009).
// Si le client envoie déjà un X-Request-ID, celui-ci est préservé.
//
// Permet de corréler la ligne d'accès middleware avec les `slog.*Context`
// émis par les services pour la même requête. Sans ça, debug prod cassé
// en multi-user.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(headerRequestID)
		if id == "" {
			id = uuid.New().String()
		}
		w.Header().Set(headerRequestID, id)
		// Propager dans le ctx pour les services downstream.
		ctx := ctxkeys.WithRequestID(r.Context(), id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
