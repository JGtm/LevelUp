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
type AssetNameSweepCron struct {
	run       AssetNameSweepRunner
	titleSlug string
	interval  time.Duration
	bootDelay time.Duration
	log       *slog.Logger // module=sync → logs/sync.log (cohérent avec la résolution in-sync)
}

// NewAssetNameSweepCron construit le cron. interval <= 0 → hebdomadaire ;
// titleSlug == "" → titre par défaut. Le logger est tagué module=sync.
func NewAssetNameSweepCron(run AssetNameSweepRunner, titleSlug string, interval time.Duration) *AssetNameSweepCron {
	if interval <= 0 {
		interval = DefaultAssetNameSweepInterval
	}
	if titleSlug == "" {
		titleSlug = titlePkg.DefaultSlug
	}
	return &AssetNameSweepCron{
		run:       run,
		titleSlug: titleSlug,
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
		"interval", c.interval, "boot_delay", c.bootDelay, "title", c.titleSlug)

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
func (c *AssetNameSweepCron) RunOnce(ctx context.Context) {
	if c == nil || c.run == nil {
		return
	}
	start := time.Now()
	res, err := c.run(ctx, c.titleSlug)
	if err != nil {
		c.log.WarnContext(ctx, "asset_name_sweep_cron: balayage échoué (best-effort)",
			"err", err, "duration", time.Since(start))
		return
	}
	if res.Requested > 0 {
		c.log.InfoContext(ctx, "asset_name_sweep_cron: balayage terminé",
			"requested", res.Requested, "resolved", res.Resolved, "skipped", res.Skipped,
			"capped", res.Capped, "errors", res.Errors, "duration", time.Since(start))
	}
}
