package games

import "sync"

// Resolver d'hosts d'ingestion partagé, câblé une fois au boot (server.go) depuis
// la mappings.Registry. Vit ici (package games, bas niveau) pour être consultable
// par TOUTES les couches d'ingestion — internal/sync ET internal/platform/halo
// ET internal/assets — sans dépendance croisée. Nil tant que non câblé → les
// call-sites retombent sur leur const Halo legacy (byte-identique).
var (
	defaultEndpointMu       sync.RWMutex
	defaultEndpointResolver EndpointResolver
)

// SetDefaultEndpointResolver câble le resolver partagé (appelé au boot). Idempotent.
func SetDefaultEndpointResolver(r EndpointResolver) {
	defaultEndpointMu.Lock()
	defaultEndpointResolver = r
	defaultEndpointMu.Unlock()
}

// DefaultEndpointResolver retourne le resolver partagé (nil tant que non câblé).
func DefaultEndpointResolver() EndpointResolver {
	defaultEndpointMu.RLock()
	defer defaultEndpointMu.RUnlock()
	return defaultEndpointResolver
}
