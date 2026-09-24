// Package duckdb — player_matches_cache.go : l'historique canonique ENRICHI d'un
// joueur derrière le cache des lectures joueur (plan perf 2026-09-23, lot L5b,
// décision D5b.4).
//
// CachedPlayerMatchesRepo est le port.PlayerMatchesRepository câblé par le registre
// (wire.playerMatchesAdapterFor). Il enveloppe PlayerMatchesAdapter, donc met en cache
// les lignes APRÈS l'enrichissement FR/EN des libellés : un hit ne relance ni les
// quatre requêtes du chargement ni les lectures metadata de l'enrichissement. Un
// enrichissement en échec (étape consignée par noteDegraded) ou une requête annulée
// en cours de chargement rend les lignes dégradées : jamais mises en cache.
//
// Clé : (xuid, titre, base) + une variante par jeu de filtres (filtersCacheKey,
// SHA-256 du JSON canonique des filtres, slices triés). TTL, coalescence,
// invalidation et copie à la lecture : player_read_cache.go.
package duckdb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"maps"
	"sort"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// playerMatchesReadCache : historique canonique enrichi, une variante par jeu de
// filtres.
var playerMatchesReadCache = newPlayerReadCache("player_matches_cache", clonePlayerMatchRows)

// playerMatchesSource est l'adapter que le cache enveloppe : *PlayerMatchesAdapter
// en production (lignes enrichies), un faux en test.
type playerMatchesSource interface {
	LoadPlayerMatches(ctx context.Context, slug, gamertag string, filters port.PlayerMatchFilters) ([]canonical.PlayerMatchRow, error)
	LobbySizesAtCompletion(ctx context.Context, slug string, matchIDs []string) (map[string]int, error)
}

// CachedPlayerMatchesRepo implémente port.PlayerMatchesRepository (plus la
// capacité optionnelle LobbySizesAtCompletion, lue par SessionPageService) derrière
// le cache process-wide des lectures joueur. Threadsafe.
type CachedPlayerMatchesRepo struct {
	inner playerMatchesSource
	cache *playerReadCache[[]canonical.PlayerMatchRow]
	scope playerCacheScope
}

// NewCachedPlayerMatchesRepo construit l'historique canonique enrichi du joueur
// derrière le cache process-wide.
func NewCachedPlayerMatchesRepo(pdb *PlayerDB) *CachedPlayerMatchesRepo {
	return &CachedPlayerMatchesRepo{
		inner: NewPlayerMatchesAdapter(NewPlayerMatchesRepo(pdb), pdb.TitleSlug, pdb.Gamertag),
		cache: playerMatchesReadCache,
		scope: scopeOf(pdb),
	}
}

// LoadPlayerMatches rend une copie des lignes cachées pour ces filtres, ou les
// charge (chargement + enrichissement) via l'adapter.
func (c *CachedPlayerMatchesRepo) LoadPlayerMatches(
	ctx context.Context,
	slug string,
	gamertag string,
	filters port.PlayerMatchFilters,
) ([]canonical.PlayerMatchRow, error) {
	return c.cache.load(ctx, c.scope, filtersCacheKey(filters), func(ctx context.Context) ([]canonical.PlayerMatchRow, error) {
		return c.inner.LoadPlayerMatches(ctx, slug, gamertag, filters)
	})
}

// LobbySizesAtCompletion délègue à l'adapter (lecture non cachée).
func (c *CachedPlayerMatchesRepo) LobbySizesAtCompletion(ctx context.Context, slug string, matchIDs []string) (map[string]int, error) {
	return c.inner.LobbySizesAtCompletion(ctx, slug, matchIDs)
}

// InvalidatePlayer retire du cache toutes les variantes du joueur de ce repo (les
// arguments sont ignorés, comme par l'adapter : le joueur est fixé au constructeur).
func (c *CachedPlayerMatchesRepo) InvalidatePlayer(_, _ string) {
	c.cache.invalidate(c.scope.xuid, c.scope.titleSlug)
}

