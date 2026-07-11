package games

import (
	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/games/mappings"
)

// engagement_calibration.go — résolution title-aware des poids d'events du score
// d'engagement (chantier F7, DE-4). Externalise les poids (levier de calibration
// dépendant du gameplay) de constants.toml [engagement] ; le défaut reste
// temporal.DefaultEventWeights (byte-identique Infinite). Un titre déclare SES poids
// dans SON constants.toml ; le moteur temporal reste agnostic (il reçoit les poids
// en entrée). Miroir du pattern games.EffectiveHpToKill (damage_model).

// EngagementResolver est l'extension OPTIONNELLE d'EndpointResolver qui résout les
// poids d'engagement d'un titre (constants.toml [engagement]). Implémentée par
// *MappingsEndpointResolver. Un resolver qui ne l'implémente pas (stub de test) →
// fallback temporal.DefaultEventWeights.
type EngagementResolver interface {
	EngagementFor(slug string) (mappings.EngagementConstants, bool)
}

// EngagementWeightsFromResolver résout les poids d'engagement d'un titre via un
// resolver, avec fallback temporal.DefaultEventWeights (resolver nil / titre inconnu
// / section non déclarée). Point d'injection testable.
func EngagementWeightsFromResolver(res EndpointResolver, slug string) temporal.EventWeights {
	if er, ok := res.(EngagementResolver); ok {
		if e, found := er.EngagementFor(slug); found {
			return temporal.EventWeights{
				Objective: e.Objective,
				Assist:    e.Assist,
				Death:     e.Death,
				Default:   e.Default,
			}
		}
	}
	return temporal.DefaultEventWeights()
}

// EngagementWeightsFor résout les poids d'engagement du titre via le resolver
// partagé posé au boot (DefaultEndpointResolver), fallback temporal.DefaultEventWeights.
// Point d'entrée des points de collecte (service + sync) qui disposent du slug.
func EngagementWeightsFor(slug string) temporal.EventWeights {
	return EngagementWeightsFromResolver(DefaultEndpointResolver(), slug)
}
