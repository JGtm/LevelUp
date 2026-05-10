// Package halo — compare_provider.go : stats d'un joueur distant via Waypoint.
//
// Sprint 54 C : PlayerStatsProvider pour la comparaison joueur vs joueur.
// FetchRemoteStats : stats agrégées depuis l'endpoint career-stats.
// FetchCSRDirect : CSR actuel + meilleur via découverte dynamique des playlist IDs.
package halo

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

const defaultSkillHost = "https://skill.svc.halowaypoint.com"

// compareStatsResponse est la réponse brute de l'endpoint career stats Waypoint.
type compareStatsResponse struct {
	Gamertag           string  `json:"Gamertag"`
	TotalMatchesPlayed int     `json:"TotalMatchesPlayed"`
	TotalWins          int     `json:"TotalWins"`
	TotalKills         int     `json:"TotalKills"`
	TotalDeaths        int     `json:"TotalDeaths"`
	TotalAssists       int     `json:"TotalAssists"`
	ShotsFired         float64 `json:"ShotsFired"`
	ShotsHit           float64 `json:"ShotsHit"`
	TotalDamageDealt   float64 `json:"TotalDamageDealt"`
	TotalDamageTaken   float64 `json:"TotalDamageTaken"`
}

// rankedPlaylistIDs liste les playlists ranked connues d'Halo Infinite (hardcodé comme SpartanRecord).
// Essayées dans l'ordre pour récupérer le CSR d'un joueur sans lookup BDD.
var rankedPlaylistIDs = []string{
	"edfef3ac-9cbe-4fa2-b949-8f29deafd483", // Ranked Arena (Open Crossplay)
	"f7f30787-f607-436b-bdec-44c65bc2ecef", // Ranked Arena (Solo-Duo Controller)
	"f7eb8c71-fedb-4696-8c0f-96025e285ffd", // Ranked Arena (Solo-Duo MnK)
}

// csrResponse est la réponse brute de skill.svc.halowaypoint.com/hi/playlist/{id}/csrs.
type csrResponse struct {
	Value []csrEntry `json:"Value"`
}

type csrEntry struct {
	Id         string    `json:"Id"`
	ResultCode int       `json:"ResultCode"`
	Result     csrResult `json:"Result"`
}

type csrResult struct {
	Current    csrRating `json:"Current"`
	AllTimeMax csrRating `json:"AllTimeMax"`
}

type csrRating struct {
	Value int `json:"Value"`
}

// FetchRemoteStats retourne les stats normalisées d'un joueur Waypoint.
// Les tokens sont lus depuis le contexte via ctxkeys.
// Retourne une erreur si le joueur est introuvable ou si l'auth est absente.
func (p *HaloProvider) FetchRemoteStats(ctx context.Context, gamertag, titleSlug string) (*domain.NormalizedPlayerStats, error) {
	tokens := ctxkeys.HaloTokens(ctx)
	if tokens == nil {
		return nil, fmt.Errorf("FetchRemoteStats: tokens absents du contexte")
	}

	url := fmt.Sprintf(
		"%s/hi/players/%s/career-stats",
		defaultStatsHost,
		gamertag,
	)
	body, err := p.doGet(ctx, url, tokens)
	if err != nil {
		return nil, fmt.Errorf("FetchRemoteStats(%s): %w", gamertag, err)
	}

	var resp compareStatsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("FetchRemoteStats(%s) parse: %w", gamertag, err)
	}

	stats := domain.NormalizedPlayerStats{
		TitleSlug: titleSlug,
		Gamertag:  gamertag,
		Matches:   resp.TotalMatchesPlayed,
	}
	if resp.TotalMatchesPlayed > 0 {
		n := float64(resp.TotalMatchesPlayed)
		stats.WinRate = float64(resp.TotalWins) / n
		stats.KillsPerGame = float64(resp.TotalKills) / n
		stats.DeathsPerGame = float64(resp.TotalDeaths) / n
		stats.AssistsPerGame = float64(resp.TotalAssists) / n
		stats.DamagePerGame = resp.TotalDamageDealt / n
		stats.DamageTakenPerGame = resp.TotalDamageTaken / n
		if resp.TotalDeaths > 0 {
			stats.KDR = float64(resp.TotalKills) / float64(resp.TotalDeaths)
			stats.KDA = (float64(resp.TotalKills) + 0.33*float64(resp.TotalAssists)) / float64(resp.TotalDeaths)
		}
		if resp.ShotsFired > 0 {
			stats.Accuracy = resp.ShotsHit / resp.ShotsFired
		}
	}
	return &stats, nil
}

