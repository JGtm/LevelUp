package games

import "levelup/go-api/internal/games/mappings"

// damage_model.go — résolution title-aware du modèle de dégâts (PV effectifs pour
// tuer un joueur), baseline des KPI rendement/résistance (offensive_conversion =
// effective_hp × frag-équivalents / dégâts ; defensive_resistance = dégâts_subis /
// (effective_hp × morts)). Externalise le littéral `225` câblé Infinite (cf.
// PLAN_DAMAGE_MODEL_PER_TITLE.md) : un titre déclare SA valeur dans
// constants.toml [damage_model] ; le défaut reste Halo Infinite, byte-identique.

// DefaultEffectiveHpToKill — PV effectifs pour tuer un Spartan Halo Infinite
// (90 vie + 135 bouclier, échelle de dégâts de l'API). Défaut byte-identique
// quand le resolver n'est pas câblé ou que le titre ne déclare pas son
// [damage_model] (ex. mono-titre, tests). Halo 5 = 115 (bouclier 70 + armure 45).
const DefaultEffectiveHpToKill = 225.0

// DamageModelResolver est l'extension OPTIONNELLE d'EndpointResolver qui résout les
// constantes de modèle de dégâts d'un titre (constants.toml [damage_model]).
// Implémentée par *MappingsEndpointResolver. Un resolver qui ne l'implémente pas
// (stub de test) → fallback DefaultEffectiveHpToKill via les helpers ci-dessous.
// Séparée d'EndpointResolver pour ne pas casser les implémentations existantes.
type DamageModelResolver interface {
	DamageModelFor(slug string) (mappings.DamageModelConstants, bool)
}

// EffectiveHpToKillFromResolver résout les PV-pour-tuer d'un titre via un resolver,
// avec fallback DefaultEffectiveHpToKill (resolver nil / ne supportant pas
// l'extension / titre inconnu / modèle non déclaré). Point d'injection testable.
func EffectiveHpToKillFromResolver(res EndpointResolver, slug string) float64 {
	if dmr, ok := res.(DamageModelResolver); ok {
		if dm, found := dmr.DamageModelFor(slug); found && dm.EffectiveHpToKill > 0 {
			return dm.EffectiveHpToKill
		}
	}
	return DefaultEffectiveHpToKill
}

// EffectiveHpToKill résout les PV-pour-tuer du titre via le resolver partagé posé
// au boot (DefaultEndpointResolver), fallback DefaultEffectiveHpToKill. Point
// d'entrée des callers compute (combat_yield, squad_breakdown, post-sync) qui
// disposent du slug du titre mais pas du resolver.
func EffectiveHpToKill(slug string) float64 {
	return EffectiveHpToKillFromResolver(DefaultEndpointResolver(), slug)
}

// ProvidesNativeKDA indique si le titre fournit un KDA per-match via son API (donc
// utilisable/affichable tel quel). Faux pour les titres qui n'en renvoient pas —
// Halo 5 : forme native = FDA NET ((k+a/3)−d)/games, distincte du quotient KDA ;
// fabriquer un KDA façon Infinite pour eux produirait une valeur fausse. Ces titres
// déclarent no_native_kda = true dans constants.toml [damage_model]. Défaut true
// (byte-identique Infinite). Source = config, JAMAIS de slug== (règle title-agnostic).
//
// RÈGLE ABSOLUE associée : on ne CALCULE jamais le KDA d'un titre qui le fournit
// (on lit l'API) ; pour un titre qui ne le fournit pas, on le laisse nil plutôt que
// d'appliquer une formule étrangère.
func ProvidesNativeKDA(slug string) bool {
	return ProvidesNativeKDAFromResolver(DefaultEndpointResolver(), slug)
}

// ProvidesNativeKDAFromResolver est la forme testable de ProvidesNativeKDA (point
// d'injection du resolver). Défaut true si le resolver est nil / ne supporte pas
// l'extension / le titre ne déclare pas son [damage_model].
func ProvidesNativeKDAFromResolver(res EndpointResolver, slug string) bool {
	if dmr, ok := res.(DamageModelResolver); ok {
		if dm, found := dmr.DamageModelFor(slug); found {
			return !dm.NoNativeKDA
		}
	}
	return true
}
