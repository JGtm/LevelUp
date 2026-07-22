// Package patterns — moteur d'analyse de patterns de jeu (Pattern Engine v3).
//
// Ce package est entièrement stateless : aucun accès DB, aucun import HTTP.
// L'entrée est un []MatchRow et la sortie est un PatternReport.
//
// Ref : .ai/PLAN_PATTERN_ENGINE_V3.md
package patterns

import "time"

// MatchRow est la ligne d'entrée pour l'analyse de patterns.
// Toutes les métriques sont déjà calculées avant d'arriver ici.
type MatchRow struct {
	MatchID       string
	PlayedAt      time.Time
	Mode          string // normalisé
	MapID         string
	Outcome       int // 2=WIN, 3=LOSS, 1=DRAW, 4=DNF
	IsRanked      bool
	DurationSec   int
	SessionID     string
	KDA           float64
	Kills         int
	Deaths        int
	Assists       int
	Accuracy      float64
	OC            float64 // offensive_conversion
	DR            float64 // defensive_resistance
	HSRate        float64 // headshot_kills / kills
	FirstKills    int
	MMRDelta      float64
	PerfScore     *float64
	EngageScore   *float64
	ResidualBrut  *float64
	DeltaLUSR     *float64
	DeltaCSR      *float64
	CSRValue      *float64
	IsWithFriends bool
}

// PatternConfig contient les seuils injectables (toutes les valeurs ont
// des défauts via DefaultPatternConfig).
type PatternConfig struct {
	MinMatchesPerGroup   int     // 5
	StrengthWinRateDelta float64 // 0.12
	WeaknessWinRateDelta float64 // 0.12
	TiltLossRun          int     // 3
	TiltKDADropPct       float64 // 0.25
	EngageDeltaTilt      float64 // 0.20
	FatigueMinSession    int     // 4
	FatigueSessionCovPct float64 // 0.60
	AccuracyPlateauStd   float64 // 0.02
	AccuracyPlateauMax   float64 // 0.45
	// EngagementDrop
	EngageDropWindow        int // 10 — fenêtre de matchs récents analysés
	EngageDropHighThreshold int // 10 — seuil dropCount pour SeverityHigh
	// PerfCeiling
	PerfCeilingMinRows         int     // 20 — minimum de rows avec PerfScore
	PerfCeilingWindow          int     // 30 — fenêtre d'analyse LOWESS
	PerfCeilingTopN            int     // 10 — nombre de scores top pour meanTop
	PerfCeilingLowessAlpha     float64 // 0.4
	PerfCeilingFlatSlopeThresh float64 // 2.0 — pente LOWESS considérée plate
}

// DefaultPatternConfig retourne la configuration avec les seuils recommandés.
func DefaultPatternConfig() PatternConfig {
	return PatternConfig{
		MinMatchesPerGroup:         5,
		StrengthWinRateDelta:       0.12,
		WeaknessWinRateDelta:       0.12,
		TiltLossRun:                3,
		TiltKDADropPct:             0.25,
		EngageDeltaTilt:            0.20,
		FatigueMinSession:          4,
		FatigueSessionCovPct:       0.60,
		AccuracyPlateauStd:         0.02,
		AccuracyPlateauMax:         0.45,
		EngageDropWindow:           10,
		EngageDropHighThreshold:    10,
		PerfCeilingMinRows:         20,
		PerfCeilingWindow:          30,
		PerfCeilingTopN:            10,
		PerfCeilingLowessAlpha:     0.4,
		PerfCeilingFlatSlopeThresh: 2.0,
	}
}

// ContextType identifie le type de contexte d'un pattern.
type ContextType string

const (
	ContextByMode  ContextType = "by_mode"
	ContextByMap   ContextType = "by_map"
	ContextBySquad ContextType = "by_squad"
)

// Signal identifie le signal d'un pattern contextuel.
type Signal string

const (
	SignalStrength Signal = "strength"
	SignalWeakness Signal = "weakness"
	SignalNeutral  Signal = "neutral"
)

