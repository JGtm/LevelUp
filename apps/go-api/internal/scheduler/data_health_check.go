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
// Aucune action de repair automatique : l'admin déclenche manuellement les
// actions correctives de la page Qualité des données (POST
// /admin/actions/lying-bits/reset pour les bits menteurs, POST
// /admin/actions/registry-names/backfill pour les noms du registre), ou
// `cmd/diag_db_health` pour investiguer un compteur qui bouge.
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	gosync "sync"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync/skill"
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
	// LUSRInteriorGaps : total de trous d'INTÉRIEUR LUSR détectés ce cycle (matchs
	// éligibles, non notés, sous le watermark → notes définitivement absentes). >0 =
	// replay requis. LUSRPendingRecent : matchs éligibles non notés AU-DESSUS du
	// watermark (simple attente, pas une anomalie). LUSRPlayersScanned : # de joueurs
	// effectivement scannés (diagnostic couverture).
	LUSRInteriorGaps   int
	LUSRPendingRecent  int
	LUSRPlayersScanned int
	// LUSRPlayersUnmeasured : # de joueurs (ou de titres entiers) dont les trous LUSR
	// n'ont PAS pu être mesurés ce cycle : player DB tenue RW par un sync donc
	// inouvrable, scan SQL en échec, ou os.ReadDir du répertoire joueurs KO. >0 = scan
	// LUSR PARTIEL → LUSRInteriorGaps est sous-compté et la jauge expvar NE DOIT PAS
	// être republiée (sinon le badge /admin/data s'éteindrait à tort — « unmeasured ≠
	// sain »). Distinct de ProbeErrors : une player DB tenue RW est transitoire/normale,
	// elle ne fait pas échouer le cron mais gèle la jauge le temps du sync.
	LUSRPlayersUnmeasured int
	// KillSummariesPending : matchs dont la passe courante du journal des morts n'a pas de résumés
	// (tête) à jour, tous titres mesurés confondus (ADR 0039). Signal distinct : hors WarningsTotal,
	// toujours loggé (WARN par titre quand non nul) et publié en expvar.
	KillSummariesPending int
	// Les gardes « index ART désynchronisé » ont été RETIRÉES avec les index qu'elles
	// surveillaient : personal_score_awards le 2026-09-20 (step
	// drop_psa_secondary_art_indexes_v1), match_skill_rank le 2026-09-27 (step
	// drop_msr_secondary_art_indexes_v1) — plus d'index, plus rien à surveiller.
	//
	// Passes : jauge « passes par match » de chaque table du registre de compaction présente,
	// par titre (data_health_passes.go). PassesUnmeasured : tables dont la mesure a échoué (> 0 =
	// rien n'est publié ce cycle). PassesOverThreshold : tables au-delà de
	// migration.SeuilPassesParMatch, renseigné à la publication.
	Passes              []PassesTitre
	PassesUnmeasured    int
	PassesOverThreshold int
	Duration            time.Duration
}

// HealthScheduler orchestre l'audit santé DB périodique. N'émet pas de
// notification utilisateur — uniquement des logs structurés pour
// permettre le diag depuis `logs/scheduler.log`.
type HealthScheduler struct {
	repoRoot      string
	intervalHours int
	enabled       bool

	// store : lectures des bases de l'audit (sondes, trous LUSR, jauge des passes) ;
	// now : horloge du fichier de jauge (data_health_passes.go).
	store port.DataHealthStore
	now   func() time.Time

	// compactAtStartup : l'instance compacte elle-même les passes au démarrage (capacité
	// CompactPassesAtStartup) ; sans elle, le script de nuit de l'hôte s'en charge. Ne change que le
	// texte de l'avertissement d'une table au-delà du seuil.
	compactAtStartup bool

	// Dernier audit COMPLET (dashboard monitoring admin). Les cycles avortés
	// (shared absente/illisible) ne l'écrasent pas : on garde le dernier
	// signal réel plutôt qu'un faux « tout vert ».
	lastMu     gosync.RWMutex
	lastResult *DataHealthCheckResult
	lastRunAt  time.Time
}

// Le volet « garde-fou trous LUSR » (scan par joueur)
// vit dans data_health_lusr.go — même package.

// NewDataHealthScheduler crée un scheduler avec les défauts (24h, enabled) ; store lit les bases de l'audit.
func NewDataHealthScheduler(repoRoot string, store port.DataHealthStore) *HealthScheduler {
	return &HealthScheduler{
		store:         store,
		repoRoot:      repoRoot,
		intervalHours: 24,
		enabled:       true,
		now:           time.Now,
	}
}

// WithInterval permet de surcharger l'intervalle (utile pour tests).
func (s *HealthScheduler) WithInterval(hours int) *HealthScheduler {
	if hours > 0 {
		s.intervalHours = hours
	}
	return s
}

