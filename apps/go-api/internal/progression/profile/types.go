// Package profile — service PlayerProfile (V1 Ascension).
//
// Depuis 2026-05-18 (V1 PlayerProfile commit 4) ce package porte le service
// complet décrit dans `PLAN_PLAYER_PROFILE_ASCENSION.md` §4-§5 :
//   - Section A1 : radar narrative 6 axes + rôles
//   - Section A2 : style FK/FD + engagement
//   - Section B : LUSR tier + 8 composantes + tendance LOWESS
//   - Section C : leviers + suggestions de défis
//
// Historique : précédente version V2-commit-6 ne portait que LUSRState +
// TierState + MuTrend (mini-service workaround). V1 commit 4 enrichit
// le `PlayerProfile` avec les nouveaux champs sans casser les callers V2
// (la closure post-sync continue à lire `p.LUSR.Mu`, `p.NextTier.Label`
// et `p.MuTrend`).
package profile

import (
	"time"
)

// PlayerProfile rassemble l'état de progression d'un joueur sur un titre.
//
// Conformément au plan §4 :
//   - Section A1 : DominantRole + SecondaryRole + RadarAxes + Strengths + ImprovementAreas
//   - Section A2 : StyleSignature + EngagementSnap
//   - Section B  : SkillRating + LUSR + LUSRComponents + MuTrend
//   - Section C  : Leverages + SuggestedChallenges
type PlayerProfile struct {
	UserID    string    `json:"user_id"`
	TitleSlug string    `json:"title_slug"`
	UpdatedAt time.Time `json:"updated_at"`

	// HasEnoughData : true si MatchesAnalyzed >= MinMatchesForProfile (30).
	// Si false, les sections suivantes peuvent être vides ou partielles —
	// le frontend doit afficher un état "données insuffisantes".
	HasEnoughData   bool `json:"has_enough_data"`
	MatchesAnalyzed int  `json:"matches_analyzed"`

	// ── Section A1 : narrative ────────────────────────────────────────────

	DominantRole     string                   `json:"dominant_role,omitempty"` // narrative.ImpactRole
	SecondaryRole    string                   `json:"secondary_role,omitempty"`
	RadarAxes        []ParticipationAxisValue `json:"radar_axes,omitempty"`        // 6 axes, valeurs 0..100
	Strengths        []RadarAxisInsight       `json:"strengths,omitempty"`         // top 3
	ImprovementAreas []RadarAxisInsight       `json:"improvement_areas,omitempty"` // bottom 3

	// ── Section A2 : style & discipline ───────────────────────────────────

	StyleSignature StyleSignature     `json:"style_signature"`
	EngagementSnap EngagementSnapshot `json:"engagement_snap"`

	// ── Section B : LUSR ──────────────────────────────────────────────────

	// LUSR : état courant. Empty si pas assez de matchs (MinMatchesForRating).
	LUSR LUSRState `json:"lusr"`

	// SkillRating : tier + sub-tier formatés pour affichage UI.
	SkillRating SkillRatingSnapshot `json:"skill_rating"`

	// Tier + NextTier conservés pour compat V2 callers (post-sync coach).
	Tier     TierState `json:"-"`
	NextTier TierState `json:"-"`

	// LUSRComponents : 8 composantes avec courant/top20%/cible.
	LUSRComponents []LUSRComponentBreakdown `json:"lusr_components,omitempty"`

	// MuTrend : tendance LOWESS sur μ (composite). Compat V2 (coach LOWESSPositive).
	MuTrend LOWESSTrend `json:"mu_trend"`

	// SkillTrend : sparkline 90 j de la tendance LUSR (μ lissé LOWESS, en points
	// LUSR — même échelle que « {mu} pts LUSR »). Fenêtre FIXE (SkillTrendWindowDays),
	// indépendante de window_days. Vide si < 3 points de rating dans la fenêtre
	// (LOWESS non fiable) → le front n'affiche rien. Sert UNIQUEMENT du lissé, jamais
	// le μ brut (DEC-5/D2, DEC-6). Cf. journal Lot D.
	SkillTrend []SkillTrendPoint `json:"skill_trend,omitempty"`

	// ── Section C : coaching ──────────────────────────────────────────────

	Leverages           []ProgressionLeverage `json:"leverages,omitempty"`
	SuggestedChallenges []SuggestedChallenge  `json:"suggested_challenges,omitempty"`
}