// ContextualPattern est le résultat pour une clé de contexte.
//
// Label est le libellé lisible de la clé, résolu côté service/handler (le
// package reste pur : aucun accès au référentiel ici). Rempli pour les
// patterns by_map (nom de carte résolu depuis metadata, repli localisé si
// inconnu — jamais le GUID nu). Vide pour by_mode (la clé est déjà un libellé
// normalisé) et by_squad (libellé i18n côté front).
type ContextualPattern struct {
	Type  ContextType `json:"type"`
	Key   string      `json:"key"`
	Label string      `json:"label,omitempty"`
	// FilterKey est la clé de filtrage STABLE d'un lien pattern→Solo (F7) : la
	// valeur EXACTE que le pipeline de filtres matche, indépendante de la locale
	// de la requête. by_map : nom de carte FR-first (fr→en fixe, = mapUI) ;
	// by_mode : libellé de mode normalisé (= modeUI = Key). Rempli au handler.
	// Vide pour by_squad (non lié). Distinct de Label (affichage localisé).
	FilterKey    string   `json:"filter_key,omitempty"`
	MatchCount   int      `json:"match_count"`
	WinRate      float64  `json:"win_rate"`
	AvgKDA       float64  `json:"avg_kda"`
	AvgOC        float64  `json:"avg_oc"`
	AvgDR        float64  `json:"avg_dr"`
	AvgPerf      *float64 `json:"avg_perf,omitempty"`
	AvgDeltaCSR  *float64 `json:"avg_delta_csr,omitempty"`
	AvgDeltaLUSR *float64 `json:"avg_delta_lusr,omitempty"`
	Delta        float64  `json:"delta"`
	Signal       Signal   `json:"signal"`
}

// BehaviorType identifie le type de pattern comportemental.
type BehaviorType string

const (
	BehaviorTilt            BehaviorType = "tilt"
	BehaviorSessionFatigue  BehaviorType = "session_fatigue"
	BehaviorEngagementDrop  BehaviorType = "engagement_drop"
	BehaviorAccuracyPlateau BehaviorType = "accuracy_plateau"
	BehaviorPerfCeiling     BehaviorType = "perf_ceiling"
)

// Severity identifie la sévérité d'un pattern comportemental.
type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

// BehavioralPattern est le résultat d'une détection comportementale.
type BehavioralPattern struct {
	Type      BehaviorType `json:"type"`
	Trigger   string       `json:"trigger"`
	Evidence  string       `json:"evidence"`
	Severity  Severity     `json:"severity"`
	Confirmed bool         `json:"confirmed"`
}

// Axes de leviers — clés stables consommées par le gabarit i18n front (un
// gabarit de phrase FR/EN par axe, F3). Le backend ne sert JAMAIS de phrase
// (title-agnostic + multilingue) : il sert l'axe + les données structurées du
// contexte visé, le front compose. Les 3 premiers portent un contexte
// (mode/carte/escouade), les 5 suivants sont des axes comportementaux (phrase
// fixe par axe).
const (
	AxisModeSelection     = "mode_selection"
	AxisMapAvoidance      = "map_avoidance"
	AxisSquadPlay         = "squad_play"
	AxisSessionManagement = "session_management"
	AxisSessionLength     = "session_length"
	AxisEngagement        = "engagement"
	AxisAccuracy          = "accuracy"
	AxisRadarAxis         = "radar_axis"
)

// Lever est une cible d'amélioration calibrée.
//
// F3 : le backend ne sert PLUS de phrase (`label` supprimé) — le front compose
// la phrase via un gabarit i18n FR/EN par `Axis`. Pour les leviers contextuels
// (by_mode/by_map/by_squad), le contexte visé est servi en données structurées :
// ContextKey (clé brute — libellé de mode, « with_friends »/« solo », ou GUID de
// carte) et ContextLabel (nom d'asset résolu, title-agnostic, rempli au handler
// pour by_map — jamais un littéral). Les leviers comportementaux n'ont pas de
// contexte (ContextKey/ContextLabel vides).
type Lever struct {
	Rank         int     `json:"rank"`
	Axis         string  `json:"axis"`
	ContextKey   string  `json:"context_key,omitempty"`
	ContextLabel string  `json:"context_label,omitempty"`
	CurrentVal   float64 `json:"current_val"`
	TargetVal    float64 `json:"target_val"`
	Horizon      int     `json:"horizon"`
	Impact       float64 `json:"impact"`
}

// PatternReport est la sortie de Analyze().
type PatternReport struct {
	WindowSize       int                 `json:"window_size"`
	ContextPatterns  []ContextualPattern `json:"context_patterns"`
	BehaviorPatterns []BehavioralPattern `json:"behavior_patterns"`
	Levers           []Lever             `json:"levers"`
	ComputedAt       time.Time           `json:"computed_at"`
	// MinMatchesForSignal : seuil de matchs par groupe sous lequel le front
	// affiche « Échantillon faible » au lieu de Force/Faiblesse (DEC-8). Servi
	// pour éviter un seuil codé en dur côté client.
	MinMatchesForSignal int `json:"min_matches_for_signal"`
}

// AnalyzeInput est l'entrée de la fonction Analyze.
type AnalyzeInput struct {
	Rows   []MatchRow
	N      int // nombre max de rows à utiliser (0 = toutes)
	Config PatternConfig
	Now    time.Time
}
