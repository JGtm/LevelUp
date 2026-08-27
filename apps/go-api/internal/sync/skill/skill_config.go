// Package sync — skill_config.go : constantes et tiers LUSR (TrueSkill 2).
//
// Portage de src/analysis/skill_rating_config.py + src/analysis/playlist_groups.py.
// Centralise tous les paramètres numériques de l'algorithme LUSR.
package skill

import "math"

// ── TrueSkill 2 paramètres ─────────────────────────────────────────────────

const (
	// InitialMU est le rating initial (centre de l'échelle).
	InitialMU = 1500.0
	// InitialSigma est la déviation initiale (incertitude maximale).
	InitialSigma = 350.0
	// MinSigma est le plancher de sigma (jamais en dessous).
	MinSigma = 60.0
	// MaxSigma est le plafond de sigma.
	MaxSigma = 350.0
	// Beta est le bruit de performance (β) — sépare skill de variance.
	Beta = 200.0
	// Tau est la dérive dynamique (τ) — sigma augmente de τ entre chaque match.
	Tau = 25.0
	// DrawProbability est la probabilité implicite d'égalité.
	DrawProbability = 0.06
	// MinMatchesForRating est le nombre minimum de matchs pour un rating fiable.
	// (Algorithme local LUSR ; séparé du seuil placement CSR Microsoft.)
	MinMatchesForRating = 10
	// CSRPlacementThresholdDefault est le seuil de matchs de placement CSR par
	// défaut (depuis Season 3, 2023-03-07). Lookup par season_id via
	// metadata.csr_placement_thresholds pour les saisons antérieures (S1-S2 = 10).
	CSRPlacementThresholdDefault = 5
	// LUSRPlacementThreshold est le seuil de matchs requis avant le calcul d'un
	// LUSR stable. Inchangé (algorithme local). Exposé pour parité avec CSR.
	LUSRPlacementThreshold = 10
	// MinRating est le rating minimum global.
	MinRating = 200.0
	// KElo est l'amplitude du changement mu par match.
	KElo = 32.0
)

// ── Inactivité ──────────────────────────────────────────────────────────────

const (
	InactivitySigmaPerDay  = 1.0
	MaxInactivityDays      = 14
	InactivityThresholdDay = 1.0
)

// ── Clés canoniques des métriques skill/performance ─────────────────────────
//
// Utilisées comme clés dans CompositeWeights, RelativeWeights, et le dispatch
// des contributions individuelles (skill_rating.go). Externalisées pour éviter
// les doublons goconst et garantir la cohérence des noms entre les map keys
// et les case du switch.
const (
	MetricKeyKillsVsExpected  = "kills_vs_expected"
	MetricKeyDeathsVsExpected = "deaths_vs_expected"
	MetricKeyWinFactor        = "win_factor"
	MetricKeyDamageEfficiency = "damage_efficiency"
	MetricKeyAccuracyDelta    = "accuracy_delta"
	MetricKeyMedalExploit     = "medal_exploit"
	MetricKeyOffensiveConv    = "offensive_conversion"
	MetricKeyDefensiveResist  = "defensive_resistance"
	MetricKeyKPM              = "kpm"
	MetricKeyDPMDeaths        = "dpm_deaths"
	MetricKeyAPM              = "apm"
	MetricKeyKDA              = "kda"
	MetricKeyAccuracy         = "accuracy"
	MetricKeyPSPM             = "pspm"
	MetricKeyDPMDamage        = "dpm_damage"
	MetricKeyRankPerf         = "rank_perf"
	// MetricKeyObjectiveParticipation (ospm) — points d'awards de catégorie
	// `objective` par minute (source personal_score_awards, DB joueur). Active
	// UNIQUEMENT sur les chaînes de famille objectif (cf. WeightsForChain).
	MetricKeyObjectiveParticipation = "objective_participation"
)

// ── Score composite ─────────────────────────────────────────────────────────

// CompositeWeights pondère les composantes du score composite [0,1].
// Portage de compositeWeights v5 (analysis/skill_rating.go).
// La somme dépasse 1.0 (1.02) : computeCompositeScore renormalise par totalWeight.
var CompositeWeights = map[string]float64{
	MetricKeyKillsVsExpected:  0.27,
	MetricKeyDeathsVsExpected: 0.24,
	MetricKeyWinFactor:        0.05,
	MetricKeyDamageEfficiency: 0.10,
	MetricKeyAccuracyDelta:    0.10,
	MetricKeyMedalExploit:     0.04,
	MetricKeyOffensiveConv:    0.16,
	MetricKeyDefensiveResist:  0.06,
}

