// Package halo — player_token_cache.go : cache process-level des HaloTokens par XUID.
//
// Évite d'appeler MSAL + l'échange XBL/XSTS/Spartan à chaque requête.
// TTL calé sur l'expiry du SpartanToken (~1h) avec une marge de 10 min.
package halo

import (
	"log/slog"
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

// InvalidateCachedPlayerTokens supprime l'entrée d'un xuid (force un nouveau
// refresh complet au prochain GetCachedPlayerTokens). À appeler quand un
// événement externe (rotation RT Microsoft, révocation manuelle) rend les
// tokens en cache potentiellement obsolètes — le TTL automatique de 50 min
// est trop long pour ces cas. No-op si l'xuid n'est pas en cache.
func InvalidateCachedPlayerTokens(xuid string) {
	if xuid == "" {
		return
	}
	playerTokenStore.mu.Lock()
	_, existed := playerTokenStore.store[xuid]
	delete(playerTokenStore.store, xuid)
	playerTokenStore.mu.Unlock()
	// Observabilité : tracer une invalidation EFFECTIVE (rotation RT / re-login)
	// — utile pour diagnostiquer la fraîcheur des fetchs live token-gated.
	if existed {
		slog.Debug("halo: cache HaloTokens invalidé — re-dérivation forcée au prochain Get", "xuid", xuid)
	}
}
