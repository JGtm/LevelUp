// Package sync — skill_reexport.go : ré-exports (alias) des symboles du package feuille
// internal/sync/skill (K3c — extraction du cluster skill hors de la racine sync/).
//
// Le cluster skill (rating LUSR/TrueSkill v2, chaîne, métriques) a migré dans
// internal/sync/skill (package FEUILLE : durable_progress + exclusion_filter y sont aussi →
// aucune dépendance skill→sync). sync-root et les consommateurs externes qui utilisaient ces
// symboles via le package sync continuent de fonctionner INCHANGÉS grâce à ces alias ; skill
// est importé UNI-DIRECTIONNELLEMENT (sync → skill, jamais l'inverse).
package sync

import "levelup/go-api/internal/sync/skill"

// Constantes ré-exportées (métriques, seuils, chaînes, tiers).
const (
	MetricKeyKillsVsExpected  = skill.MetricKeyKillsVsExpected
	MetricKeyDeathsVsExpected = skill.MetricKeyDeathsVsExpected
	MetricKeyWinFactor        = skill.MetricKeyWinFactor
	MetricKeyDamageEfficiency = skill.MetricKeyDamageEfficiency
	MetricKeyAccuracyDelta    = skill.MetricKeyAccuracyDelta
	MetricKeyMedalExploit     = skill.MetricKeyMedalExploit
	MetricKeyOffensiveConv    = skill.MetricKeyOffensiveConv
	MetricKeyDefensiveResist  = skill.MetricKeyDefensiveResist
	MetricKeyKPM              = skill.MetricKeyKPM
	MetricKeyDPMDeaths        = skill.MetricKeyDPMDeaths
	MetricKeyAPM              = skill.MetricKeyAPM
	MetricKeyKDA              = skill.MetricKeyKDA
	MetricKeyAccuracy         = skill.MetricKeyAccuracy
	MetricKeyPSPM             = skill.MetricKeyPSPM
	MetricKeyDPMDamage        = skill.MetricKeyDPMDamage
	MetricKeyRankPerf         = skill.MetricKeyRankPerf

	MetricKeyObjectiveParticipation = skill.MetricKeyObjectiveParticipation

	MinMatchesForRating           = skill.MinMatchesForRating
	MinMatchesForAccuracyDelta    = skill.MinMatchesForAccuracyDelta
	MinMatchesForRelative         = skill.MinMatchesForRelative
	MinMatchesPerChainForRelative = skill.MinMatchesPerChainForRelative

	LUSRPlacementThreshold = skill.LUSRPlacementThreshold
	LUSRMaxDelta           = skill.LUSRMaxDelta
	LUSRChainArenaSlayer   = skill.LUSRChainArenaSlayer
	LUSRChainArenaObjectif = skill.LUSRChainArenaObjectif
	LUSRChainBTB           = skill.LUSRChainBTB
	LUSRChainChaos         = skill.LUSRChainChaos

	PerfChainRanked         = skill.PerfChainRanked // valeur stockée historique (cf. skill_config.go)
	PerfChainRankedSlayer   = skill.PerfChainRankedSlayer
	PerfChainRankedObjectif = skill.PerfChainRankedObjectif
	PerfChainFirefight      = skill.PerfChainFirefight

	TierBronze   = skill.TierBronze
	TierSilver   = skill.TierSilver
	TierGold     = skill.TierGold
	TierPlatinum = skill.TierPlatinum
	TierDiamond  = skill.TierDiamond
	TierOnyx     = skill.TierOnyx

	TierLabelBronze    = skill.TierLabelBronze
	TierLabelArgent    = skill.TierLabelArgent
	TierLabelOr        = skill.TierLabelOr
	TierLabelPlatine   = skill.TierLabelPlatine
	TierLabelDiamant   = skill.TierLabelDiamant
	TierLabelOnyx      = skill.TierLabelOnyx
	TierLabelPlacement = skill.TierLabelPlacement

	InitialMU              = skill.InitialMU
	InitialSigma           = skill.InitialSigma
	Beta                   = skill.Beta
	AccuracyHistorySize    = skill.AccuracyHistorySize
	InactivitySigmaPerDay  = skill.InactivitySigmaPerDay
	MaxInactivityDays      = skill.MaxInactivityDays
	InactivityThresholdDay = skill.InactivityThresholdDay

	Tau                  = skill.Tau
	KElo                 = skill.KElo
	MaxSigma             = skill.MaxSigma
	MinSigma             = skill.MinSigma
	MinRating            = skill.MinRating
	DefaultOpponentSigma = skill.DefaultOpponentSigma
	IndividualMUAlpha    = skill.IndividualMUAlpha
)

