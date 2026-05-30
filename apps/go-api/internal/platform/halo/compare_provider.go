// Package halo — compare_provider.go : stats d'un joueur distant via Waypoint.
//
// Sprint 54 C : PlayerStatsProvider pour la comparaison joueur vs joueur.
// FetchRemoteStats : stats agrégées depuis l'endpoint service record Waypoint
// (/hi/players/{player}/Matchmade/servicerecord). L'ancien chemin `/career-stats`
// n'existe pas dans l'API officielle (HTTP 404 pour tout joueur) — schéma et
// chemin corrects confirmés via le wrapper Grunt (Stats_GetPlayerServiceRecord).
package halo

import (
	"context"
	"encoding/json"
	"fmt"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// serviceRecordResponse est la réponse de l'endpoint service record Waypoint.
// Schéma : Wins/Losses/MatchesCompleted au top-level, le reste sous CoreStats
// (cf. Grunt Stats_GetPlayerServiceRecord).
type serviceRecordResponse struct {
	MatchesCompleted int `json:"MatchesCompleted"`
	Wins             int `json:"Wins"`
	Losses           int `json:"Losses"`
	CoreStats        struct {
		Kills       int     `json:"Kills"`
		Deaths      int     `json:"Deaths"`
		Assists     int     `json:"Assists"`
		ShotsFired  float64 `json:"ShotsFired"`
		ShotsHit    float64 `json:"ShotsHit"`
		DamageDealt float64 `json:"DamageDealt"`
		DamageTaken float64 `json:"DamageTaken"`
	} `json:"CoreStats"`
}

// FetchRemoteStats retourne les stats normalisées d'un joueur Waypoint depuis
// son service record matchmade. Les tokens sont lus depuis le contexte via
// ctxkeys. Retourne une erreur si le joueur est introuvable ou si l'auth est
// absente.
func (p *HaloProvider) FetchRemoteStats(ctx context.Context, gamertag, titleSlug string) (*domain.NormalizedPlayerStats, error) {
	tokens := ctxkeys.HaloTokens(ctx)
	if tokens == nil {
		return nil, fmt.Errorf("FetchRemoteStats: tokens absents du contexte")
	}

	// /hi/players/{gamertag}/Matchmade/servicerecord — LifecycleMode "Matchmade"
	// (PascalCase) ; pas de season marker → record cumulatif (lifetime).
	url := fmt.Sprintf(
		"%s/hi/players/%s/Matchmade/servicerecord",
		defaultStatsHost,
		gamertag,
	)
	body, err := p.doGet(ctx, url, tokens)
	if err != nil {
		return nil, fmt.Errorf("FetchRemoteStats(%s): %w", gamertag, err)
	}

	var resp serviceRecordResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("FetchRemoteStats(%s) parse: %w", gamertag, err)
	}

	cs := resp.CoreStats
	stats := domain.NormalizedPlayerStats{
		TitleSlug: titleSlug,
		Gamertag:  gamertag,
		Matches:   resp.MatchesCompleted,
	}
	if resp.MatchesCompleted > 0 {
		n := float64(resp.MatchesCompleted)
		stats.WinRate = float64(resp.Wins) / n
		stats.KillsPerGame = float64(cs.Kills) / n
		stats.DeathsPerGame = float64(cs.Deaths) / n
		stats.AssistsPerGame = float64(cs.Assists) / n
		stats.DamagePerGame = cs.DamageDealt / n
		stats.DamageTakenPerGame = cs.DamageTaken / n
		if cs.Deaths > 0 {
			stats.KDR = float64(cs.Kills) / float64(cs.Deaths)
			stats.KDA = (float64(cs.Kills) + 0.33*float64(cs.Assists)) / float64(cs.Deaths)
		}
		if cs.ShotsFired > 0 {
			stats.Accuracy = cs.ShotsHit / cs.ShotsFired
		}
	}
	return &stats, nil
}
