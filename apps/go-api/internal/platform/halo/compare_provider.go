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
	"net/url"
	"strconv"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/halo/duration"
)

// serviceRecordResponse est la réponse de l'endpoint service record Waypoint.
// Schéma : Wins/Losses/MatchesCompleted + TimePlayed (ISO-8601) au top-level,
// CoreStats (+ Medals) en sous-objet (cf. Grunt Stats_GetPlayerServiceRecord).
type serviceRecordResponse struct {
	MatchesCompleted int    `json:"MatchesCompleted"`
	Wins             int    `json:"Wins"`
	Losses           int    `json:"Losses"`
	TimePlayed       string `json:"TimePlayed"` // ex: "P1DT7H50M24.6360455S"
	CoreStats        struct {
		Kills       int     `json:"Kills"`
		Deaths      int     `json:"Deaths"`
		Assists     int     `json:"Assists"`
		ShotsFired  float64 `json:"ShotsFired"`
		ShotsHit    float64 `json:"ShotsHit"`
		DamageDealt float64 `json:"DamageDealt"`
		DamageTaken float64 `json:"DamageTaken"`
		Medals      []struct {
			NameID int64 `json:"NameId"`
			Count  int   `json:"Count"`
		} `json:"Medals"`
	} `json:"CoreStats"`
	// Subqueries (service record lifetime uniquement) : dimensions disponibles
	// pour re-filtrer le service record. SeasonIds liste les saisons réellement
	// jouées par le joueur (cf. Grunt SubqueryContainer / SPNKr ServiceRecordSubqueries).
	Subqueries struct {
		SeasonIDs        []string `json:"SeasonIds"`
		IsRanked         []bool   `json:"IsRanked"`
		PlaylistAssetIDs []string `json:"PlaylistAssetIds"`
	} `json:"Subqueries"`
}

// FetchRemoteStats retourne les stats normalisées d'un joueur Waypoint (sans
// les médailles). Conservé pour Compare ; délègue à FetchServiceRecord.
func (p *HaloProvider) FetchRemoteStats(ctx context.Context, gamertag, titleSlug string) (*domain.NormalizedPlayerStats, error) {
	rec, err := p.FetchServiceRecord(ctx, gamertag, titleSlug)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, nil
	}
	s := rec.Stats
	return &s, nil
}

// FetchServiceRecord fetch le service record matchmade (lifetime) d'un joueur et
// retourne stats normalisées (incl. TimePlayedSeconds) + médailles agrégées.
// Un seul appel réseau. Les tokens sont lus depuis le contexte via ctxkeys.
// Enrobé du filet 401 (defense-in-depth) : re-mint + retry unique sur révocation.
func (p *HaloProvider) FetchServiceRecord(ctx context.Context, gamertag, titleSlug string) (*domain.RemoteServiceRecord, error) {
	return retryOnAuth(ctx, func(c context.Context) (*domain.RemoteServiceRecord, error) {
		return p.fetchServiceRecordOnce(c, gamertag, titleSlug)
	})
}

