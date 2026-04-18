// Package domain — types pour l'exclusion manuelle de matchs.
package domain

import "time"

// ExcludedMatch représente un match marqué comme non pertinent par l'utilisateur.
type ExcludedMatch struct {
	MatchID   string    `json:"match_id"`
	StartTime time.Time `json:"start_time"`
	MapName   string    `json:"map_name"`
	ModeName  string    `json:"mode_name"`
}

// SetMatchExclusionRequest est le corps de PATCH .../matches/{match_id}/exclusion.
type SetMatchExclusionRequest struct {
	Excluded bool `json:"excluded"`
}
