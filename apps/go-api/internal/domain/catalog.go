package domain

// catalog.go — types domain pour le catalogue Playlists/Pairs/Maps (Phase C plan catalogue).
//
// Ces types représentent une vue de lecture du catalogue persisté dans
// metadata.duckdb, retournée par port.CatalogRepo.

// CatalogPlaylist représente une playlist du catalogue, vue de lecture.
type CatalogPlaylist struct {
	TitleSlug        string `json:"title_slug"`
	PlaylistAssetID  string `json:"playlist_asset_id"`
	CurrentVersionID string `json:"current_version_id,omitempty"`
	Name             string `json:"name"`       // localisé selon ?lang= ou name_canonical fallback
	Experience       string `json:"experience"` // ranked/social/btb/firefight/...
	IsRanked         bool   `json:"is_ranked"`
	MatchCount       int    `json:"match_count,omitempty"` // si onlyPlayed=true, joint avec match_participants
}

// CatalogPair représente un pair map+mode du catalogue, vue de lecture.
type CatalogPair struct {
	TitleSlug          string  `json:"title_slug"`
	PairAssetID        string  `json:"pair_asset_id"`
	Name               string  `json:"name"`
	MapAssetID         string  `json:"map_asset_id"`
	GameVariantAssetID string  `json:"game_variant_asset_id"`
	ModeCategory       string  `json:"mode_category"`
	ModeLabel          string  `json:"mode_label,omitempty"` // localisé via pair_mode_label_translations
	Weight             float64 `json:"weight,omitempty"`     // depuis playlist_pair_links
}

// CatalogMap représente une map du catalogue, vue de lecture.
type CatalogMap struct {
	TitleSlug  string `json:"title_slug"`
	MapAssetID string `json:"map_asset_id"`
	Name       string `json:"name"`
	ImageURL   string `json:"image_url,omitempty"`
	MatchCount int    `json:"match_count,omitempty"`
}

// CatalogQueueEntry est une entrée PENDING de catalog_fetch_queue (asset pas encore
// présent dans la table catalogue de son type). Consommée par le drain du
// CatalogFetcherService (K1j), résolue via l'adapter puis persistée par le CatalogWriter.
type CatalogQueueEntry struct {
	AssetType string
	AssetID   string
	VersionID string
}
