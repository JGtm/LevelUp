// Package analysis — combat_yield.go : calcul du rendement combat (S56).
//
// Portage des formules définies dans DAMAGE_EFFICIENCY_INTEGRATION.md.
// offensive_conversion = effectiveHpToKill * (kills + assists/3) / damage_dealt
// defensive_resistance = damage_taken / (effectiveHpToKill * deaths)
// offensive_finishing   = effectiveHpToKill * kills / damage_dealt  (diagnostic uniquement)
//
// effectiveHpToKill = PV effectifs pour tuer un joueur (baseline title-spécifique,
// 225 Halo Infinite). PARAMÈTRE : la fonction reste pure ; le caller résout la
// valeur du titre via games.EffectiveHpToKill(slug) et l'injecte (cf.
// PLAN_DAMAGE_MODEL_PER_TITLE.md). Externalise le littéral 225 câblé.
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

// CombatDRReferenceThreshold — seuil de référence DR (résistance défensive)
// partagé par le coach (alerte « combat fragile » + traduction en signal), le
// détecteur de jalons combat (combat_endurance / combat_excellence) et le
// backfill des dates de franchissement de ces jalons.
//
// Volontairement DISTINCT de DefensiveResistanceP80 (1,65) ci-dessus : ce
// dernier normalise l'échelle visuelle des jauges, alors que celui-ci déclenche
// un conseil ou débloque un jalon et vise donc un niveau ATTEIGNABLE. Les
// aligner changerait la sensibilité des alertes — ce n'est PAS une dérive à
// corriger. Le suffixe « P80 » des noms locaux historiques (combatDRP80,
// drMilestoneThreshold…) se lit « seuil de référence », pas « percentile 80 de
// la population ».
//
// Source unique (CLAUDE.md règle 6) — garde-rail interdisant toute nouvelle
// copie du littéral : internal/archlint/no_raw_combat_dr_threshold_test.go.
const CombatDRReferenceThreshold = 1.59

// CombatOCReferenceThreshold — seuil de référence OC (conversion offensive),
// jumeau de CombatDRReferenceThreshold ci-dessus. Partagé par le coach (alertes
// « combat actif » / « combat précis »), le détecteur de jalons combat
// (combat_precision / combat_excellence) et le backfill des dates de
// franchissement de ces jalons.
//
// Volontairement DISTINCT de OffensiveConversionP80 (0,90) : ce dernier
// normalise l'échelle visuelle des jauges (NormalizeForBar), alors que celui-ci
// déclenche un conseil ou débloque un palier de progression et vise donc un
// niveau ATTEIGNABLE régulièrement. Les aligner changerait la difficulté des
// paliers et la sensibilité des alertes — ce n'est PAS une dérive à corriger.
// Le suffixe « P80 » des noms locaux historiques (combatOCP80,
// ocMilestoneThreshold…) se lit « seuil de référence », pas « percentile 80 de
// la population ».
//
// Source unique (CLAUDE.md règle 6) — garde-rail interdisant toute nouvelle
// copie du littéral : internal/archlint/no_raw_combat_oc_threshold_test.go.
const CombatOCReferenceThreshold = 0.83

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
// rendement offensif normalisé : OffensiveConversion = effectiveHpToKill / DamagePerFragEquivalent.
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
// effectiveHpToKill = PV effectifs pour tuer (baseline title-spécifique, 225 Infinite ;
// le caller résout via games.EffectiveHpToKill(slug)). Cas limites : retourne 0 si les
// dénominateurs sont nuls ou négatifs. Le coefficient 1/3 pour les assists est la
// convention officielle Halo Infinite.
func ComputeCombatYield(kills, assists int, damageDlt, damageTkn float64, deaths int, effectiveHpToKill float64) CombatYield {
	return ComputeCombatYieldFloat(float64(kills), float64(assists), damageDlt, damageTkn, float64(deaths), effectiveHpToKill)
}

// ComputeCombatYieldFloat est la variante en flottants des mêmes formules, pour
// des entrées déjà agrégées (ex. moyennes par partie d'un service record / des
// stats normalisées). effectiveHpToKill = baseline title-spécifique (225 Infinite).
// Mêmes garde-fous : dénominateurs nuls/négatifs → 0.
func ComputeCombatYieldFloat(kills, assists, damageDlt, damageTkn, deaths, effectiveHpToKill float64) CombatYield {
	var cy CombatYield
	if damageDlt > 0 {
		cy.OffensiveConversion = effectiveHpToKill * FragEquivalents(kills, assists) / damageDlt
		cy.OffensiveFinishing = effectiveHpToKill * kills / damageDlt
	}
	if damageTkn > 0 && deaths > 0 {
		cy.DefensiveResistance = damageTkn / (effectiveHpToKill * deaths)
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