// clonePlayerMatchRows copie l'historique rendu : nouvelle tranche (trier ou
// modifier les lignes reste sans effet sur le cache) et copie des références que
// des consommateurs modifient — les quatre AssetReference du résumé : Home et
// Synthèse ré-appliquent EnrichCanonicalAssetTranslations aux lignes reçues, qui
// écrit Labels[...], DefaultLabel et IconURL (une écriture de map partagée entre
// requêtes serait une erreur fatale « concurrent map writes »). Les autres
// références (pointeurs de scalaires, Teams, FriendsXUIDs, SkillSnapshot) restent
// partagées : aucun consommateur n'écrit à travers elles (grep du 2026-09-23 sur
// internal/service et internal/analysis). Inventaire figé par
// TestClonePlayerMatchRows_ReferenceInventory.
func clonePlayerMatchRows(rows []canonical.PlayerMatchRow) []canonical.PlayerMatchRow {
	if rows == nil {
		return nil
	}
	out := make([]canonical.PlayerMatchRow, len(rows))
	copy(out, rows)
	for i := range out {
		s := &out[i].Summary
		s.Playlist = cloneAssetRef(s.Playlist)
		s.Map = cloneAssetRef(s.Map)
		s.GameVariant = cloneAssetRef(s.GameVariant)
		s.PairMode = cloneAssetRef(s.PairMode)
	}
	return out
}

func cloneAssetRef(ref *canonical.AssetReference) *canonical.AssetReference {
	if ref == nil {
		return nil
	}
	c := *ref
	c.Labels = maps.Clone(ref.Labels)
	return &c
}

// cacheMetrics compte les hits/misses d'un cache décorateur. Reste consommé par
// highlight_events_cache.go (CachedHighlightEventsRepo).
type cacheMetrics struct {
	hits   int64
	misses int64
}

// filtersCacheKey produit une clé stable pour des filtres équivalents,
// indépendante de l'ordre des slices. SHA-256 hex sur JSON canonique.
func filtersCacheKey(f port.PlayerMatchFilters) string {
	canonicalFilters := struct {
		Period               *string  `json:"period,omitempty"`
		OutcomeIn            []string `json:"outcome_in,omitempty"`
		HadBotTeammate       *bool    `json:"had_bot_teammate,omitempty"`
		IsFirefight          *bool    `json:"is_firefight,omitempty"`
		IsRanked             *bool    `json:"is_ranked,omitempty"`
		MinTimePlayedSeconds *int     `json:"min_time_played_seconds,omitempty"`
		ExcludeFriendsXUIDs  []string `json:"exclude_friends_xuids,omitempty"`
		BTBExcluded          bool     `json:"btb_excluded,omitempty"`
		PlaylistKind         *string  `json:"playlist_kind,omitempty"`
		MapIDs               []string `json:"map_ids,omitempty"`
		Limit                int      `json:"limit,omitempty"`
		OrderBy              string   `json:"order_by,omitempty"`
	}{
		HadBotTeammate:       f.HadBotTeammate,
		IsFirefight:          f.IsFirefight,
		IsRanked:             f.IsRanked,
		MinTimePlayedSeconds: f.MinTimePlayedSeconds,
		BTBExcluded:          f.BTBExcluded,
		PlaylistKind:         f.PlaylistKind,
		Limit:                f.Limit,
		OrderBy:              f.OrderBy,
	}
	if f.Period != nil {
		s := string(*f.Period)
		canonicalFilters.Period = &s
	}
	for _, o := range f.OutcomeIn {
		canonicalFilters.OutcomeIn = append(canonicalFilters.OutcomeIn, string(o))
	}
	canonicalFilters.ExcludeFriendsXUIDs = append([]string{}, f.ExcludeFriendsXUIDs...)
	canonicalFilters.MapIDs = append([]string{}, f.MapIDs...)
	sort.Strings(canonicalFilters.OutcomeIn)
	sort.Strings(canonicalFilters.ExcludeFriendsXUIDs)
	sort.Strings(canonicalFilters.MapIDs)

	buf, _ := json.Marshal(canonicalFilters)
	sum := sha256.Sum256(buf)
	return hex.EncodeToString(sum[:])
}
