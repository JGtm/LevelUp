// Package domain — types du gestionnaire de filtres.
package domain

import (
	"fmt"
	"time"
)

// LabelValue est une paire label/value pour les options de sélection.
//
// Parent (optionnel) sert à grouper les options dans un dropdown hiérarchique
// (HTML <optgroup>). Utilisé par le filtre Mode des médias où les sous-modes
// (Slayer, CTF, Team Slayer…) sont regroupés sous leur catégorie parente
// (Assassin, Fiesta, …). Si vide, l'option est un header de catégorie ou
// hors hiérarchie.
type LabelValue struct {
	Label string `json:"label"`
	Value string `json:"value"`
	// Count : nombre de matchs si on AJOUTE cette option à la sélection courante
	// de la catégorie (sémantique OR). Pour une option déjà cochée, vaut le total
	// post-cascade actuel. Pour une option non cochée, vaut le count si on la coche.
	// Permet à l'UI d'afficher "Mode (42)" et de griser/masquer les options à 0.
	Count  int    `json:"count"`
	Parent string `json:"parent,omitempty"`
}

// FilterMatchRow représente une ligne minimale pour la résolution des filtres.
// Type de transfert entre platform/duckdb et les services.
type FilterMatchRow struct {
	MatchID        string
	StartTime      *time.Time
	MapName        *string // nom EN brut
	MapNameFR      *string // COALESCE(map_name_fr, map_name), enrichi par applyMapFRTranslations
	MapID          *string // UUID asset — clé de résolution metadata (titres sans noms registry)
	PairName       *string // nom EN brut
	PairNameFR     *string // COALESCE, enrichi par applyModeFRTranslations
	PairID         *string // UUID asset, clé de lookup asset_translations (fallback enrichissement FR)
	PlaylistName   *string // COALESCE(playlist_name_fr, playlist_name), enrichi par applyPlaylistFRTranslations
	PlaylistNameEN *string // playlist_name EN brut — clé de migration cascade
	PlaylistID     *string // UUID asset — clé de résolution metadata (titres sans noms registry)
	// game_variant : source de MODE des titres sans pair (Halo 5), où pair_id/pair_name
	// sont NULL. Résolu read-side depuis asset_translations via applyAssetNamesFromMetadata.
	GameVariantID     *string // UUID asset — clé de résolution metadata
	GameVariantName   *string // EN — source de MODE des titres sans pair (Halo 5)
	GameVariantNameFR *string // FR (sinon EN) — enrichi par applyAssetNamesFromMetadata
	IsFirefight       bool
	IsRanked          bool
	SessionID         *string
	SessionLabel      *string
	IsWithFriends     bool
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

	// MatchContext restreint les matchs selon le contexte de la page appelante (Phase C plan catalogue).
	// Consommé par FiltersService.Resolve() pour appliquer le filtre is_with_friends en Phase I.
	//
	//   "solo"  : is_with_friends = false (matchs solo uniquement)
	//   "squad" : is_with_friends = true  (matchs en groupe uniquement, pages escouade)
	//   "all"   : pas de filtre supplémentaire (défaut, comportement actuel)
	MatchContext string `json:"match_context,omitempty"`
}

// MatchContext valid values, exposed for service-layer validation.
const (
	MatchContextSolo  = "solo"
	MatchContextSquad = "squad"
	MatchContextAll   = "all"
)

// IsValidMatchContext valide une valeur MatchContext (string vide acceptée = "all").
func IsValidMatchContext(s string) bool {
	switch s {
	case "", MatchContextSolo, MatchContextSquad, MatchContextAll:
		return true
	}
	return false
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
	if !IsValidMatchContext(f.MatchContext) {
		return fmt.Errorf("FilterContextInput: match_context invalide %q (attendu : solo|squad|all)", f.MatchContext)
	}
	return nil
}

// SessionOption représente une session disponible.
type SessionOption struct {
	Label      string `json:"label"`
	SessionID  string `json:"session_id"`
	MatchCount int    `json:"match_count"`
	// MatchCountFiltered : nombre de matchs de la session si elle était sélectionnée
	// avec les autres filtres actifs (cascade + match_context). Permet à l'UI de
	// masquer les sessions retournant 0 sous la sélection courante.
	MatchCountFiltered int  `json:"match_count_filtered"`
	IsSquad            bool `json:"is_squad"`
	// StartedAtUTC / EndedAtUTC : timestamps du premier et dernier match de la
	// session. Permettent à l'UI (PeriodSessionRail) de formater des labels
	// localisés type « Session du 6 avril 2026 de 21:43 à 23:40 » sans dépendre
	// du label backend qui peut être tronqué/anglicisé.
	StartedAtUTC time.Time `json:"started_at_utc"`
	EndedAtUTC   time.Time `json:"ended_at_utc"`
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

// PeriodPresetCount donne, pour un preset de période (7j/30j/90j/Toutes),
// le nombre de matchs qu'il contiendrait si l'utilisateur l'activait
// (en gardant la cascade et le match_context actifs).
type PeriodPresetCount struct {
	PresetID string `json:"preset_id"` // "7d" | "30d" | "90d" | "all"
	Days     int    `json:"days"`      // 7, 30, 90, 0 (=all)
	Count    int    `json:"count"`
}

// SeasonCount donne, pour une saison du catalog (kind="season" dans
// assets.toml), le nombre de matchs qu'elle contiendrait si l'utilisateur
// la sélectionnait via la SaisonPill (cascade et match_context appliqués).
//
// Symétrique de PeriodPresetCount : sert au folding "+N saisons sans matchs"
// côté frontend.
type SeasonCount struct {
	SeasonID string `json:"season_id"` // ex: "season6", "season10_op1"
	Count    int    `json:"count"`
}

// FilterContextResolved est la réponse de POST filters/resolve.
type FilterContextResolved struct {
	Effective        FilterContextInput     `json:"effective"`
	AvailableOptions AvailableFilterOptions `json:"available_options"`
	SessionOptions   SessionOptions         `json:"session_options"`
	Counts           FilterCounts           `json:"counts"`
	PeriodPresets    []PeriodPresetCount    `json:"period_presets"`
	SeasonCounts     []SeasonCount          `json:"season_counts,omitempty"` // nil si le titre n'a pas de kind "season"
}

// FilterMatchIDsResponse est la réponse de POST filters/match-ids : la liste
// ordonnée (start_time DESC, récent d'abord) des match_id de la sélection
// courante. Alimente le bouton "Voir les matchs" (L2) qui ouvre le 1er match
// et parcourt la liste via prev/next. Calculée par le MÊME pipeline que
// FilterContextResolved.Counts → respecte match_context (solo/squad), sessions,
// période et cascade, là où le fallback /neighbors (shared-only) ne le peut pas.
type FilterMatchIDsResponse struct {
	MatchIDs []string `json:"match_ids"`
}
