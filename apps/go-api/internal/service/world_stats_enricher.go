// Package service — world_stats_enricher.go : adaptateur reliant l'agrégateur
// multi-tokens au cron du classement mondial (Phase C, wiring runtime).
//
// L'agrégateur est paramétré par saison (TargetSeasons) ; l'enricher mémorise les
// dépendances stables (client multi-tokens + résolveur xuid + config de base) et
// construit un agrégateur ciblé sur LA saison active à chaque cycle de cron.
package service

import (
	"context"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

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

// EnrichSeason agrège les stats des `gamertags` pour la saison `season` (id brut
// ou chemin — normalisé en interne). Best-effort par joueur (erreurs collectées).
func (e *WorldStatsEnricher) EnrichSeason(ctx context.Context, season string, gamertags []string) ([]domain.WorldPlayerSeasonStats, []error) {
	if e == nil || len(gamertags) == 0 {
		return nil, nil
	}
	cfg := e.baseCfg
	cfg.TargetSeasons = map[string]bool{analysis.NormalizeSeasonID(season): true}
	agg := NewWorldStatsAggregator(e.src, e.resolver, cfg)
	return agg.Run(ctx, gamertags)
}
