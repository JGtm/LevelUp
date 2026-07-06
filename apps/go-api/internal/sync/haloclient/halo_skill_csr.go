// Package sync — halo_skill_csr.go : CSR par playlist (endpoint skill
// /hi/playlist/{id}/csrs). Mécanisme canonique Grunt (Skill.GetPlaylistCsr) /
// SPNKr (skill.get_playlist_csr) : fonctionne pour n'importe quelle playlist et
// n'importe quelle saison, renvoie "Non classé" si jamais jouée. Permet
// d'afficher toutes les playlists classées sans dériver de l'historique.
//
// Extrait de halo_skill.go (règle 500 lignes/fichier). Les types partagés
// (PlayerPlaylistCSR, csrRankRaw, rawToCSRSnapshot) + le player-level
// GetPlayerCSRs restent dans halo_skill.go (même package).
package haloclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"levelup/go-api/internal/games"
)

// playlistCSRResponse : réponse de GET /hi/playlist/{id}/csrs?players=...&season=...
// Contrairement au player-level (/hi/players/.../csrs), la playlist est dans
// l'URL — Result ne porte donc pas PlaylistId/Queue/Input.
type playlistCSRResponse struct {
	Value []struct {
		ID         string `json:"Id"`
		ResultCode int    `json:"ResultCode"`
		Result     struct {
			Current    csrRankRaw `json:"Current"`
			SeasonMax  csrRankRaw `json:"SeasonMax"`
			AllTimeMax csrRankRaw `json:"AllTimeMax"`
		} `json:"Result"`
	} `json:"Value"`
}

// GetPlaylistCsr récupère le CSR d'un joueur pour UNE playlist classée et une
// saison donnée. Contrairement au player-level GetPlayerCSRs (qui ne renvoie que
// les playlists ENGAGÉES), cet endpoint répond pour N'IMPORTE QUELLE playlist —
// y compris "Non classé" si le joueur ne l'a jamais jouée.
//
// Retourne (nil, nil) si l'API répond 404 ou ne renvoie pas de résultat pour le
// xuid (joueur non classé sans entrée). PlaylistName/Queue/Input restent vides :
// le caller les renseigne depuis la référence rankedplaylists.
func (c *HaloAPIClient) GetPlaylistCsr(ctx context.Context, playlistID, xuid, seasonID string) (*PlayerPlaylistCSR, error) {
	if strings.TrimSpace(playlistID) == "" {
		return nil, fmt.Errorf("GetPlaylistCsr: playlistID vide")
	}
	if strings.TrimSpace(xuid) == "" {
		return nil, fmt.Errorf("GetPlaylistCsr: xuid vide")
	}

	endpoint := fmt.Sprintf("%s/%s/playlist/%s/csrs?players=xuid(%s)",
		c.hostFor(ctx, games.EndpointSkill, haloSkillHost), c.gamePrefix(ctx), url.PathEscape(playlistID), url.PathEscape(xuid))
	if s := strings.TrimSpace(seasonID); s != "" {
		endpoint += "&season=" + url.QueryEscape(s)
	}

	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		if isNotFoundErr(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetPlaylistCsr(%s): %w", playlistID, err)
	}

	csr, err := parsePlaylistCSR(body, playlistID, xuid)
	if err != nil {
		return nil, fmt.Errorf("GetPlaylistCsr decode(%s): %w", playlistID, err)
	}
	return csr, nil
}

// parsePlaylistCSR décode la réponse /hi/playlist/{id}/csrs et retourne le CSR du
// xuid demandé. (nil, nil) si le xuid n'a pas d'entrée (non classé sans data).
// Helper pur (sans IO) pour faciliter les tests du parser.
func parsePlaylistCSR(body []byte, playlistID, xuid string) (*PlayerPlaylistCSR, error) {
	var resp playlistCSRResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	for _, v := range resp.Value {
		if v.ResultCode != 0 || unwrapXUID(v.ID) != xuid {
			continue
		}
		return &PlayerPlaylistCSR{
			PlaylistID: playlistID,
			Current:    rawToCSRSnapshot(v.Result.Current),
			Season:     rawToCSRSnapshot(v.Result.SeasonMax),
			AllTime:    rawToCSRSnapshot(v.Result.AllTimeMax),
		}, nil
	}
	return nil, nil
}
