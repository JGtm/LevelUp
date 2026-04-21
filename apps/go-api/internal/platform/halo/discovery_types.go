// Package halo — discovery_types.go : types pour l'API Discovery UGC (assets multilingues).
//
// Sprint 54 : peuplement asset_translations.
package halo

// AssetType représente les types d'assets disponibles dans Discovery UGC.
type AssetType string

const (
	AssetTypeMap         AssetType = "map"          // MapVariants
	AssetTypePlaylist    AssetType = "playlist"     // Playlists
	AssetTypePair        AssetType = "pair"         // PlaylistMapModePairs
	AssetTypeGameVariant AssetType = "game_variant" // UgcGameVariants
)

// AssetTypeToEndpoint convertit un AssetType en segment d'URL Discovery UGC.
var AssetTypeToEndpoint = map[AssetType]string{
	AssetTypeMap:         "map-variants",
	AssetTypePlaylist:    "playlists",
	AssetTypePair:        "playlist-map-mode-pairs",
	AssetTypeGameVariant: "ugc-game-variants",
}

// AssetTypeToMatchInfoKey convertit un AssetType en clé JSON MatchInfo.
// Utilisé pour extraire les version_ids depuis les stats de match.
var AssetTypeToMatchInfoKey = map[AssetType]string{
	AssetTypeMap:         "MapVariant",
	AssetTypePlaylist:    "Playlist",
	AssetTypePair:        "PlaylistMapModePair",
	AssetTypeGameVariant: "UgcGameVariant",
}

// DiscoveryAsset représente un asset récupéré depuis l'API Discovery UGC.
type DiscoveryAsset struct {
	AssetID     string `json:"AssetId"`
	VersionID   string `json:"VersionId"`
	PublicName  string `json:"PublicName"`
	Description string `json:"Description,omitempty"`
}

// MatchInfoAsset représente la structure {AssetId, VersionId} dans MatchInfo.
type MatchInfoAsset struct {
	AssetID   string `json:"AssetId"`
	VersionID string `json:"VersionId"`
}
