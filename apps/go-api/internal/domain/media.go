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
	Page          int               `json:"page"`                  // numéro de page legacy (défaut 1)
	PageSize      int               `json:"page_size,omitempty"`   // taille de page legacy (défaut 24)
	Pagination    PaginationRequest `json:"pagination,omitempty"`  // pagination moderne React
	Kind          string            `json:"kind,omitempty"`        // filtre type legacy : "clip" | "screenshot"
	KindFilter    string            `json:"kind_filter,omitempty"` // filtre type moderne
	SectionFilter string            `json:"section_filter,omitempty"`
	MapFilter     string            `json:"map_filter,omitempty"`
	ModeFilter    string            `json:"mode_filter,omitempty"`
	GroupBy       string            `json:"group_by,omitempty"`
	Sort          string            `json:"sort,omitempty"`
	LikedOnly     bool              `json:"liked_only,omitempty"`
}

// MediaFilters regroupe les paramètres de filtrage/tri pour le repository.
type MediaFilters struct {
	KindFilter string // "clip" | "screenshot" | "" (aucun)
	MapFilter  string // filtre ILIKE sur le libellé de carte exposé à l'UI
	ModeFilter string // filtre ILIKE sur le mode normalisé exposé à l'UI
	LikedOnly  bool   // restreindre aux médias likés
	Sort       string // "date_desc" | "date_asc" | "map_asc" | "mode_asc"
}

// ResolvePage retourne le numéro de page normalisé.
func (r MediaPageRequest) ResolvePage() int {
	if r.Pagination.Page > 0 {
		return r.Pagination.Page
	}
	if r.Page > 0 {
		return r.Page
	}
	return 1
}

// ResolvePageSize retourne la taille de page normalisée et bornée.
func (r MediaPageRequest) ResolvePageSize(defaultPageSize, maxPageSize int) int {
	pageSize := r.Pagination.PageSize
	if pageSize <= 0 {
		pageSize = r.PageSize
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if maxPageSize > 0 && pageSize > maxPageSize {
		return maxPageSize
	}
	return pageSize
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
	Liked          bool
	MapName        *string
	ModeName       *string
	MapID          *string
}

// MediaLikersInfo contient les likers d'un média (max 3 noms + total).
type MediaLikersInfo struct {
	Names []string // max 3 premiers gamertags
	Total int
}

// MediaSectionTotals regroupe les compteurs par section (S56).
type MediaSectionTotals struct {
	Mine       int
	Unassigned int
}

// ---------------------------------------------------------------------------
// Types de réponse — Médias
// ---------------------------------------------------------------------------

// MediaItem est un média enrichi pour l'affichage en galerie.
type MediaItem struct {
	Basename       string     `json:"basename"`
	FilePath       string     `json:"file_path"`
	Kind           string     `json:"kind"`
	ThumbnailPath  *string    `json:"thumbnail_path,omitempty"`
	CaptureEndUTC  *time.Time `json:"capture_end_utc,omitempty"`
	MatchID        *string    `json:"match_id,omitempty"`
	MatchStartTime *time.Time `json:"match_start_time,omitempty"`
	Section        string     `json:"section"`
	OwnerGamertag  *string    `json:"owner_gamertag,omitempty"`
	MapName        *string    `json:"map_name,omitempty"`
	ModeName       *string    `json:"mode_name,omitempty"`
	Liked          bool       `json:"liked"`
	LikeCount      int        `json:"like_count"`
	// Likes sociaux : noms des 3 premiers likers + total
	Likers      []string `json:"likers,omitempty"`
	TotalLikers int      `json:"total_likers"`
}

// MediaItemsPage représente la table paginée de la galerie médias.
type MediaItemsPage struct {
	Items      []MediaItem    `json:"items"`
	Pagination PaginationMeta `json:"pagination"`
	Freshness  interface{}    `json:"freshness,omitempty"`
}

// MediaFilterOptions représente les options de filtres dédiées à la galerie.
type MediaFilterOptions struct {
	Maps  []LabelValue `json:"maps"`
	Modes []LabelValue `json:"modes"`
}

// MediaPageResponse est la réponse paginée de la galerie médias.
type MediaPageResponse struct {
	Items            MediaItemsPage     `json:"items"`
	TotalMine        int                `json:"total_mine"`
	TotalTeammates   int                `json:"total_teammates"`
	TotalUnassigned  int                `json:"total_unassigned"`
	AvailableFilters MediaFilterOptions `json:"available_filters"`
}

// MediaLikeRequest représente une mise à jour explicite de l'état liked.
type MediaLikeRequest struct {
	FilePath      string `json:"file_path"`
	Liked         bool   `json:"liked"`
	LikerSlug     string `json:"liker_slug,omitempty"`     // slug du joueur qui like
	LikerGamertag string `json:"liker_gamertag,omitempty"` // gamertag affiché dans les likers
}

// MediaLikeResponse confirme l'état liked persisté côté backend.
type MediaLikeResponse struct {
	FilePath    string   `json:"file_path"`
	Liked       bool     `json:"liked"`
	LikeCount   int      `json:"like_count"`
	Likers      []string `json:"likers,omitempty"`
	TotalLikers int      `json:"total_likers"`
}

// ---------------------------------------------------------------------------
// Upload
// ---------------------------------------------------------------------------

// UploadedFile est un fichier reçu via multipart/form-data.
type UploadedFile struct {
	OriginalName    string
	Data            []byte
	CaptureTimeUnix *int64 // mtime du fichier côté client (secondes Unix), optionnel
}

// UploadRequest regroupe les paramètres d'un upload multi-fichiers.
type UploadRequest struct {
	Files               []UploadedFile
	CapturesDir         string // chemin absolu résolu par le handler
	DBPath              string // chemin vers stats.duckdb du joueur (fallback)
	SharedSocialDBPath  string // chemin vers shared_social.duckdb (cible principale)
	SharedMatchesDBPath string // chemin vers shared_matches_v2.duckdb (lecture match_registry)
	Tolerance           int    // buffer association match (minutes, défaut 2)
}

// ReassociateRequest configure une ré-association forcée des médias.
type ReassociateRequest struct {
	DBPath              string
	SharedSocialDBPath  string
	SharedMatchesDBPath string
	BufferMin           int // 0 → défaut 2 min
}

// ReassociateResult résume le résultat de la ré-association.
type ReassociateResult struct {
	BackupTable  string   `json:"backup_table"`         // nom de la table snapshot
	DeletedAssoc int      `json:"deleted_associations"` // lignes supprimées
	NewAssoc     int      `json:"new_associations"`     // nouvelles associations créées
	Errors       []string `json:"errors,omitempty"`
}

// UploadResult résume le résultat d'un upload multi-fichiers.
type UploadResult struct {
	Saved      int      `json:"saved"`            // fichiers écrits sur disque
	NewIndexed int      `json:"new_indexed"`      // nouvelles entrées media_files
	Associated int      `json:"associated"`       // associations matchs créées
	Thumbnails int      `json:"thumbnails"`       // miniatures générées
	Errors     []string `json:"errors,omitempty"` // erreurs non-bloquantes
}

// MatchFavoriteRequest représente une demande de bascule favori.
type MatchFavoriteRequest struct {
	PlayerSlug string `json:"player_slug"`
	MatchID    string `json:"match_id"`
	Favorited  bool   `json:"favorited"`
}

// MatchFavoriteResponse représente la réponse après une bascule favori.
type MatchFavoriteResponse struct {
	PlayerSlug string `json:"player_slug"`
	MatchID    string `json:"match_id"`
	Favorited  bool   `json:"favorited"`
}
