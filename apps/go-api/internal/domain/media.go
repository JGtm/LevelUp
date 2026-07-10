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
	Page           int               `json:"page"`                  // numéro de page legacy (défaut 1)
	PageSize       int               `json:"page_size,omitempty"`   // taille de page legacy (défaut 24)
	Pagination     PaginationRequest `json:"pagination,omitempty"`  // pagination moderne React
	Kind           string            `json:"kind,omitempty"`        // filtre type legacy : "clip" | "screenshot"
	KindFilter     string            `json:"kind_filter,omitempty"` // filtre type moderne
	SectionFilter  string            `json:"section_filter,omitempty"`
	AuthorSlugs    []string          `json:"author_slugs,omitempty"`    // si non vide : whitelist player_slug (prend le pas sur SectionFilter)
	PlaylistFilter string            `json:"playlist_filter,omitempty"` // filtre par playlist_id (Partie rapide, Super Fiesta, Ranked…)
	MapFilter      string            `json:"map_filter,omitempty"`
	ModeFilter     string            `json:"mode_filter,omitempty"`
	GroupBy        string            `json:"group_by,omitempty"`
	Sort           string            `json:"sort,omitempty"`
	LikedOnly      bool              `json:"liked_only,omitempty"`
	UnassignedOnly bool              `json:"unassigned_only,omitempty"`
}

// MediaFilters regroupe les paramètres de filtrage/tri pour le repository.
type MediaFilters struct {
	KindFilter     string   // "clip" | "screenshot" | "" (aucun)
	SectionFilter  string   // "" (toutes sources) | "mine" (player_slug courant) | "teammate" (autres)
	AuthorSlugs    []string // whitelist explicite de player_slug (prend le pas sur SectionFilter si non vide)
	PlaylistFilter string   // filtre par playlist_id (UUID ou label brut) — match WHERE playlist_id = ? OR LOWER(label) = LOWER(?)
	MapFilter      string   // filtre par map (map_id ou label canonique)
	ModeFilter     string   // filtre par CATÉGORIE custom (Assassin/Fiesta/BTB/Ranked/Firefight/Other) — cf. analysis.InferModeCategoryFromPairName
	LikedOnly      bool     // restreindre aux médias likés
	UnassignedOnly bool     // restreindre aux médias sans match associé
	Sort           string   // "date_desc" | "date_asc" | "map_asc" | "mode_asc"
	GroupBy        string   // "" | "owner" | "map" | "mode" | "session" | "liked"
}

// MediaAuthor décrit un auteur sélectionnable dans le filtre Auteurs.
type MediaAuthor struct {
	PlayerSlug string `json:"player_slug"`
	Gamertag   string `json:"gamertag"`
	IsSelf     bool   `json:"is_self"`
	MediaCount int    `json:"media_count"` // nombre de fichiers détectés dans son dossier (best-effort)
}

// MediaAuthorsResponse regroupe la liste d'auteurs disponibles pour le filtre.
type MediaAuthorsResponse struct {
	Authors []MediaAuthor `json:"authors"`
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
	ModeName       *string // catégorie custom inférée côté Go depuis PairNameRaw
	MapID          *string
	// PairNameRaw : valeur brute (EN) de match_registry.pair_name. Sert à inférer
	// la mode_category custom (Assassin/Fiesta/BTB/Ranked/Firefight/Other) côté Go.
	PairNameRaw *string
	// PlayerSlug identifie le propriétaire du média (schéma shared_social uniquement, nil en legacy).
	// Dans ce projet playerSlug == gamertag.
	PlayerSlug *string
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
	Playlists []LabelValue `json:"playlists"`
	Maps      []LabelValue `json:"maps"`
	Modes     []LabelValue `json:"modes"`
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
	CapturesDir         string // chemin absolu résolu par le handler ({CapturesBase}/{Gamertag})
	CapturesBase        string // racine multi-player ({CapturesBase}/{slug}/...) — vide en mode legacy
	Gamertag            string // owner gamertag, requis pour construire les paths relatifs en DB
	DBPath              string // chemin vers stats.duckdb du joueur (fallback)
	SharedSocialDBPath  string // chemin vers shared_social.duckdb (cible principale)
	SharedMatchesDBPath string // chemin vers shared_matches_v2.duckdb (lecture match_registry)
	Tolerance           int    // buffer association match (minutes, défaut 2)
}

