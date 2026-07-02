// Package sync — halo_service_record.go : service record matchmade d'un joueur
// filtré par (saison, playlist) en UN appel (endpoint
// /hi/players/xuid(N)/Matchmade/servicerecord?seasonId=&playlistAssetId=).
//
// Parité Grunt (Stats_GetPlayerServiceRecord) / SPNKr (get_service_record) : renvoie
// l'agrégat CoreStats de la saison×playlist. Sert au classement mondial (B2) pour
// remplacer l'agrégation par-match (des milliers de GetMatchStats) par un appel par
// (joueur, playlist) — moins d'appels, moins de trous. Fonctionne pour un xuid
// arbitraire (endpoint public, comme la feature Comparaison).
package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/platform/halo/duration"
)

// serviceRecordRaw : sous-ensemble de la réponse service record (top-level
// Wins/Losses/MatchesCompleted/TimePlayed + CoreStats). Aligné sur le schéma validé
// par platform/halo.compare_provider (Stats_GetPlayerServiceRecord).
type serviceRecordRaw struct {
	MatchesCompleted int    `json:"MatchesCompleted"`
	Wins             int    `json:"Wins"`
	Losses           int    `json:"Losses"`
	TimePlayed       string `json:"TimePlayed"`
	CoreStats        struct {
		Kills       int64   `json:"Kills"`
		Deaths      int64   `json:"Deaths"`
		Assists     int64   `json:"Assists"`
		ShotsFired  float64 `json:"ShotsFired"`
		ShotsHit    float64 `json:"ShotsHit"`
		DamageDealt float64 `json:"DamageDealt"`
		DamageTaken float64 `json:"DamageTaken"`
		Medals      []struct {
			Count int `json:"Count"`
		} `json:"Medals"`
	} `json:"CoreStats"`
}

// GetSeasonPlaylistServiceRecord récupère l'agrégat de stats matchmade d'un joueur
// pour une (saison, playlist) donnée. `playlistID` vide → total de la saison (toutes
// playlists). Retourne (nil, nil) si aucune donnée : 404 (jamais jouée), 403 (record
// gated/privé), ou MatchesCompleted == 0 — le joueur est alors conservé sans stats.
func (c *HaloAPIClient) GetSeasonPlaylistServiceRecord(ctx context.Context, xuid, seasonID, playlistID string) (*domain.WorldServiceRecord, error) {
	if strings.TrimSpace(xuid) == "" {
		return nil, fmt.Errorf("GetSeasonPlaylistServiceRecord: xuid vide")
	}
	if strings.TrimSpace(seasonID) == "" {
		return nil, fmt.Errorf("GetSeasonPlaylistServiceRecord: seasonID vide")
	}

	q := url.Values{}
	q.Set("seasonId", seasonID)
	if s := strings.TrimSpace(playlistID); s != "" {
		q.Set("playlistAssetId", s)
	}
	endpoint := fmt.Sprintf("%s/%s/players/xuid(%s)/Matchmade/servicerecord?%s",
		c.hostFor(ctx, games.EndpointStats, haloStatsHost), c.gamePrefix(ctx), url.PathEscape(xuid), q.Encode())

	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		// 404 = jamais jouée ; 401/403 = record gated/privé → pas de stats, joueur
		// conservé (best-effort). Toute autre erreur remonte.
		if isNotFoundErr(err) || IsAuthError(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetSeasonPlaylistServiceRecord(%s,%s): %w", seasonID, playlistID, err)
	}
	return parseSeasonPlaylistServiceRecord(body)
}

// parseSeasonPlaylistServiceRecord décode la réponse et retourne (nil, nil) si la
// (saison, playlist) n'a aucun match. Helper pur (sans IO) pour tests hors réseau.
func parseSeasonPlaylistServiceRecord(body []byte) (*domain.WorldServiceRecord, error) {
	var raw serviceRecordRaw
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("GetSeasonPlaylistServiceRecord decode: %w", err)
	}
	if raw.MatchesCompleted <= 0 {
		return nil, nil
	}
	var medals int64
	for _, m := range raw.CoreStats.Medals {
		medals += int64(m.Count)
	}
	cs := raw.CoreStats
	return &domain.WorldServiceRecord{
		MatchesCompleted: raw.MatchesCompleted,
		Wins:             raw.Wins,
		Losses:           raw.Losses,
		TimePlayedSec:    duration.SecondsInt64(raw.TimePlayed),
		Kills:            cs.Kills,
		Deaths:           cs.Deaths,
		Assists:          cs.Assists,
		ShotsFired:       cs.ShotsFired,
		ShotsHit:         cs.ShotsHit,
		DamageDealt:      cs.DamageDealt,
		DamageTaken:      cs.DamageTaken,
		MedalCount:       medals,
	}, nil
}
