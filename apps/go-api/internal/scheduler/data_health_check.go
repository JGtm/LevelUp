// Package scheduler — data_health_check.go : audit santé DB périodique.
//
// DataHealthScheduler exécute périodiquement les invariants de
// `cmd/diag_db_health` directement en mémoire (pas via os/exec) et émet
// une notification admin (Category = `data_health_warning`) quand des
// anomalies sont détectées.
//
// Pattern : réutilise le ticker du AutoSyncScheduler. Lit
// `app_settings.json:data_health_check_enabled` (default true) et
// `data_health_check_interval_hours` (default 24h).
//
// Anomalies suivies :
//   - match_registry.map_name / pair_name UUID brut (régression sync)
//   - bits MENTEURS MBitEvents/MBitWeaponKills sans data
//   - xuids orphelins (sans alias en DB)
//   - banner garbage URLs (`/Waypoint/file/images/`) résiduelles
//
// Aucune action de repair automatique : émet juste la notif. L'admin
// déclenche manuellement `cmd/repair_data_consistency` si besoin.
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/duckdb"
)

// DataHealthCheckResult agrège les compteurs d'un cycle d'audit.
type DataHealthCheckResult struct {
	UUIDsRawCount        int
	LyingBitsEvents      int
	LyingBitsWeaponKills int
	OrphanXUIDs          int
	GarbageBannerURLs    int
	WarningsTotal        int
	Duration             time.Duration
}

// HealthScheduler orchestre l'audit santé DB périodique.
//
// Notifier peut être nil → la santé est calculée et loggée mais aucune notif
// n'est émise (mode test ou setup où le service notif n'est pas câblé).
type HealthScheduler struct {
	repoRoot      string
	notif         notifications.Emitter
	intervalHours int
	enabled       bool
}

// NewDataHealthScheduler crée un scheduler avec les défauts (24h, enabled).
func NewDataHealthScheduler(repoRoot string, notif notifications.Emitter) *HealthScheduler {
	return &HealthScheduler{
		repoRoot:      repoRoot,
		notif:         notif,
		intervalHours: 24,
		enabled:       true,
	}
}

// WithInterval permet de surcharger l'intervalle (utile pour tests).
func (s *HealthScheduler) WithInterval(hours int) *HealthScheduler {
	if hours > 0 {
		s.intervalHours = hours
	}
	return s
}

// SetEnabled permet de désactiver le scheduler (par config).
func (s *HealthScheduler) SetEnabled(enabled bool) *HealthScheduler {
	s.enabled = enabled
	return s
}

// SetEmitter permet d'injecter l'emitter notifications après l'initialisation
// du scheduler — utile car le notifications.Service est créé par
// ServiceRegistry après le boot du scheduler dans main.go (ordre
// d'initialisation dépendances). Thread-safe via le fait que les cycles
// runCycle() lisent s.notif sans lock (pattern lazy : nil = no-op).
func (s *HealthScheduler) SetEmitter(notif notifications.Emitter) {
	s.notif = notif
}

// Run lance la boucle périodique. Doit être appelé en goroutine.
func (s *HealthScheduler) Run(ctx context.Context) {
	if !s.enabled {
		slog.InfoContext(ctx, "data_health: scheduler désactivé par config")
		return
	}
	interval := time.Duration(s.intervalHours) * time.Hour
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.InfoContext(ctx, "data_health: scheduler démarré",
		"interval_hours", s.intervalHours)

	// Premier tick immédiat (sans attendre le 1er interval) pour catch les
	// régressions au boot.
	s.runCycle(ctx)

	for {
		select {
		case <-ticker.C:
			s.runCycle(ctx)
		case <-ctx.Done():
			slog.InfoContext(ctx, "data_health: arrêt scheduler")
			return
		}
	}
}

// RunOnce exécute un cycle d'audit unique (utilisable manuellement / tests).
func (s *HealthScheduler) RunOnce(ctx context.Context) *DataHealthCheckResult {
	return s.runCycle(ctx)
}

