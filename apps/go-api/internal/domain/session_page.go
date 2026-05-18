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
}
