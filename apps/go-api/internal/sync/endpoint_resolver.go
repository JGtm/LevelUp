// Package sync — endpoint_resolver.go : résolution title-aware des hosts
// d'ingestion API (MT-01 / PMT-1 Contract).
//
// Les HaloAPIClient consultent un games.EndpointResolver pour router chaque
// appel vers l'host du titre courant (lu dans le ctx). Source de vérité : la
// section [endpoints] de config/titles/{slug}/constants.toml, chargée au boot.
//
// Le resolver est câblé UNE fois au boot (server.go → SetDefaultEndpointResolver)
// et partagé par tous les clients. Les const Halo legacy (haloStatsHost, …)
// restent comme FALLBACK byte-identique pour les chemins non câblés (binaires
// CLI hors boot, tests) — pour halo_infinite, le resolver rend exactement la
// même valeur (golden de parité), donc zéro changement de comportement.
package sync

import (
	"context"
	"log/slog"
	"sync"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games"
)

var (
	endpointResolverMu sync.RWMutex
	sharedEndpoints    games.EndpointResolver
)

// SetDefaultEndpointResolver câble le resolver d'hosts partagé (appelé au boot,
// après chargement de la mappings.Registry). Idempotent.
func SetDefaultEndpointResolver(r games.EndpointResolver) {
	endpointResolverMu.Lock()
	sharedEndpoints = r
	endpointResolverMu.Unlock()
}

// sharedEndpointResolver retourne le resolver partagé (nil tant que non câblé).
func sharedEndpointResolver() games.EndpointResolver {
	endpointResolverMu.RLock()
	defer endpointResolverMu.RUnlock()
	return sharedEndpoints
}

// hostFor résout l'host d'un endpoint pour le titre courant (lu dans le ctx).
// Précédence : resolver d'instance (override de test) → resolver partagé (boot).
// Fallback sur `legacy` (const Halo) si aucun resolver n'est câblé OU si le titre
// ne déclare pas cet endpoint (warn `endpoint_missing` ; transition — la
// validation boot PMT-12 garantit qu'un titre ACTIF déclare ses endpoints requis).
func (c *HaloAPIClient) hostFor(ctx context.Context, key games.EndpointKey, legacy string) string {
	res := c.endpoints
	if res == nil {
		res = sharedEndpointResolver()
	}
	if res == nil {
		return legacy
	}
	slug := ctxkeys.TitleSlug(ctx)
	if host, ok := res.HostFor(slug, key); ok {
		return host
	}
	slog.WarnContext(ctx, "endpoint_missing", "title", slug, "endpoint_key", string(key))
	return legacy
}
