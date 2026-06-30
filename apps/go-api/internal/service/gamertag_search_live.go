// Package service — gamertag_search_live.go : décorateur de recherche de gamertag
// qui ajoute un FALLBACK LIVE (résolution gamertag→xuid via l'API Halo/Xbox) quand
// la recherche locale (xuid_aliases) ne trouve pas le joueur cherché.
//
// Problème résolu : la recherche locale (Q11GamertagSearch) ne couvre que les
// joueurs déjà croisés (présents dans xuid_aliases). Chercher un joueur JAMAIS
// croisé renvoyait 0 résultat — y compris en prod. Le décorateur résout alors le
// gamertag en xuid via un résolveur live (PeopleHub→profil Xbox, cf.
// worldenrich.BuildDirectoryResolver) et ajoute un résultat synthétique afin que la
// cible apparaisse comme suggestion exacte (Explorer + Face-à-face partagent le
// même endpoint, donc un fix backend couvre les deux pages).
//
// Garde-fous : ne se déclenche QUE si le résolveur est câblé (nil en démo/offline),
// la query ressemble à un gamertag complet, et aucun match exact local n'existe
// déjà. 429/throttle et "introuvable" dégradent silencieusement (jamais 500) : la
// liste locale est toujours une réponse valide.
package service

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// GamertagXUIDResolver résout un gamertag public en xuid numérique (live, hors DB
// locale). Satisfait par les résolveurs worldenrich (PeopleHub→profil Xbox).
// Interface locale (duck-typing) pour éviter que le package service importe
// worldenrich. nil → aucun fallback live.
type GamertagXUIDResolver interface {
	ResolveXUID(ctx context.Context, gamertag string) (string, error)
}

const (
	liveFallbackMinChars = 3
	liveFallbackMaxChars = 30
	liveFallbackNegTTL   = 60 * time.Second
	liveFallbackTimeout  = 6 * time.Second
)

// LiveFallbackGamertagSearch enveloppe une recherche locale (port.GamertagSearchService)
// et ajoute un fallback live optionnel. Implémente port.GamertagSearchService.
type LiveFallbackGamertagSearch struct {
	local    port.GamertagSearchService
	resolver GamertagXUIDResolver // nil → pur passthrough local

	mu       sync.Mutex
	negCache map[string]time.Time // lower(gamertag) → expiry (échecs : not-found/throttle)
	now      func() time.Time
}

// NewLiveFallbackGamertagSearch crée le décorateur. resolver nil (démo/offline) →
// le décorateur se comporte exactement comme la recherche locale.
func NewLiveFallbackGamertagSearch(local port.GamertagSearchService, resolver GamertagXUIDResolver) *LiveFallbackGamertagSearch {
	return &LiveFallbackGamertagSearch{
		local:    local,
		resolver: resolver,
		negCache: make(map[string]time.Time),
		now:      time.Now,
	}
}

// Search délègue d'abord au local, puis tente le fallback live si pertinent.
func (s *LiveFallbackGamertagSearch) Search(ctx context.Context, query string) ([]domain.GamertagSearchResult, error) {
	results, err := s.local.Search(ctx, query)
	if err != nil {
		return results, err // erreur locale réelle : propager (comportement d'origine)
	}
	if s.resolver == nil {
		return results, nil
	}
	q := strings.TrimSpace(query)
	if !isPlausibleGamertag(q) || hasExactMatch(results, q) {
		return results, nil
	}
	key := strings.ToLower(q)
	if s.negCached(key) {
		return results, nil
	}

	rctx, cancel := context.WithTimeout(ctx, liveFallbackTimeout)
	defer cancel()
	xuid, rerr := s.resolver.ResolveXUID(rctx, q)
	if rerr != nil {
		s.setNeg(key)
		if strings.Contains(rerr.Error(), "429") {
			slog.WarnContext(ctx, "gamertag_live_fallback_throttled", "q", q)
		} else {
			slog.DebugContext(ctx, "gamertag_live_fallback_miss", "q", q, "err", rerr)
		}
		return results, nil
	}
	if hasXUID(results, xuid) {
		return results, nil // déjà présent localement sous un autre libellé
	}
	// Info : événement rare (gaté) et notable — un joueur JAMAIS croisé résolu via
	// l'API live. Visible dans logs/service.log (observabilité de la feature + usage API).
	slog.InfoContext(ctx, "gamertag_live_fallback_hit", "q", q, "xuid", xuid)
	return append(results, domain.GamertagSearchResult{
		Gamertag:   q,
		XUID:       xuid,
		Score:      0, // sous les hits locaux dans l'ordre front
		ExactMatch: true,
	}), nil
}

// negCached indique si key a un échec récent encore valide (purge à la lecture).
func (s *LiveFallbackGamertagSearch) negCached(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.negCache[key]
	if !ok {
		return false
	}
	if s.now().After(exp) {
		delete(s.negCache, key)
		return false
	}
	return true
}

func (s *LiveFallbackGamertagSearch) setNeg(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.negCache[key] = s.now().Add(liveFallbackNegTTL)
}

// isPlausibleGamertag : filtre les query partielles/junk avant tout appel réseau.
// Un gamertag Xbox = lettres/chiffres/espaces (+ suffixe #NNNN), 3..30 caractères,
// au moins un caractère non-espace.
func isPlausibleGamertag(q string) bool {
	if len(q) < liveFallbackMinChars || len(q) > liveFallbackMaxChars {
		return false
	}
	hasNonSpace := false
	for _, r := range q {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			hasNonSpace = true
		case r == ' ':
		case r == '#':
			hasNonSpace = true
		default:
			return false
		}
	}
	return hasNonSpace
}

// hasExactMatch : un résultat local correspond déjà exactement (casse ignorée).
func hasExactMatch(results []domain.GamertagSearchResult, q string) bool {
	for _, r := range results {
		if strings.EqualFold(r.Gamertag, q) {
			return true
		}
	}
	return false
}

// hasXUID : l'xuid figure déjà dans les résultats locaux.
func hasXUID(results []domain.GamertagSearchResult, xuid string) bool {
	for _, r := range results {
		if r.XUID == xuid {
			return true
		}
	}
	return false
}
