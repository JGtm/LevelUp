// Package domain — media.go : types pour la page Galerie Médias.
//
// Sprint 13 :
//
//	POST /api/v1/players/{slug}/pages/media  → MediaPageResponse
package domain

import "time"

// ---------------------------------------------------------------------------
// Types de requête — Médias (POST body)
// ---------------------------------------------------------------------------

// MediaPageRequest : corps de POST /pages/media.
// Tous les champs sont optionnels.
type MediaPageRequest struct {
	Page     int    `json:"page"`                // numéro de page (défaut 1)
	PageSize int    `json:"page_size,omitempty"` // taille de page (défaut 24)
	Kind     string `json:"kind,omitempty"`      // filtre type : "clip" | "screenshot" | "" (tous)
}

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

// ---------------------------------------------------------------------------
// Upload
// ---------------------------------------------------------------------------

// UploadedFile est un fichier reçu via multipart/form-data.
type UploadedFile struct {
	OriginalName string
	Data         []byte
}

// UploadRequest regroupe les paramètres d'un upload multi-fichiers.
type UploadRequest struct {
	Files       []UploadedFile
	CapturesDir string // chemin absolu résolu par le handler
	DBPath      string // chemin vers stats.duckdb du joueur
	Tolerance   int    // tolérance association match (minutes)
}

// UploadResult résume le résultat d'un upload multi-fichiers.
type UploadResult struct {
	Saved      int      `json:"saved"`            // fichiers écrits sur disque
	NewIndexed int      `json:"new_indexed"`      // nouvelles entrées media_files
	Associated int      `json:"associated"`       // associations matchs créées
	Thumbnails int      `json:"thumbnails"`       // miniatures générées
	Errors     []string `json:"errors,omitempty"` // erreurs non-bloquantes
}
