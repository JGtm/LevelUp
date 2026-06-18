package games

import "sync"

// Resolver d'issues (outcome) partagé, câblé une fois au boot (server.go) depuis
// la mappings.Registry — miroir du resolver d'endpoints. Vit ici (package games,
// bas niveau) pour être consultable par les repos platform/duckdb sans dépendance
// croisée. Nil tant que non câblé → les call-sites retombent sur leur littéral
// `outcome = N` legacy (byte-identique Halo).
var (
	defaultOutcomeMu       sync.RWMutex
	defaultOutcomeResolver OutcomeResolver
)

// SetDefaultOutcomeResolver câble le resolver d'issues partagé (appelé au boot).
func SetDefaultOutcomeResolver(r OutcomeResolver) {
	defaultOutcomeMu.Lock()
	defaultOutcomeResolver = r
	defaultOutcomeMu.Unlock()
}

// DefaultOutcomeResolver retourne le resolver d'issues partagé (nil tant que non câblé).
func DefaultOutcomeResolver() OutcomeResolver {
	defaultOutcomeMu.RLock()
	defer defaultOutcomeMu.RUnlock()
	return defaultOutcomeResolver
}
