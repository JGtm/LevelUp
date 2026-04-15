// Package domain — media.go : types pour la page Galerie Médias.
//
// Sprint 13 :
//   GET /api/v1/players/{slug}/pages/media  → MediaPageResponse
package domain

import "time"

// ---------------------------------------------------------------------------
// Lignes brutes DuckDB — Médias
// ---------------------------------------------------------------------------

// MediaFileRow est une ligne brute chargée depuis Q37 (media_files + associations).
type MediaFileRow struct {
	FilePath       string
	FileName       string
	Kind           string
	ThumbnailPath  *string
	CaptureEndUTC  *time.Time
	MatchID        *string
	MatchStartTime *time.Time
}

// ---------------------------------------------------------------------------
// Types de réponse — Médias
// ---------------------------------------------------------------------------

// MediaItem est un média enrichi pour l'affichage en galerie.
type MediaItem struct {
	FileName       string     `json:"file_name"`
	FilePath       string     `json:"file_path"`
	Kind           string     `json:"kind"`
	ThumbnailPath  *string    `json:"thumbnail_path,omitempty"`
	CaptureEndUTC  *time.Time `json:"capture_end_utc,omitempty"`
	MatchID        *string    `json:"match_id,omitempty"`
	MatchStartTime *time.Time `json:"match_start_time,omitempty"`
}

// MediaPageResponse est la réponse paginée de la galerie médias.
type MediaPageResponse struct {
	Items      []MediaItem `json:"items"`
	TotalCount int         `json:"total_count"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	HasMore    bool        `json:"has_more"`
}
