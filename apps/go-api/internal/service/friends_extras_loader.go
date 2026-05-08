// Package service — friends_extras_loader.go : loader d'extras per-friend
// pour le panneau d'expander du scoreboard Match View.
//
// Contexte : le panneau Python `match_view_scoreboard_detail.py` charge à la
// volée la player DB de l'ami (via player_db_exists + duckdb_read_only) pour
// afficher performance_score + match_skill_rank + had_bot_teammate dans la
// section "Local". Le port Go reproduit ce comportement en s'appuyant sur le
// pool DuckDB (cfg.ResolvePlayer met en cache les PlayerDB par slug).
//
// Best-effort : un xuid sans player DB configurée → simplement absent du
// résultat. Erreurs per-friend loggées en Debug et silenciées (le scoreboard
// reste utilisable même si la DB d'un ami est verrouillée / corrompue).
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// FriendDBOpener ouvre une PlayerDB pour un (titleSlug, gamertag) donné. Pattern
// idempotent : opens le pool si pas déjà cachée, retourne la même instance pour
// les appels suivants. Implémenté par registry.makeTitlePlayerResolver.
type FriendDBOpener func(ctx context.Context, titleSlug, gamertag string) (FriendMatchExtrasRepo, error)

// FriendMatchExtrasRepo : sous-ensemble du MatchViewRepository nécessaire pour
// charger les extras per-friend (perf score + skill rank + enrich + assists model).
// Permet de découpler le loader du repo concret côté tests.
type FriendMatchExtrasRepo interface {
	GetMatchEnrichment(ctx context.Context, matchID string) (*domain.MatchEnrichmentRaw, error)
	GetMatchSkillRank(ctx context.Context, matchID string) (*domain.SkillRankRaw, error)
	GetPlayerAssistsModel(ctx context.Context, gameVariantName string) (*domain.PlayerAssistsModel, error)
}

// FriendProfile : couple (xuid, gamertag, titleSlug) issu de cfg.LoadPlayers,
// indexé par xuid pour O(1) lookup. titleSlug est le slug du titre courant.
type FriendProfile struct {
	XUID      string
	Gamertag  string
	TitleSlug string
}

// NewFriendsExtrasResolver retourne un port.FriendsExtrasResolver qui ouvre
// la player DB de chaque ami configuré et charge l'enrichissement + skill rank
// pour le matchID donné.
//
// `friendsByXUID` : map xuid → FriendProfile pré-construit au boot par le
// registry (évite de re-parser db_profiles.json à chaque request).
//
// `opener` : ouvre la player DB d'un ami via le pool DuckDB (cached).
func NewFriendsExtrasResolver(
	friendsByXUID map[string]FriendProfile,
	opener FriendDBOpener,
) port.FriendsExtrasResolver {
	return func(ctx context.Context, matchID string, gameVariantName string, xuids []string) map[string]port.FriendMatchExtras {
		out := make(map[string]port.FriendMatchExtras, len(xuids))
		for _, xuid := range xuids {
			profile, ok := friendsByXUID[xuid]
			if !ok {
				continue
			}
			repo, err := opener(ctx, profile.TitleSlug, profile.Gamertag)
			if err != nil {
				slog.DebugContext(ctx, "friends_extras: open player db failed",
					"xuid", xuid, "gamertag", profile.Gamertag, "err", err)
				continue
			}
			extras := loadOneFriendExtras(ctx, repo, matchID, gameVariantName, xuid)
			if extras != nil {
				out[xuid] = *extras
			}
		}
		return out
	}
}

// loadOneFriendExtras charge enrichissement + skill rank pour un ami. Retourne
// nil si AUCUNE des deux sources n'a remonté de données (rien à afficher).
//
// Stratégie de logging : toute erreur de lecture remontée par le repo est
// loggée en Warn (pas Error — la dégradation est visuelle et non bloquante)
// avec match_id + xuid. Une absence de données (err nil + result nil/empty)
// est loggée en Debug pour distinguer "DB inaccessible" de "match non joué
// par cet ami".
func loadOneFriendExtras(
	ctx context.Context,
	repo FriendMatchExtrasRepo,
	matchID, gameVariantName, xuid string,
) *port.FriendMatchExtras {
	var (
		extras  port.FriendMatchExtras
		hasData bool
	)
	if enrich, err := repo.GetMatchEnrichment(ctx, matchID); err != nil {
		slog.WarnContext(ctx, "friends_extras: load enrichment failed",
			"match_id", matchID, "xuid", xuid, "err", err)
	} else if enrich != nil && enrich.PerformanceScore != nil {
		v := *enrich.PerformanceScore
		extras.PerformanceScore = &v
		hasData = true
	}
	if rank, err := repo.GetMatchSkillRank(ctx, matchID); err != nil {
		slog.WarnContext(ctx, "friends_extras: load skill rank failed",
			"match_id", matchID, "xuid", xuid, "err", err)
	} else if rank != nil {
		extras.SkillRank = &domain.MatchScoreboardSkillRank{
			RatingType:  rank.RatingType,
			TierLabel:   rank.TierLabel,
			RatingValue: rank.RatingValue,
			RatingDelta: rank.RatingDelta,
		}
		hasData = true
	}
	if gameVariantName != "" {
		if m, err := repo.GetPlayerAssistsModel(ctx, gameVariantName); err == nil && m != nil {
			extras.AssistsModel = m
			hasData = true
		}
	}
	if !hasData {
		slog.DebugContext(ctx, "friends_extras: no data for friend on match",
			"match_id", matchID, "xuid", xuid)
		return nil
	}
	return &extras
}