// WithCompactionAtStartup dit que l'instance compacte les passes au démarrage (et non la nuit) : le
// texte de l'avertissement de la jauge nomme alors le prochain démarrage.
func (s *HealthScheduler) WithCompactionAtStartup(at bool) *HealthScheduler {
	s.compactAtStartup = at
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
	runHeavy(ctx, "data_health", s.cycle)

	for {
		select {
		case <-ticker.C:
			runHeavy(ctx, "data_health", s.cycle)
		case <-ctx.Done():
			slog.InfoContext(ctx, "data_health: arrêt scheduler")
			return
		}
	}
}

// cycle est runCycle sans son résultat : la forme qu'attend la porte du mode jeu (runHeavy).
func (s *HealthScheduler) cycle(ctx context.Context) { s.runCycle(ctx) }

// RunOnce exécute un cycle d'audit unique (utilisable manuellement / tests).
func (s *HealthScheduler) RunOnce(ctx context.Context) *DataHealthCheckResult {
	return s.runCycle(ctx)
}

func (s *HealthScheduler) runCycle(ctx context.Context) *DataHealthCheckResult {
	start := time.Now()
	res := &DataHealthCheckResult{}
	// Statut des crons (A6/DC-5, décision D1) : les WARNINGS data-health (UUIDs bruts,
	// bits menteurs, banner garbage) sont un RÉSULTAT, PAS un échec du cron. En
	// revanche des SONDES en échec (ProbeErrors > 0 : le cycle n'a pas pu tout mesurer)
	// = échec avec cause → rapporté à ReportCronRun. Défer sur `res` (renseigné par
	// auditTitle) pour couvrir les retours anticipés.
	defer func() {
		var cronErr error
		if res.ProbeErrors > 0 {
			cronErr = fmt.Errorf("%d sonde(s) data-health en échec", res.ProbeErrors)
		}
		observability.ReportCronRun("data_health", start, cronErr, time.Since(start).Milliseconds())
	}()

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
		// La jauge ne dépend pas de l'audit (elle prend ses propres lectures courtes, une fois le
		// handle de l'audit rendu) : un titre dont l'audit échoue est mesuré quand même, ou compté
		// non mesuré — jamais retiré du fichier en silence.
		s.auditTitlePasses(ctx, pr, td.Slug, res)
	}

	// Aucune shared DB lisible sur AUCUN titre ⇒ cycle avorté : on ne mémorise pas
	// (sinon « tout vert » trompeur écraserait le dernier signal réel, cf. doc de
	// lastResult). Identique au comportement mono-titre historique (early-return
	// quand la shared était absente), généralisé à N titres.
	if auditedTitles == 0 {
		slog.DebugContext(ctx, "data_health: aucune shared DB lisible — cycle avorté")
		return res
	}

	res.WarningsTotal = res.UUIDsRawCount + res.LyingBitsEvents + res.LyingBitsWeaponKills +
		res.GarbageBannerURLs
	res.Duration = time.Since(start)

	// Publie la jauge expvar des trous LUSR (dernier scan complet uniquement). Les
	// trous LUSR ne rentrent PAS dans WarningsTotal (signal distinct : panneau
	// monitoring + auto-heal), mais sont toujours loggés.
	publishLUSRGaugeIfComplete(ctx, res)
	s.publierJaugePasses(ctx, pr, res)

	// Log structuré uniquement — pas d'émission de notif (cf. décision
	// 2026-05-20 dans le commentaire de tête du package).
	// orphan_xuids est TOUJOURS loggé (même cycle propre) : c'est le signal
	// data-quality derrière les gamertags masqués "Joueur ####" (fix XUID
	// 2026-05-30). Reste informatif — hors WarningsTotal par décision.
	// Lot B (audit #10) : un cycle avec des sondes en échec (ProbeErrors > 0) est
	// loggué en WARN — il n'a pas pu tout mesurer et ne doit pas passer pour « sain ».
	// Idem si des joueurs LUSR n'ont pas pu être mesurés (scan partiel, jauge gelée) :
	// « unmeasured ≠ sain ».
	unmeasured := res.LUSRPlayersUnmeasured
	logHealth := slog.InfoContext
	if res.ProbeErrors > 0 || unmeasured > 0 {
		logHealth = slog.WarnContext
	}
	if res.WarningsTotal == 0 && res.ProbeErrors == 0 && unmeasured == 0 {
		slog.InfoContext(ctx, "data_health: cycle terminé",
			"warnings_total", 0,
			"probe_errors", 0,
			"orphan_xuids", res.OrphanXUIDs,
			"lusr_interior_gaps", res.LUSRInteriorGaps,
			"lusr_pending_recent", res.LUSRPendingRecent,
			"lusr_players_scanned", res.LUSRPlayersScanned,
			"kill_summaries_pending", res.KillSummariesPending,
			"lusr_players_unmeasured", 0,
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
			"kill_summaries_pending", res.KillSummariesPending,
			"lusr_interior_gaps", res.LUSRInteriorGaps,
			"lusr_pending_recent", res.LUSRPendingRecent,
			"lusr_players_scanned", res.LUSRPlayersScanned,
			"lusr_players_unmeasured", res.LUSRPlayersUnmeasured,
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

	db, err := s.store.OpenShared(sharedPath)
	if err != nil {
		slog.WarnContext(ctx, "data_health: ouverture shared DB échouée",
			"titleSlug", slug, "err", err)
		return false
	}
	defer db.Close() //nolint:errcheck // ref-count : best-effort

	// 1. UUIDs bruts résiduels (map_name + pair_name)
	res.UUIDsRawCount += s.countShared(ctx, db, slug, port.ProbeUUIDsMapName, &res.ProbeErrors)
	res.UUIDsRawCount += s.countShared(ctx, db, slug, port.ProbeUUIDsPairName, &res.ProbeErrors)

	// 2. Bits menteurs (la sonde des armes ne s'applique que si la base porte la table de preuve)
	res.LyingBitsEvents += s.countShared(ctx, db, slug, port.ProbeLyingBitsEvents, &res.ProbeErrors)
	res.LyingBitsWeaponKills += s.countShared(ctx, db, slug, port.ProbeLyingBitsWeapons, &res.ProbeErrors)

	// 3. xuids orphelins (alias absent shared)
	res.OrphanXUIDs += s.countShared(ctx, db, slug, port.ProbeOrphanXUIDs, &res.ProbeErrors)

	// 4. Banner garbage URLs (toutes les player DBs du titre)
	res.GarbageBannerURLs += s.auditPlayerBanners(ctx, pr, slug, &res.ProbeErrors)

	// 4bis. Résumés par match du journal des morts (jauge ADR 0039, read-only).
	s.auditKillSummaries(ctx, db, slug, res)

	// 5. Trous d'intérieur LUSR (garde-fou notes LUSR — read-only, best-effort ;
	// implémentation dans data_health_lusr.go).
	s.auditTitleLUSRGaps(ctx, pr, slug, db, res)

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
		n, err := s.store.CountPlayerProbe(ctx, port.ProbeGarbageBannerURLs, playerPath)
		switch {
		case errors.Is(err, port.ErrDataHealthUnavailable):
			continue // player DB inouvrable : ignorée (best-effort)
		case err != nil:
			s.probeFailed(ctx, slug, port.ProbeGarbageBannerURLs, err, probeErrors)
		}
		total += n
	}
	return total
}