const (
	AccuracyHistorySize        = 50
	MinMatchesForAccuracyDelta = 5
	IndividualMUAlpha          = 150.0
	DefaultOpponentSigma       = 150.0
	// LUSRMaxDelta est le guard-rail : cap ±100 pts par match (LUSR).
	LUSRMaxDelta = 100.0
)

// ── Performance score (relatif) ─────────────────────────────────────────────

const MinMatchesForRelative = 10

// MinMatchesPerChainForRelative est le seuil minimum de matchs **dans une
// chaîne** avant de calculer un score relatif pour cette chaîne. Symétrique au
// LUSR (MinMatchesForRating). Pas de fallback global : sans 10 matchs dans la
// chaîne du match courant, on ne calcule pas de score pour préserver la
// sémantique "relatif à ta chaîne".
const MinMatchesPerChainForRelative = 10

// RelativeWeights pondère les métriques du score relatif (0-100).
// Portage de relativeWeights v5 (analysis/performance_score.go).
// Somme = 1.01 → renormalisé automatiquement si des métriques sont absentes.
var RelativeWeights = map[string]float64{
	MetricKeyKPM:              0.14,
	MetricKeyDPMDeaths:        0.10,
	MetricKeyAPM:              0.06,
	MetricKeyKDA:              0.11,
	MetricKeyAccuracy:         0.04,
	MetricKeyPSPM:             0.10,
	MetricKeyDPMDamage:        0.06,
	MetricKeyRankPerf:         0.04,
	MetricKeyKillsVsExpected:  0.09,
	MetricKeyDeathsVsExpected: 0.07,
	MetricKeyMedalExploit:     0.06,
	MetricKeyOffensiveConv:    0.09,
	MetricKeyDefensiveResist:  0.05,
}

// objectiveChainWeights — profil de poids des chaînes de FAMILLE OBJECTIF
// (LUSRChainArenaObjectif, PerfChainRankedObjectif).
//
// FIGÉ au gate 0 du 2026-08-27 (décision D-J de .ai/PLAN_PERF_NOTE_OBJECTIFS.md,
// argumentaire chiffré dans RAPPORT_SIM_PERF_NOTE_2026-08.md) : la participation à
// l'objectif entre à 0.12, financée par les quatre métriques de combat pur
// (kpm 0.14→0.10, kda 0.11→0.09, accuracy 0.04→0.03, pspm 0.10→0.08). TOUTES les
// autres métriques sont identiques à RelativeWeights. Somme = 1.04, renormalisée
// par computeRelativePerformanceScore comme le profil par défaut.
//
// À 0.16 les porteurs de combat perdaient jusqu'à 12 points et la marge de
// non-inversion tombait au niveau du bruit : ne pas modifier ces poids sans
// re-simuler (cmd/diag_perfsim) et les re-figer avec l'utilisateur.
var objectiveChainWeights = map[string]float64{
	MetricKeyObjectiveParticipation: 0.12,
	MetricKeyKPM:                    0.10,
	MetricKeyKDA:                    0.09,
	MetricKeyAccuracy:               0.03,
	MetricKeyPSPM:                   0.08,
	MetricKeyDPMDeaths:              0.10,
	MetricKeyAPM:                    0.06,
	MetricKeyDPMDamage:              0.06,
	MetricKeyRankPerf:               0.04,
	MetricKeyKillsVsExpected:        0.09,
	MetricKeyDeathsVsExpected:       0.07,
	MetricKeyMedalExploit:           0.06,
	MetricKeyOffensiveConv:          0.09,
	MetricKeyDefensiveResist:        0.05,
}

// WeightsForChain rend le profil de poids du score relatif applicable à une chaîne
// de performance.
//
// Les deux chaînes de famille objectif intègrent `objective_participation` ; toutes
// les autres (arena_slayer, btb, chaos, firefight, ranked_slayer) gardent le profil
// historique, où la métrique est ABSENTE — et une métrique absente du profil n'est
// ni calculée ni comptée dans la renormalisation par la somme des poids présents.
// Une chaîne inconnue retombe sur le profil par défaut (jamais de profil vide : un
// score reste calculable).
func WeightsForChain(chain string) map[string]float64 {
	if chain == LUSRChainArenaObjectif || chain == PerfChainRankedObjectif {
		return objectiveChainWeights
	}
	return RelativeWeights
}

// ── Chaînes LUSR ─────────────────────────────────────────────────────────────

