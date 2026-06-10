// Package service — world_player_stats_aggregator.go : agrégateur one-shot
// MULTI-TOKENS des stats joueur du classement mondial (Phase C,
// PLAN_WORLD_LEADERBOARD_ENRICHED.md).
//
// Pipeline par joueur :
//  1. résolution gamertag -> xuid (PeopleHub, single-token RTA — bas volume)
//  2. pagination GetMatchHistory (matchmaking) via le PooledHaloClient
//     (PolicyAnyPublic = round-robin sur TOUS les tokens du pool → parallélisme)
//  3. GetMatchStats par match -> extraction pure (analysis.ExtractPlayerMatchStat)
//  4. accumulation par (saison CSR, playlist) -> analysis.AccumulateWorldStats
//
// Le fan-out entre joueurs est borné par Concurrency ; le RPS effectif global est
// déjà plafonné par le pool (PerTokenRPS × nb tokens). La persistance
// (InsertPlayerSeasonStats) est laissée au caller (cron/CLI) pour garder
// l'agrégateur testable sans DB.
package service

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	syncpkg "levelup/go-api/internal/sync"
)

const worldMatchPageSize = 25 // plafond API Halo GetMatchHistory

// Garantit que le client multi-tokens du pool satisfait la surface attendue.
var _ WorldMatchSource = (*syncpkg.PooledHaloClient)(nil)

// WorldMatchSource : sous-ensemble de la surface client suffisant pour
// l'agrégation. Satisfait par *syncpkg.PooledHaloClient (multi-tokens).
type WorldMatchSource interface {
	GetMatchHistory(ctx context.Context, gamertag, matchType string, start, count int) ([]syncpkg.MatchHistoryEntry, error)
	GetMatchStats(ctx context.Context, matchID string) (map[string]any, error)
}

// WorldXUIDResolver résout gamertag -> xuid numérique (PeopleHub, single-token).
// Satisfait par *auth.PeopleHubResolver.
type WorldXUIDResolver interface {
	ResolveXUID(ctx context.Context, gamertag string) (string, error)
}

// WorldStatsAggregatorConfig paramètre l'agrégation.
type WorldStatsAggregatorConfig struct {
	// TargetSeasons : saisons CSR normalisées à conserver (ex {"csrseason13-2":true}).
	// Vide = conserver toutes les saisons rencontrées (backfill complet, Phase D).
	TargetSeasons map[string]bool
	// MaxPages : plafond de pages d'historique par joueur. 0 = défaut 40 (~1000 matchs).
	MaxPages int
	// StopAfterNonTarget : arrête la pagination après N matchs consécutifs hors
	// TargetSeasons (historique chronologique décroissant → une fois sous les saisons
	// cibles on ne remonte plus). 0 = défaut 50. **Négatif = désactivé** (scan
	// jusqu'à MaxPages — requis pour backfiller une VIEILLE saison, sinon l'arrêt
	// se déclenche sur les matchs récents avant d'atteindre la cible). Ignoré si
	// TargetSeasons vide.
	StopAfterNonTarget int
	// Concurrency : nb de joueurs traités en parallèle. 0 = défaut 8.
	Concurrency int
}

func (c *WorldStatsAggregatorConfig) withDefaults() {
	if c.MaxPages <= 0 {
		c.MaxPages = 40
	}
	if c.StopAfterNonTarget == 0 {
		c.StopAfterNonTarget = 50 // négatif laissé tel quel = désactivé (backfill)
	}
	if c.Concurrency <= 0 {
		c.Concurrency = 8
	}
}

// WorldStatsAggregator agrège les stats brutes par (saison, playlist) pour un
// ensemble de joueurs, via un client multi-tokens.
type WorldStatsAggregator struct {
	src      WorldMatchSource
	resolver WorldXUIDResolver
	cfg      WorldStatsAggregatorConfig
}

