package scheduler

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
)

// DefaultCatalogRefreshInterval : le catalogue (noms localisés des playlists, couples
// map-mode, maps et modes) change rarement → un rafraîchissement HEBDOMADAIRE suffit.
// La fiabilité vient de ce ticker, PAS d'un redémarrage : même si le serveur n'est
// jamais relancé (cas d'une web app), le cron retourne toutes les semaines.
const DefaultCatalogRefreshInterval = 7 * 24 * time.Hour

// CatalogDrainRunner exécute le drain catalogue (recensement match_registry →
// DiscoveryUGC → upsert catalogue + asset_translations) pour un titre.
// Satisfait par (*api.ServiceRegistry).RunCatalogUGCDrain — on passe par une func
// pour éviter au package scheduler de dépendre de internal/api.
type CatalogDrainRunner func(ctx context.Context, titleSlug string) (domain.CatalogUGCDrainResult, error)

// CatalogRefreshCron rafraîchit périodiquement le catalogue d'assets et leurs noms
// localisés via le drain testé (même chemin que l'action admin catalog/ugc-drain).
// Les assets que le drain ne parvient pas à normaliser (nom resté UUID brut, FR
// manquant) remontent automatiquement dans la page admin data-quality (sections
// RawAssets / UntranslatedModes), où ils sont corrigeables à la main — la boucle se
// ferme sans UI dédiée.
type CatalogRefreshCron struct {
	run       CatalogDrainRunner
	titleSlug string
	interval  time.Duration
}

// NewCatalogRefreshCron construit le cron. interval <= 0 → hebdomadaire ;
// titleSlug == "" → titre par défaut.
func NewCatalogRefreshCron(run CatalogDrainRunner, titleSlug string, interval time.Duration) *CatalogRefreshCron {
	if interval <= 0 {
		interval = DefaultCatalogRefreshInterval
	}
	if titleSlug == "" {
		titleSlug = titlePkg.DefaultSlug
	}
	return &CatalogRefreshCron{run: run, titleSlug: titleSlug, interval: interval}
}

// Run lance le cron : 1er tick immédiat (catalogue frais après un déploiement) puis
// toutes les `interval`. La régularité ne dépend PAS d'un redémarrage : le ticker
// tourne tant que le process vit. Bloque jusqu'à ctx.Done().
func (c *CatalogRefreshCron) Run(ctx context.Context) {
	if c == nil || c.run == nil {
		slog.WarnContext(ctx, "catalog_refresh_cron: noop (runner nil)")
		return
	}
	slog.InfoContext(ctx, "catalog_refresh_cron: started", "interval", c.interval, "title", c.titleSlug)

	c.RunOnce(ctx)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "catalog_refresh_cron: stopped (ctx done)")
			return
		case <-ticker.C:
			c.RunOnce(ctx)
		}
	}
}

// RunOnce exécute un cycle de rafraîchissement. Idempotent (le drain fait des UPSERT /
// INSERT OR IGNORE et porte son propre mutex anti-concurrence). Exporté pour un
// éventuel endpoint admin force-refresh et pour les tests.
func (c *CatalogRefreshCron) RunOnce(ctx context.Context) {
	if c == nil || c.run == nil {
		return
	}
	start := time.Now()
	res, err := c.run(ctx, c.titleSlug)
	if err != nil {
		slog.ErrorContext(ctx, "catalog_refresh_cron: drain échoué", "err", err,
			"duration", time.Since(start))
		return
	}
	slog.InfoContext(ctx, "catalog_refresh_cron: terminé",
		"seeded", res.Seeded, "playlists", res.Playlists, "pairs", res.Pairs,
		"maps", res.Maps, "game_variants", res.GameVariants, "errors", res.Errors,
		"duration", time.Since(start))
}
