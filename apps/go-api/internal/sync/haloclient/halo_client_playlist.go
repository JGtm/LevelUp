package haloclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"levelup/go-api/internal/games"
)

// PlaylistEntry : un couple map-mode enfant d'une playlist + sa version + son poids
// relatif dans la rotation matchmaking (cf. blog den.dev "Halo Infinite Playlist
// Weights"). La VersionID permet de fetcher ensuite la config du couple lui-même.
type PlaylistEntry struct {
	MapModePairAssetID string
	VersionID          string
	Weight             float64
}

// PlaylistConfig : config d'une playlist via discovery-infiniteugc — son nom + la liste
// de ses couples map-mode enfants (même ceux jamais joués). C'est la découverte « A à Z »
// que match_registry seul ne donne pas.
type PlaylistConfig struct {
	PublicName string
	Entries    []PlaylistEntry
}

// playlistConfigRaw mappe la réponse JSON réelle de discovery-infiniteugc (validée
// contre l'API 2026-06-12). Les couples enfants + poids sont dans RotationEntries
// (AssetId/VersionId + Metadata.Weight) ; CustomData.PlaylistEntries (den.dev) sert de
// repli (IDs seuls, sans version).
type playlistConfigRaw struct {
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

// GetPlaylistConfig récupère la config d'une playlist depuis discovery-infiniteugc.
// versionID est REQUIS (l'API 404 sans, et "latest" renvoie 400) — le fournir depuis
// match_registry.playlist_version_id ou le catalogue. Auth Spartan + 343-clearance
// (gérés par doGet). Réutilise le host UGC du client film.
func (c *HaloAPIClient) GetPlaylistConfig(ctx context.Context, playlistID, versionID string) (*PlaylistConfig, error) {
	if versionID == "" {
		return nil, fmt.Errorf("GetPlaylistConfig(%s): version_id requis", playlistID)
	}
	endpoint := fmt.Sprintf("%s/%s/playlists/%s/versions/%s",
		c.hostFor(ctx, games.EndpointDiscoveryUGC, haloUGCHost), c.gamePrefix(ctx), url.PathEscape(playlistID), url.PathEscape(versionID))
	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("GetPlaylistConfig(%s): %w", playlistID, err)
	}
	return parsePlaylistConfig(body)
}

// parsePlaylistConfig décode la réponse JSON. Séparé pour être testable sans réseau.
func parsePlaylistConfig(body []byte) (*PlaylistConfig, error) {
	var raw playlistConfigRaw
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsePlaylistConfig decode: %w", err)
	}
	cfg := &PlaylistConfig{PublicName: decodeLocalizedName(raw.PublicName)}
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
	// Repli : si RotationEntries absent (vieille forme), prendre les IDs de
	// CustomData.PlaylistEntries (sans version ni poids).
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

// decodeLocalizedName tolère un PublicName string brut OU un objet {value: "..."}.
func decodeLocalizedName(raw json.RawMessage) string {
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