// ParticipationAxisValue est une valeur scorée sur un axe radar narrative.
type ParticipationAxisValue struct {
	Axis  string  `json:"axis"`  // "combat" | "survival" | ...
	Value float64 `json:"value"` // 0..100
	Raw   float64 `json:"raw,omitempty"`
}

// RadarAxisInsight est un axe radar avec un message d'interprétation court.
type RadarAxisInsight struct {
	Axis    string  `json:"axis"`
	Value   float64 `json:"value"`
	Message string  `json:"message,omitempty"` // i18n key ou direct
}

// StyleSignature décrit le style offensif via First Kill / First Death.
type StyleSignature struct {
	FirstKillCount  int     `json:"first_kill_count"`
	FirstDeathCount int     `json:"first_death_count"`
	FKFDRatio       float64 `json:"fkfd_ratio"`          // FK / max(FD, 1)
	StyleKey        string  `json:"style_key,omitempty"` // "opportunistic_finisher" | "overextended" | "hyper_engaged" | "passive"
}

// EngagementSnapshot capture le score d'engagement + régularité.
type EngagementSnapshot struct {
	Score            float64 `json:"score"` // 0-100
	Tier             string  `json:"tier"`  // "low" | "regular" | "high" | "intense"
	MatchesPerDayAvg float64 `json:"matches_per_day_avg"`
	MaxGapDays       int     `json:"max_gap_days"`
	RegularityCoach  string  `json:"regularity_coach,omitempty"` // i18n key
}

// SkillRatingSnapshot capture le tier LUSR + progression vers le suivant.
type SkillRatingSnapshot struct {
	TierName      string  `json:"tier_name"`    // "Diamond"
	TierNameFR    string  `json:"tier_name_fr"` // "Diamant"
	SubTier       int     `json:"sub_tier"`     // 1..6
	Label         string  `json:"label"`        // "Diamond III"
	Mu            float64 `json:"mu"`
	Sigma         float64 `json:"sigma"`
	NextTierLabel string  `json:"next_tier_label,omitempty"`
	NextTierMu    float64 `json:"next_tier_mu,omitempty"`
	GapToNext     float64 `json:"gap_to_next,omitempty"`    // NextTierMu - Mu
	ProgressRatio float64 `json:"progress_ratio,omitempty"` // 0..1 dans le sub-tier courant
}

// LUSRComponentBreakdown décrit une des 8 composantes LUSR.
type LUSRComponentBreakdown struct {
	Name          string  `json:"name"`            // "kills_vs_expected"
	Weight        float64 `json:"weight"`          // 0.27
	CurrentAvg    float64 `json:"current_avg"`     // 0.52
	PersonalTop20 float64 `json:"personal_top_20"` // 0.78
	TargetForTier float64 `json:"target_for_tier"` // via RequiredCompositeForTier
	Trend         float64 `json:"trend"`           // slope LOWESS sur 30j (positif = amélioration)
}

// ProgressionLeverage identifie un axe d'amélioration prioritaire.
type ProgressionLeverage struct {
	Component       string   `json:"component"`        // "deaths_vs_expected"
	LeverageValue   float64  `json:"leverage_value"`   // (1 - current) × weight
	NarrativeAxes   []string `json:"narrative_axes"`   // ["survival"]
	CoachingMessage string   `json:"coaching_message"` // i18n key
}