// UploadResult résume le résultat d'un upload multi-fichiers.
type UploadResult struct {
	Saved      int      `json:"saved"`             // fichiers écrits sur disque
	Skipped    int      `json:"skipped,omitempty"` // ré-uploads ignorés (même nom + même contenu)
	NewIndexed int      `json:"new_indexed"`       // nouvelles entrées media_files
	Associated int      `json:"associated"`        // associations matchs créées
	Thumbnails int      `json:"thumbnails"`        // miniatures générées
	Errors     []string `json:"errors,omitempty"`  // erreurs non-bloquantes
}

// MediaMatchCandidate représente un match potentiel pour une réassociation manuelle.
// Inclut le résultat du joueur + le lobby pour aider à reconnaître le bon match.
type MediaMatchCandidate struct {
	MatchID      string     `json:"match_id"`
	StartTime    *time.Time `json:"start_time,omitempty"`
	EndTime      *time.Time `json:"end_time,omitempty"`
	MapName      *string    `json:"map_name,omitempty"`      // FR si dispo
	MapImageURL  *string    `json:"map_image_url,omitempty"` // /static/maps/X.png (flat) ou /static/maps/{titleSlug}/X.png (title-scoped — Phase 6 finition multi-titres)
	ModeName     *string    `json:"mode_name,omitempty"`     // FR normalisé (préfixes/suffixes strippés)
	PlaylistName *string    `json:"playlist_name,omitempty"` // FR
	IsCurrent    bool       `json:"is_current"`              // true si ce match est l'association actuelle
	DeltaSeconds *int       `json:"delta_seconds,omitempty"` // écart capture vs start_time
	Outcome      *int       `json:"outcome,omitempty"`       // 2=victoire 3=défaite 1=égalité 4=DNF
	// Scores d'équipe, du POV du joueur courant (basés sur mp.team_id).
	// Nil si l'un des champs match_registry.team_X_score est NULL (FFA, certains modes objectifs).
	OwnScore   *int `json:"own_score,omitempty"`
	EnemyScore *int `json:"enemy_score,omitempty"`
	// Lobby : autres joueurs du match (max 8, équipes incluses)
	Lobby []MediaMatchLobbyEntry `json:"lobby,omitempty"`
}

// MediaMatchLobbyEntry décrit un joueur du lobby d'un match candidat.
type MediaMatchLobbyEntry struct {
	Gamertag string `json:"gamertag"`
	TeamID   *int   `json:"team_id,omitempty"`
	IsSelf   bool   `json:"is_self"` // true si gamertag == player courant
	IsBot    bool   `json:"is_bot"`  // détecté par xuid LIKE 'bid(%' — pour badge UI
}

// MediaMatchCandidatesResponse regroupe les matchs candidats pour un média.
type MediaMatchCandidatesResponse struct {
	FilePath      string                `json:"file_path"`
	CaptureUTC    *time.Time            `json:"capture_utc,omitempty"` // capture_start_utc utilisé pour la recherche
	WindowMinutes int                   `json:"window_minutes"`        // fenêtre ±X min utilisée
	Candidates    []MediaMatchCandidate `json:"candidates"`
}

// MediaAssociateRequest force l'association d'un média à un match précis.
type MediaAssociateRequest struct {
	FilePath string `json:"file_path"`
	MatchID  string `json:"match_id"`
}

// MediaAssociateResponse confirme la nouvelle association.
type MediaAssociateResponse struct {
	FilePath string  `json:"file_path"`
	MatchID  string  `json:"match_id"`
	MapName  *string `json:"map_name,omitempty"`
	ModeName *string `json:"mode_name,omitempty"`
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
