package sharedprovider

import (
	"context"
	"time"
)

// readBudgetKeyType est une clé de contexte privée (évite les collisions).
type readBudgetKeyType struct{}

var readBudgetKey readBudgetKeyType

// WithSwapWaitBudget attache au contexte un budget MAX d'attente d'un swap RW→RO
// pour les Get() servis avec ce contexte. Il borne UNIQUEMENT le temps passé à
// attendre la fin d'un swap (sync tenant le writer RW) — jamais la durée
// d'exécution des requêtes elles-mêmes. Le handle *sql.DB retourné continue
// d'utiliser le contexte du caller pour ses requêtes.
//
// Usage : un middleware HTTP pose un budget court (~2-3s) sur toutes les requêtes
// user-facing, pour qu'une page échoue vite (503 Retry-After) au lieu de pendre
// jusqu'au readyTimeout (30s) quand un sync tient le writer. Les lectures du sync
// (in-process, sans contexte HTTP) n'ont pas ce budget → conservent le readyTimeout
// complet, robuste face aux swaps légitimes.
//
// budget <= 0 est ignoré (pas de plafonnement).
func WithSwapWaitBudget(ctx context.Context, budget time.Duration) context.Context {
	if budget <= 0 {
		return ctx
	}
	return context.WithValue(ctx, readBudgetKey, budget)
}

// swapWaitBudget lit le budget d'attente de swap posé sur le contexte, s'il existe.
func swapWaitBudget(ctx context.Context) (time.Duration, bool) {
	b, ok := ctx.Value(readBudgetKey).(time.Duration)
	if !ok || b <= 0 {
		return 0, false
	}
	return b, true
}
