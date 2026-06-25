// Package halo — player_token_cache.go : cache process-level des HaloTokens par XUID.
//
// Évite d'appeler MSAL + l'échange XBL/XSTS/Spartan à chaque requête.
//
// Principe « expiry-aware » (anti-401-par-péremption) : le cache ne sert JAMAIS un
// token qu'il ne peut pas garantir valide. L'entrée porte l'expiry RÉEL du Spartan
// token (`HaloTokens.SpartanExpiresAt`, champ `ExpiresUtc` de la réponse /spartan-token).
// Si l'expiry réel est inconnu (sources legacy/import), on retombe sur un TTL conservateur.
// `GetCachedPlayerTokens` applique une marge avant expiry : un token dans sa fenêtre de
// marge est traité comme expiré → le caller re-mint. C'est CETTE garantie (et non un
// retry sur 401) qui empêche de servir un Spartan périmé.
package halo

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"levelup/go-api/internal/domain"
)

// fallbackTokenTTL : durée de vie supposée quand l'expiry réel du Spartan est inconnu
// (token issu d'une source qui ne fournit pas `ExpiresUtc`). Conservateur.
const fallbackTokenTTL = 50 * time.Minute

// tokenExpiryMargin : marge de sécurité avant l'expiry réel. Un token dont l'expiry
// tombe dans cette fenêtre est considéré comme déjà expiré (re-mint anticipé) — couvre
// la latence réseau et une légère dérive d'horloge sans jamais émettre un appel avec
// un token sur le point d'expirer.
const tokenExpiryMargin = 5 * time.Minute

type cachedTokenEntry struct {
	tokens *domain.HaloTokens
	// expiresAt : expiry RÉEL du Spartan token (ou now+fallbackTokenTTL si inconnu).
	expiresAt time.Time
}

// playerTokenStore est le cache global (singleton de processus).
var playerTokenStore = &struct {
	mu    sync.RWMutex
	store map[string]cachedTokenEntry
}{store: make(map[string]cachedTokenEntry)}

// playerTokenRefresher est le hook (nil-safe) de re-dérivation des tokens d'un joueur.
// Câblé au boot sur la résolution registry (MSAL/OAuth → Spartan frais). Doit MINTER
// frais (ignorer le cache) et NE PAS écrire le cache — `ResolveFreshPlayerTokens` s'en
// charge. nil → pas de re-mint possible (comportement pré-fix : on rend l'erreur).
var (
	playerTokenRefresherMu sync.RWMutex
	playerTokenRefresher   func(ctx context.Context, xuid string) (*domain.HaloTokens, error)
)

// tokenRefreshSF déduplique les re-mint concurrents pour un même xuid (anti
// thundering-herd : N requêtes HTTP + watcher sur le même joueur → un seul échange OAuth).
var tokenRefreshSF singleflight.Group

// SetPlayerTokenRefresher câble le hook global de re-dérivation. Appelé une fois au boot.
func SetPlayerTokenRefresher(fn func(ctx context.Context, xuid string) (*domain.HaloTokens, error)) {
	playerTokenRefresherMu.Lock()
	playerTokenRefresher = fn
	playerTokenRefresherMu.Unlock()
}

// spartanExpiryLog formate l'expiry du Spartan pour les logs ("unknown" si zéro).
func spartanExpiryLog(tokens *domain.HaloTokens) string {
	if tokens == nil || tokens.SpartanExpiresAt.IsZero() {
		return "unknown"
	}
	return tokens.SpartanExpiresAt.UTC().Format(time.RFC3339)
}

// cacheExpiryFor calcule l'instant d'expiry d'une entrée à partir de l'expiry réel du
// Spartan (si fourni) ou du TTL de repli.
func cacheExpiryFor(tokens *domain.HaloTokens) time.Time {
	if tokens != nil && !tokens.SpartanExpiresAt.IsZero() {
		return tokens.SpartanExpiresAt
	}
	return time.Now().Add(fallbackTokenTTL)
}

// GetCachedPlayerTokens retourne les HaloTokens en cache s'ils sont encore valides
// AVEC marge (now < expiry - tokenExpiryMargin), nil sinon. Ne sert jamais un token
// dans sa fenêtre de marge → garantit qu'on n'émet pas d'appel avec un Spartan périmé.
func GetCachedPlayerTokens(xuid string) *domain.HaloTokens {
	playerTokenStore.mu.RLock()
	defer playerTokenStore.mu.RUnlock()
	entry, ok := playerTokenStore.store[xuid]
	if !ok || !time.Now().Before(entry.expiresAt.Add(-tokenExpiryMargin)) {
		return nil
	}
	return entry.tokens
}