func (p *HaloProvider) fetchServiceRecordOnce(ctx context.Context, gamertag, titleSlug string) (*domain.RemoteServiceRecord, error) {
	tokens := ctxkeys.HaloTokens(ctx)
	if tokens == nil {
		return nil, fmt.Errorf("FetchServiceRecord: tokens absents du contexte")
	}

	// /hi/players/{gamertag}/Matchmade/servicerecord — LifecycleMode "Matchmade"
	// (PascalCase) ; pas de season marker → record cumulatif (lifetime).
	url := fmt.Sprintf(
		"%s/%s/players/%s/Matchmade/servicerecord",
		defaultStatsHost,
		p.gamePrefix(ctx),
		gamertag,
	)
	body, err := p.doGet(ctx, url, tokens)
	if err != nil {
		return nil, fmt.Errorf("FetchServiceRecord(%s): %w", gamertag, err)
	}

	var resp serviceRecordResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("FetchServiceRecord(%s) parse: %w", gamertag, err)
	}

	cs := resp.CoreStats
	stats := domain.NormalizedPlayerStats{
		TitleSlug:         titleSlug,
		Gamertag:          gamertag,
		Matches:           resp.MatchesCompleted,
		TimePlayedSeconds: duration.SecondsInt64(resp.TimePlayed),
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
		}
		// KDA = agrégat NET par match : ((Σkills + Σassists/3) − Σdeaths) / nb_matchs.
		// JAMAIS un quotient par les morts (le KDA est net, pas un ratio).
		stats.KDA = (float64(cs.Kills) + float64(cs.Assists)/3.0 - float64(cs.Deaths)) / n
		if cs.ShotsFired > 0 {
			stats.Accuracy = cs.ShotsHit / cs.ShotsFired
		}
	}

	medals := make([]domain.RemoteMedalCount, 0, len(cs.Medals))
	for _, m := range cs.Medals {
		if m.NameID == 0 || m.Count <= 0 {
			continue
		}
		medals = append(medals, domain.RemoteMedalCount{NameID: m.NameID, Count: m.Count})
	}

	return &domain.RemoteServiceRecord{
		Stats:            stats,
		Medals:           medals,
		SeasonIDs:        resp.Subqueries.SeasonIDs,
		PlaylistAssetIDs: resp.Subqueries.PlaylistAssetIDs,
	}, nil
}

// FetchSeasonServiceRecord retourne le nombre de matchs matchmade complétés par
// un joueur sur UNE saison donnée, optionnellement filtré classé/non-classé.
//
// Endpoint officiel (parité Grunt GetPlayerServiceRecordByXuid / SPNKr
// get_service_record) : /hi/players/{gamertag}/Matchmade/servicerecord avec les
// query params seasonId (chemin CMS, ex. "Seasons/Season7.json") et, si fourni,
// isRanked. Fonctionne pour un xuid/gamertag arbitraire. isRanked=nil → total
// de la saison (pas de filtre). Retourne (0, nil) si la saison n'a pas de donnée.
func (p *HaloProvider) FetchSeasonServiceRecord(ctx context.Context, gamertag, seasonID string, isRanked *bool) (int, error) {
	return retryOnAuth(ctx, func(c context.Context) (int, error) {
		return p.fetchSeasonServiceRecordOnce(c, gamertag, seasonID, isRanked)
	})
}

func (p *HaloProvider) fetchSeasonServiceRecordOnce(ctx context.Context, gamertag, seasonID string, isRanked *bool) (int, error) {
	tokens := ctxkeys.HaloTokens(ctx)
	if tokens == nil {
		return 0, fmt.Errorf("FetchSeasonServiceRecord: tokens absents du contexte")
	}
	if seasonID == "" {
		return 0, fmt.Errorf("FetchSeasonServiceRecord: seasonID vide")
	}

	rawURL := buildSeasonServiceRecordURL(p.gamePrefix(ctx), gamertag, seasonID, isRanked)
	body, err := p.doGet(ctx, rawURL, tokens)
	if err != nil {
		return 0, fmt.Errorf("FetchSeasonServiceRecord(%s, %s): %w", gamertag, seasonID, err)
	}

	var resp serviceRecordResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("FetchSeasonServiceRecord(%s, %s) parse: %w", gamertag, seasonID, err)
	}
	return resp.MatchesCompleted, nil
}

// buildSeasonServiceRecordURL construit l'URL du service record filtré par
// saison (et optionnellement par isRanked). Helper pur (testable sans réseau) :
// le seasonID contient des slashes ("Seasons/Season7.json") → encodés en query
// par url.Values. Casing seasonId/isRanked aligné sur le wrapper Grunt.
func buildSeasonServiceRecordURL(gamePrefix, gamertag, seasonID string, isRanked *bool) string {
	q := url.Values{}
	q.Set("seasonId", seasonID)
	if isRanked != nil {
		q.Set("isRanked", strconv.FormatBool(*isRanked))
	}
	return fmt.Sprintf("%s/%s/players/%s/Matchmade/servicerecord?%s", defaultStatsHost, gamePrefix, gamertag, q.Encode())
}
