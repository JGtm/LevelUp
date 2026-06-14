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

// AssetTypeToEndpoint convertit un AssetType en segment d'URL discovery-infiniteugc.
// Segments camelCase EXACTS de l'API (cf. SPNKr/Grunt + sync.GetPlaylistConfig validé
// contre l'API réelle 2026-06-12) : /hi/{segment}/{assetId}/versions/{versionId}.
// NB : les anciens segments hyphénés (map-variants…) + le host gamecms-hacs étaient
// FAUX (403 systématique → asset_translations jamais peuplé au runtime).
var AssetTypeToEndpoint = map[AssetType]string{
	AssetTypeMap:         "maps",
	AssetTypePlaylist:    "playlists",
	AssetTypePair:        "mapModePairs",
	AssetTypeGameVariant: "ugcGameVariants",
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
	// ImageURL : URL CDN de la miniature (construite depuis le bloc Files :
	// Prefix + chemin d'image). Vide si l'asset n'expose pas d'image. Sert au
	// peuplement de map_images_registry.image_url pour les cartes inconnues.
	ImageURL string `json:"-"`
}

// MatchInfoAsset représente la structure {AssetId, VersionId} dans MatchInfo.
type MatchInfoAsset struct {
	AssetID   string `json:"AssetId"`
	VersionID string `json:"VersionId"`
}