// TokensFresh indique si un jeu de tokens est encore exploitable (now < expiry - marge).
// Expiry inconnu (0) → true : on ne casse pas les sources legacy/import sans expiry ;
// le filet 401 (defense-in-depth) couvre le cas où un tel token serait en fait périmé.
func TokensFresh(tokens *domain.HaloTokens) bool {
	if tokens == nil || tokens.SpartanToken == "" {
		return false
	}
	if tokens.SpartanExpiresAt.IsZero() {
		return true
	}
	return time.Now().Before(tokens.SpartanExpiresAt.Add(-tokenExpiryMargin))
}

// TokensFreshStrict est comme TokensFresh mais EXIGE une expiry CONNUE : un token d'expiry
// inconnue (zéro) est traité comme NON frais (à re-résoudre). À utiliser là où le déterminisme
// prime — typiquement la réutilisation d'un token de SESSION (sessions pré-A1 sans
// SpartanExpiresAt) : on ne le garde que si on peut PROUVER qu'il est encore valide, sinon on
// re-minte un token frais. Évite l'intermittence « parfois frais / parfois 401 » des chemins
// token-gated sans filet (rang carrière). TokensFresh (tolérant au zéro) reste pour les chemins
// qui ont un filet 401 en aval.
func TokensFreshStrict(tokens *domain.HaloTokens) bool {
	if tokens == nil || tokens.SpartanExpiresAt.IsZero() {
		return false
	}
	return TokensFresh(tokens)
}

// SetCachedPlayerTokens stocke les HaloTokens, avec un instant d'expiry calé sur
// l'expiry RÉEL du Spartan (fallback TTL conservateur si inconnu).
func SetCachedPlayerTokens(xuid string, tokens *domain.HaloTokens) {
	playerTokenStore.mu.Lock()
	defer playerTokenStore.mu.Unlock()
	playerTokenStore.store[xuid] = cachedTokenEntry{
		tokens:    tokens,
		expiresAt: cacheExpiryFor(tokens),
	}
}

// InvalidateCachedPlayerTokens supprime l'entrée d'un xuid (force un nouveau
// refresh complet au prochain GetCachedPlayerTokens). À appeler quand un
// événement externe (rotation RT Microsoft, révocation manuelle) OU un 401/403
// live (filet defense-in-depth) rend les tokens en cache potentiellement obsolètes.
// No-op si l'xuid n'est pas en cache.
func InvalidateCachedPlayerTokens(xuid string) {
	if xuid == "" {
		return
	}
	playerTokenStore.mu.Lock()
	_, existed := playerTokenStore.store[xuid]
	delete(playerTokenStore.store, xuid)
	playerTokenStore.mu.Unlock()
	// Observabilité : tracer une invalidation EFFECTIVE (rotation RT / re-login / 401)
	// — utile pour diagnostiquer la fraîcheur des fetchs live token-gated.
	if existed {
		slog.Debug("halo: cache HaloTokens invalidé — re-dérivation forcée au prochain Get", "xuid", xuid)
	}
}

// ResolveFreshPlayerTokens retourne des tokens valides pour le joueur : sert le cache
// s'il est frais (avec marge), sinon re-mint via le hook global, dédupliqué par xuid
// (singleflight) et persisté au cache. C'est le point d'entrée unique du chemin
// per-player pour obtenir un token « valide par construction ».
func ResolveFreshPlayerTokens(ctx context.Context, xuid string) (*domain.HaloTokens, error) {
	if xuid == "" {
		return nil, fmt.Errorf("halo: ResolveFreshPlayerTokens xuid vide")
	}
	if t := GetCachedPlayerTokens(xuid); t != nil {
		return t, nil
	}
	playerTokenRefresherMu.RLock()
	refresher := playerTokenRefresher
	playerTokenRefresherMu.RUnlock()
	if refresher == nil {
		return nil, fmt.Errorf("halo: aucun refresher token câblé (xuid=%s)", xuid)
	}
	v, err, _ := tokenRefreshSF.Do(xuid, func() (interface{}, error) {
		// Double-check : un autre appel concurrent a pu re-minter pendant l'attente.
		if t := GetCachedPlayerTokens(xuid); t != nil {
			return t, nil
		}
		tokens, rerr := refresher(ctx, xuid)
		if rerr != nil {
			return nil, rerr
		}
		if tokens == nil {
			return nil, fmt.Errorf("halo: refresher a renvoyé des tokens nil (xuid=%s)", xuid)
		}
		SetCachedPlayerTokens(xuid, tokens)
		// Observabilité (Info) : un re-mint = échange MSAL/OAuth coûteux mais peu fréquent
		// (1× par joueur par ~durée de vie Spartan, grâce au cache + singleflight). C'est LE
		// signal qui prouve que le chemin déterministe (token frais à expiry connue) est servi
		// — utile pour vérifier en prod l'absence d'intermittence du rang carrière.
		slog.InfoContext(ctx, "halo: token Spartan re-minté (cache expiry-aware)",
			"xuid", xuid, "spartan_expires_at", spartanExpiryLog(tokens))
		return tokens, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*domain.HaloTokens), nil
}
