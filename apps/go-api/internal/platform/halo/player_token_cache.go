// Package halo — player_token_cache.go : cache process-level des HaloTokens par XUID.
//
// Évite d'appeler MSAL + l'échange XBL/XSTS/Spartan à chaque requête.
// TTL calé sur l'expiry du SpartanToken (~1h) avec une marge de 10 min.
package halo

import (
	"sync"
	"time"

	"levelup/go-api/internal/domain"
)

const playerTokenTTL = 50 * time.Minute

type cachedTokenEntry struct {
	tokens    *domain.HaloTokens
	expiresAt time.Time
}

// playerTokenStore est le cache global (singleton de processus).
var playerTokenStore = &struct {
	mu    sync.RWMutex
	store map[string]cachedTokenEntry
}{store: make(map[string]cachedTokenEntry)}

// GetCachedPlayerTokens retourne les HaloTokens en cache si encore valides, nil sinon.
func GetCachedPlayerTokens(xuid string) *domain.HaloTokens {
	playerTokenStore.mu.RLock()
	defer playerTokenStore.mu.RUnlock()
	entry, ok := playerTokenStore.store[xuid]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil
	}
	return entry.tokens
}

// SetCachedPlayerTokens stocke les HaloTokens avec un TTL de playerTokenTTL.
func SetCachedPlayerTokens(xuid string, tokens *domain.HaloTokens) {
	playerTokenStore.mu.Lock()
	defer playerTokenStore.mu.Unlock()
	playerTokenStore.store[xuid] = cachedTokenEntry{
		tokens:    tokens,
		expiresAt: time.Now().Add(playerTokenTTL),
	}
}
