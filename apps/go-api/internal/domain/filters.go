// Package domain — types du gestionnaire de filtres.
package domain

import (
	"fmt"
	"time"
)

// LabelValue est une paire label/value pour les options de sélection.
type LabelValue struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// FilterMatchRow représente une ligne minimale pour la résolution des filtres.
// Type de transfert entre platform/duckdb et les services.
type FilterMatchRow struct {
	MatchID       string
	StartTime     *time.Time
	MapName       *string
	MapNameFR     *string
	PairName      *string
	PairNameFR    *string
	PlaylistName  *string
	IsFirefight   bool
	IsRanked      bool
	SessionID     *string
	SessionLabel  *string
	IsWithFriends bool
}

// PeriodInput représente le filtre de période.
type PeriodInput struct {
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
}

// CascadeFilter représente les filtres en cascade (playlist, mode, carte).
type CascadeFilter struct {
	ExperienceTypes []string `json:"experience_types"`
	Playlists       []string `json:"playlists"`
	Modes           []string `json:"modes"`
	Maps            []string `json:"maps"`
}

// SessionsFilter représente le filtre de session.
type SessionsFilter struct {
	PickedSessionLabel      *string  `json:"picked_session_label"`
	PickedSoloSessionLabel  *string  `json:"picked_solo_session_label"`
	PickedSquadSessionLabel *string  `json:"picked_squad_session_label"`
	PickedSessions          []string `json:"picked_sessions"`
	GapMinutes              int      `json:"gap_minutes"`
}

// FilterContextInput est le corps de la requête POST filters/resolve.
type FilterContextInput struct {
	FilterMode string         `json:"filter_mode"` // "period" | "sessions"
	Period     PeriodInput    `json:"period"`
	Sessions   SessionsFilter `json:"sessions"`
	Cascade    CascadeFilter  `json:"cascade"`
}

// validFilterModes est l'ensemble des modes de filtre valides.
var validFilterModes = map[string]bool{
	"period":   true,
	"sessions": true,
}

// Validate vérifie la cohérence des paramètres du filtre.
// Une valeur vide de FilterMode est acceptée (défaut : "period").
func (f FilterContextInput) Validate() error {
	if f.FilterMode != "" && !validFilterModes[f.FilterMode] {
		return fmt.Errorf("FilterContextInput: filter_mode invalide %q (attendu : period|sessions)", f.FilterMode)
	}
	if f.Period.StartDate != nil && f.Period.EndDate != nil {
		if !f.Period.StartDate.Before(*f.Period.EndDate) {
			return fmt.Errorf("FilterContextInput: start_date (%s) doit être antérieure à end_date (%s)",
				f.Period.StartDate.Format(time.RFC3339), f.Period.EndDate.Format(time.RFC3339))
		}
	}
	if f.Sessions.GapMinutes < 0 {
		return fmt.Errorf("FilterContextInput: gap_minutes doit être ≥ 0 (reçu %d)", f.Sessions.GapMinutes)
	}
	return nil
}

// SessionOption représente une session disponible.
type SessionOption struct {
	Label      string `json:"label"`
	SessionID  string `json:"session_id"`
	MatchCount int    `json:"match_count"`
	IsSquad    bool   `json:"is_squad"`
}

// SessionOptions contient toutes les sessions par catégorie.
type SessionOptions struct {
	AllSessions []SessionOption `json:"all_sessions"`
	SoloLabels  []string        `json:"solo_labels"`
	SquadLabels []string        `json:"squad_labels"`
}

// AvailableFilterOptions représente les options disponibles après résolution.
type AvailableFilterOptions struct {
	ExperienceTypes []LabelValue `json:"experience_types"`
	Playlists       []LabelValue `json:"playlists"`
	Modes           []LabelValue `json:"modes"`
	Maps            []LabelValue `json:"maps"`
}

// FilterCounts représente les comptages avant/après filtrage.
type FilterCounts struct {
	TotalMatchesBeforeFilters int `json:"total_matches_before_filters"`
	TotalMatchesAfterFilters  int `json:"total_matches_after_filters"`
}

// FilterContextResolved est la réponse de POST filters/resolve.
type FilterContextResolved struct {
	Effective        FilterContextInput     `json:"effective"`
	AvailableOptions AvailableFilterOptions `json:"available_options"`
	SessionOptions   SessionOptions         `json:"session_options"`
	Counts           FilterCounts           `json:"counts"`
}