// publishLUSRGaugeIfComplete republie la jauge expvar des trous d'intérieur —
// SEULEMENT si le scan LUSR a été COMPLET (aucun joueur non mesuré). Un scan partiel
// (player DB tenue RW par un sync, ReadDir KO) sous-compte LUSRInteriorGaps :
// republier écraserait le badge /admin/data avec un total trompeur (voire 0) et
// l'éteindrait à tort — « unmeasured ≠ sain ». On conserve alors la dernière valeur
// publiée (dernier scan complet ou heal manuel, cf. skill.AddLUSRInteriorGapsGauge)
// et on logge le motif.
func publishLUSRGaugeIfComplete(ctx context.Context, res *DataHealthCheckResult) {
	if res.LUSRPlayersUnmeasured == 0 {
		skill.SetLUSRInteriorGapsGauge(res.LUSRInteriorGaps)
		return
	}
	slog.WarnContext(ctx, "data_health: jauge LUSR non mise à jour (scan partiel)",
		"lusr_players_unmeasured", res.LUSRPlayersUnmeasured,
		"lusr_interior_gaps_scanned", res.LUSRInteriorGaps)
}

// countShared exécute une sonde de la base partagée. Sur erreur, la LOGGUE (Warn) et incrémente
// *probeErrors (Lot B, audit #10 : plus de sonde avalée en silence → un cycle qui n'a rien pu mesurer ne passe
// plus pour « sain »). Retourne 0, aussi pour une sonde qui ne s'applique pas à cette base.
func (s *HealthScheduler) countShared(ctx context.Context, db port.DataHealthShared, slug string, probe port.HealthProbe, probeErrors *int) int {
	n, applicable, err := db.CountProbe(ctx, probe)
	if err != nil {
		s.probeFailed(ctx, slug, probe, err, probeErrors)
		return 0
	}
	if !applicable {
		return 0
	}
	return n
}

// probeFailed journalise une sonde en échec et la compte dans *probeErrors.
func (s *HealthScheduler) probeFailed(ctx context.Context, slug string, probe port.HealthProbe, err error, probeErrors *int) {
	slog.WarnContext(ctx, "data_health: sonde échouée", "probe", string(probe), "titleSlug", slug, "err", err)
	*probeErrors++
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