// LUSRChainConfig associe une chaîne TrueSkill à ses labels UI (FR + EN).
// Remplace PlaylistGroupConfig : le WeightFactor est supprimé — chaque chaîne
// est homogène par construction, tous les matchs pèsent 1.0.
type LUSRChainConfig struct {
	LabelFR string
	LabelEN string
}

// Clés canoniques des chaînes LUSR (valeur stockée dans match_skill_rank.playlist_group).
const (
	LUSRChainArenaSlayer   = "arena_slayer"
	LUSRChainArenaObjectif = "arena_objectif"
	LUSRChainBTB           = "btb"
	LUSRChainChaos         = "chaos"
)

// Clés canoniques additionnelles pour le score de performance (superset LUSR).
// Le LUSR exclut Ranked (→ CSR) et Firefight (→ PvE non classé) ; le score de
// performance, lui, doit couvrir tous les matchs joués → ces deux chaînes
// supplémentaires garantissent qu'aucun match n'est orphelin de score.
//
// Le classé est scindé PAR FAMILLE depuis le 2026-08-27 (plan
// .ai/PLAN_PERF_NOTE_OBJECTIFS.md, décision D-A) : comparer un match d'objectif à
// un historique majoritairement slayer décalait sa note (corpus : pspm médian 206
// en objectif contre 161 en slayer sur le classé). Une note reste « relative aux
// 50 derniers matchs de la MÊME chaîne » — la scission rend cette population
// homogène.
const (
	PerfChainRankedSlayer   = "ranked_slayer"
	PerfChainRankedObjectif = "ranked_objectif"
	PerfChainFirefight      = "firefight"
)

// PerfChainRanked ("ranked") n'est PLUS ÉMISE comme chaîne de performance depuis
// la scission par famille (2026-08-27) : GetPerformanceChain retourne désormais
// ranked_slayer ou ranked_objectif. La constante SURVIT pour deux raisons, dont
// aucune n'est une classification de performance :
//
//  1. valeur HISTORIQUE encore stockée dans player_match_enrichment.performance_chain
//     pour les matchs classés notés avant la scission. Aucune migration de données
//     n'est requise : batchComputePerformanceScores ne skippe un match que si la
//     chaîne STOCKÉE égale la chaîne RECALCULÉE (sync/performance.go) — "ranked" ne
//     peut plus être recalculée, donc chaque match concerné est renoté dans sa
//     nouvelle chaîne au premier batch ;
//  2. usages VIVANTS sans rapport, qui empruntent seulement le littéral :
//     playlist_group des lignes CSR (sync/csr_writes.go, table match_skill_rank) et
//     détection de playlist classée par sous-chaîne du nom (sync/transforms_helpers.go).
//
// Ne pas la réintroduire dans une classification de chaîne de performance.
const PerfChainRanked = "ranked"

// LUSRChains mappe clé interne → labels UI FR/EN.
var LUSRChains = map[string]LUSRChainConfig{
	LUSRChainArenaSlayer:   {LabelFR: "Social · Slayer", LabelEN: "Social · Slayer"},
	LUSRChainArenaObjectif: {LabelFR: "Social · Objectif", LabelEN: "Social · Objective"},
	LUSRChainBTB:           {LabelFR: "Grande Équipe", LabelEN: "Big Team Battle"},
	LUSRChainChaos:         {LabelFR: "Chaos", LabelEN: "Chaos"},
}

// GetLUSRChain (classification pair_name → chaîne LUSR) est désormais le seam
// title-aware défini dans skill_chain_provider.go (dispatcher vers le classifier
// title-owned skillchain.ClassifyLUSRChain). La logique Halo a été déplacée dans
// internal/games/halo_infinite/skillchain (MT-15).

// GetPerformanceChain détermine la chaîne du score de performance d'un match.
// Contrairement à GetLUSRChain (qui exclut Ranked/Firefight pour CSR/PvE), cette
// fonction garantit qu'aucun match n'est orphelin : tout match est rattaché à
// l'une des 7 chaînes possibles. La sémantique du score 0-100 devient ainsi
// "relatif aux 50 derniers matchs de la même chaîne".
//
// Priorité :
//  1. isRanked       → "ranked_objectif" si le sous-mode est de la famille
//     objectif, "ranked_slayer" sinon (scission par famille, D-A)
//  2. isFirefight    → "firefight"
//  3. GetLUSRChain() → arena_slayer / arena_objectif / btb / chaos
//  4. fallback       → arena_slayer (cohérent avec lusrChainForAssassin)
//
// titleSlug (C6) rend la classification title-aware : un titre avec classifier
// dédié (Halo 5) classe ses propres modes au lieu de les collapser dans la grille
// Infinite. ""/halo_infinite → classifier défaut → byte-identique HINF.
//
// Famille du classé : le test passe par le seam IsObjectiveFamilyForTitle (le
// classifier LUSR ne peut pas répondre — sur un pair_name Ranked il retourne ""
// puisque le classé va au CSR). pair_name NULL/vide, sous-mode inconnu, classifier
// absent, ou titre sans notion de famille (Halo 5 : pas de pair_name) → famille
// slayer, donc PerfChainRankedSlayer.
func GetPerformanceChain(titleSlug, pairName string, isRanked, isFirefight bool) string {
	if isRanked {
		if IsObjectiveFamilyForTitle(titleSlug, pairName) {
			return PerfChainRankedObjectif
		}
		return PerfChainRankedSlayer
	}
	if isFirefight {
		return PerfChainFirefight
	}
	if c := GetLUSRChainForTitle(titleSlug, pairName); c != "" {
		return c
	}
	return LUSRChainArenaSlayer
}