// NewWorldStatsAggregator construit l'agrégateur. `src` doit être un client
// multi-tokens (typiquement *syncpkg.PooledHaloClient construit avec PolicyAnyPublic).
func NewWorldStatsAggregator(src WorldMatchSource, resolver WorldXUIDResolver, cfg WorldStatsAggregatorConfig) *WorldStatsAggregator {
	cfg.withDefaults()
	return &WorldStatsAggregator{src: src, resolver: resolver, cfg: cfg}
}

// AggregatePlayer résout l'xuid, collecte l'historique et accumule les stats du
// joueur par (saison, playlist).
func (a *WorldStatsAggregator) AggregatePlayer(ctx context.Context, gamertag string) ([]domain.WorldPlayerSeasonStats, error) {
	xuid, err := a.resolver.ResolveXUID(ctx, gamertag)
	if err != nil {
		return nil, fmt.Errorf("resolve xuid %q: %w", gamertag, err)
	}
	stats, err := a.collectPlayerMatches(ctx, xuid)
	if err != nil {
		return nil, err
	}
	return analysis.AccumulateWorldStats(gamertag, stats), nil
}

// collectPlayerMatches pagine l'historique matchmaking et extrait les stats par
// match. S'arrête à MaxPages, en fin d'historique, ou après StopAfterNonTarget
// matchs consécutifs hors saisons cibles.
func (a *WorldStatsAggregator) collectPlayerMatches(ctx context.Context, xuid string) ([]analysis.PlayerMatchStat, error) {
	player := "xuid(" + xuid + ")"
	var collected []analysis.PlayerMatchStat
	nonTarget := 0
	for page := 0; page < a.cfg.MaxPages; page++ {
		if err := ctx.Err(); err != nil {
			return collected, err
		}
		hist, err := a.src.GetMatchHistory(ctx, player, "matchmaking", page*worldMatchPageSize, worldMatchPageSize)
		if err != nil {
			return collected, fmt.Errorf("match history xuid(%s) page %d: %w", xuid, page, err)
		}
		if len(hist) == 0 {
			break
		}
		for _, h := range hist {
			raw, err := a.src.GetMatchStats(ctx, h.MatchID)
			if err != nil {
				return collected, fmt.Errorf("match stats %s: %w", h.MatchID, err)
			}
			st, ok := analysis.ExtractPlayerMatchStat(raw, xuid)
			if !ok {
				continue
			}
			if len(a.cfg.TargetSeasons) > 0 && !a.cfg.TargetSeasons[st.SeasonID] {
				nonTarget++
				continue
			}
			nonTarget = 0
			collected = append(collected, st)
		}
		if a.cfg.StopAfterNonTarget > 0 && len(a.cfg.TargetSeasons) > 0 && nonTarget >= a.cfg.StopAfterNonTarget {
			break
		}
		if len(hist) < worldMatchPageSize {
			break
		}
	}
	return collected, nil
}

// Run agrège tous les gamertags en parallèle (borné par Concurrency) et retourne
// les stats fusionnées. Un joueur en échec n'interrompt pas le batch : son erreur
// est collectée dans `errs` (best-effort par joueur).
func (a *WorldStatsAggregator) Run(ctx context.Context, gamertags []string) ([]domain.WorldPlayerSeasonStats, []error) {
	type playerResult struct {
		stats []domain.WorldPlayerSeasonStats
		err   error
		gt    string
	}
	results := make([]playerResult, len(gamertags))

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(a.cfg.Concurrency)
	for i, gt := range gamertags {
		i, gt := i, gt
		g.Go(func() error {
			s, err := a.AggregatePlayer(gctx, gt)
			results[i] = playerResult{stats: s, err: err, gt: gt}
			return nil // best-effort : on ne propage pas l'échec d'un joueur
		})
	}
	_ = g.Wait()

	var all []domain.WorldPlayerSeasonStats
	var errs []error
	for _, r := range results {
		if r.err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", r.gt, r.err))
			continue
		}
		all = append(all, r.stats...)
	}
	return all, errs
}
