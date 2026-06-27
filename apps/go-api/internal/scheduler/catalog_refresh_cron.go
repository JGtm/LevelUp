package scheduler

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability/logging"
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
//
// Title-aware : itère les titres ACTIFS du registre et ne lance le drain QUE pour
// ceux dont le catalogue se résout via le mécanisme discovery-infiniteugc, signalé
// par CapForge (UGC « HINF-shaped » : playlists / map-mode pairs / maps /
// ugcGameVariants). Un titre actif SANS cette cap (ex. Halo 5, dont le catalogue est
// peuplé metadata-side via l'API officielle www.haloapi.com — cf.
// cmd/h5-metadata-fetch, HORS de ce cron) est SKIPPÉ proprement (slog Debug, jamais
// d'erreur) : on ne lance JAMAIS le drain HINF (fetcher /hi/ hardcodé) sur des GUIDs
// d'un autre titre. Le runner reste Infinite-spécifique par construction (le drain
// construit halo_infinite.NewCatalogAdapter) ; la brique itère mais n'invente aucune
// source pour un titre sans la cap.
type CatalogRefreshCron struct {
	run      CatalogDrainRunner
	registry *titlePkg.Registry // titres actifs à itérer (capability-gated)
	interval time.Duration
	log      *slog.Logger // tagué module=catalog → logs/catalog.log
}

// NewCatalogRefreshCron construit le cron. interval <= 0 → hebdomadaire ;
// titleSlug == "" → titre par défaut, conservé pour rétro-compat de la signature
// (le ciblage des titres se fait désormais via le registre + CapForge ; ce paramètre
// n'est plus utilisé pour restreindre le drain à un seul titre). Le logger est
// capturé ici (après l'install du handler au boot) et tagué module=catalog → tous
// les logs du cron vont dans logs/catalog.log.
func NewCatalogRefreshCron(run CatalogDrainRunner, _ string, interval time.Duration) *CatalogRefreshCron {
	if interval <= 0 {
		interval = DefaultCatalogRefreshInterval
	}
	return &CatalogRefreshCron{
		run:      run,
		registry: titlePkg.DefaultRegistry(),
		interval: interval,
		log:      slog.Default().With("module", logging.ModuleCatalog),
	}
}

// Run lance le cron : 1er tick immédiat (catalogue frais après un déploiement) puis
// toutes les `interval`. La régularité ne dépend PAS d'un redémarrage : le ticker
// tourne tant que le process vit. Bloque jusqu'à ctx.Done().
func (c *CatalogRefreshCron) Run(ctx context.Context) {
	if c == nil || c.run == nil {
		slog.WarnContext(ctx, "catalog_refresh_cron: noop (runner nil)", "module", logging.ModuleCatalog)
		return
	}
	c.log.InfoContext(ctx, "catalog_refresh_cron: started", "interval", c.interval)

	c.RunOnce(ctx)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			c.log.InfoContext(ctx, "catalog_refresh_cron: stopped (ctx done)")
			return
		case <-ticker.C:
			c.RunOnce(ctx)
		}
	}
}

// RunOnce exécute un cycle de rafraîchissement. Idempotent (le drain fait des UPSERT /
// INSERT OR IGNORE et porte son propre mutex anti-concurrence). Exporté pour un
// éventuel endpoint admin force-refresh et pour les tests.
//
// Title-aware : itère les titres ACTIFS et ne draine QUE ceux déclarant CapForge
// (catalogue résolu via discovery-infiniteugc). Un titre sans la cap est skippé
// proprement (slog Debug, pas d'erreur) — voir la doc du type.
func (c *CatalogRefreshCron) RunOnce(ctx context.Context) {
	if c == nil || c.run == nil {
		return
	}
	reg := c.registry
	if reg == nil {
		reg = titlePkg.DefaultRegistry()
	}
	for _, desc := range reg.Active() {
		if desc == nil {
			continue
		}
		if !desc.HasCapability(titlePkg.CapForge) {
			// Dégradation gracieuse : titre actif mais sans catalogue UGC
			// discovery-infiniteugc (ex. Halo 5 → metadata-side, hors de ce cron).
			// On le saute, ce n'est PAS une erreur. Ne JAMAIS draîner /hi/ sur ses GUIDs.
			c.log.DebugContext(ctx, "catalog_refresh_cron: no catalog adapter for title — skip",
				"titleSlug", desc.Slug)
			continue
		}
		c.runOnceForTitle(ctx, desc.Slug)
	}
}

// runOnceForTitle exécute le drain catalogue pour UN titre déclarant CapForge.
// Best-effort : une erreur n'interrompt pas l'itération sur les autres titres.
func (c *CatalogRefreshCron) runOnceForTitle(ctx context.Context, titleSlug string) {
	start := time.Now()
	c.log.InfoContext(ctx, "catalog_refresh_cron: cycle démarré", "titleSlug", titleSlug)
	res, err := c.run(ctx, titleSlug)
	if err != nil {
		c.log.ErrorContext(ctx, "catalog_refresh_cron: cycle échoué", "titleSlug", titleSlug,
			"err", err, "duration", time.Since(start))
		return
	}
	c.log.InfoContext(ctx, "catalog_refresh_cron: cycle terminé", "titleSlug", titleSlug,
		"seeded", res.Seeded, "playlists", res.Playlists, "pairs", res.Pairs,
		"maps", res.Maps, "game_variants", res.GameVariants, "errors", res.Errors,
		"duration", time.Since(start))
}
