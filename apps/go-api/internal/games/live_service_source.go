package games

// live_service_source.go — gating title-aware des SURFACES LIVE-SERVICE
// (Battle Pass / Challenges) re-fetchées périodiquement par le watcher pendant
// la présence active d'un joueur (internal/watcher live-refresh).
//
// Pourquoi ce gating : les endpoints live Battle Pass (economy
// `rewardtracks/operations`) et Challenges (`decks`) sont des surfaces
// live-service Halo Infinite. Elles n'existent PAS pour tous les titres :
// Halo 5 n'expose ni Battle Pass (pas de pass saisonnier) ni Challenges — les
// sonder avec le token Spartan partagé renvoie HTTP 404 (bruit de logs + charge
// inutile toutes les 5 min). Le seul garde-fou correct est de NE PAS appeler
// ces endpoints pour un titre qui ne les déclare pas.
//
// Discriminants = capabilities CapBattlePass (battlepass.progression) et
// CapChallenges (challenges.surface). 100% title-agnostic : on lit la capability
// du titre via le resolver partagé, jamais son slug (archlint
// no_slug_comparison). Même pattern que career_progression_source.go (resolver
// optionnel + helper package-level + défaut byte-identique Infinite).

// ProvidesBattlePassFromResolver indique si le titre expose la surface Battle
// Pass live (endpoint economy rewardtracks/operations), résolu via la capability
// CapBattlePass du titre.
//
// Défaut TRUE (byte-identique Halo Infinite) quand le resolver est nil / ne
// supporte pas l'extension CapabilityResolver / le titre n'a aucune capability
// déclarée : préserve le chemin live historique pour les tests et instances
// mono-titre qui ne câblent pas de resolver de capabilities. Un titre qui
// DÉCLARE ses capabilities sans battlepass.progression (ex. Halo 5) → FALSE : on
// court-circuite le fetch live (qui renverrait 404 via le token partagé).
func ProvidesBattlePassFromResolver(res EndpointResolver, slug string) bool {
	cr, ok := res.(CapabilityResolver)
	if !ok {
		return true
	}
	caps, found := cr.CapabilitiesFor(slug)
	if !found {
		// Titre sans capabilities déclarées : supposer Infinite (défaut sûr).
		return true
	}
	return caps.Has(CapBattlePass)
}

// ProvidesBattlePass résout via le resolver partagé posé au boot
// (DefaultEndpointResolver). Point d'entrée des callers (watcher live-refresh)
// qui disposent du slug du titre mais pas du resolver.
func ProvidesBattlePass(slug string) bool {
	return ProvidesBattlePassFromResolver(DefaultEndpointResolver(), slug)
}

// ProvidesChallengesFromResolver indique si le titre expose la surface
// Challenges live (endpoint decks), résolu via la capability CapChallenges.
// Même sémantique de défaut que ProvidesBattlePassFromResolver.
func ProvidesChallengesFromResolver(res EndpointResolver, slug string) bool {
	cr, ok := res.(CapabilityResolver)
	if !ok {
		return true
	}
	caps, found := cr.CapabilitiesFor(slug)
	if !found {
		return true
	}
	return caps.Has(CapChallenges)
}

// ProvidesChallenges résout via le resolver partagé posé au boot
// (DefaultEndpointResolver). Point d'entrée des callers (watcher live-refresh).
func ProvidesChallenges(slug string) bool {
	return ProvidesChallengesFromResolver(DefaultEndpointResolver(), slug)
}
