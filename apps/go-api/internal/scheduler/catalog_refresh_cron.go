package scheduler

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
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

// CatalogAdapterChecker répond à la VRAIE question du gate : « ce titre a-t-il un
// catalog adapter discovery-infiniteugc résolvable ? ». C'est le test sémantique
// précis qui remplace le proxy CapForge (Forge ≠ catalogue UGC) : il est satisfait
// par (*api.ServiceRegistry).HasCatalogAdapter, qui construit l'adapter par le MÊME
// chemin que le drain (rules TOML config/titles/<slug>/catalog/experience_rules.toml
// + halo_infinite.NewCatalogAdapter). Un titre dont l'adapter ne se résout pas
// (ex. Halo 5, catalogue metadata-side, pas de experience_rules.toml) renvoie false
// — équivalent fonctionnel de `_, err := resolver.Catalog(slug); err == nil`.
//
// Le package scheduler ne dépend ainsi NI de internal/api NI de internal/games :
// la closure est injectée au boot via WithCatalogAdapterCheck.
type CatalogAdapterChecker func(titleSlug string) bool

// CatalogRefreshCron rafraîchit périodiquement le catalogue d'assets et leurs noms
// localisés via le drain testé (même chemin que l'action admin catalog/ugc-drain).
// Les assets que le drain ne parvient pas à normaliser (nom resté UUID brut, FR
// manquant) remontent automatiquement dans la page admin data-quality (sections
// RawAssets / UntranslatedModes), où ils sont corrigeables à la main — la boucle se
// ferme sans UI dédiée.
//
// Title-aware : itère les titres ACTIFS du registre et ne lance le drain QUE pour
// ceux dont le catalogue se résout RÉELLEMENT via discovery-infiniteugc, testé par
// la présence d'un catalog adapter résolvable (hasCatalogAdapter, injecté au boot).
// C'est le test sémantique précis qui remplace le proxy CapForge (Forge ≠ catalogue
// UGC). Un titre actif SANS adapter (ex. Halo 5, dont le catalogue est peuplé
// metadata-side via l'API officielle www.haloapi.com — cf. cmd/h5-metadata-fetch,
// HORS de ce cron) est SKIPPÉ proprement (slog Debug, jamais d'erreur) : on ne lance
// JAMAIS le drain HINF (fetcher /hi/ hardcodé) sur des GUIDs d'un autre titre. Le
// runner reste Infinite-spécifique par construction (le drain construit
// halo_infinite.NewCatalogAdapter) ; la brique itère mais n'invente aucune source
// pour un titre sans adapter.
//
// Fallback rétro-compat : si AUCUN checker n'est injecté (hasCatalogAdapter nil —
// tests historiques, wiring partiel), le gate retombe sur le proxy CapForge. Le
// comportement prod est identique (HINF a l'adapter ET CapForge ; H5 n'a ni l'un ni
// l'autre), seul le signal diffère (présence d'adapter vs cap déclarée).
type CatalogRefreshCron struct {
	run               CatalogDrainRunner
	registry          *titlePkg.Registry    // titres actifs à itérer
	hasCatalogAdapter CatalogAdapterChecker // gate précis (nil → fallback CapForge)
	interval          time.Duration
	log               *slog.Logger // tagué module=catalog → logs/catalog.log
}

// WithCatalogAdapterCheck injecte le test « ce titre a-t-il un catalog adapter
// résolvable ? » qui remplace le proxy CapForge dans le gate. check == nil est
// nil-safe (retombe sur le fallback CapForge). Retourne le cron pour chaînage au boot.
func (c *CatalogRefreshCron) WithCatalogAdapterCheck(check CatalogAdapterChecker) *CatalogRefreshCron {
	if c != nil {
		c.hasCatalogAdapter = check
	}
	return c
}

// NewCatalogRefreshCron construit le cron. interval <= 0 → hebdomadaire ;
// titleSlug == "" → titre par défaut, conservé pour rétro-compat de la signature
// (le ciblage des titres se fait désormais via le registre + le test de présence
// d'un catalog adapter ; ce paramètre n'est plus utilisé pour restreindre le drain à
// un seul titre). Le gate précis s'injecte via WithCatalogAdapterCheck. Le logger est
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
// Title-aware : itère les titres ACTIFS et ne draine QUE ceux dont le catalog adapter
// discovery-infiniteugc est résolvable (test précis hasCatalogAdapter ; fallback
// CapForge si aucun checker injecté). Un titre sans adapter est skippé proprement
// (slog Debug, pas d'erreur) — voir la doc du type.
func (c *CatalogRefreshCron) RunOnce(ctx context.Context) {
	if c == nil || c.run == nil {
		return
	}
	// Statut unifie des crons (A6/DC-5) : liveness du cycle — les erreurs par
	// titre restent best-effort internes (loguees).
	start := time.Now()
	defer func() {
		observability.ReportCronRun("catalog_refresh", start, nil, time.Since(start).Milliseconds())
	}()
	reg := c.registry
	if reg == nil {
		reg = titlePkg.DefaultRegistry()
	}
	for _, desc := range reg.Active() {
		if desc == nil {
			continue
		}
		if !c.titleHasCatalogAdapter(desc) {
			// Dégradation gracieuse : titre actif mais sans catalog adapter UGC
			// discovery-infiniteugc résolvable (ex. Halo 5 → metadata-side, hors de
			// ce cron). On le saute, ce n'est PAS une erreur. Ne JAMAIS draîner /hi/
			// sur ses GUIDs.
			c.log.DebugContext(ctx, "catalog_refresh_cron: no catalog adapter for title — skip",
				"titleSlug", desc.Slug)
			continue
		}
		c.runOnceForTitle(ctx, desc.Slug)
	}
}

// titleHasCatalogAdapter applique le gate : test RÉEL de présence d'un catalog
// adapter résolvable si un checker est injecté, sinon fallback proxy CapForge
// (rétro-compat). Comportement prod identique entre les deux chemins.
func (c *CatalogRefreshCron) titleHasCatalogAdapter(desc *titlePkg.TitleDescriptor) bool {
	if c.hasCatalogAdapter != nil {
		return c.hasCatalogAdapter(desc.Slug)
	}
	return desc.HasCapability(titlePkg.CapForge)
}

// runOnceForTitle exécute le drain catalogue pour UN titre dont l'adapter est
// résolvable. Best-effort : une erreur n'interrompt pas l'itération sur les autres.
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
