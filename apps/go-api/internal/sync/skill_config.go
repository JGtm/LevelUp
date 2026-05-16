// Package sync — skill_config.go : constantes et tiers LUSR (TrueSkill 2).
//
// Portage de src/analysis/skill_rating_config.py + src/analysis/playlist_groups.py.
// Centralise tous les paramètres numériques de l'algorithme LUSR.
package sync

import (
	"math"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/games/halo_infinite"
)

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
	MinMatchesForRating = 10
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

// ── Score composite ─────────────────────────────────────────────────────────

// CompositeWeights pondère les composantes du score composite [0,1].
// Portage de compositeWeights v5 (analysis/skill_rating.go).
// La somme dépasse 1.0 (1.02) : computeCompositeScore renormalise par totalWeight.
var CompositeWeights = map[string]float64{
	"kills_vs_expected":    0.27,
	"deaths_vs_expected":   0.24,
	"win_factor":           0.05,
	"damage_efficiency":    0.10,
	"accuracy_delta":       0.10,
	"medal_exploit":        0.04,
	"offensive_conversion": 0.16,
	"defensive_resistance": 0.06,
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

// RelativeWeights pondère les métriques du score relatif (0-100).
// Portage de relativeWeights v5 (analysis/performance_score.go).
// Somme = 1.01 → renormalisé automatiquement si des métriques sont absentes.
var RelativeWeights = map[string]float64{
	"kpm":                  0.14,
	"dpm_deaths":           0.10,
	"apm":                  0.06,
	"kda":                  0.11,
	"accuracy":             0.04,
	"pspm":                 0.10,
	"dpm_damage":           0.06,
	"rank_perf":            0.04,
	"kills_vs_expected":    0.09,
	"deaths_vs_expected":   0.07,
	"medal_exploit":        0.06,
	"offensive_conversion": 0.09,
	"defensive_resistance": 0.05,
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
const (
	PerfChainRanked    = "ranked"
	PerfChainFirefight = "firefight"
)

// LUSRChains mappe clé interne → labels UI FR/EN.
var LUSRChains = map[string]LUSRChainConfig{
	LUSRChainArenaSlayer:   {LabelFR: "Social · Slayer", LabelEN: "Social · Slayer"},
	LUSRChainArenaObjectif: {LabelFR: "Social · Objectif", LabelEN: "Social · Objective"},
	LUSRChainBTB:           {LabelFR: "Grande Équipe", LabelEN: "Big Team Battle"},
	LUSRChainChaos:         {LabelFR: "Chaos", LabelEN: "Chaos"},
}

// GetLUSRChain détermine la chaîne TrueSkill LUSR depuis le pair_name d'un match.
// Retourne "" si le match est exclu du LUSR (Ranked → CSR, Firefight → PvE).
//
// Classification :
//   - Ranked, Firefight                          → exclu ("")
//   - BTB, BTB Heavies                           → btb
//   - Fiesta, Super Fiesta, Husky Raid           → chaos
//   - Other : Infection/Griffball/Rocket Hog/Action Sack/Event → chaos
//     Rumble Pit + préfixes inconnus     → arena_slayer (fallback)
//   - Assassin (Arena/Tactical/Assault/Community) :
//     sous-mode objectif (CTF, Strongholds…)  → arena_objectif
//     tout le reste                            → arena_slayer
func GetLUSRChain(pairName string) string {
	category := halo_infinite.InferModeCategoryFromPairName(pairName)
	switch category {
	case halo_infinite.ModeCategoryRanked, halo_infinite.ModeCategoryFirefight:
		return ""
	case halo_infinite.ModeCategoryBTB:
		if containsI(pairName, "rocket hog") {
			return LUSRChainChaos
		}
		return LUSRChainBTB
	case halo_infinite.ModeCategoryFiesta, halo_infinite.ModeCategorySuperFiesta, halo_infinite.ModeCategoryHuskyRaid:
		return LUSRChainChaos
	case halo_infinite.ModeCategoryOther:
		return lusrChainForOther(pairName)
	default: // ModeCategoryAssassin
		return lusrChainForAssassin(pairName)
	}
}

// lusrChainForOther classe les modes de catégorie Other.
// Chaos : Infection, Griffball, Rocket Hog Race, Action Sack, Event.
// Fallback arena_slayer : Rumble Pit et tout préfixe inconnu.
func lusrChainForOther(pairName string) string {
	if containsI(pairName, "infection") || containsI(pairName, "griffball") ||
		containsI(pairName, "rocket hog") || containsI(pairName, "action sack") ||
		containsI(pairName, "event") {
		return LUSRChainChaos
	}
	return LUSRChainArenaSlayer
}

// GetPerformanceChain détermine la chaîne du score de performance d'un match.
// Contrairement à GetLUSRChain (qui exclut Ranked/Firefight pour CSR/PvE), cette
// fonction garantit qu'aucun match n'est orphelin : tout match est rattaché à
// l'une des 6 chaînes possibles. La sémantique du score 0-100 devient ainsi
// "relatif aux 50 derniers matchs de la même chaîne".
//
// Priorité :
//  1. isRanked       → "ranked"
//  2. isFirefight    → "firefight"
//  3. GetLUSRChain() → arena_slayer / arena_objectif / btb / chaos
//  4. fallback       → arena_slayer (cohérent avec lusrChainForAssassin)
func GetPerformanceChain(pairName string, isRanked, isFirefight bool) string {
	if isRanked {
		return PerfChainRanked
	}
	if isFirefight {
		return PerfChainFirefight
	}
	if c := GetLUSRChain(pairName); c != "" {
		return c
	}
	return LUSRChainArenaSlayer
}

// lusrChainForAssassin classe les sous-modes Arena/Tactical/Assault/Community.
// Objectif reconnus : CTF, Oddball, Strongholds, KotH, Total Control,
// Land Grab, Extraction, Stockpile, One Flag CTF, Covert One Flag.
// Tout le reste (Slayer, Attrition, Elimination, inconnu) → arena_slayer.
func lusrChainForAssassin(pairName string) string {
	subMode := toLowerASCII(analysis.NormalizeModeLabel(pairName))
	switch subMode {
	case "ctf", "capture the flag", "neutral flag ctf", "one flag ctf", "covert one flag",
		"strongholds", "oddball", "king of the hill",
		"total control", "land grab", "extraction", "stockpile":
		return LUSRChainArenaObjectif
	default:
		return LUSRChainArenaSlayer
	}
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

// SkillTiers définit l'échelle LUSR (Bronze → Onyx).
var SkillTiers = []SkillTier{
	{"Bronze", "Bronze", 1000.0, 1200.0, "#CD7F32", 6},
	{"Silver", "Argent", 1200.0, 1400.0, "#C0C0C0", 6},
	{"Gold", "Or", 1400.0, 1600.0, "#FFD700", 6},
	{"Platinum", "Platine", 1600.0, 1800.0, "#00CED1", 6},
	{"Diamond", "Diamant", 1800.0, 2000.0, "#B9F2FF", 6},
	{"Onyx", "Onyx", 2000.0, 9999.0, "#1C1C1C", 1},
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

func clampF(v, lo, hi float64) float64 {
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
	return clampF(r/(1.0+r), 0.0, 1.0)
}

// containsI est un contains case-insensitive simplifié.
func containsI(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	s = toLowerASCII(s)
	substr = toLowerASCII(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLowerASCII(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		b[i] = c
	}
	return string(b)
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
