// Package service — world_stats_enricher.go : adaptateur reliant l'agrégateur
// multi-tokens au cron du classement mondial (Phase C, wiring runtime).
//
// L'agrégateur est paramétré par saison (TargetSeasons) ; l'enricher mémorise les
// dépendances stables (client multi-tokens + résolveur xuid + config de base) et
// construit un agrégateur ciblé sur LA saison active à chaque cycle de cron.
package service

import (
	"context"
	"expvar"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
)

// worldEnrichXUIDFromSnapshot compte les xuid réutilisés du snapshot Waypoint (donc
// NON re-résolus via PeopleHub, cf. B1). Exposé sur /debug/vars pour mesurer le
// court-circuit du goulot de résolution.
var worldEnrichXUIDFromSnapshot = expvar.NewInt("world_enrich.xuid_from_snapshot")

// RankedPlaylistSet retourne l'ensemble des asset IDs des playlists CLASSÉES
// actives. Sert à filtrer le backfill/enrichissement en ranked-only (l'historique
// matchmaking mêle classé et social ; le classement mondial est CSR/classé).
func RankedPlaylistSet() map[string]bool {
	pls := rankedplaylists.Active()
	out := make(map[string]bool, len(pls))
	for _, pl := range pls {
		out[pl.AssetID] = true
	}
	return out
}

// WorldStatsEnricher produit les stats agrégées des joueurs d'une saison donnée.
type WorldStatsEnricher struct {
	src      WorldMatchSource
	resolver WorldXUIDResolver
	baseCfg  WorldStatsAggregatorConfig
}

// NewWorldStatsEnricher construit l'enricher. `src` est typiquement un
// *syncpkg.PooledHaloClient (multi-tokens, PolicyAnyPublic). baseCfg fournit les
// bornes (Concurrency, MaxPages, StopAfterNonTarget) ; TargetSeasons est ignoré
// (fixé par EnrichSeason à la saison demandée).
func NewWorldStatsEnricher(src WorldMatchSource, resolver WorldXUIDResolver, baseCfg WorldStatsAggregatorConfig) *WorldStatsEnricher {
	return &WorldStatsEnricher{src: src, resolver: resolver, baseCfg: baseCfg}
}

// EnrichSeason agrège les stats des `players` pour la saison `season` (id brut ou
// chemin — normalisé en interne). Le xuid déjà connu (scrapé du snapshot Waypoint) est
// pré-seedé dans l'agrégateur : la résolution PeopleHub n'est faite QUE pour les
// joueurs sans xuid (lignes de snapshot antérieures à B1). Best-effort par joueur
// (erreurs collectées).
func (e *WorldStatsEnricher) EnrichSeason(ctx context.Context, season string, players []domain.WorldPlayerRef) ([]domain.WorldPlayerSeasonStats, []error) {
	if e == nil || len(players) == 0 {
		return nil, nil
	}
	cfg := e.baseCfg
	cfg.TargetSeasons = map[string]bool{analysis.NormalizeSeasonID(season): true}
	if len(cfg.RankedPlaylists) == 0 {
		cfg.RankedPlaylists = RankedPlaylistSet() // ranked-only (cron : toujours classé)
	}
	if cfg.XUIDResolveDelay <= 0 {
		// Throttle PeopleHub (~10 req/15s/compte) : l'enrichissement auto cape le débit
		// de résolution xuid, sinon une saison à 200+ joueurs SANS xuid pré-seedé
		// déclenche des 429 qui les skippent (cf. PrepareWorldPlayers).
		cfg.XUIDResolveDelay = 1600 * time.Millisecond
	}

	// Sépare gamertags (ordre de traitement) et xuid déjà connus (pré-seed → PeopleHub
	// court-circuité pour ces joueurs).
	gamertags := make([]string, 0, len(players))
	known := make(map[string]string, len(players))
	for _, p := range players {
		gamertags = append(gamertags, p.Gamertag)
		if p.XUID != "" {
			known[p.Gamertag] = p.XUID
		}
	}

	agg := NewWorldStatsAggregator(e.src, e.resolver, cfg)
	if n := agg.SeedKnownXUIDs(known); n > 0 {
		worldEnrichXUIDFromSnapshot.Add(int64(n))
		slog.DebugContext(ctx, "world_enrich: xuid pré-seedés depuis le snapshot (PeopleHub court-circuité)",
			"season", season, "seeded", n, "total", len(gamertags))
	}
	return agg.Run(ctx, gamertags)
}
