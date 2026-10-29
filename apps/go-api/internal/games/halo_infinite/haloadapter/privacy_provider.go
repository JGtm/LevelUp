// Package haloadapter — privacy_provider.go : confidentialité des matchs d'un compte Halo
// (`/{jeu}/players/xuid(X)/matches-privacy`), avec un cache de processus par xuid (la base du
// joueur est ouverte en lecture seule dans le pool : le cache vit en mémoire).
//
// Succès gardé PrivacyCacheTTL (30 min) ; l'ÉCHEC de l'appel (statut HTTP, réseau, réponse
// illisible) est lui aussi mémorisé, PrivacyFailureTTL : le repli « fetch_error » est resservi
// sans appel réseau. Une fin de contexte (annulation de la requête, budget de l'appelant écoulé)
// n'est pas un verdict de Waypoint et n'est pas mémorisée ici — /bootstrap mémorise son propre
// dépassement de budget.
package haloadapter

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/haloapi/infinite"
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

// GetMatchPrivacy retourne la confidentialité du compte Halo `xuid`. L'appel n'est fait que si le
// cache est vide ou expiré (succès : 30 min, échec : 5 min). Rend toujours un MatchPrivacyInfo
// non nil (repli gracieux). Marqueurs timing `privacy_cache_hit` / `privacy_cache_miss`.
func (p *Provider) GetMatchPrivacy(ctx context.Context, xuid string) (*domain.MatchPrivacyInfo, error) {
	if xuid == "" {
		return &domain.MatchPrivacyInfo{Hint: errHintAuthRequired}, nil
	}
	if cached, ok := p.privacyCache.get(xuid); ok {
		timing.FromContext(ctx).Section("privacy_cache_hit")()
		return cached, nil
	}
	tokens := ctxkeys.HaloTokens(ctx)
	if tokens == nil {
		return &domain.MatchPrivacyInfo{Hint: errHintAuthRequired}, nil
	}
	timing.FromContext(ctx).Section("privacy_cache_miss")()

	resp, err := p.api(ctx, surfaceProviderGet, infinite.RetryProvider()).GetMatchPrivacy(ctx, authDe(tokens), xuid)
	if err != nil {
		hint := errHintFetchError
		if estErreurDeDecodage(err) {
			hint = privacyHintParseError
		}
		// Confidentialité non critique — repli partiel, échec mémorisé.
		return p.privacyFailure(ctx, xuid, hint, err), nil
	}
	info := parsePrivacyResponse(resp)
	p.privacyCache.set(xuid, info)
	return info, nil
}

// estErreurDeDecodage dit si `err` vient du décodage JSON d'une réponse reçue (et non de l'appel).
func estErreurDeDecodage(err error) bool {
	var syntaxe *json.SyntaxError
	var typage *json.UnmarshalTypeError
	return errors.As(err, &syntaxe) || errors.As(err, &typage)
}

// privacyFailure journalise un échec de l'appel et rend le repli partiel, mémorisé
// PrivacyFailureTTL — sauf fin de contexte (annulation ou échéance de l'appelant), qui n'est pas
// un verdict de Waypoint : la requête suivante réessaie.
func (p *Provider) privacyFailure(ctx context.Context, xuid, hint string, err error) *domain.MatchPrivacyInfo {
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

// Constantes de confidentialité.
const (
	privacyStatusPrivate   = "Private"
	privacyHintFullPrivate = "full_private"
	privacyHintParseError  = "parse_error"
)

// parsePrivacyResponse convertit la réponse brute en MatchPrivacyInfo.
func parsePrivacyResponse(resp *infinite.PrivacyDTO) *domain.MatchPrivacyInfo {
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
