// Package domain — session_page.go : types pour la page détail de session.
package domain

import (
	"fmt"
	timepkg "time"
)

// SessionPageRequest est le body de POST .../pages/sessions/detail.
type SessionPageRequest struct {
	Filters             FilterContextInput `json:"filters,omitempty"`
	SessionLabel        *string            `json:"session_label,omitempty"`
	CompareSessionLabel *string            `json:"compare_session_label,omitempty"`
	EnableCompare       bool               `json:"enable_compare,omitempty"`
	// Locale ("fr" | "en") pour la résolution des libellés cartes/modes/playlists
	// (aligné Home/Explorer). Vide → FR par défaut.
	Locale string `json:"locale,omitempty"`
}

// Validate valide les paramètres de SessionPageRequest.
func (r SessionPageRequest) Validate() error {
	if err := r.Filters.Validate(); err != nil {
		return err
	}
	if r.SessionLabel != nil && *r.SessionLabel == "" {
		return fmt.Errorf("SessionPageRequest: session_label vide")
	}
	if r.CompareSessionLabel != nil && *r.CompareSessionLabel == "" {
		return fmt.Errorf("SessionPageRequest: compare_session_label vide")
	}
	return nil
}

// SessionDetailMatchRow représente une ligne détaillée de match au sein d'une session.
type SessionDetailMatchRow struct {
	MatchID          string       `json:"match_id"`
	StartTime        timepkg.Time `json:"start_time"`
	Outcome          *int         `json:"outcome,omitempty"`
	PlaylistName     string       `json:"playlist_name"`
	PairName         string       `json:"pair_name"`
	IsRanked         bool         `json:"is_ranked"`
	Kills            int          `json:"kills"`
	Deaths           int          `json:"deaths"`
	Assists          int          `json:"assists"`
	KDA              *float64     `json:"kda,omitempty"`
	Accuracy         *float64     `json:"accuracy,omitempty"`
	PersonalScore    *int         `json:"personal_score,omitempty"`
	PerformanceScore *float64     `json:"performance_score,omitempty"`
	SessionLabel     *string      `json:"session_label,omitempty"`
	DominantCategory *string      `json:"dominant_category,omitempty"`
	OffensiveConv    *float64     `json:"offensive_conversion,omitempty"`
	DefensiveResist  *float64     `json:"defensive_resistance,omitempty"`
	// Dégâts infligés / subis du match (pour la barre composite par match).
	DamageDealt *float64 `json:"damage_dealt,omitempty"`
	DamageTaken *float64 `json:"damage_taken,omitempty"`
	// Placement du joueur (rang API, = "Rang" du scoreboard) + taille du lobby à la
	// fin (participants present_at_completion, bots inclus) — pour le breakdown des placements.
	Placement *int `json:"placement,omitempty"`
	LobbySize *int `json:"lobby_size,omitempty"`
	// Champs enrichis pour le tableau détail (Phase 3), projetés depuis StatsMatchRow.
	MapName          string   `json:"map_name,omitempty"`
	DurationSeconds  *int     `json:"duration_seconds,omitempty"`
	TeamMMR          *float64 `json:"team_mmr,omitempty"`
	EnemyMMR         *float64 `json:"enemy_mmr,omitempty"`
	DeltaMMR         *float64 `json:"delta_mmr,omitempty"`
	PerfTier         int      `json:"perf_tier,omitempty"`
	SkillRatingType  string   `json:"skill_rating_type,omitempty"`
	SkillRatingValue *float64 `json:"skill_rating_value,omitempty"`
	SkillRatingDelta *float64 `json:"skill_rating_delta,omitempty"` // gain/perte de rating du match
	// SkillExpectedWinProb : proba de victoire pré-match de l'équipe du joueur
	// (LUSR v2, ∈ [0,1]). nil pour les matchs pré-v2 / non-LUSR. Le front la
	// catégorise (attendu / surprise / belle perf) via categorizeWinProb.
	SkillExpectedWinProb *float64 `json:"expected_win_prob,omitempty"`
	// SkillTierLabel : libellé du palier ranked/LUSR (ex. "Or III", "Diamant V"),
	// construit comme l'Explorer (analysis.BuildSkillTierLabel). Nil si non rankée /
	// placement. La colonne "Rang" du tableau affiche ça (pas la valeur brute).
	SkillTierLabel *string `json:"skill_tier_label,omitempty"`
	// PlacementDone/PlacementTotal : progression de placement (X/Y). Si présents,
	// la colonne "Rang" affiche "X/Y" à la place du palier (comme l'Explorer).
	PlacementDone  *int `json:"placement_done,omitempty"`
	PlacementTotal *int `json:"placement_total,omitempty"`
	// ModeUI : libellé de mode normalisé + traduit (comme l'Explorer), via analysis.ResolveModeUI.
	ModeUI string `json:"mode_ui,omitempty"`
	// Stats attendues (écart CUMULÉ au FDA attendu sur la session). KdaExpected =
	// kills_expected + assists_expected/3 − deaths_expected (analysis.ExpectedFDA).
	// Tous nil hors titre à CapExpectedStats (Halo 5) ou match sans attendu → chart
	// gaté par la capability côté front.
	KillsExpected   *float64 `json:"kills_expected,omitempty"`
	DeathsExpected  *float64 `json:"deaths_expected,omitempty"`
	AssistsExpected *float64 `json:"assists_expected,omitempty"`
	KdaExpected     *float64 `json:"kda_expected,omitempty"`
	// CareerXPEstimated — XP de carrière (Career Rank) ESTIMÉE gagnée sur ce match,
	// MIROIR EXACT du calcul Timeseries (analysis.EstimateCareerXP via
	// estimateMatchCareerXP, capability analytics.career_xp_estimate). nil hors
	// capability / match Firefight / personal_score absent (V72-13).
	CareerXPEstimated *int `json:"career_xp_estimated,omitempty"`
}

