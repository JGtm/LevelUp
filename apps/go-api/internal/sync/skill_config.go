// Package sync — skill_config.go : constantes et tiers LUSR (TrueSkill 2).
//
// Portage de src/analysis/skill_rating_config.py + src/analysis/playlist_groups.py.
// Centralise tous les paramètres numériques de l'algorithme LUSR.
package sync

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

// ── Playlist groups ─────────────────────────────────────────────────────────

// PlaylistGroupConfig définit un groupe de playlists LUSR.
type PlaylistGroupConfig struct {
	WeightFactor float64
}

// Identifiants des groupes LUSR utilisés dans PlaylistGroups et GetPlaylistGroup.
const (
	playlistGroupRanked = "ranked"
	playlistGroupArena  = "arena"
	playlistGroupBTB    = "btb"
	playlistGroupFun    = "fun"
)

// PlaylistGroups mappe le nom de groupe → config.
// Portage de src/analysis/playlist_groups.py.
var PlaylistGroups = map[string]PlaylistGroupConfig{
	playlistGroupRanked: {WeightFactor: 1.0},
	playlistGroupArena:  {WeightFactor: 0.8},
	playlistGroupBTB:    {WeightFactor: 0.7},
	playlistGroupFun:    {WeightFactor: 0.25},
}

// GetPlaylistGroup détermine le groupe LUSR d'une playlist.
func GetPlaylistGroup(playlistName, pairName *string) string {
	pn := ""
	if playlistName != nil {
		pn = *playlistName
	}
	pp := ""
	if pairName != nil {
		pp = *pairName
	}
	// Détection simplifiée basée sur des keywords.
	for _, s := range []string{pn, pp} {
		if containsI(s, "ranked") || containsI(s, "classé") {
			return playlistGroupRanked
		}
		if containsI(s, "btb") || containsI(s, "big team") {
			return playlistGroupBTB
		}
		if containsI(s, "fiesta") || containsI(s, "rumble") ||
			containsI(s, "action sack") || containsI(s, "swat") ||
			containsI(s, "griffball") || containsI(s, "infection") {
			return playlistGroupFun
		}
	}
	return playlistGroupArena // default social
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
