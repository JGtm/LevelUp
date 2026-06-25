package games

// career_progression_source.go — gating title-aware de la SOURCE LIVE de
// progression carrière (endpoint economy `careerranks`).
//
// Pourquoi ce gating : l'endpoint live `GET economy/<prefix>/careerranks/...`
// (sync.HaloAPIClient.GetCareerProgress) est le catalogue de rangs de carrière
// Halo Infinite (paliers + XP-pour-palier-suivant). Il n'existe PAS pour tous
// les titres : Halo 5 dérive son Spartan Rank (SR) de la carnage
// (halo_5.SpartanRankProgression / applySpartanRank), persisté directement dans
// career_progression au sync — il n'a pas de catalogue economy careerranks.
//
// Le token Spartan d'un titre comme Halo 5 reste un token Infinite VALIDE :
// appeler l'endpoint careerranks Infinite avec ce token RÉUSSIT et renvoie la
// carrière INFINITE du joueur → contamination cross-titre (rang/XP Infinite
// écrits dans la player DB du titre courant). Le seul garde-fou correct est de
// NE PAS appeler cet endpoint pour un titre qui n'expose pas le catalogue.
//
// Discriminant = capability CapCareerRankCatalog (career.rank_catalog), qui
// décrit EXACTEMENT cette source ("catalogue table-backed avec icône par
// palier ; Infinite ; absent pour h5, SR numérique", cf. adapter.go). 100%
// title-agnostic : on lit la capability du titre via le resolver partagé,
// jamais son slug (archlint no_slug_comparison). Suit le pattern de
// damage_model.go (resolver optionnel + helper package-level + défaut
// byte-identique Infinite).

// CapabilityResolver est l'extension OPTIONNELLE d'EndpointResolver qui résout
// la CapabilityMap d'un titre (capabilities.toml). Implémentée par
// *MappingsEndpointResolver. Un resolver qui ne l'implémente pas (stub de test,
// resolver nil) → fallback "supposer Infinite" via les helpers ci-dessous.
// Séparée d'EndpointResolver pour ne pas casser les implémentations existantes.
type CapabilityResolver interface {
	CapabilitiesFor(slug string) (CapabilityMap, bool)
}

// ProvidesLiveCareerProgressionFromResolver indique si le titre expose la SOURCE
// LIVE de progression carrière (endpoint economy careerranks), résolu via la
// capability CapCareerRankCatalog du titre.
//
// Défaut TRUE (byte-identique Halo Infinite) quand le resolver est nil / ne
// supporte pas l'extension / le titre n'a aucune capability déclarée : préserve
// le chemin live historique pour les tests et instances mono-titre qui ne
// câblent pas de resolver de capabilities. Un titre qui DÉCLARE ses capabilities
// sans career.rank_catalog (ex. Halo 5) → FALSE : on court-circuite le fetch
// live careerranks (qui renverrait la carrière Infinite via le token partagé).
func ProvidesLiveCareerProgressionFromResolver(res EndpointResolver, slug string) bool {
	cr, ok := res.(CapabilityResolver)
	if !ok {
		return true
	}
	caps, found := cr.CapabilitiesFor(slug)
	if !found {
		// Titre sans capabilities déclarées : supposer Infinite (défaut sûr).
		return true
	}
	return caps.Has(CapCareerRankCatalog)
}

// ProvidesLiveCareerProgression résout via le resolver partagé posé au boot
// (DefaultEndpointResolver). Point d'entrée des callers (CareerLiveService) qui
// disposent du slug du titre mais pas du resolver.
func ProvidesLiveCareerProgression(slug string) bool {
	return ProvidesLiveCareerProgressionFromResolver(DefaultEndpointResolver(), slug)
}
