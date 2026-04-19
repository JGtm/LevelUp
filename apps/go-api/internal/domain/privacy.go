// Package domain — privacy.go : types pour la privacy des matchs Halo.
//
// Sprint 54 B : MatchPrivacyInfo, MatchPrivacyWarning.
// Sprint 55 E4 : PlayerPrivacyState (persistance DB).
package domain

import "time"

// MatchPrivacyInfo représente l'état de privacy d'un compte Halo.
// Chargé au bootstrap et au chargement de l'historique.
type MatchPrivacyInfo struct {
	IsPrivate bool   `json:"is_private"`
	IsPartial bool   `json:"is_partial"`
	Hint      string `json:"hint"` // "auth_required" | "partial_history" | ""
}

// MatchPrivacyWarning est intégré dans MatchHistoryResponse et MatchViewResponse.
// Level : "none" | "partial" | "full"
type MatchPrivacyWarning struct {
	Level   string `json:"level"`             // "none" | "partial" | "full"
	Message string `json:"message,omitempty"` // message localisé
}

// PrivacyWarningNone est un warning vide — pas de privacy.
var PrivacyWarningNone = &MatchPrivacyWarning{Level: "none"}

// PlayerPrivacyState est l'état de privacy persisté pour un joueur (Sprint 55 E4).
// Table stats.duckdb : player_privacy_state.
type PlayerPrivacyState struct {
	XUID       string    `json:"xuid"`
	IsPrivate  bool      `json:"is_private"`
	ObservedAt time.Time `json:"observed_at"`
	Source     string    `json:"source"` // "waypoint" | "api" | "manual"
}

// NewPrivacyWarning crée un MatchPrivacyWarning à partir d'un MatchPrivacyInfo.
func NewPrivacyWarning(info MatchPrivacyInfo) *MatchPrivacyWarning {
	switch {
	case info.IsPrivate:
		return &MatchPrivacyWarning{
			Level:   "full",
			Message: "Ce compte a des matchs privés — l'historique peut être incomplet.",
		}
	case info.IsPartial:
		return &MatchPrivacyWarning{
			Level:   "partial",
			Message: "Certains matchs ne sont pas accessibles — l'historique peut être partiel.",
		}
	default:
		return PrivacyWarningNone
	}
}
