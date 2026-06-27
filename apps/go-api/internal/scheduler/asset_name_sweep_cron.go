package scheduler

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/assetnames"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability/logging"
)

// DefaultAssetNameSweepInterval : les assets restés en UUID brut (jamais résolus
// in-sync ET jamais rejoués → jamais re-tentés) sont rares et changent lentement →
// un balayage de rattrapage HEBDOMADAIRE suffit. La régularité vient du ticker, PAS
// d'un redémarrage : même si le serveur tourne des semaines, le filet repasse.
const DefaultAssetNameSweepInterval = 7 * 24 * time.Hour

// assetSweepBootDelay : délai avant le 1er balayage, le temps que le pool de tokens
// se réchauffe au boot (le fetch GameCMS exige un token Spartan, cf. asset_name_fetcher.go).
const assetSweepBootDelay = 60 * time.Second

// AssetNameSweepRunner résout vers asset_translations les noms d'assets restés en
// UUID dans match_registry pour un titre. ART-safe (ops.UpsertAssetTranslation,
// SELECT-then-write). Satisfait par une closure capturant
// (*api.ServiceRegistry).ResolveUnresolvedAssetNames + le pool de tokens — le package
// scheduler ne dépend ainsi ni de internal/api ni du pool.
type AssetNameSweepRunner func(ctx context.Context, titleSlug string) (assetnames.Result, error)

// AssetNameSweepCron : filet de rattrapage autonome pour la traîne d'assets non
// résolus. DÉCOUPLÉ du drain catalogue (CatalogRefreshCron) à dessein : le balayage
// de NOMS est ART-safe et n'a aucune raison d'être coupé quand le drain catalogue
// l'est (bug ART « Failed to delete all rows from index » sur les UPSERT catalogue).
// Gaté séparément par LEVELUP_SYNC_RESOLVE_ASSETS — le même interrupteur que la
// résolution in-sync (halo.AssetNameResolutionEnabled). La résorption des noms reste
// donc autonome même catalogue coupé.
//
// Title-aware : itère les titres ACTIFS du registre et ne sweep QUE ceux dont la
// résolution de NOMS passe par le fetcher discovery-infiniteugc, signalé par CapForge
// (même mécanisme UGC « HINF-shaped » que le drain catalogue). Un titre actif SANS
// cette cap (ex. Halo 5, dont les noms sont résolus metadata-side via l'API officielle
// — cf. cmd/h5-metadata-fetch, HORS de ce cron) est SKIPPÉ proprement (slog Debug,
// jamais d'erreur) : on ne lance JAMAIS le fetcher /hi/ hardcodé sur ses GUIDs.
type AssetNameSweepCron struct {
	run       AssetNameSweepRunner
	registry  *titlePkg.Registry // titres actifs à itérer (capability-gated)
	interval  time.Duration
	bootDelay time.Duration
	log       *slog.Logger // module=sync → logs/sync.log (cohérent avec la résolution in-sync)
}

// NewAssetNameSweepCron construit le cron. interval <= 0 → hebdomadaire ;
// titleSlug == "" → titre par défaut, conservé pour rétro-compat de la signature
// (le ciblage des titres se fait désormais via le registre + CapForge). Le logger est
// tagué module=sync.
func NewAssetNameSweepCron(run AssetNameSweepRunner, _ string, interval time.Duration) *AssetNameSweepCron {
	if interval <= 0 {
		interval = DefaultAssetNameSweepInterval
	}
	return &AssetNameSweepCron{
		run:       run,
		registry:  titlePkg.DefaultRegistry(),
		interval:  interval,
		bootDelay: assetSweepBootDelay,
		log:       slog.Default().With("module", logging.ModuleSync),
	}
}

// Run : 1er balayage après bootDelay (laisse le pool de tokens se réchauffer), puis
// toutes les `interval`. Bloque jusqu'à ctx.Done().
func (c *AssetNameSweepCron) Run(ctx context.Context) {
	if c == nil || c.run == nil {
		slog.WarnContext(ctx, "asset_name_sweep_cron: noop (runner nil)", "module", logging.ModuleSync)
		return
	}
	c.log.InfoContext(ctx, "asset_name_sweep_cron: started",
		"interval", c.interval, "boot_delay", c.bootDelay)

	select {
	case <-ctx.Done():
		return
	case <-time.After(c.bootDelay):
		c.RunOnce(ctx)
	}

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			c.log.InfoContext(ctx, "asset_name_sweep_cron: stopped (ctx done)")
			return
		case <-ticker.C:
			c.RunOnce(ctx)
		}
	}
}

// RunOnce exécute un balayage. Best-effort, idempotent (skip-fresh +
// ops.UpsertAssetTranslation ART-safe). Exporté pour les tests / un éventuel
// endpoint admin force-sweep.
//
// Title-aware : itère les titres ACTIFS et ne sweep QUE ceux déclarant CapForge
// (résolution de noms via discovery-infiniteugc). Un titre sans la cap est skippé
// proprement (slog Debug, pas d'erreur) — voir la doc du type.
func (c *AssetNameSweepCron) RunOnce(ctx context.Context) {
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
			// Dégradation gracieuse : titre actif mais dont les noms ne se résolvent
			// pas via discovery-infiniteugc (ex. Halo 5 → metadata-side, hors de ce
			// cron). On le saute, ce n'est PAS une erreur.
			c.log.DebugContext(ctx, "asset_name_sweep_cron: no discovery-infiniteugc resolver for title — skip",
				"titleSlug", desc.Slug)
			continue
		}
		c.runOnceForTitle(ctx, desc.Slug)
	}
}

// runOnceForTitle exécute le balayage de noms pour UN titre déclarant CapForge.
// Best-effort : une erreur n'interrompt pas l'itération sur les autres titres.
func (c *AssetNameSweepCron) runOnceForTitle(ctx context.Context, titleSlug string) {
	start := time.Now()
	res, err := c.run(ctx, titleSlug)
	if err != nil {
		c.log.WarnContext(ctx, "asset_name_sweep_cron: balayage échoué (best-effort)",
			"titleSlug", titleSlug, "err", err, "duration", time.Since(start))
		return
	}
	if res.Requested > 0 {
		c.log.InfoContext(ctx, "asset_name_sweep_cron: balayage terminé", "titleSlug", titleSlug,
			"requested", res.Requested, "resolved", res.Resolved, "skipped", res.Skipped,
			"capped", res.Capped, "errors", res.Errors, "duration", time.Since(start))
	}
}