// SessionCompareSuggestion décrit la session proposée pour une comparaison rapide.
type SessionCompareSuggestion struct {
	SessionLabel string `json:"session_label"`
	Strategy     string `json:"strategy"`
	Reason       string `json:"reason"`
}

// SessionPageResponse est la réponse de POST .../pages/sessions/detail.
//
// CompareMatches : rows détaillées de la session comparée (alimente les charts
// du drawer compare côté front). Vide si compare_enabled = false.
// PreviousSessionLabel / NextSessionLabel : navigation chronologique dans
// available_sessions (older/newer) pour les boutons ←/→ du drawer.
type SessionPageResponse struct {
	CurrentSession       *SessionCompareEntry      `json:"current_session"`
	AvailableSessions    []string                  `json:"available_sessions"`
	Matches              []SessionDetailMatchRow   `json:"matches"`
	SuggestedCompare     *SessionCompareSuggestion `json:"suggested_compare,omitempty"`
	CompareEnabled       bool                      `json:"compare_enabled"`
	CompareSession       *SessionCompareEntry      `json:"compare_session,omitempty"`
	CompareMatches       []SessionDetailMatchRow   `json:"compare_matches"`
	CompareMetrics       []SessionCompareMetricRow `json:"compare_metrics"`
	PreviousSessionLabel *string                   `json:"previous_session_label,omitempty"`
	NextSessionLabel     *string                   `json:"next_session_label,omitempty"`
	// IntensityRows / CompareIntensityRows : profil d'intensité (frags par phase de
	// match) de la session courante et de la session comparée — MIROIR du calcul
	// Timeseries (buildIntensityRows). Best-effort : nil si le repo highlight events
	// n'est pas câblé ou sans events (le front affiche l'état vide). Alimente le chart
	// « Intensité » (profil médian + enveloppe P25–P75) du drawer et de la vue principale.
	IntensityRows        []IntensityMatchRow `json:"intensity_rows,omitempty"`
	CompareIntensityRows []IntensityMatchRow `json:"compare_intensity_rows,omitempty"`
}
