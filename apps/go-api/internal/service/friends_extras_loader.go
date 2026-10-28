// Package service — friends_extras_loader.go : loader d'extras per-friend
// pour le panneau d'expander du scoreboard Match View.
//
// Contexte : le panneau Python `match_view_scoreboard_detail.py` charge à la
// volée la player DB de l'ami (via player_db_exists + duckdb_read_only) pour
// afficher performance_score + match_skill_rank + had_bot_teammate dans la
// section "Local". Le port Go reproduit ce comportement en s'appuyant sur le
// pool DuckDB (cfg.ResolvePlayer met en cache les PlayerDB par slug).
//
// Un xuid sans profil suivi (pas de player DB configurée) ou sans donnée sur ce match est une
// ABSENCE : simplement absent du résultat. Une player DB d'ami qui ne s'ouvre pas ou dont la
// lecture échoue est une PANNE : journalisée en ERREUR et REMONTÉE (le résultat reste servi pour
// les autres amis, le scoreboard reste utilisable).
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
)

// FriendDBOpener ouvre une PlayerDB pour un (titleSlug, gamertag) donné. Pattern
// idempotent : opens le pool si pas déjà cachée, retourne la même instance pour
// les appels suivants. Implémenté par registry.makeTitlePlayerResolver.
type FriendDBOpener func(ctx context.Context, titleSlug, gamertag string) (FriendMatchExtrasRepo, error)

// ErrFriendWithoutPlayerDB : le FriendDBOpener rend cette erreur pour un ami dont la player DB
// n'existe pas encore (profil suivi, jamais synchronisé). C'est une ABSENCE : le résolveur ne le
// compte ni comme panne ni comme erreur de section.
var ErrFriendWithoutPlayerDB = errors.New("friends_extras: aucune player DB pour cet ami")

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
//
// Contrat de retour : la map porte les extras des amis lus ; l'erreur (errors.Join) porte les
// pannes de lecture ou d'ouverture, une par ami en cause. Les deux sont valides ensemble.
func NewFriendsExtrasResolver(
	friendsByXUID map[string]FriendProfile,
	opener FriendDBOpener,
	assetURL games.TitleAssetURLAdapter,
) port.FriendsExtrasResolver {
	return func(ctx context.Context, matchID string, gameVariantName string, xuids []string) (map[string]port.FriendMatchExtras, error) {
		out := make(map[string]port.FriendMatchExtras, len(xuids))
		var failures []error
		for _, xuid := range xuids {
			profile, ok := friendsByXUID[xuid]
			if !ok {
				continue
			}
			repo, err := opener(ctx, profile.TitleSlug, profile.Gamertag)
			if err != nil {
				if errors.Is(err, ErrFriendWithoutPlayerDB) {
					slog.DebugContext(ctx, "friends_extras: ami sans player DB (absence, pas une panne)",
						"match_id", matchID, "xuid", xuid, "gamertag", profile.Gamertag)
					continue
				}
				slog.Log(ctx, observability.LevelUnlessCanceled(ctx, err, slog.LevelError), "friends_extras: open player db failed",
					"match_id", matchID, "xuid", xuid, "gamertag", profile.Gamertag, "err", err)
				failures = append(failures, fmt.Errorf("ami %s: ouverture de la player DB: %w", xuid, err))
				continue
			}
			extras, err := loadOneFriendExtras(ctx, repo, matchID, gameVariantName, xuid, assetURL)
			if err != nil {
				failures = append(failures, err)
			}
			if extras != nil {
				out[xuid] = *extras
			}
		}
		return out, errors.Join(failures...)
	}
}

// loadOneFriendExtras charge enrichissement + skill rank + modèle d'assists pour un ami. Retourne
// nil si AUCUNE source n'a remonté de données (rien à afficher).
//
// Stratégie d'erreur : une lecture en échec est journalisée en ERREUR (match_id + xuid) et
// REMONTÉE (errors.Join, une entrée par source) ; les autres sources sont tout de même lues et
// leurs données retournées avec l'erreur. Une absence de données (err nil + result nil/empty) est
// loggée en Debug pour distinguer "DB inaccessible" de "match non joué par cet ami".
func loadOneFriendExtras(
	ctx context.Context,
	repo FriendMatchExtrasRepo,
	matchID, gameVariantName, xuid string,
	assetURL games.TitleAssetURLAdapter,
) (*port.FriendMatchExtras, error) {
	var (
		extras   port.FriendMatchExtras
		hasData  bool
		failures []error
	)
	fail := func(source string, err error) {
		slog.Log(ctx, observability.LevelUnlessCanceled(ctx, err, slog.LevelError), "friends_extras: load "+source+" failed",
			"match_id", matchID, "xuid", xuid, "err", err)
		failures = append(failures, fmt.Errorf("ami %s: %s: %w", xuid, source, err))
	}
	if enrich, err := repo.GetMatchEnrichment(ctx, matchID); err != nil {
		fail("enrichment", err)
	} else if enrich != nil && enrich.PerformanceScore != nil {
		v := *enrich.PerformanceScore
		extras.PerformanceScore = &v
		hasData = true
	}
	if rank, err := repo.GetMatchSkillRank(ctx, matchID); err != nil {
		fail("skill rank", err)
	} else if rank != nil {
		labelSkillRank(ctx, rank)
		iconURL := resolveSkillIconURL(rank.Tier, rank.SubTier, assetURL)
		var iconURLPtr *string
		if iconURL != "" {
			iconURLPtr = &iconURL
		}
		extras.SkillRank = &domain.MatchScoreboardSkillRank{
			RatingType:  rank.RatingType,
			TierLabel:   rank.TierLabel,
			RatingValue: rank.RatingValue,
			RatingDelta: rank.RatingDelta,
			IconURL:     iconURLPtr,
		}
		hasData = true
	}
	if gameVariantName != "" {
		if m, err := repo.GetPlayerAssistsModel(ctx, gameVariantName); err != nil {
			fail("assists model", err)
		} else if m != nil {
			extras.AssistsModel = m
			hasData = true
		}
	}
	err := errors.Join(failures...)
	if !hasData {
		slog.DebugContext(ctx, "friends_extras: no data for friend on match",
			"match_id", matchID, "xuid", xuid)
		return nil, err
	}
	return &extras, err
}