// ── Tiers LUSR ──────────────────────────────────────────────────────────────

// SkillTier définit un tier du LUSR.
type SkillTier struct {
	Name      string
	NameFR    string
	MinRating float64
	MaxRating float64
	Color     string
	SubTiers  int
}

// Noms canoniques EN des tiers CSR/LUSR (réutilisés par csr_writes).
const (
	TierBronze   = "Bronze"
	TierSilver   = "Silver"
	TierGold     = "Gold"
	TierPlatinum = "Platinum"
	TierDiamond  = "Diamond"
	TierOnyx     = "Onyx"
	// Labels FR pour SkillTiers et UI. Bronze/Onyx coïncident avec EN.
	TierLabelBronze  = "Bronze"
	TierLabelArgent  = "Argent"
	TierLabelOr      = "Or"
	TierLabelPlatine = "Platine"
	TierLabelDiamant = "Diamant"
	TierLabelOnyx    = "Onyx"
	// Label spécifique CSR placement.
	TierLabelPlacement = "Placement"
)

// SkillTiers définit l'échelle LUSR (Bronze → Onyx).
var SkillTiers = []SkillTier{
	{TierBronze, TierLabelBronze, 1000.0, 1200.0, "#CD7F32", 6},
	{TierSilver, TierLabelArgent, 1200.0, 1400.0, "#C0C0C0", 6},
	{TierGold, TierLabelOr, 1400.0, 1600.0, "#FFD700", 6},
	{TierPlatinum, TierLabelPlatine, 1600.0, 1800.0, "#00CED1", 6},
	{TierDiamond, TierLabelDiamant, 1800.0, 2000.0, "#B9F2FF", 6},
	{TierOnyx, TierLabelOnyx, 2000.0, 9999.0, "#1C1C1C", 1},
}

var romanNumerals = map[int]string{1: "I", 2: "II", 3: "III", 4: "IV", 5: "V", 6: "VI"}

// GetTierForRating retourne le tier et sous-tier pour un rating donné.
func GetTierForRating(rating float64) (*SkillTier, int) {
	for i := range SkillTiers {
		t := &SkillTiers[i]
		if rating >= t.MinRating && rating < t.MaxRating {
			if t.SubTiers <= 1 {
				return t, 0
			}
			rangePerSub := (t.MaxRating - t.MinRating) / float64(t.SubTiers)
			sub := int((rating-t.MinRating)/rangePerSub) + 1
			if sub > t.SubTiers {
				sub = t.SubTiers
			}
			return t, sub
		}
	}
	return nil, 0
}

// FormatTierLabel formate le label complet du tier en français.
func FormatTierLabel(rating float64) string {
	tier, sub := GetTierForRating(rating)
	if tier == nil {
		return "Non classé"
	}
	if sub == 0 {
		return tier.NameFR
	}
	r, ok := romanNumerals[sub]
	if !ok {
		return tier.NameFR
	}
	return tier.NameFR + " " + r
}

// ── Helpers math ────────────────────────────────────────────────────────────

func ClampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func sigmoidRatio(num, denom float64) float64 {
	if denom <= 0 {
		return 0.5
	}
	r := num / denom
	return ClampF(r/(1.0+r), 0.0, 1.0)
}

// drawMargin calcule la marge d'égalité à partir de la probabilité de draw.
func drawMargin(beta float64) float64 {
	if DrawProbability <= 0 {
		return 0.0
	}
	p := (DrawProbability + 1.0) / 2.0
	if p >= 1.0 {
		return 8.0 * beta
	}
	t := math.Sqrt(-2.0 * math.Log(1.0-p))
	return t * beta
}
