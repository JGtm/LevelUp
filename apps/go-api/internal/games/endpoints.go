package games

import "levelup/go-api/internal/games/mappings"

// EndpointKey ré-exporte mappings.EndpointKey afin que les consommateurs du port
// (platform/halo, internal/sync) n'importent que le package games.
type EndpointKey = mappings.EndpointKey

// Clés d'endpoint canoniques (alias des constantes mappings, MT-01).
const (
	EndpointStats        = mappings.EndpointStats
	EndpointGameCMS      = mappings.EndpointGameCMS
	EndpointEconomy      = mappings.EndpointEconomy
	EndpointSkill        = mappings.EndpointSkill
	EndpointUGCFilm      = mappings.EndpointUGCFilm
	EndpointDiscoveryUGC = mappings.EndpointDiscoveryUGC
	EndpointChallenges   = mappings.EndpointChallenges
	EndpointNameplate    = mappings.EndpointNameplate
)

// EndpointResolver résout l'host d'ingestion d'un titre par clé d'endpoint (MT-01).
//
// Port consommé par platform/halo + internal/sync ; zéro accès DB. La résolution
// se fait TOUJOURS par clé d'endpoint, jamais par comparaison de slug (archlint
// no_slug_comparison). HostFor retourne ok=false si le titre ne déclare pas cet
// axe d'ingestion → le caller dégrade (skip + warn), sans fallback silencieux
// vers l'host d'un autre titre.
type EndpointResolver interface {
	HostFor(slug string, key EndpointKey) (string, bool)
}

// MappingsEndpointResolver adapte une mappings.Registry au port EndpointResolver.
type MappingsEndpointResolver struct {
	reg         *mappings.Registry
	defaultSlug string
}

// NewMappingsEndpointResolver construit le resolver autour d'une Registry chargée.
// defaultSlug ("" → "halo_infinite") sert uniquement à résoudre un slug vide
// (ctx sans titre) vers le titre par défaut ; un slug NON vide mais inconnu
// retourne ok=false (pas de fallback cross-titre).
func NewMappingsEndpointResolver(reg *mappings.Registry, defaultSlug string) *MappingsEndpointResolver {
	if defaultSlug == "" {
		defaultSlug = "halo_infinite"
	}
	return &MappingsEndpointResolver{reg: reg, defaultSlug: defaultSlug}
}

// HostFor résout l'host d'un endpoint pour un titre. Voir EndpointResolver.
func (r *MappingsEndpointResolver) HostFor(slug string, key EndpointKey) (string, bool) {
	if r == nil || r.reg == nil {
		return "", false
	}
	if slug == "" {
		slug = r.defaultSlug
	}
	set, ok := r.reg.GetEndpoints(slug)
	if !ok {
		return "", false
	}
	return set.Host(key)
}

// GamePrefixFor résout le segment d'URL de jeu d'un titre (constants.toml
// [meta].game_prefix). Implémente games.GamePrefixResolver (MT-01). Même
// précédence de slug vide que HostFor. (_, false) si le titre est inconnu ou ne
// déclare pas de préfixe → le caller applique son défaut "hi" byte-identique.
func (r *MappingsEndpointResolver) GamePrefixFor(slug string) (string, bool) {
	if r == nil || r.reg == nil {
		return "", false
	}
	if slug == "" {
		slug = r.defaultSlug
	}
	set, ok := r.reg.GetEndpoints(slug)
	if !ok {
		return "", false
	}
	return set.GamePrefix()
}
