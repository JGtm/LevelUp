// Package domain — fanout.go : types pour l'enrichissement multi-joueur.
//
// Sprint 42 : quand le joueur A est synchronisé, les joueurs B/C/D
// qui partagent des matchs communs peuvent être enrichis en fanout.
package domain

import "time"

// FanoutTarget identifie un joueur à enrichir après la sync d'un autre.
type FanoutTarget struct {
	Gamertag     string `json:"gamertag"`
	XUID         string `json:"xuid"`
	CommonCount  int    `json:"common_count"`  // matchs en commun découverts
	MissingCount int    `json:"missing_count"` // matchs sans enrichissement
}

// FanoutPlan décrit les enrichissements à effectuer après une sync.
type FanoutPlan struct {
	SourceGamertag string         `json:"source_gamertag"`
	Targets        []FanoutTarget `json:"targets"`
	MatchIDs       []string       `json:"match_ids"` // matchs nouvellement insérés
	CreatedAt      time.Time      `json:"created_at"`
}

// FanoutResult résume les résultats de l'enrichissement fanout.
type FanoutResult struct {
	TargetsProcessed int      `json:"targets_processed"`
	MatchesEnriched  int      `json:"matches_enriched"`
	Errors           []string `json:"errors,omitempty"`
}
