package infinite

// discovery.go — actifs Discovery UGC (playlists, cartes, couples carte-mode, variantes de jeu)
// avec leur nom localisé. Copié de `platform/halo/discovery_client.go` (`FetchAsset`) et
// `discovery_types.go`. Le service exige les jetons Spartan et 343-clearance ; version_id est
// requis (404 sans, 400 pour « latest »). Politique de reprise d'origine : [RetryDiscovery].

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
)

// AssetType : les types d'actifs de Discovery UGC.
type AssetType string

// Types d'actifs.
const (
	AssetTypeMap         AssetType = "map"
	AssetTypePlaylist    AssetType = "playlist"
	AssetTypePair        AssetType = "pair"
	AssetTypeGameVariant AssetType = "game_variant"
)

// SegmentDAsset rend le segment d'URL (camelCase exact du service) d'un type d'actif.
func SegmentDAsset(t AssetType) (string, bool) {
	switch t {
	case AssetTypeMap:
		return "maps", true
	case AssetTypePlaylist:
		return "playlists", true
	case AssetTypePair:
		return "mapModePairs", true
	case AssetTypeGameVariant:
		return "ugcGameVariants", true
	}
	return "", false
}

// CleMatchInfo rend la clé JSON de `MatchInfo` (stats d'un match) qui porte l'actif d'un type.
func CleMatchInfo(t AssetType) (string, bool) {
	switch t {
	case AssetTypeMap:
		return "MapVariant", true
	case AssetTypePlaylist:
		return "Playlist", true
	case AssetTypePair:
		return "PlaylistMapModePair", true
	case AssetTypeGameVariant:
		return "UgcGameVariant", true
	}
	return "", false
}

// DiscoveryAsset : un actif lu dans Discovery UGC, nom et description décodés, URL de miniature
// construite depuis la liste réelle de ses fichiers.
type DiscoveryAsset struct {
	AssetID     string `json:"AssetId"`
	VersionID   string `json:"VersionId"`
	PublicName  string `json:"PublicName"`
	Description string `json:"Description,omitempty"`
	// ImageURL : URL CDN de la miniature (bloc Files : Prefix + chemin d'image) ; vide si aucune.
	ImageURL string `json:"-"`
}

// MatchInfoAsset : la forme {AssetId, VersionId} d'un actif dans `MatchInfo`.
type MatchInfoAsset struct {
	AssetID   string `json:"AssetId"`
	VersionID string `json:"VersionId"`
}

// DiscoveryAssetDTO : la réponse brute de `GET /{jeu}/{segment}/{assetId}/versions/{versionId}`.
type DiscoveryAssetDTO struct {
	AssetID     string          `json:"AssetId"`
	VersionID   string          `json:"VersionId"`
	PublicName  json.RawMessage `json:"PublicName"`
	Description json.RawMessage `json:"Description"`
	Files       struct {
		Prefix            string   `json:"Prefix"`
		FileRelativePaths []string `json:"FileRelativePaths"`
	} `json:"Files"`
}

// RequeteActif désigne un actif de Discovery UGC et la langue de sa lecture.
type RequeteActif struct {
	Type      AssetType
	AssetID   string
	VersionID string
	// Langue : valeur d'`Accept-Language` ("" : pas d'en-tête).
	Langue string
}

// FetchAsset récupère un actif de Discovery UGC ([CleDiscoveryUGC]) dans la langue de la requête
// (donnée par l'appelant) et rend son nom localisé.
func (c *Client) FetchAsset(ctx context.Context, auth Auth, r RequeteActif) (*DiscoveryAsset, error) {
	assetType, assetID, versionID, langue := r.Type, r.AssetID, r.VersionID, r.Langue
	segment, ok := SegmentDAsset(assetType)
	if !ok {
		return nil, fmt.Errorf("asset type invalide : %s", assetType)
	}
	if versionID == "" {
		return nil, fmt.Errorf("fetch asset %s/%s: version_id requis (discovery-infiniteugc 404 sans)", assetType, assetID)
	}
	endpoint := fmt.Sprintf("%s/%s/%s/versions/%s", c.endpoints.base(CleDiscoveryUGC), segment,
		url.PathEscape(assetID), url.PathEscape(versionID))
	body, err := c.get(ctx, requete{url: endpoint, auth: auth, langue: langue})
	if err != nil {
		slog.DebugContext(ctx, "FetchAsset: request failed", "asset_type", assetType, "asset_id", assetID, "lang", langue, "err", err)
		return nil, fmt.Errorf("fetch asset %s/%s [%s]: %w", assetType, assetID, langue, err)
	}
	var raw DiscoveryAssetDTO
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode asset %s/%s: %w", assetType, assetID, err)
	}
	return &DiscoveryAsset{
		AssetID:     raw.AssetID,
		VersionID:   raw.VersionID,
		PublicName:  DecodeLocalizedName(raw.PublicName),
		Description: DecodeLocalizedName(raw.Description),
		ImageURL:    BuildAssetImageURL(raw.Files.Prefix, raw.Files.FileRelativePaths),
	}, nil
}

// DecodeLocalizedName lit un nom localisé : chaîne brute OU objet {value: "..."} ; "" sinon.
func DecodeLocalizedName(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var obj struct {
		Value string `json:"value"`
	}
	if json.Unmarshal(raw, &obj) == nil {
		return obj.Value
	}
	return ""
}

// BuildAssetImageURL construit l'URL CDN de la miniature d'un actif depuis son bloc Files : un
// chemin « thumbnail », sinon « hero » / « screenshot », sinon la première image
// (.jpg/.jpeg/.png/.webp) ; "" si rien d'exploitable. Fonction pure.
func BuildAssetImageURL(prefix string, relPaths []string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" || len(relPaths) == 0 {
		return ""
	}
	pick := func(pred func(string) bool) string {
		for _, p := range relPaths {
			if pred(strings.ToLower(strings.TrimSpace(p))) {
				return strings.TrimSpace(p)
			}
		}
		return ""
	}
	isImage := func(p string) bool {
		return strings.HasSuffix(p, ".jpg") || strings.HasSuffix(p, ".jpeg") ||
			strings.HasSuffix(p, ".png") || strings.HasSuffix(p, ".webp")
	}
	img := pick(func(p string) bool { return strings.Contains(p, "thumbnail") && isImage(p) })
	if img == "" {
		img = pick(func(p string) bool {
			return (strings.Contains(p, "hero") || strings.Contains(p, "screenshot")) && isImage(p)
		})
	}
	if img == "" {
		img = pick(isImage)
	}
	if img == "" {
		return ""
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return prefix + img
}
