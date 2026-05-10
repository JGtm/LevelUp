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

// matchSkillResponse est la réponse brute de skill.svc.halowaypoint.com/hi/matches/{id}/skill.
type matchSkillResponse struct {
	Value []matchSkillEntry `json:"Value"`
}

type matchSkillEntry struct {
	Id         string           `json:"Id"`
	ResultCode int              `json:"ResultCode"`
	Result     matchSkillResult `json:"Result"`
}

type matchSkillResult struct {
	RankRecap matchSkillRankRecap `json:"RankRecap"`
}

type matchSkillRankRecap struct {
	PostMatchCsr matchSkillCsrValue `json:"PostMatchCsr"`
}

type matchSkillCsrValue struct {
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

// FetchCSRFromMatch retourne le CSR actuel (PostMatchCsr) depuis l'endpoint match skill Waypoint.
// Endpoint : GET skill.svc.halowaypoint.com/hi/matches/{matchID}/skill?players=xuid({xuid})
// Retourne (0, nil) si le joueur n'a pas de données CSR pour ce match (placement ou non rankédé).
func (p *HaloProvider) FetchCSRFromMatch(ctx context.Context, matchID, xuid string) (current int, err error) {
	tokens := ctxkeys.HaloTokens(ctx)
	if tokens == nil {
		return 0, fmt.Errorf("FetchCSRFromMatch: tokens absents du contexte")
	}
	if matchID == "" {
		return 0, fmt.Errorf("FetchCSRFromMatch: matchID vide")
	}

	url := fmt.Sprintf(
		"%s/hi/matches/%s/skill?players=xuid(%s)",
		defaultSkillHost,
		matchID,
		xuid,
	)
	body, err := p.doGet(ctx, url, tokens)
	if err != nil {
		if strings.Contains(err.Error(), "HTTP 404") || strings.Contains(err.Error(), "HTTP 410") {
			return 0, nil
		}
		return 0, fmt.Errorf("FetchCSRFromMatch(%s): %w", xuid, err)
	}

	var resp matchSkillResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("FetchCSRFromMatch(%s) parse: %w", xuid, err)
	}
	if len(resp.Value) == 0 {
		return 0, nil
	}

	entry := resp.Value[0]
	// ResultCode != 0 = joueur sans rang compétitif pour ce match.
	if entry.ResultCode != 0 {
		return 0, nil
	}

	// Value <= 0 = en placement (non encore classé).
	if entry.Result.RankRecap.PostMatchCsr.Value > 0 {
		current = entry.Result.RankRecap.PostMatchCsr.Value
	}
	return current, nil
}
