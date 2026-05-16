// Package domain — types pour l'exclusion manuelle de matchs.
package domain

import (
	"errors"
	"time"
)

// ErrMatchNotFound : le match_id n'existe pas dans shared.match_registry.
// Mappé en HTTP 404 par le handler.
var ErrMatchNotFound = errors.New("match introuvable dans le registre")

// ErrRankedMatchNotExcludable : tentative d'exclure un match classé (CSR).
// Mappé en HTTP 422 par le handler. La réactivation reste autorisée.
var ErrRankedMatchNotExcludable = errors.New("les matchs classés ne peuvent pas être exclus")

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

// MatchRegistryInfo : sous-ensemble de shared.match_registry consulté avant
// d'autoriser une exclusion (garde "match classé" → refus si IsRanked) et de
// déterminer la chaîne perf / playlist_group LUSR pour le recompute.
type MatchRegistryInfo struct {
	MatchID     string
	StartTime   time.Time
	IsRanked    bool
	IsFirefight bool
	PairName    string
}
