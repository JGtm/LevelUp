package games

import (
	"time"

	"levelup/go-api/internal/games/mappings"
)

// career_xp.go — résolution title-aware des éras de multiplicateur d'XP de carrière
// (constants.toml [[career_xp_eras]]) et de la capability analytics.career_xp_estimate.
// Miroir du pattern games.EffectiveHpToKill (damage_model) : un titre déclare SES éras
// dans SON constants.toml ; le défaut reste Halo Infinite, byte-identique. La fonction
// de calcul est PURE (analysis.EstimateCareerXP) — ce package ne fait que la résolution.

// careerXPDoublingCutover — bascule ×1 → ×2 de l'XP de carrière Halo Infinite :
// Operation: Infinite, 18 novembre 2025, minuit UTC (« the Applied Score multipliers
// for all matchmaking playlists will be doubled », doublement permanent). Borne à
// minuit UTC : l'imprécision d'heure de déploiement est négligeable par match.
var careerXPDoublingCutover = time.Date(2025, 11, 18, 0, 0, 0, 0, time.UTC)

// DefaultCareerXPEras — éras d'XP de carrière Halo Infinite (défaut byte-identique
// quand le resolver n'est pas câblé ou que le titre ne déclare pas [[career_xp_eras]]).
// ×1 avant le 2025-11-18, ×2 depuis.
func DefaultCareerXPEras() []mappings.CareerXPEra {
	return []mappings.CareerXPEra{
		{From: time.Time{}, To: careerXPDoublingCutover, Multiplier: 1.0},
		{From: careerXPDoublingCutover, To: time.Time{}, Multiplier: 2.0},
	}
}

// CareerXPErasResolver est l'extension OPTIONNELLE d'EndpointResolver qui résout les
// éras d'XP de carrière d'un titre (constants.toml [[career_xp_eras]]). Implémentée
// par *MappingsEndpointResolver. Un resolver qui ne l'implémente pas (stub de test) →
// fallback DefaultCareerXPEras.
type CareerXPErasResolver interface {
	CareerXPErasFor(slug string) ([]mappings.CareerXPEra, bool)
}

// CareerXPErasFromResolver résout les éras d'un titre via un resolver, avec fallback
// DefaultCareerXPEras (resolver nil / ne supportant pas l'extension / titre inconnu /
// éras non déclarées). Point d'injection testable.
func CareerXPErasFromResolver(res EndpointResolver, slug string) []mappings.CareerXPEra {
	if r, ok := res.(CareerXPErasResolver); ok {
		if eras, found := r.CareerXPErasFor(slug); found && len(eras) > 0 {
			return eras
		}
	}
	return DefaultCareerXPEras()
}

// CareerXPErasFor résout les éras d'XP de carrière du titre via le resolver partagé
// posé au boot (DefaultEndpointResolver), fallback DefaultCareerXPEras. Point d'entrée
// du service Timeseries (qui dispose du slug).
func CareerXPErasFor(slug string) []mappings.CareerXPEra {
	return CareerXPErasFromResolver(DefaultEndpointResolver(), slug)
}

// ProvidesCareerXPEstimate indique si le titre EXPOSE la série « XP de carrière
// (estimée) » (capability analytics.career_xp_estimate). Défaut FALSE : c'est un
// analytic OPT-IN, déclaré explicitement (Halo Infinite uniquement) ; un titre dont
// les capabilities ne portent pas cette clé (Halo 5 : système Spartan Rank distinct)
// ne l'expose pas. Source = capability, JAMAIS de slug== (règle title-agnostic).
func ProvidesCareerXPEstimate(slug string) bool {
	return ProvidesCareerXPEstimateFromResolver(DefaultEndpointResolver(), slug)
}

// ProvidesCareerXPEstimateFromResolver est la forme testable de ProvidesCareerXPEstimate.
// Défaut FALSE si le resolver est nil / ne supporte pas l'extension CapabilityResolver /
// le titre n'a aucune capability déclarée (opt-in strict, contrairement aux traits du
// modèle de dégâts qui supposent Infinite par défaut).
func ProvidesCareerXPEstimateFromResolver(res EndpointResolver, slug string) bool {
	cr, ok := res.(CapabilityResolver)
	if !ok {
		return false
	}
	caps, found := cr.CapabilitiesFor(slug)
	if !found {
		return false
	}
	return caps.Has(CapAnalyticsCareerXPEstimate)
}
