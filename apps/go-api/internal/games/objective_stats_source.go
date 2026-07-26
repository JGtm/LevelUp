package games

// objective_stats_source.go — gating title-aware de la SOURCE DE DONNÉES
// match_objective_stats (stats objectifs par joueur/match, shared DB).
//
// Discriminant = capability CapMatchObjectiveStats (match.objective.stats).
// Halo Infinite la déclare supported ; Halo 5 not_exposed (le carnage report h5
// n'a aucun sous-objet par mode → la table reste vide pour ce titre). Les
// surfaces dont l'axe « Objectifs » dépend de cette source (profil de
// participation Ascension notamment) court-circuitent le calcul et RETIRENT
// l'axe au lieu d'interroger une table structurellement vide. 100 %
// title-agnostic (jamais de slug== — archlint no_slug_comparison). Même pattern
// que live_service_source.go (resolver optionnel + helper package-level +
// défaut byte-identique Infinite).

// ProvidesObjectiveStatsFromResolver indique si le titre expose la source
// match_objective_stats, résolu via la capability CapMatchObjectiveStats.
//
// Défaut TRUE (byte-identique Halo Infinite) quand le resolver est nil / ne
// supporte pas l'extension CapabilityResolver / le titre n'a aucune capability
// déclarée. Un titre qui DÉCLARE ses capabilities sans match.objective.stats
// (ex. Halo 5) → FALSE.
func ProvidesObjectiveStatsFromResolver(res EndpointResolver, slug string) bool {
	cr, ok := res.(CapabilityResolver)
	if !ok {
		return true
	}
	caps, found := cr.CapabilitiesFor(slug)
	if !found {
		return true
	}
	return caps.Has(CapMatchObjectiveStats)
}

// ProvidesObjectiveStats résout via le resolver partagé posé au boot
// (DefaultEndpointResolver). Point d'entrée des callers qui disposent du slug
// du titre mais pas du resolver (progression/profile).
func ProvidesObjectiveStats(slug string) bool {
	return ProvidesObjectiveStatsFromResolver(DefaultEndpointResolver(), slug)
}
