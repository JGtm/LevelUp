package middleware

import (
	"net/http"
	"time"

	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// DefaultUserFacingReadBudget est le budget max d'attente d'un swap RW→RO de la
// base partagée pour une requête user-facing. Court volontairement : une page doit
// répondre vite — ou échouer proprement en 503 Retry-After — plutôt que pendre
// jusqu'au readyTimeout du provider (30s) quand un sync tient le writer RW.
const DefaultUserFacingReadBudget = 3 * time.Second

// UserFacingReadBudget pose sur le contexte de la requête un budget max d'attente
// d'un swap de la base partagée (cf. sharedprovider.WithSwapWaitBudget). Il borne
// UNIQUEMENT l'attente d'un swap en cours, JAMAIS l'exécution des requêtes (une
// page lourde mais légitime va au bout). Les lectures du sync (in-process, hors
// chaîne HTTP) ne traversent pas ce middleware → conservent le readyTimeout complet,
// robuste face aux swaps légitimes.
//
// budget <= 0 retombe sur DefaultUserFacingReadBudget.
func UserFacingReadBudget(budget time.Duration) func(http.Handler) http.Handler {
	if budget <= 0 {
		budget = DefaultUserFacingReadBudget
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := sharedprovider.WithSwapWaitBudget(r.Context(), budget)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