// Types ré-exportés (alias).
type (
	FormulaSimReport = skill.FormulaSimReport
	LUSRDryRunReport = skill.LUSRDryRunReport
	LUSRChainConfig  = skill.LUSRChainConfig
	SkillTier        = skill.SkillTier
	lusrMatchData    = skill.LusrMatchData
)

// Vars ré-exportées.
var (
	LUSRChains      = skill.LUSRChains
	SkillTiers      = skill.SkillTiers
	RelativeWeights = skill.RelativeWeights
)

// Fonctions ré-exportées (valeur de fonction — les appelants sync-root gardent leur nom
// d'origine ; les impl minuscules sont exportées côté skill sans collision).
var (
	IsLUSRV2Canonical          = skill.IsLUSRV2Canonical
	IsLUSRV2Enabled            = skill.IsLUSRV2Enabled
	RunFormulaSim              = skill.RunFormulaSim
	RunLUSRV2ShadowOwnerOnly   = skill.RunLUSRV2ShadowOwnerOnly
	BatchComputeLUSR           = skill.BatchComputeLUSR
	batchComputeLUSR           = skill.BatchComputeLUSRWithMedals
	batchComputeLUSRPreview    = skill.BatchComputeLUSRPreview
	loadExistingRatingIDs      = skill.LoadExistingRatingIDs
	slugHasLUSR                = skill.SlugHasLUSR
	GetPerformanceChain        = skill.GetPerformanceChain
	RunDualRowSentinel         = skill.RunDualRowSentinel
	computeCombatYield         = skill.ComputeCombatYield
	clampF                     = skill.ClampF
	loadExcludedMatchIDs       = skill.LoadExcludedMatchIDs
	loadObjectiveParticipation = skill.LoadObjectiveParticipation

	CompositeWeights               = skill.CompositeWeights
	WeightsForChain                = skill.WeightsForChain
	FormatTierLabel                = skill.FormatTierLabel
	GetLUSRChain                   = skill.GetLUSRChain
	GetTierForRating               = skill.GetTierForRating
	RunLUSRV2Shadow                = skill.RunLUSRV2Shadow
	SetLUSRChainClassifier         = skill.SetLUSRChainClassifier
	SetLUSRChainClassifierForTitle = skill.SetLUSRChainClassifierForTitle

	// Seam famille objectif (chaîne de performance classée, scission D-A).
	SetObjectiveFamilyClassifier         = skill.SetObjectiveFamilyClassifier
	SetObjectiveFamilyClassifierForTitle = skill.SetObjectiveFamilyClassifierForTitle
	IsObjectiveFamilyForTitle            = skill.IsObjectiveFamilyForTitle

	SimulationVariants                     = skill.SimulationVariants
	DefaultLUSRModeIfUnset                 = skill.DefaultLUSRModeIfUnset
	LogLUSRModeAtBoot                      = skill.LogLUSRModeAtBoot
	ValidateLUSRChainClassifierWired       = skill.ValidateLUSRChainClassifierWired
	ValidateObjectiveFamilyClassifierWired = skill.ValidateObjectiveFamilyClassifierWired
)
