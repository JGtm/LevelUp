// Package halo — privacy_provider.go : interrogation de la privacy d'un compte Halo.
//
// Sprint 54 B : GET /hi/players/{xuid}/matches-privacy
// Sprint 54 B5 : cache process-level avec TTL (privacyTTLCache) pour éviter
//
//	un appel Waypoint à chaque requête bootstrap.
//	TTL par défaut : 30 minutes.
//	La player DB étant ouverte en read-only dans le pool, le cache
//	est maintenu en mémoire (sync.RWMutex) plutôt qu'en DuckDB.
package halo

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// PrivacyCacheTTL est la durée de validité d'une entrée dans le cache privacy.
// 30 minutes : suffisant pour éviter le spam Waypoint, court pour rester frais.
const PrivacyCacheTTL = 30 * time.Minute

// privacyCacheEntry est une entrée du cache TTL.
type privacyCacheEntry struct {
	info       *domain.MatchPrivacyInfo
	observedAt time.Time
}

// privacyTTLCache est un cache thread-safe de la privacy par xuid.
type privacyTTLCache struct {
	mu      sync.RWMutex
	entries map[string]privacyCacheEntry
}

// get retourne l'entrée si elle existe et n'est pas expirée.
func (c *privacyTTLCache) get(xuid string) (*domain.MatchPrivacyInfo, bool) {
	c.mu.RLock()
	e, ok := c.entries[xuid]
	c.mu.RUnlock()
	if !ok || time.Since(e.observedAt) > PrivacyCacheTTL {
		return nil, false
	}
	return e.info, true
}

// set stocke une entrée dans le cache.
func (c *privacyTTLCache) set(xuid string, info *domain.MatchPrivacyInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = make(map[string]privacyCacheEntry)
	}
	c.entries[xuid] = privacyCacheEntry{info: info, observedAt: time.Now()}
}

// privacyResponse est la réponse brute de l'endpoint matches-privacy.
type privacyResponse struct {
	AllMatchesPrivacy    string `json:"AllMatchesPrivacy"`
	PublicMatchesPrivacy string `json:"PublicMatchesPrivacy"`
	RankedMatchesPrivacy string `json:"RankedMatchesPrivacy"`
	CustomMatchesPrivacy string `json:"CustomMatchesPrivacy"`
}

// defaultStatsHost — host service-record/privacy. PMT-1 (MT-01) : NON migré sur
// EndpointResolver à dessein. Sa valeur (sans `:443`) est byte-distincte de
// EndpointStats (`https://halostats…:443`, utilisé par halo_client) ; les router
// sous une même clé introduirait un `:443` sur le Host header de ce fetch prod
// (cosmétique mais non byte-identique), et privacy/servicerecord ne mérite pas
// une 9e clé d'endpoint. Reste sur la const jusqu'à ce qu'un 2e titre expose
// réellement cette surface (capability-gated). Cf. .ai/thought_log.md.
const defaultStatsHost = "https://halostats.svc.halowaypoint.com"

// GetMatchPrivacy retourne la privacy du compte Halo pour le xuid donné.
// L'appel Waypoint n'est effectué que si le cache est vide ou expiré (TTL=30min).
// Retourne un MatchPrivacyInfo non-nil même en cas d'erreur (fallback gracieux).
func (p *HaloProvider) GetMatchPrivacy(ctx context.Context, xuid string) (*domain.MatchPrivacyInfo, error) {
	if xuid == "" {
		return &domain.MatchPrivacyInfo{Hint: errHintAuthRequired}, nil
	}

	// Sprint 54 B5 : vérifier le cache avant tout appel Waypoint.
	if cached, ok := p.privacyCache.get(xuid); ok {
		return cached, nil
	}

	tokens := ctxkeys.HaloTokens(ctx)
	if tokens == nil {
		return &domain.MatchPrivacyInfo{Hint: errHintAuthRequired}, nil
	}

	url := fmt.Sprintf("%s/%s/players/xuid(%s)/matches-privacy", defaultStatsHost, p.gamePrefix(ctx), xuid)
	body, err := p.doGet(ctx, url, tokens)
	if err != nil {
		// Privacy non critique — fallback partiel sans mettre en cache l'erreur.
		return &domain.MatchPrivacyInfo{IsPartial: true, Hint: errHintFetchError}, nil
	}

	var resp privacyResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return &domain.MatchPrivacyInfo{IsPartial: true, Hint: "parse_error"}, nil
	}

	info := parsePrivacyResponse(&resp)

	// Sprint 54 B5 : mettre en cache le résultat (succès uniquement).
	p.privacyCache.set(xuid, info)

	return info, nil
}

// Constantes privacy Waypoint.
const (
	privacyStatusPrivate   = "Private"
	privacyHintFullPrivate = "full_private"
)

// parsePrivacyResponse convertit la réponse brute Waypoint en MatchPrivacyInfo.
func parsePrivacyResponse(resp *privacyResponse) *domain.MatchPrivacyInfo {
	info := &domain.MatchPrivacyInfo{}
	if resp.AllMatchesPrivacy == privacyStatusPrivate {
		info.IsPrivate = true
		info.Hint = privacyHintFullPrivate
	} else if resp.RankedMatchesPrivacy == privacyStatusPrivate || resp.PublicMatchesPrivacy == privacyStatusPrivate {
		info.IsPartial = true
		info.Hint = "partial_private"
	}
	return info
}
