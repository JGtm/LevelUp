package infinite

// playlist.go — configuration d'une playlist (Discovery UGC). Copié de
// `sync/haloclient/halo_client_playlist.go` (GET, DTO et lecture des couples carte-mode).

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// PlaylistEntry : un couple carte-mode d'une playlist, sa version et son poids dans la rotation.
type PlaylistEntry struct {
	MapModePairAssetID string
	VersionID          string
	Weight             float64
}

// PlaylistConfig : le nom d'une playlist et ses couples carte-mode (même jamais joués).
type PlaylistConfig struct {
	PublicName string
	Entries    []PlaylistEntry
}

// PlaylistConfigDTO : la réponse brute de `GET /{jeu}/playlists/{id}/versions/{version}`. Les
// couples et leurs poids sont dans `RotationEntries` ; `CustomData.PlaylistEntries` (identifiants
// seuls) sert de repli pour la forme ancienne.
type PlaylistConfigDTO struct {
	PublicName      json.RawMessage `json:"PublicName"`
	RotationEntries []struct {
		AssetID   string `json:"AssetId"`
		VersionID string `json:"VersionId"`
		Metadata  struct {
			Weight float64 `json:"Weight"`
		} `json:"Metadata"`
	} `json:"RotationEntries"`
	CustomData struct {
		PlaylistEntries []struct {
			MapModePairAssetID string `json:"MapModePairAssetId"`
		} `json:"PlaylistEntries"`
	} `json:"CustomData"`
}

// GetPlaylistConfig récupère la configuration d'une playlist ([CleDiscoveryUGC]). `versionID` est
// requis (le service rend 404 sans, 400 pour « latest »).
func (c *Client) GetPlaylistConfig(ctx context.Context, auth Auth, playlistID, versionID string) (*PlaylistConfig, error) {
	if versionID == "" {
		return nil, fmt.Errorf("GetPlaylistConfig(%s): version_id requis", playlistID)
	}
	endpoint := fmt.Sprintf("%s/playlists/%s/versions/%s", c.endpoints.base(CleDiscoveryUGC),
		url.PathEscape(playlistID), url.PathEscape(versionID))
	body, err := c.get(ctx, requete{url: endpoint, auth: auth})
	if err != nil {
		return nil, fmt.Errorf("GetPlaylistConfig(%s): %w", playlistID, err)
	}
	return ParsePlaylistConfig(body)
}

// ParsePlaylistConfig décode une configuration de playlist : couples de `RotationEntries`, sinon
// (forme ancienne) identifiants de `CustomData.PlaylistEntries`, sans version ni poids.
func ParsePlaylistConfig(body []byte) (*PlaylistConfig, error) {
	var raw PlaylistConfigDTO
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsePlaylistConfig decode: %w", err)
	}
	cfg := &PlaylistConfig{PublicName: DecodeLocalizedName(raw.PublicName)}
	for _, e := range raw.RotationEntries {
		if e.AssetID == "" {
			continue
		}
		cfg.Entries = append(cfg.Entries, PlaylistEntry{
			MapModePairAssetID: e.AssetID,
			VersionID:          e.VersionID,
			Weight:             e.Metadata.Weight,
		})
	}
	if len(cfg.Entries) == 0 {
		for _, e := range raw.CustomData.PlaylistEntries {
			if e.MapModePairAssetID == "" {
				continue
			}
			cfg.Entries = append(cfg.Entries, PlaylistEntry{MapModePairAssetID: e.MapModePairAssetID})
		}
	}
	return cfg, nil
}
