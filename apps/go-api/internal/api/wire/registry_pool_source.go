// Package api — registry_pool_source.go : source de token SAIN issue du pool
// partagé pour les LECTURES PUBLIQUES de tiers (player-query Explorer).
//
// Contexte (A1, diagnostic Explorer 2026-07-22) : tous les fetchs live du
// player-query portaient jusqu'ici les tokens du PROFIL sélectionné
// (enrichWithHaloTokens → ResolveFreshPlayerTokens(pdb.XUID)). Quand le RT du
// profil sélectionné est mort (AADSTS70000), TOUTES les sections live tombaient
// alors qu'un pool de tokens sains existait déjà (utilisé par le sync). Cet
// adaptateur expose un token sain du pool (PolicyAnyPublic, santé/cooldowns gérés
// par le pool) pour porter une lecture PUBLIQUE de tiers — sans jamais "réparer"
// une auth (ADR 0023 : aucune re-capture ; un RT mort reste mort).
package wire

import (
	"context"
	"errors"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/auth/pool"
)

// publicReadPoolAcquireTimeout borne l'acquisition d'un token du pool sur le
// chemin de lecture publique : l'acquisition est en mémoire (round-robin sur un
// canal bufferisé) donc quasi instantanée, mais on borne pour ne jamais pendre
// la requête si tous les slots sont temporairement pris.
const publicReadPoolAcquireTimeout = 3 * time.Second

// Compteurs expvar de provenance du token servant les lectures live du
// player-query Explorer (observabilité du fallback pool — A1.2).
const (
	explorerTokenSrcSession = "explorer_live_token_source_session"
	explorerTokenSrcProfile = "explorer_live_token_source_profile"
	explorerTokenSrcPool    = "explorer_live_token_source_pool"
	explorerTokenSrcNone    = "explorer_live_token_source_none"
)

// errPoolTokensEmpty : le pool a rendu un slot mais sans Spartan token
// exploitable (slot non encore résolu) — traité comme "pool indisponible".
var errPoolTokensEmpty = errors.New("wire: pool a rendu un lease sans Spartan token")

// pooledTokenSource fournit un token Halo SAIN issu du pool partagé pour les
// LECTURES PUBLIQUES de tiers. Découplé de pool.Pool (interface locale) pour la
// testabilité — les tests injectent un stub sans dépendre du pool concret.
type pooledTokenSource interface {
	// ResolveAny retourne un token sain du pool (round-robin, santé/cooldowns
	// gérés par le pool) et le gamertag du compte porteur pour la traçabilité.
	// Le pool n'expose que le gamertag du slot (pas le xuid) → provenance tracée
	// par gamertag. Erreur si aucun slot sain n'est disponible.
	ResolveAny(ctx context.Context) (tokens *domain.HaloTokens, sourceGamertag string, err error)
}

// poolPublicReadAdapter adapte un pool.Pool en pooledTokenSource.
type poolPublicReadAdapter struct{ p pool.Pool }

// ResolveAny acquiert un token round-robin (PolicyAnyPublic), copie le pointeur
// de tokens (immuable côté pool) puis rend IMMÉDIATEMENT le slot : on ne tient
// pas le lease pendant tout l'errgroup de fetchs live de l'encart cible (sinon
// on sérialiserait la page sur un seul slot). Les tokens restent valides ~4 h.
func (a poolPublicReadAdapter) ResolveAny(ctx context.Context) (*domain.HaloTokens, string, error) {
	lease, err := a.p.Acquire(ctx, pool.PolicyAnyPublic, "")
	if err != nil {
		return nil, "", err
	}
	tokens := lease.Tokens
	gamertag := lease.Gamertag
	lease.Release()
	if tokens == nil || tokens.SpartanToken == "" {
		return nil, "", errPoolTokensEmpty
	}
	return tokens, gamertag, nil
}

// WithTokenPool câble le pool de tokens partagé comme source de fallback pour les
// lectures publiques de tiers (Explorer). p nil → no-op (comportement historique
// conservé : le player-query reste sur les tokens du profil sélectionné).
func (r *ServiceRegistry) WithTokenPool(p pool.Pool) *ServiceRegistry {
	if p != nil {
		r.publicReadTokenSrc = poolPublicReadAdapter{p: p}
	}
	return r
}

// incExplorerTokenSource incrémente le compteur expvar de provenance du token.
func incExplorerTokenSource(name string) { observability.IncCounter(name) }
