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
// [damage_model] (ex. mono-titre, tests). Halo 5 = 115 (design : bouclier 70 +
// armure 45 ; PV-pour-tuer réels, PAS la moyenne empirique dégâts/kill —
// cf. config/titles/halo_5/constants.toml).
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

// DefaultOffensiveConversionP80 — frontière élite OC (80e percentile) Halo Infinite,
// repère de normalisation des barres/radars de rendement. Défaut quand le titre ne
// déclare pas son offensive_conversion_p80. Aligné sur analysis.OffensiveConversionP80
// (mais games ne peut pas importer analysis → littéral dupliqué, gardé en sync).
const DefaultOffensiveConversionP80 = 0.90

// OffensiveConversionP80FromResolver résout la frontière élite OC d'un titre via un
// resolver, fallback DefaultOffensiveConversionP80. Point d'injection testable.
func OffensiveConversionP80FromResolver(res EndpointResolver, slug string) float64 {
	if dmr, ok := res.(DamageModelResolver); ok {
		if dm, found := dmr.DamageModelFor(slug); found && dm.OffensiveConversionP80 > 0 {
			return dm.OffensiveConversionP80
		}
	}
	return DefaultOffensiveConversionP80
}

// OffensiveConversionP80 résout la frontière élite OC du titre (repère des barres/radars
// de rendement) via le resolver partagé, fallback DefaultOffensiveConversionP80. Point
// d'entrée des callers d'AFFICHAGE qui disposent du slug. NB : le chemin LUSR garde la
// constante analysis.OffensiveConversionP80 (LUSR = Infinite-only, ne doit pas varier).
func OffensiveConversionP80(slug string) float64 {
	return OffensiveConversionP80FromResolver(DefaultEndpointResolver(), slug)
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

// ProvidesDamageTaken indique si le titre fournit damage_taken (dégâts subis) via
// son API. Faux pour Halo 5 (no_damage_taken=true) → la résistance défensive
// (DR = dégâts_subis/(hp×morts)) n'est pas calculable ; les surfaces qui en
// dépendent (profil de combat axe défensif, coaching « fragile », milestones
// endurance/excellence, KPI résistance) la neutralisent plutôt que d'afficher un
// DR=0 trompeur. Défaut true (Infinite). Source = config, JAMAIS de slug==.
func ProvidesDamageTaken(slug string) bool {
	return ProvidesDamageTakenFromResolver(DefaultEndpointResolver(), slug)
}

// ProvidesDamageTakenFromResolver est la forme testable de ProvidesDamageTaken.
// Défaut true si resolver nil / sans extension / titre sans [damage_model].
func ProvidesDamageTakenFromResolver(res EndpointResolver, slug string) bool {
	if dmr, ok := res.(DamageModelResolver); ok {
		if dm, found := dmr.DamageModelFor(slug); found {
			return !dm.NoDamageTaken
		}
	}
	return true
}

// ProvidesMMR indique si le titre fournit un MMR (matchmaking rating) via son API.
// Faux pour Halo 5 (no_mmr=true) : PreMatch/PostMatchRatings sont servis null partout
// (vérifié 25/25 matchs, classés inclus — sonde 2026-06-26), aucun MMR brut. Les
// surfaces qui en dépendent (team_mmr/enemy_mmr/delta_mmr du MatchSummary, highlight
// « plus belle victoire underdog ») doivent l'OMETTRE plutôt que d'afficher du vide.
// Défaut true (Infinite). Source = config, JAMAIS de slug== (règle title-agnostic).
func ProvidesMMR(slug string) bool {
	return ProvidesMMRFromResolver(DefaultEndpointResolver(), slug)
}

// ProvidesMMRFromResolver est la forme testable de ProvidesMMR. Défaut true si
// resolver nil / sans extension / titre sans [damage_model].
func ProvidesMMRFromResolver(res EndpointResolver, slug string) bool {
	if dmr, ok := res.(DamageModelResolver); ok {
		if dm, found := dmr.DamageModelFor(slug); found {
			return !dm.NoMMR
		}
	}
	return true
}
