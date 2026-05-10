// Package halo — compare_provider.go : stats d'un joueur distant via Waypoint.
//
// Sprint 54 C : PlayerStatsProvider pour la comparaison joueur vs joueur.
// FetchRemoteStats : stats agrégées depuis l'endpoint career-stats.
// FetchCSR : CSR actuel + meilleur depuis skill.svc.halowaypoint.com.
package halo

import (
	"context"
	"encoding/json"
	"fmt"

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

// csrResponse est la réponse brute de skill.svc.halowaypoint.com/hi/playlist/{id}/csrs.
type csrResponse struct {
	Value []csrEntry `json:"Value"`
}

type csrEntry struct {
	Id     string    `json:"Id"`
	Result csrResult `json:"Result"`
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

// FetchCSR retourne le CSR actuel et le meilleur CSR historique depuis Waypoint.
// Endpoint : GET skill.svc.halowaypoint.com/hi/playlist/{playlistID}/csrs?players=xuid({xuid})
// Si Current.Value == -1 (placement), current est retourné comme 0.
func (p *HaloProvider) FetchCSR(ctx context.Context, xuid, playlistID string) (current, best int, err error) {
	tokens := ctxkeys.HaloTokens(ctx)
	if tokens == nil {
		return 0, 0, fmt.Errorf("FetchCSR: tokens absents du contexte")
	}
	if playlistID == "" {
		return 0, 0, fmt.Errorf("FetchCSR: playlistID vide")
	}

	url := fmt.Sprintf(
		"%s/hi/playlist/%s/csrs?players=xuid(%s)",
		defaultSkillHost,
		playlistID,
		xuid,
	)
	body, err := p.doGet(ctx, url, tokens)
	if err != nil {
		return 0, 0, fmt.Errorf("FetchCSR(%s): %w", xuid, err)
	}

	var resp csrResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, 0, fmt.Errorf("FetchCSR(%s) parse: %w", xuid, err)
	}
	if len(resp.Value) == 0 {
		return 0, 0, fmt.Errorf("FetchCSR(%s): réponse vide", xuid)
	}

	result := resp.Value[0].Result
	// Value == -1 signifie que le joueur est en matchs de placement.
	if result.Current.Value > 0 {
		current = result.Current.Value
	}
	if result.AllTimeMax.Value > 0 {
		best = result.AllTimeMax.Value
	}
	return current, best, nil
}
