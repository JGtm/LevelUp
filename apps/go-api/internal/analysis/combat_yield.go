// Package analysis — combat_yield.go : calcul du rendement combat (S56).
//
// Portage des formules définies dans DAMAGE_EFFICIENCY_INTEGRATION.md.
// offensive_conversion = 225 * (kills + assists/3) / damage_dealt
// defensive_resistance = damage_taken / (225 * deaths)
// offensive_finishing   = 225 * kills / damage_dealt  (diagnostic uniquement)
package analysis

import (
	"sync/atomic"

	"levelup/go-api/internal/domain"
)

// excludeAssistsFromYield : reglage global Settings (rendement_exclude_assists).
// Quand true, OffensiveConversion = 225*kills/damage (assists ignores) sur TOUS
// les composants rendement (Home, Timeseries, Sessions, Explorer, Escouade,
// Match view). C'est un reglage app UNIQUE (pas par-user) ; on le porte donc en
// variable globale atomique volontaire, mise a jour au boot et a chaque PATCH
// /settings (cf. SetExcludeAssistsFromYield), pour eviter de threader le flag
// dans ~13 agregateurs purs et leurs appelants. Ce n'est PAS un guard de
// compatibilite. Defaut false (assists comptes a 1/3, convention Halo).
var excludeAssistsFromYield atomic.Bool

// SetExcludeAssistsFromYield met a jour le reglage global du rendement combat.
// Appele au boot (depuis app_settings) et apres chaque PATCH /settings.
func SetExcludeAssistsFromYield(v bool) { excludeAssistsFromYield.Store(v) }

// AssistsExcludedFromYield retourne l'etat courant du reglage (lecture seule,
// ex. pour adapter un libelle UI). Thread-safe.
func AssistsExcludedFromYield() bool { return excludeAssistsFromYield.Load() }

// CombatYield regroupe les métriques de rendement combat d'un joueur pour un match.
type CombatYield struct {
	OffensiveConversion float64 // conversion offensive (assists/3 inclus)
	DefensiveResistance float64 // résistance défensive
	OffensiveFinishing  float64 // variante stricte kills-only (diagnostic)
}

// Repère de normalisation des barres OC/DR — frontière élite mondiale (cf.
// .ai/PLAN_COMBAT_PROFILE_RECALIBRATION.md). Sert UNIQUEMENT à NormalizeForBar
// (échelle visuelle des jauges) ; distinct des bandes de classification.
const (
	OffensiveConversionP80 = 0.90
	DefensiveResistanceP80 = 1.65
	CombatYieldClipFactor  = 1.5 // clippage à 1.5× le repère
)

// assistFragWeight : convention officielle Halo Infinite — 1 assist = 1/3 de frag.
// Coefficient unique partagé par OffensiveConversion (numérateur) et le dégâts par
// frag-équivalent (dénominateur), pour que % et chiffre affiché restent l'inverse exact.
const assistFragWeight = 3.0

// FragEquivalents = frags + assists/3. Dénominateur commun au rendement offensif
// (OffensiveConversion) et au dégâts par frag-équivalent affiché.
//
// Si le réglage global excludeAssistsFromYield est actif, les assists sont
// ignorés (FragEquivalents = kills) → OffensiveConversion = OffensiveFinishing
// sur tous les composants rendement, sans re-câblage par site.
func FragEquivalents(kills, assists float64) float64 {
	if excludeAssistsFromYield.Load() {
		return kills
	}
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

// bandThreshold associe une borne basse (incluse) à un style. Les bandes sont
// triées par borne décroissante ; classifyBand retient la première satisfaite.
type bandThreshold struct {
	min   float64
	style domain.CombatStyle
}

// Bornes de classification — 5 bandes par axe, calibrées sur la distribution des
// world leaders (top-100 mondial, table world_player_season_stats) + validation
// terrain (cf. .ai/PLAN_COMBAT_PROFILE_RECALIBRATION.md). DISTINCTES des
// constantes P80 ci-dessus, qui ne servent qu'à la normalisation visuelle des
// barres (NormalizeForBar).
var (
	offensiveBands = []bandThreshold{
		{0.90, domain.CombatStyleOffensiveChirurgical},
		{0.85, domain.CombatStyleOffensivePrecis},
		{0.81, domain.CombatStyleOffensiveEquilibre},
		{0.78, domain.CombatStyleOffensiveIrregulier},
		{0, domain.CombatStyleOffensiveDisperse},
	}
	defensiveBands = []bandThreshold{
		{1.65, domain.CombatStyleDefensiveInebranlable},
		{1.50, domain.CombatStyleDefensiveResistant},
		{1.35, domain.CombatStyleDefensiveSolide},
		{1.20, domain.CombatStyleDefensiveExpose},
		{0, domain.CombatStyleDefensiveFragile},
	}
	// Activité = engagement absolu pace_joueur/pace_lobby (1.0 = au rythme du lobby).
	activityBands = []bandThreshold{
		{1.25, domain.CombatStyleActivityAgressif},
		{1.08, domain.CombatStyleActivityActif},
		{0.92, domain.CombatStyleActivityMesure},
		{0.80, domain.CombatStyleActivityDiscret},
		{0, domain.CombatStyleActivityPassif},
	}
)

// classifyBand retourne le style de la première bande dont la borne basse est
// atteinte (bandes triées décroissant). Fallback : la bande la plus basse.
func classifyBand(v float64, bands []bandThreshold) domain.CombatStyle {
	for _, b := range bands {
		if v >= b.min {
			return b.style
		}
	}
	return bands[len(bands)-1].style
}

// ClassifyCombatProfile construit un CombatProfileBlock depuis des métriques agrégées.
//
// avgPaceRatio = engagement absolu (pace_joueur / pace_lobby moyen ; 1.0 = au
// rythme du lobby). Nil si les paces d'engagement ne sont pas disponibles →
// StyleActivity reste nil (dégradation gracieuse, ex. titre sans events).
func ClassifyCombatProfile(avgOC, avgDR float64, avgPaceRatio *float64, matchCount int) domain.CombatProfileBlock {
	block := domain.CombatProfileBlock{
		AvgOC:        avgOC,
		AvgDR:        avgDR,
		AvgPaceRatio: avgPaceRatio,
		MatchCount:   matchCount,
	}
	if matchCount < minMatchesForCombatStyle {
		return block
	}
	off := classifyOffensive(avgOC)
	block.StyleOffensive = &off
	def := classifyDefensive(avgDR)
	block.StyleDefensive = &def
	if avgPaceRatio != nil {
		act := classifyActivity(*avgPaceRatio)
		block.StyleActivity = &act
	}
	return block
}

// classifyOffensive / classifyDefensive / classifyActivity — 5 bandes chacune.
func classifyOffensive(avgOC float64) domain.CombatStyle { return classifyBand(avgOC, offensiveBands) }
func classifyDefensive(avgDR float64) domain.CombatStyle { return classifyBand(avgDR, defensiveBands) }
func classifyActivity(avgPaceRatio float64) domain.CombatStyle {
	return classifyBand(avgPaceRatio, activityBands)
}