func (s *HealthScheduler) runCycle(ctx context.Context) *DataHealthCheckResult {
	start := time.Now()
	res := &DataHealthCheckResult{}

	titleDir := filepath.Join(s.repoRoot, "data", "titles", titlePkg.DefaultSlug)
	sharedPath := filepath.Join(titleDir, "warehouse", "shared_matches_v2.duckdb")

	if _, err := os.Stat(sharedPath); err != nil {
		slog.DebugContext(ctx, "data_health: shared DB absente — skip", "err", err)
		return res
	}

	db, err := openDBShared(sharedPath)
	if err != nil {
		slog.WarnContext(ctx, "data_health: ouverture shared DB échouée", "err", err)
		return res
	}
	defer db.Close() //nolint:errcheck // ref-count : best-effort

	// 1. UUIDs bruts résiduels (map_name + pair_name)
	const uuidPattern = `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
	var mapUUIDs, pairUUIDs int
	_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM match_registry WHERE map_name ~ ?`, uuidPattern).Scan(&mapUUIDs)
	_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM match_registry WHERE pair_name ~ ?`, uuidPattern).Scan(&pairUUIDs)
	res.UUIDsRawCount = mapUUIDs + pairUUIDs

	// 2. Bits menteurs
	const mbitEvents = 1 << 16
	const mbitWeaponKills = 1 << 21
	_ = db.QueryRow(ctx, fmt.Sprintf(`
		SELECT COUNT(*) FROM match_registry r
		WHERE (COALESCE(r.backfill_completed, 0) & %d) != 0
		  AND NOT EXISTS (SELECT 1 FROM highlight_events h WHERE h.match_id = r.match_id)
	`, mbitEvents)).Scan(&res.LyingBitsEvents)
	_ = db.QueryRow(ctx, fmt.Sprintf(`
		SELECT COUNT(*) FROM match_registry r
		WHERE (COALESCE(r.backfill_completed, 0) & %d) != 0
		  AND NOT EXISTS (SELECT 1 FROM weapon_kills w WHERE w.match_id = r.match_id)
	`, mbitWeaponKills)).Scan(&res.LyingBitsWeaponKills)

	// 3. xuids orphelins (alias absent shared)
	_ = db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT mp.xuid)
		FROM match_participants mp
		LEFT JOIN xuid_aliases xa ON xa.xuid = mp.xuid
		WHERE mp.xuid NOT LIKE 'bid(%'
		  AND (xa.xuid IS NULL OR xa.gamertag IS NULL OR xa.gamertag = '')
	`).Scan(&res.OrphanXUIDs)

	// 4. Banner garbage URLs (toutes les player DBs)
	playersDir := filepath.Join(titleDir, "players")
	if entries, err := os.ReadDir(playersDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			playerPath := filepath.Join(playersDir, e.Name(), "stats.duckdb")
			if _, err := os.Stat(playerPath); err != nil {
				continue
			}
			pdb, err := openDBShared(playerPath)
			if err != nil {
				continue
			}
			var n int
			_ = pdb.QueryRow(ctx, `
				SELECT COUNT(*) FROM career_progression
				WHERE banner_image_url LIKE '%/Waypoint/file/images/%'
				   OR emblem_image_url LIKE '%/Waypoint/file/images/%'
				   OR backdrop_image_url LIKE '%/Waypoint/file/images/%'
			`).Scan(&n)
			res.GarbageBannerURLs += n
			_ = pdb.Close() //nolint:errcheck // ref-count : best-effort
		}
	}

	res.WarningsTotal = res.UUIDsRawCount + res.LyingBitsEvents + res.LyingBitsWeaponKills + res.OrphanXUIDs + res.GarbageBannerURLs
	res.Duration = time.Since(start)

	slog.InfoContext(ctx, "data_health: cycle terminé",
		"warnings_total", res.WarningsTotal,
		"uuids_raw", res.UUIDsRawCount,
		"lying_bits_events", res.LyingBitsEvents,
		"lying_bits_weapons", res.LyingBitsWeaponKills,
		"orphan_xuids", res.OrphanXUIDs,
		"garbage_banner_urls", res.GarbageBannerURLs,
		"duration", res.Duration.Round(time.Millisecond),
	)

	if res.WarningsTotal > 0 && s.notif != nil {
		s.emitWarningNotification(ctx, res)
	}
	return res
}

// emitWarningNotification émet une notif unique agrégeant tous les counters.
// Catégorie : `data_health_warning`. Severity = warning.
func (s *HealthScheduler) emitWarningNotification(ctx context.Context, res *DataHealthCheckResult) {
	in := notifications.EmitInput{
		Category: notifications.CategoryDataHealthWarning,
		Severity: notifications.SeverityWarn,
		TitleKey: "notif.data_health_warning.title",
		BodyKey:  "notif.data_health_warning.body",
		Params: map[string]any{
			"warnings_total":      res.WarningsTotal,
			"uuids_raw":           res.UUIDsRawCount,
			"lying_bits_events":   res.LyingBitsEvents,
			"lying_bits_weapons":  res.LyingBitsWeaponKills,
			"orphan_xuids":        res.OrphanXUIDs,
			"garbage_banner_urls": res.GarbageBannerURLs,
			"hint":                "Relancer cmd/repair_data_consistency pour résoudre",
		},
		TargetRoute: "/admin/data-health",
		Source:      "data_health_scheduler",
	}
	if err := s.notif.Emit(ctx, in); err != nil {
		slog.WarnContext(ctx, "data_health: échec émission notif", "err", err)
	}
}

// openDBShared ouvre une DuckDB via le cache de connexions partagé du package
// duckdb (clé "rw:path"). Crucial pour éviter le conflit DuckDB
// "Can't open a connection to same database file with a different configuration"
// quand le serveur principal a déjà ouvert la même DB en read-write via le pool
// joueur ou les migrations. Un sql.Open direct créerait une 2e config DuckDB
// sur le même fichier, ce que le moteur refuse.
//
// La connexion retournée est ref-comptée : Close() décrémente le compteur
// sans fermer la DB si d'autres handles sont en cours d'utilisation.
func openDBShared(path string) (*duckdb.DB, error) {
	return duckdb.OpenReadWriteShared(path)
}
