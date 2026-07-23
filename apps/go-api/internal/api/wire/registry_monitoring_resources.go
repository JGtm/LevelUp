// Package api — registry_monitoring_resources.go : runner du monitoring
// ressources machine & process (plan monitoring A5, DC-4).
//
// Sources : runtime Go (ops.CollectRuntimeStats), tailles fichiers DB + WAL via
// os.Stat sur les chemins PathResolver (jamais de filepath.Join à la main),
// disque libre via la façade platform/diskfree (build tags), snapshots expvar
// existants duckdb_budgets + pool stats (enfin surfacés dans l'UI), uptime +
// compteur de restarts persistant (marqueur server_boot dans cron_runs, A1).
// Tout est best-effort : une source indisponible dégrade sa section.
package wire

import (
	"context"
	"os"
	"time"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/diskfree"
	"levelup/go-api/internal/platform/duckdb"
)

// ResourcesReport agrège l'état ressources machine & process (A5.1).
func (r *ServiceRegistry) ResourcesReport(ctx context.Context) (domain.AdminResourcesResponse, error) {
	now := time.Now().UTC()
	resp := domain.AdminResourcesResponse{
		GeneratedAt: now.Format(time.RFC3339),
		Runtime:     ops.CollectRuntimeStats(),
		UptimeS:     int64(time.Since(r.startedAt).Seconds()),
		Databases:   []domain.ResourceDBFile{},
		Budgets:     duckdb.BudgetsSnapshot(),
		PoolStats:   map[string]interface{}{},
	}
	for k, v := range duckdb.PoolStatsSnapshot() {
		resp.PoolStats[k] = v
	}
	resp.Disk = r.resourceDisk()
	r.fillResourceDatabases(ctx, &resp)
	if r.monitoringStore != nil {
		if n, err := r.monitoringStore.CronRunCount(ctx, "server_boot"); err == nil {
			resp.Restarts = n
		} else {
			monitoringLog.WarnContext(ctx, "admin_resources: restarts illisibles", "err", err)
		}
	}
	return resp, nil
}

// resourceDisk mesure l'espace libre du volume data (seuils nommés A5.3).
func (r *ServiceRegistry) resourceDisk() domain.ResourceDisk {
	pr := titlePkg.NewPathResolver(r.cfg.RepoRoot)
	dataDir := pr.TitleDataDir(titlePkg.DefaultSlug) // volume data (data/titles/...)
	out := domain.ResourceDisk{Path: dataDir}
	free, total, err := diskfree.Free(dataDir)
	if err != nil {
		out.Status = domain.FreshnessStatusUnknown
		out.Error = err.Error()
		return out
	}
	out.FreeBytes = free
	out.TotalBytes = total
	out.Status = ops.EvaluateDiskStatus(free, total)
	return out
}

// fillResourceDatabases mesure les bases par titre actif + les bases globales.
// Sonde d'abord la racine data : introuvable/illisible (RepoRoot mal résolu,
// volume non monté) → statut "unavailable" + log ERROR, plutôt qu'une table de
// tailles nulles silencieuse (os.Stat avalé) qui a l'air d'un rendu cassé.
func (r *ServiceRegistry) fillResourceDatabases(ctx context.Context, resp *domain.AdminResourcesResponse) {
	pr := titlePkg.NewPathResolver(r.cfg.RepoRoot)
	dataRoot := pr.TitlesRootDir()
	if info, statErr := os.Stat(dataRoot); statErr != nil || !info.IsDir() {
		resp.DBInventoryStatus = domain.DBInventoryUnavailable
		monitoringLog.ErrorContext(ctx, "admin_resources: racine data introuvable — inventaire des bases indisponible",
			"data_root", dataRoot, "repo_root", r.cfg.RepoRoot, "err", statErr)
		return
	}
	resp.DBInventoryStatus = domain.DBInventoryOK
	for _, desc := range titlePkg.DefaultRegistry().NonArchived() {
		if desc.IsInternal || !desc.IsActive() {
			continue
		}
		slug := desc.Slug
		resp.Databases = append(resp.Databases,
			ops.DBFileSize(slug+"/shared_matches_v2", pr.SharedDBPath(slug)),
			ops.DBFileSize(slug+"/metadata", pr.MetadataDBPath(slug)),
			ops.DBFileSize(slug+"/shared_pve", pr.SharedPVEDBPath(slug)),
			ops.DBFileSize(slug+"/shared_social", pr.SharedSocialDBPath(slug)),
			domain.ResourceDBFile{
				Name:      slug + "/players (agrégé)",
				Path:      pr.PlayersRootDir(slug),
				SizeBytes: ops.DirTotalSize(pr.PlayersRootDir(slug)),
			},
		)
	}
	resp.Databases = append(resp.Databases,
		ops.DBFileSize("global/xbox_aliases", pr.GlobalXuidAliasesDBPath()),
		ops.DBFileSize("global/monitoring", pr.GlobalMonitoringDB()),
	)
	for _, db := range resp.Databases {
		resp.DBTotalBytes += db.SizeBytes + db.WalBytes
	}
}