// FetchCSRDirect retourne le CSR actuel et le meilleur CSR depuis Waypoint.
// Découverte dynamique des playlist IDs depuis l'historique des matchs pour
// ne pas dépendre d'IDs hardcodés qui changent à chaque saison Halo.
func (p *HaloProvider) FetchCSRDirect(ctx context.Context, xuid string) (current, best int, err error) {
	tokens := ctxkeys.HaloTokens(ctx)
	if tokens == nil {
		return 0, 0, fmt.Errorf("FetchCSRDirect: tokens absents du contexte")
	}

	playlistIDs := p.discoverPlaylistIDs(ctx, xuid, tokens)

	for _, playlistID := range playlistIDs {
		csrURL := fmt.Sprintf("%s/hi/playlist/%s/csrs?players=xuid(%s)",
			defaultSkillHost, playlistID, xuid)
		body, fetchErr := p.doGet(ctx, csrURL, tokens)
		if fetchErr != nil {
			isGone := strings.Contains(fetchErr.Error(), "HTTP 404") ||
				strings.Contains(fetchErr.Error(), "HTTP 410")
			if !isGone {
				slog.WarnContext(ctx, "FetchCSRDirect: erreur CSR playlist",
					"playlist", playlistID, "xuid", xuid, "err", fetchErr)
			}
			continue
		}
		var resp csrResponse
		if parseErr := json.Unmarshal(body, &resp); parseErr != nil {
			continue
		}
		if len(resp.Value) == 0 || resp.Value[0].ResultCode != 0 {
			continue // joueur non classé dans cette playlist
		}
		entry := resp.Value[0]
		if entry.Result.Current.Value > 0 {
			current = entry.Result.Current.Value
		}
		if entry.Result.AllTimeMax.Value > 0 {
			best = entry.Result.AllTimeMax.Value
		}
		if current > 0 || best > 0 {
			return current, best, nil
		}
	}
	slog.DebugContext(ctx, "FetchCSRDirect: aucun CSR trouvé",
		"xuid", xuid, "playlists_essayées", len(playlistIDs))
	return 0, 0, nil
}

// discoverPlaylistIDs récupère les playlist IDs depuis l'historique des matchs
// du joueur (dynamique) et complète avec les IDs ranked connus (fallback).
func (p *HaloProvider) discoverPlaylistIDs(ctx context.Context, xuid string, tokens *domain.HaloTokens) []string {
	histURL := fmt.Sprintf(
		"%s/hi/players/xuid(%s)/matches?type=matchmaking&count=25&start=0",
		defaultStatsHost, xuid)
	histBody, histErr := p.doGet(ctx, histURL, tokens)
	if histErr != nil {
		slog.WarnContext(ctx, "FetchCSRDirect: historique matchmaking indisponible — IDs hardcodés",
			"xuid", xuid, "err", histErr)
		return rankedPlaylistIDs
	}

	var history struct {
		Results []struct {
			MatchInfo struct {
				Playlist *struct {
					AssetId string `json:"AssetId"`
				} `json:"Playlist"`
			} `json:"MatchInfo"`
		} `json:"Results"`
	}
	if json.Unmarshal(histBody, &history) != nil {
		return rankedPlaylistIDs
	}

	seen := make(map[string]bool)
	ids := make([]string, 0, 8)
	for _, r := range history.Results {
		if r.MatchInfo.Playlist == nil {
			continue
		}
		if pid := r.MatchInfo.Playlist.AssetId; pid != "" && !seen[pid] {
			seen[pid] = true
			ids = append(ids, pid)
		}
	}
	// Ajouter les IDs hardcodés non déjà présents comme fallback.
	for _, pid := range rankedPlaylistIDs {
		if !seen[pid] {
			ids = append(ids, pid)
		}
	}
	slog.DebugContext(ctx, "FetchCSRDirect: playlists découvertes",
		"xuid", xuid, "depuis_historique", len(ids)-len(rankedPlaylistIDs), "total", len(ids))
	return ids
}
