package games

// DefaultGamePrefix est le segment d'URL de jeu de Halo Infinite. C'est le
// fallback byte-identique : tout call-site d'ingestion qui ne peut pas résoudre
// le préfixe du titre courant (resolver non câblé, titre sans game_prefix
// déclaré, resolver ne supportant pas l'extension) retombe sur "hi" — donc zéro
// changement de comportement pour Halo Infinite.
const DefaultGamePrefix = "hi"

// GamePrefixResolver est l'extension OPTIONNELLE d'EndpointResolver qui résout le
// segment d'URL de jeu ("hi"/"h5") d'un titre (axe MT-01). Implémentée par
// *MappingsEndpointResolver. Un resolver qui ne l'implémente pas (stub de test,
// resolver legacy) est traité comme « préfixe non déclaré » → fallback "hi".
//
// Séparée de EndpointResolver pour ne pas casser les implémentations existantes
// (type-assertion à l'usage), tout en partageant la même source de vérité
// (constants.toml [meta].game_prefix → mappings.EndpointSet).
type GamePrefixResolver interface {
	GamePrefixFor(slug string) (string, bool)
}

// GamePrefixFromResolver résout le préfixe de jeu d'un titre via un resolver
// donné, avec fallback DefaultGamePrefix. res peut être nil ou ne pas implémenter
// GamePrefixResolver → "hi". Un préfixe déclaré mais vide est ignoré (fallback).
func GamePrefixFromResolver(res EndpointResolver, slug string) string {
	if gp, ok := res.(GamePrefixResolver); ok {
		if prefix, ok := gp.GamePrefixFor(slug); ok && prefix != "" {
			return prefix
		}
	}
	return DefaultGamePrefix
}

// GamePrefix résout le préfixe de jeu du titre courant via le resolver partagé de
// boot (DefaultEndpointResolver), avec fallback DefaultGamePrefix. Point d'entrée
// des couches d'ingestion (internal/sync, internal/platform/halo, internal/assets)
// qui injectent le segment dans les chemins d'API.
func GamePrefix(slug string) string {
	return GamePrefixFromResolver(DefaultEndpointResolver(), slug)
}
