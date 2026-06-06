// Package analysis — combat_yield.go : calcul du rendement combat (S56).
//
// Portage des formules définies dans DAMAGE_EFFICIENCY_INTEGRATION.md.
// offensive_conversion = 225 * (kills + assists/3) / damage_dealt
// defensive_resistance = damage_taken / (225 * deaths)
// offensive_finishing   = 225 * kills / damage_dealt  (diagnostic uniquement)
package analysis

import "levelup/go-api/internal/domain"

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

// assistFragWeight : convention officielle Halo Infinite — 1 assist = 1/3 de frag.
// Coefficient unique partagé par OffensiveConversion (numérateur) et le dégâts par
// frag-équivalent (dénominateur), pour que % et chiffre affiché restent l'inverse exact.
const assistFragWeight = 3.0

// FragEquivalents = frags + assists/3. Dénominateur commun au rendement offensif
// (OffensiveConversion) et au dégâts par frag-équivalent affiché.
func FragEquivalents(kills, assists float64) float64 {
	return kills + assists/assistFragWeight
}

// DamagePerFragEquivalent = dégâts / (frags + assists/3). C'est l'inverse exact du
// rendement offensif normalisé : OffensiveConversion = 225 / DamagePerFragEquivalent.
// Retourne 0 si le dénominateur est nul ou négatif.
func DamagePerFragEquivalent(damageDealt, kills, assists float64) float64 {
	fe := FragEquivalents(kills, assists)
	if fe <= 0 {
		return 0
	}
	return damageDealt / fe
}

// ComputeCombatYield calcule le rendement combat depuis les stats brutes d'un match.
//
// Cas limites : retourne 0 si les dénominateurs sont nuls ou négatifs.
// Le coefficient 1/3 pour les assists est la convention officielle Halo Infinite.
func ComputeCombatYield(kills, assists int, damageDlt, damageTkn float64, deaths int) CombatYield {
	return ComputeCombatYieldFloat(float64(kills), float64(assists), damageDlt, damageTkn, float64(deaths))
}

// ComputeCombatYieldFloat est la variante en flottants des mêmes formules, pour
// des entrées déjà agrégées (ex. moyennes par partie d'un service record / des
// stats normalisées). Mêmes garde-fous : dénominateurs nuls/négatifs → 0.
func ComputeCombatYieldFloat(kills, assists, damageDlt, damageTkn, deaths float64) CombatYield {
	var cy CombatYield
	if damageDlt > 0 {
		cy.OffensiveConversion = 225.0 * FragEquivalents(kills, assists) / damageDlt
		cy.OffensiveFinishing = 225.0 * kills / damageDlt
	}
	if damageTkn > 0 && deaths > 0 {
		cy.DefensiveResistance = damageTkn / (225.0 * deaths)
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

// minMatchesForCombatStyle : nombre minimum de matchs pour afficher les descripteurs.
const minMatchesForCombatStyle = 15

// ClassifyCombatProfile construit un CombatProfileBlock depuis des métriques agrégées.
//
// avgResidualBrut est nil si engagement_score_brut n'est pas disponible
// (Phase 4 du plan PLAN_COMBAT_PROFILE_WIRING.md non encore livrée).
// Dans ce cas StyleActivity est toujours nil.
func ClassifyCombatProfile(avgOC, avgDR float64, avgResidualBrut *float64, matchCount int) domain.CombatProfileBlock {
	block := domain.CombatProfileBlock{
		AvgOC:           avgOC,
		AvgDR:           avgDR,
		AvgResidualBrut: avgResidualBrut,
		MatchCount:      matchCount,
	}
	if matchCount < minMatchesForCombatStyle {
		return block
	}
	off := classifyOffensive(avgOC)
	block.StyleOffensive = &off
	def := classifyDefensive(avgDR)
	block.StyleDefensive = &def
	if avgResidualBrut != nil {
		act := classifyActivity(*avgResidualBrut)
		block.StyleActivity = &act
	}
	return block
}

// classifyOffensive retourne le style offensif selon avgOC vs OffensiveConversionP80.
func classifyOffensive(avgOC float64) domain.CombatStyle {
	if avgOC >= OffensiveConversionP80 {
		return domain.CombatStyleOffensivePrecis
	}
	if avgOC >= OffensiveConversionP80*0.70 {
		return domain.CombatStyleOffensiveEquilibre
	}
	return domain.CombatStyleOffensiveGenereux
}

// classifyDefensive retourne le style défensif selon avgDR vs DefensiveResistanceP80.
func classifyDefensive(avgDR float64) domain.CombatStyle {
	if avgDR >= DefensiveResistanceP80 {
		return domain.CombatStyleDefensiveResistant
	}
	if avgDR >= DefensiveResistanceP80*0.70 {
		return domain.CombatStyleDefensiveSolide
	}
	return domain.CombatStyleDefensiveFragile
}

// classifyActivity retourne le style activité depuis engagement_score_brut.
// ResidualBrut > 0 = au-dessus du rythme lobby, < 0 = en retrait.
// Seuil ±5 basé sur la distribution typique (calibrable).
func classifyActivity(avgResidualBrut float64) domain.CombatStyle {
	if avgResidualBrut > 5 {
		return domain.CombatStyleActivityActif
	}
	if avgResidualBrut >= -5 {
		return domain.CombatStyleActivityModere
	}
	return domain.CombatStyleActivityDiscret
}
