// Package analysis — combat_yield.go : calcul du rendement combat (S56).
//
// Portage des formules définies dans DAMAGE_EFFICIENCY_INTEGRATION.md.
// offensive_conversion = 225 * (kills + assists/3) / damage_dealt
// defensive_resistance = damage_taken / (225 * deaths)
// offensive_finishing   = 225 * kills / damage_dealt  (diagnostic uniquement)
package analysis

// CombatYield regroupe les métriques de rendement combat d'un joueur pour un match.
type CombatYield struct {
	OffensiveConversion float64 // conversion offensive (assists/3 inclus)
	DefensiveResistance float64 // résistance défensive
	OffensiveFinishing  float64 // variante stricte kills-only (diagnostic)
}

// P80 observés sur les données réelles (avril 2026, 4 joueurs, hors bots).
// Utilisés pour la normalisation de CombatYieldBar.
const (
	OffensiveConversionP80 = 0.83
	DefensiveResistanceP80 = 1.59
	CombatYieldClipFactor  = 1.5 // clippage à 1.5× p80
)

// ComputeCombatYield calcule le rendement combat depuis les stats brutes d'un match.
//
// Cas limites : retourne 0 si les dénominateurs sont nuls ou négatifs.
// Le coefficient 1/3 pour les assists est la convention officielle Halo Infinite.
func ComputeCombatYield(kills, assists int, damageDlt, damageTkn float64, deaths int) CombatYield {
	var cy CombatYield
	if damageDlt > 0 {
		cy.OffensiveConversion = 225.0 * (float64(kills) + float64(assists)/3.0) / damageDlt
		cy.OffensiveFinishing = 225.0 * float64(kills) / damageDlt
	}
	if damageTkn > 0 && deaths > 0 {
		cy.DefensiveResistance = damageTkn / (225.0 * float64(deaths))
	}
	return cy
}

// NormalizeForBar normalise une valeur par son p80 et clipe à CombatYieldClipFactor.
// Retourne une valeur dans [0, CombatYieldClipFactor].
func NormalizeForBar(value, p80 float64) float64 {
	if p80 <= 0 {
		return 0
	}
	n := value / p80
	if n > CombatYieldClipFactor {
		return CombatYieldClipFactor
	}
	if n < 0 {
		return 0
	}
	return n
}

// TooltipDamagePer calcule les dégâts bruts par unité (pour tooltip).
// Retourne dmg/kills (offensive) ou dmg/deaths (defensive).
func TooltipDamagePer(damage float64, count int) float64 {
	if count <= 0 {
		return 0
	}
	return damage / float64(count)
}
