// Package domain — teammates.go : types pour la page Teammates (contrat FastAPI).
//
// Sprint 33 :
//
//	POST /api/v1/players/{slug}/pages/teammates → TeammatesPageResponse
package domain

import "time"

// ---------------------------------------------------------------------------
// Requête
// ---------------------------------------------------------------------------

// TeammatesQueryRequest est le corps de POST /pages/teammates.
type TeammatesQueryRequest struct {
	SelectedGamertags []string            `json:"selected_gamertags"`
	Filters           *FilterContextInput `json:"filters,omitempty"`
}

// ---------------------------------------------------------------------------
// Réponse
// ---------------------------------------------------------------------------

// TeammateOption est un coéquipier fréquent sélectionnable.
type TeammateOption struct {
	Gamertag       string     `json:"gamertag"`
	XUID           *string    `json:"xuid,omitempty"`
	EncounterCount int        `json:"encounter_count"`
	LastSeenAt     *time.Time `json:"last_seen_at,omitempty"`
}

// TeammateKPIs sont les KPIs agrégés pour un groupe de matchs.
type TeammateKPIs struct {
	MatchCount     int      `json:"match_count"`
	Wins           int      `json:"wins"`
	KDRatio        *float64 `json:"kd_ratio"`
	WinRate        float64  `json:"win_rate"`
	Accuracy       *float64 `json:"accuracy"`
	KillsPerGame   *float64 `json:"kills_per_game"`
	AssistsPerGame *float64 `json:"assists_per_game"`
}

// TeammateRow est une ligne de résultat (stats avec vs sans un coéquipier).
type TeammateRow struct {
	Gamertag       string        `json:"gamertag"`
	XUID           *string       `json:"xuid,omitempty"`
	EncounterCount int           `json:"encounter_count"`
	LastSeenAt     *time.Time    `json:"last_seen_at,omitempty"`
	WithKPIs       TeammateKPIs  `json:"with_kpis"`
	WithoutKPIs    *TeammateKPIs `json:"without_kpis,omitempty"`
}

// TeammatesPageResponse est la réponse de POST /pages/teammates.
type TeammatesPageResponse struct {
	Options       []TeammateOption `json:"options"`
	Teammates     []TeammateRow    `json:"teammates"`
	SoloReference *TeammateKPIs    `json:"solo_reference"`
	TotalMatches  int              `json:"total_matches"`
}