// SuggestedChallenge est un template recommandé par le coach.
//
// LabelFR / LabelEN / DescriptionFR / DescriptionEN sont hydratés depuis
// prestige.Template à la sélection (V2 §3). Permettent à l'UI d'afficher
// un libellé humain sans avoir à charger le catalogue séparément.
type SuggestedChallenge struct {
	TemplateID       string `json:"template_id"`
	TargetTier       string `json:"target_tier"`       // "normal" | "heroic" | "legendary"
	HistoricalStreak int    `json:"historical_streak"` // nb complétions sur 90j (placeholder 0 en V1)
	IsArcStep        bool   `json:"is_arc_step"`
	ArcID            string `json:"arc_id,omitempty"`

	// Hydratés depuis le template prestige (V2 §3). Vides si template
	// introuvable (ne devrait pas arriver — selectSuggestedChallenges les
	// récupère depuis le même catalogue).
	LabelFR       string `json:"label_fr,omitempty"`
	LabelEN       string `json:"label_en,omitempty"`
	DescriptionFR string `json:"description_fr,omitempty"`
	DescriptionEN string `json:"description_en,omitempty"`
}

// MinMatchesForProfile est le seuil sous lequel on considère que le profil
// n'a pas assez de données pour les insights (Section A2 + B + C). Section A1
// dégrade gracieusement (radar avec seulement les axes calculables).
const MinMatchesForProfile = 30

// LUSRState capture l'état LUSR courant.
type LUSRState struct {
	Mu    float64
	Sigma float64
	// MatchesCount : nombre de matchs ayant contribué au rating (pour gating).
	MatchesCount int
	// LastMatchAt : timestamp du dernier match avec rating (séparé de
	// l'horloge serveur pour gérer les imports décalés).
	LastMatchAt *time.Time
}

// TierState décrit un tier + sub-tier LUSR.
type TierState struct {
	Name    string  // "Diamond"
	NameFR  string  // "Diamant"
	SubTier int     // 1..6 (0 si tier sans sub-tier comme Onyx)
	Label   string  // "Diamond III"
	LowerMu float64 // entrée du sub-tier (inclusive)
	UpperMu float64 // sortie du sub-tier (= entrée du sub-tier suivant, exclusive)
}

// IsEmpty retourne true si le TierState n'est pas renseigné.
func (t TierState) IsEmpty() bool { return t.Name == "" }

// SkillTrendPoint est un point de la sparkline de tendance LUSR 90 j (DEC-5/D2).
// Value est le rating LUSR LISSÉ par LOWESS (échelle points, identique à la valeur
// affichée « pts LUSR ») — le μ brut per-match n'est jamais servi (DEC-6).
type SkillTrendPoint struct {
	Date  string  `json:"date"`  // jour UTC (YYYY-MM-DD) du match noté
	Value float64 `json:"value"` // rating LUSR lissé (points)
}

// muPoint = un rating LUSR daté, source de la sparkline de tendance (interne).
type muPoint struct {
	at    time.Time
	value float64
}

// SkillTrendWindowDays : fenêtre FIXE (90 j) de la sparkline de tendance LUSR
// (DEC-5). Indépendante du window_days du profil (sections A1/A2/B/C) : la
// sparkline porte une sémantique « tendance sur 90 jours » stable.
const SkillTrendWindowDays = 90

// LOWESSTrend décrit une tendance lissée sur une métrique.
type LOWESSTrend struct {
	Metric string  // "mu" pour la tendance globale LUSR
	Slope  float64 // diff (lastSmoothed - firstSmoothed) sur la fenêtre
	Window int     // nombre de points effectifs dans la fenêtre LOWESS
}

// IsPositive retourne true si la tendance est positive ET suffisamment longue.
func (t LOWESSTrend) IsPositive(minWindow int) bool {
	return t.Slope > 0 && t.Window >= minWindow
}

// MinMatchesForRating est le seuil sous lequel on considère le rating non
// fiable (aligné sur sync.MinMatchesForRating mais dupliqué ici pour éviter
// la dépendance cyclique entre progression et sync).
const MinMatchesForRating = 10

// LOWESSAlpha est le paramètre de lissage passé à temporal.LowessSmooth.
// 0.3 = fenêtre ~30% du dataset (cohérent avec defaults Python statsmodels).
const LOWESSAlpha = 0.3
