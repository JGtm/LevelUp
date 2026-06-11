// Package scheduler — data_health_check.go : audit santé DB périodique.
//
// HealthScheduler exécute périodiquement les invariants de
// `cmd/diag_db_health` directement en mémoire (pas via os/exec). Les
// compteurs sont écrits dans les logs structurés (`logs/scheduler.log`)
// pour le diagnostic manuel — **aucune notification utilisateur n'est
// émise**.
//
// Décision 2026-05-20 : la catégorie de notif `data_health_warning`
// véhiculait du jargon dev ("bits menteurs", "UUIDs bruts") sans valeur
// pour un end user lambda sur une app de stats. Le scheduler reste
// utile en interne (audit + log) mais ne pollue plus la cloche notif.
//
// Anomalies suivies :
//   - match_registry.map_name / pair_name UUID brut (régression sync)
//   - bits MENTEURS MBitEvents/MBitWeaponKills sans data
//   - xuids orphelins (informatif uniquement)
//   - banner garbage URLs (`/Waypoint/file/images/`) résiduelles
//
// Aucune action de repair automatique : l'admin déclenche manuellement
// `cmd/repair_data_consistency` ou `cmd/diag_db_health` si besoin
// d'investiguer un compteur qui bouge.
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	gosync "sync"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
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

// HealthScheduler orchestre l'audit santé DB périodique. N'émet pas de
// notification utilisateur — uniquement des logs structurés pour
// permettre le diag depuis `logs/scheduler.log`.
type HealthScheduler struct {
	repoRoot      string
	intervalHours int
	enabled       bool

	// Dernier audit COMPLET (dashboard monitoring admin). Les cycles avortés
	// (shared absente/illisible) ne l'écrasent pas : on garde le dernier
	// signal réel plutôt qu'un faux « tout vert ».
	lastMu     gosync.RWMutex
	lastResult *DataHealthCheckResult
	lastRunAt  time.Time
}

// NewDataHealthScheduler crée un scheduler avec les défauts (24h, enabled).
func NewDataHealthScheduler(repoRoot string) *HealthScheduler {
	return &HealthScheduler{
		repoRoot:      repoRoot,
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

	// Chemins via PathResolver (jamais de filepath.Join("data","titles",...) en
	// dur — règle multi-titres). Reste mono-titre (DefaultSlug) ; l'itération sur
	// registry.All() est différée (changerait la structure du résultat).
	pr := titlePkg.NewPathResolver(s.repoRoot)
	slug := titlePkg.DefaultSlug
	sharedPath := pr.SharedDBPath(slug)

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
	playersDir := filepath.Join(pr.TitleDataDir(slug), "players")
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

	res.WarningsTotal = res.UUIDsRawCount + res.LyingBitsEvents + res.LyingBitsWeaponKills + res.GarbageBannerURLs
	res.Duration = time.Since(start)

	// Log structuré uniquement — pas d'émission de notif (cf. décision
	// 2026-05-20 dans le commentaire de tête du package).
	// orphan_xuids est TOUJOURS loggé (même cycle propre) : c'est le signal
	// data-quality derrière les gamertags masqués "Joueur ####" (fix XUID
	// 2026-05-30). Reste informatif — hors WarningsTotal par décision.
	if res.WarningsTotal == 0 {
		slog.InfoContext(ctx, "data_health: cycle terminé",
			"warnings_total", 0,
			"orphan_xuids", res.OrphanXUIDs,
			"duration", res.Duration.Round(time.Millisecond),
		)
	} else {
		slog.InfoContext(ctx, "data_health: cycle terminé",
			"warnings_total", res.WarningsTotal,
			"uuids_raw", res.UUIDsRawCount,
			"lying_bits_events", res.LyingBitsEvents,
			"lying_bits_weapons", res.LyingBitsWeaponKills,
			"orphan_xuids", res.OrphanXUIDs,
			"garbage_banner_urls", res.GarbageBannerURLs,
			"duration", res.Duration.Round(time.Millisecond),
		)
	}

	s.storeLastResult(res)

	return res
}

// storeLastResult mémorise le dernier audit complet (thread-safe).
func (s *HealthScheduler) storeLastResult(res *DataHealthCheckResult) {
	s.lastMu.Lock()
	defer s.lastMu.Unlock()
	cp := *res
	s.lastResult = &cp
	s.lastRunAt = time.Now()
}

// LastResult retourne une copie du dernier audit complet et son horodatage.
// (nil, zero time) si aucun cycle complet depuis le boot — le dashboard
// affiche alors « jamais couru » et propose l'action data-health/run.
func (s *HealthScheduler) LastResult() (*DataHealthCheckResult, time.Time) {
	s.lastMu.RLock()
	defer s.lastMu.RUnlock()
	if s.lastResult == nil {
		return nil, time.Time{}
	}
	cp := *s.lastResult
	return &cp, s.lastRunAt
}

// openDBShared ouvre une DuckDB via le cache de connexions partagé du package
// duckdb (clé "ro:path"). Le data_health ne fait que des SELECT, donc RO suffit.
// Aligné sur OpenReadOnly comme main.go::sharedDB et openPlayerDB::sharedDB
// pour partager la même instance et éviter le conflit "different configuration".
// Voir commentaire dans cmd/server/main.go pour le trade-off RO vs RW.
//
// La connexion retournée est ref-comptée : Close() décrémente le compteur
// sans fermer la DB si d'autres handles sont en cours d'utilisation.
func openDBShared(path string) (*duckdb.DB, error) {
	return duckdb.OpenReadOnly(path)
}
