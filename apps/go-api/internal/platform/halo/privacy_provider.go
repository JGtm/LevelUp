// Package halo — privacy_provider.go : interrogation de la privacy d'un compte Halo.
//
// Sprint 54 B : GET /hi/players/{xuid}/matches-privacy
// Sprint 54 B5 : cache process-level avec TTL (privacyTTLCache) pour éviter
//
//	un appel Waypoint à chaque requête bootstrap.
//	TTL par défaut : 30 minutes.
//	La player DB étant ouverte en read-only dans le pool, le cache
//	est maintenu en mémoire (sync.RWMutex) plutôt qu'en DuckDB.
//
// Plan perf 2026-09-23 (lot L5b, D5b.7) : l'ÉCHEC de l'appel (statut HTTP, réseau,
// réponse illisible) est lui aussi mémorisé, PrivacyFailureTTL : le repli
// « fetch_error » est resservi sans appel réseau. Une fin de contexte (annulation de
// la requête, budget de l'appelant écoulé) n'est pas un verdict de Waypoint et n'est
// pas mémorisée ici — /bootstrap mémorise son propre dépassement de budget.
package halo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/timing"
)

// PrivacyCacheTTL est la durée de validité d'une entrée dans le cache privacy.
// 30 minutes : suffisant pour éviter le spam Waypoint, court pour rester frais.
const PrivacyCacheTTL = 30 * time.Minute

// PrivacyFailureTTL est la durée de mémorisation d'un échec de l'appel privacy : un
// Waypoint en échec n'est plus rappelé à chaque requête, et un rétablissement se voit
// au plus 5 minutes plus tard.
const PrivacyFailureTTL = 5 * time.Minute

// privacyCacheEntry est une entrée du cache TTL.
type privacyCacheEntry struct {
	info       *domain.MatchPrivacyInfo
	observedAt time.Time
	ttl        time.Duration // 0 = PrivacyCacheTTL (entrée de succès)
}

// privacyTTLCache est un cache thread-safe de la privacy par xuid.
type privacyTTLCache struct {
	mu      sync.RWMutex
	entries map[string]privacyCacheEntry
	now     func() time.Time // horloge ; nil = time.Now (injectée en test)
}

func (c *privacyTTLCache) clock() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

// get retourne l'entrée si elle existe et n'est pas expirée.
func (c *privacyTTLCache) get(xuid string) (*domain.MatchPrivacyInfo, bool) {
	c.mu.RLock()
	e, ok := c.entries[xuid]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	ttl := e.ttl
	if ttl <= 0 {
		ttl = PrivacyCacheTTL
	}
	if c.clock().Sub(e.observedAt) > ttl {
		return nil, false
	}
	return e.info, true
}

// set stocke une réponse Waypoint réussie (PrivacyCacheTTL).
func (c *privacyTTLCache) set(xuid string, info *domain.MatchPrivacyInfo) {
	c.setFor(xuid, info, PrivacyCacheTTL)
}

// setFor stocke une entrée valable `ttl`.
func (c *privacyTTLCache) setFor(xuid string, info *domain.MatchPrivacyInfo, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = make(map[string]privacyCacheEntry)
	}
	c.entries[xuid] = privacyCacheEntry{info: info, observedAt: c.clock(), ttl: ttl}
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
// L'appel Waypoint n'est effectué que si le cache est vide ou expiré (succès : 30 min,
// échec : 5 min). Retourne un MatchPrivacyInfo non-nil même en cas d'erreur
// (fallback gracieux). Marqueurs timing `privacy_cache_hit` / `privacy_cache_miss`.
func (p *HaloProvider) GetMatchPrivacy(ctx context.Context, xuid string) (*domain.MatchPrivacyInfo, error) {
	if xuid == "" {
		return &domain.MatchPrivacyInfo{Hint: errHintAuthRequired}, nil
	}

	// Sprint 54 B5 : vérifier le cache avant tout appel Waypoint.
	if cached, ok := p.privacyCache.get(xuid); ok {
		timing.FromContext(ctx).Section("privacy_cache_hit")()
		return cached, nil
	}

	tokens := ctxkeys.HaloTokens(ctx)
	if tokens == nil {
		return &domain.MatchPrivacyInfo{Hint: errHintAuthRequired}, nil
	}
	timing.FromContext(ctx).Section("privacy_cache_miss")()

	url := fmt.Sprintf("%s/%s/players/xuid(%s)/matches-privacy", defaultStatsHost, p.gamePrefix(ctx), xuid)
	body, err := p.doGet(ctx, url, tokens)
	if err != nil {
		// Privacy non critique — fallback partiel, échec mémorisé (D5b.7).
		return p.privacyFailure(ctx, xuid, errHintFetchError, err), nil
	}

	var resp privacyResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return p.privacyFailure(ctx, xuid, "parse_error", err), nil
	}

	info := parsePrivacyResponse(&resp)

	// Sprint 54 B5 : mettre en cache le résultat.
	p.privacyCache.set(xuid, info)

	return info, nil
}

// privacyFailure journalise un échec de l'appel privacy et rend le repli partiel,
// mémorisé PrivacyFailureTTL — sauf fin de contexte (annulation ou échéance de
// l'appelant), qui n'est pas un verdict de Waypoint : la requête suivante réessaie.
func (p *HaloProvider) privacyFailure(ctx context.Context, xuid, hint string, err error) *domain.MatchPrivacyInfo {
	info := &domain.MatchPrivacyInfo{IsPartial: true, Hint: hint}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		slog.DebugContext(ctx, "privacy: appel Waypoint interrompu par le contexte — non mémorisé",
			"xuid", xuid, "err", err)
		return info
	}
	p.privacyCache.setFor(xuid, info, PrivacyFailureTTL)
	slog.ErrorContext(ctx, "privacy: appel Waypoint en échec — repli mémorisé, pas de nouvel appel avant l'échéance",
		"xuid", xuid, "hint", hint, "retry_after", PrivacyFailureTTL, "err", err)
	return info
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
