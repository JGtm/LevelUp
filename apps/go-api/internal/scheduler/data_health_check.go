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
	gosync "sync"
	"time"

	"levelup/go-api/internal/analysis"
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
	// ProbeErrors (Lot B, audit #10) : nombre de sondes SQL qui ont échoué ce
	// cycle. > 0 → le cycle n'a pas tout mesuré et NE DOIT PAS passer pour « sain ».
	ProbeErrors int
	Duration    time.Duration
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
	// dur — règle multi-titres). On itère sur TOUS les titres enregistrés
	// (registry.All(), comme ops.RunHealthcheck) et on AGRÈGE les compteurs dans
	// le même résultat : un titre par-titre conserve des invariants identiques, on
	// boucle juste sur les slugs. Mono-titre (halo_infinite seul) ⇒ une seule
	// itération, sortie inchangée. Un titre dont la shared DB est absente est skippé
	// proprement (slog debug, pas de panic) — cas normal pour un titre live-only
	// pas encore backfillé.
	pr := titlePkg.NewPathResolver(s.repoRoot)
	auditedTitles := 0
	for _, td := range titlePkg.DefaultRegistry().All() {
		if s.auditTitle(ctx, pr, td.Slug, res) {
			auditedTitles++
		}
	}

	// Aucune shared DB lisible sur AUCUN titre ⇒ cycle avorté : on ne mémorise pas
	// (sinon « tout vert » trompeur écraserait le dernier signal réel, cf. doc de
	// lastResult). Identique au comportement mono-titre historique (early-return
	// quand la shared était absente), généralisé à N titres.
	if auditedTitles == 0 {
		slog.DebugContext(ctx, "data_health: aucune shared DB lisible — cycle avorté")
		return res
	}

	res.WarningsTotal = res.UUIDsRawCount + res.LyingBitsEvents + res.LyingBitsWeaponKills + res.GarbageBannerURLs
	res.Duration = time.Since(start)

	// Log structuré uniquement — pas d'émission de notif (cf. décision
	// 2026-05-20 dans le commentaire de tête du package).
	// orphan_xuids est TOUJOURS loggé (même cycle propre) : c'est le signal
	// data-quality derrière les gamertags masqués "Joueur ####" (fix XUID
	// 2026-05-30). Reste informatif — hors WarningsTotal par décision.
	// Lot B (audit #10) : un cycle avec des sondes en échec (ProbeErrors > 0) est
	// loggué en WARN — il n'a pas pu tout mesurer et ne doit pas passer pour « sain ».
	logHealth := slog.InfoContext
	if res.ProbeErrors > 0 {
		logHealth = slog.WarnContext
	}
	if res.WarningsTotal == 0 && res.ProbeErrors == 0 {
		slog.InfoContext(ctx, "data_health: cycle terminé",
			"warnings_total", 0,
			"probe_errors", 0,
			"orphan_xuids", res.OrphanXUIDs,
			"duration", res.Duration.Round(time.Millisecond),
		)
	} else {
		logHealth(ctx, "data_health: cycle terminé",
			"warnings_total", res.WarningsTotal,
			"probe_errors", res.ProbeErrors,
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

// auditTitle exécute les invariants data-health (UUIDs bruts, bits menteurs
// MBitEvents/MBitWeaponKills, xuids orphelins, banner garbage URLs) sur la shared
// DB ET les player DBs d'UN titre, et AGRÈGE les compteurs dans res. Les
// invariants sont identiques quel que soit le titre — on boucle juste sur les
// slugs (cf. runCycle). Retourne true si la shared DB du titre a pu être auditée
// (sert à runCycle pour distinguer un cycle réel d'un cycle avorté).
//
// Dégradation gracieuse : si la shared DB du titre est absente (titre live-only
// pas encore backfillé, ou jamais synchronisé), on skippe proprement (slog debug,
// pas de panic, retourne false) ; les requêtes individuelles ignorent déjà leurs
// erreurs (table manquante ⇒ compteur reste à 0), donc un schéma partiel ne casse
// pas le cycle.
func (s *HealthScheduler) auditTitle(ctx context.Context, pr *titlePkg.PathResolver, slug string, res *DataHealthCheckResult) bool {
	sharedPath := pr.SharedDBPath(slug)
	if _, err := os.Stat(sharedPath); err != nil {
		slog.DebugContext(ctx, "data_health: shared DB absente — skip titre",
			"titleSlug", slug, "err", err)
		return false
	}

	db, err := openDBShared(sharedPath)
	if err != nil {
		slog.WarnContext(ctx, "data_health: ouverture shared DB échouée",
			"titleSlug", slug, "err", err)
		return false
	}
	defer db.Close() //nolint:errcheck // ref-count : best-effort

	// 1. UUIDs bruts résiduels (map_name + pair_name)
	const uuidPattern = `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
	res.UUIDsRawCount += scanCount(ctx, db, slug, "uuids_map_name", `SELECT COUNT(*) FROM match_registry WHERE map_name ~ ?`, &res.ProbeErrors, uuidPattern)
	res.UUIDsRawCount += scanCount(ctx, db, slug, "uuids_pair_name", `SELECT COUNT(*) FROM match_registry WHERE pair_name ~ ?`, &res.ProbeErrors, uuidPattern)

	// 2. Bits menteurs
	const mbitEvents = 1 << 16
	const mbitWeaponKills = 1 << 21
	res.LyingBitsEvents += scanCount(ctx, db, slug, "lying_bits_events", fmt.Sprintf(`
		SELECT COUNT(*) FROM match_registry r
		WHERE (COALESCE(r.backfill_completed, 0) & %d) != 0
		  AND NOT EXISTS (SELECT 1 FROM highlight_events h WHERE h.match_id = r.match_id)
	`, mbitEvents), &res.ProbeErrors)
	res.LyingBitsWeaponKills += scanCount(ctx, db, slug, "lying_bits_weapons", fmt.Sprintf(`
		SELECT COUNT(*) FROM match_registry r
		WHERE (COALESCE(r.backfill_completed, 0) & %d) != 0
		  AND NOT EXISTS (SELECT 1 FROM weapon_kills w WHERE w.match_id = r.match_id)
	`, mbitWeaponKills), &res.ProbeErrors)

	// 3. xuids orphelins (alias absent shared)
	res.OrphanXUIDs += scanCount(ctx, db, slug, "orphan_xuids", `
		SELECT COUNT(DISTINCT mp.xuid)
		FROM match_participants mp
		LEFT JOIN xuid_aliases xa ON xa.xuid = mp.xuid
		WHERE `+analysis.SQLIsNotBotCol("mp.xuid")+`
		  AND (xa.xuid IS NULL OR xa.gamertag IS NULL OR xa.gamertag = '')
	`, &res.ProbeErrors)

	// 4. Banner garbage URLs (toutes les player DBs du titre)
	res.GarbageBannerURLs += s.auditPlayerBanners(ctx, pr, slug, &res.ProbeErrors)

	return true
}

// auditPlayerBanners parcourt les player DBs d'un titre et compte les URLs de
// bannière/emblème/backdrop « garbage » (chemins /Waypoint/file/images/ résiduels).
// Une player DB absente ou inouvrable est ignorée silencieusement (best-effort).
func (s *HealthScheduler) auditPlayerBanners(ctx context.Context, pr *titlePkg.PathResolver, slug string, probeErrors *int) int {
	playersDir := pr.PlayersRootDir(slug)
	entries, err := os.ReadDir(playersDir)
	if err != nil {
		return 0
	}
	total := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		playerPath := pr.PlayerDBPath(slug, e.Name())
		if _, err := os.Stat(playerPath); err != nil {
			continue
		}
		pdb, err := openDBShared(playerPath)
		if err != nil {
			continue
		}
		total += scanCount(ctx, pdb, slug, "garbage_banner_urls", `
			SELECT COUNT(*) FROM career_progression
			WHERE banner_image_url LIKE '%/Waypoint/file/images/%'
			   OR emblem_image_url LIKE '%/Waypoint/file/images/%'
			   OR backdrop_image_url LIKE '%/Waypoint/file/images/%'
		`, probeErrors)
		_ = pdb.Close() //nolint:errcheck // ref-count : best-effort
	}
	return total
}

// scanCount exécute une sonde COUNT(*) data-health. Sur erreur, la LOGGUE (Warn)
// et incrémente *probeErrors (Lot B, audit #10 : plus de sonde avalée en silence
// → un cycle qui n'a rien pu mesurer ne passe plus pour « sain »). Retourne 0.
func scanCount(ctx context.Context, db *duckdb.DB, slug, label, query string, probeErrors *int, args ...any) int {
	var n int
	if err := db.QueryRow(ctx, query, args...).Scan(&n); err != nil {
		slog.WarnContext(ctx, "data_health: sonde échouée", "probe", label, "titleSlug", slug, "err", err)
		*probeErrors++
	}
	return n
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
