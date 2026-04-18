// Package domain — privacy.go : types pour la privacy des matchs Halo.
//
// Sprint 54 B : MatchPrivacyInfo, MatchPrivacyWarning.
package domain

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
