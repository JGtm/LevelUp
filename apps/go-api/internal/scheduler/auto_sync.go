// Package scheduler — auto_sync.go : synchronisation delta automatique et périodique.
//
// AutoSyncScheduler lit app_settings.json au démarrage et à chaque tick pour :
//   - vérifier que spnkr_auto_sync_enabled est vrai avant d'agir
//   - adapter l'intervalle si spnkr_auto_sync_interval_(minutes|hours) a changé
//
// Pour chaque joueur configuré dans db_profiles.json, le cycle :
//  1. Vérifie que le joueur est présent dans le pool de tokens (Pool.HasPlayer).
//  2. Crée un PooledHaloClient pinné sur ce joueur.
//  3. Lance SyncEngine.RunDelta avec ce client (fetches parallèles internes).
//
// L'auth est entièrement déléguée au Pool/Resolver, qui :
//   - tente MSAL silent refresh puis OAuth v2 refresh sur le RT découvert par Discovery
//   - cache les tokens Halo pour ~3h30 (Spartan token lifetime)
//   - persiste les RT rotatés par Microsoft via le callback OnTokenRotated injecté
//     à NewResolver
//
// Le scheduler n'accède plus directement à os.Getenv ni à sync_meta DuckDB pour
// les tokens : c'est entièrement encapsulé dans le couple Discovery+Resolver+Pool.
package scheduler

import (
	"context"
	"log/slog"
	"os"
	gosync "sync"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/pool"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/sync"
)

const defaultIntervalHours = 6

// PlayerActivityChecker est satisfait par watcher.StateProvider.
// Défini ici pour éviter une dépendance circulaire scheduler→watcher.
type PlayerActivityChecker interface {
	IsPlayerActive(gamertag string) bool
}

// DeltaRunner abstrait l'exécution d'une sync delta pour permettre l'injection
// d'un mock dans les tests sans avoir à construire un vrai SyncEngine + pool.
// En production, c'est *sync.SyncEngine (avec engine.SetCustomClient déjà câblé)
// qui implémente cette interface via sa méthode RunDelta.
type DeltaRunner interface {
	RunDelta(ctx context.Context, opts domain.SyncOptions) (domain.SyncResult, error)
}

// DeltaRunnerFactory construit un DeltaRunner pour un joueur donné.
// La factory par défaut (defaultRunnerFactory) crée un *sync.SyncEngine
// configuré avec un PooledHaloClient pinné sur (gamertag, xuid).
// Tests : injecter une factory mock pour contrôler RunDelta sans réseau ni DB.
type DeltaRunnerFactory func(ctx context.Context, gamertag, xuid string) DeltaRunner

// RunOnceResult agrège les compteurs d'un cycle de sync.
type RunOnceResult struct {
	Total    int           `json:"total"`    // joueurs dans db_profiles
	Synced   int           `json:"synced"`   // sync delta réussie
	Skipped  int           `json:"skipped"`  // joueur absent du pool, watcher actif, etc.
	Failed   int           `json:"failed"`   // erreur pendant RunDelta
	Duration time.Duration `json:"duration_ns"`
}

// PlayerOutcomeDetail capture le résultat détaillé d'une tentative de sync pour
// un joueur, pour exposition via l'endpoint admin /api/v1/_diag/auto-sync/snapshot.
type PlayerOutcomeDetail struct {
	Gamertag        string    `json:"gamertag"`
	XUID            string    `json:"xuid"`
	Outcome         string    `json:"outcome"` // "ok", "skipped", "failed"
	Reason          string    `json:"reason"`  // texte libre expliquant le résultat
	AttemptedAt     time.Time `json:"attempted_at"`
	DurationMs      int64     `json:"duration_ms"`
	MatchesInserted int       `json:"matches_inserted,omitempty"`
	MatchesSkipped  int       `json:"matches_skipped,omitempty"`
	MedalsInserted  int       `json:"medals_inserted,omitempty"`
	SyncStatus      string    `json:"sync_status,omitempty"`
	ErrorCount      int       `json:"error_count,omitempty"`
	FirstError      string    `json:"first_error,omitempty"`
}

// SchedulerSnapshot est exposé par l'endpoint admin pour diagnostic.
type SchedulerSnapshot struct {
	LastCycleAt     time.Time             `json:"last_cycle_at"`
	LastCycleResult *RunOnceResult        `json:"last_cycle_result,omitempty"`
	IntervalMinutes int                   `json:"interval_minutes"`
	PoolSize        int                   `json:"pool_size"`
	Players         []PlayerOutcomeDetail `json:"players"`
}

// AutoSyncScheduler orchestre la sync delta périodique de tous les joueurs.
//
// Tous les appels API Halo passent par le Pool : la rotation des refresh_tokens
// est gérée par le Resolver (via callback OnTokenRotated injecté à NewResolver),
// les 429/503 déclenchent un cooldown global, et les fetches sont parallélisés
// au sein de chaque RunDelta via le PooledHaloClient.
type AutoSyncScheduler struct {
	cfg      *config.AppConfig
	settings *settings_platform.Store

	// provider est utilisé par SyncEngine.runAchievementsSync (refresh XSTS Xbox).
	// Le sync Halo lui-même passe par le pool, pas par le provider direct.
	provider auth.TokenProvider

	// pool est la source unique de tokens Halo. Peut être nil au boot si
	// Discovery n'a trouvé aucun credential — dans ce cas tous les joueurs
	// seront skipped silencieusement.
	pool pool.Pool

	// ActivityChecker est optionnel. S'il est défini, le scheduler saute le tick
	// pour les joueurs dont le watcher est en état Watching/Syncing/Cooling.
	// Doit être défini avant d'appeler Run.
	ActivityChecker PlayerActivityChecker

	// RunnerFactory permet d'injecter une fabrique de DeltaRunner alternative
	// (pour les tests). Si nil, defaultRunnerFactory est utilisé : il crée un
	// *sync.SyncEngine configuré avec un PooledHaloClient pinné.
	RunnerFactory DeltaRunnerFactory

	// Snapshot par joueur du dernier cycle — pour l'endpoint admin diagnostic.
	snapshotMu      gosync.RWMutex
	lastCycleAt     time.Time
	lastCycleResult *RunOnceResult
	playerOutcomes  map[string]PlayerOutcomeDetail // keyed by gamertag
}

// New crée un AutoSyncScheduler. tokenPool peut être nil (cas où Discovery
// n'a trouvé aucun credential au boot) — dans ce cas le scheduler tournera
// quand même mais tous les joueurs seront skipped avec un reason explicite.
func New(
	cfg *config.AppConfig,
	settings *settings_platform.Store,
	provider auth.TokenProvider,
	tokenPool pool.Pool,
) *AutoSyncScheduler {
	s := &AutoSyncScheduler{
		cfg:            cfg,
		settings:       settings,
		provider:       provider,
		pool:           tokenPool,
		playerOutcomes: make(map[string]PlayerOutcomeDetail),
	}
	s.RunnerFactory = s.defaultRunnerFactory
	return s
}

// defaultRunnerFactory construit un *sync.SyncEngine configuré avec un
// PooledHaloClient pinné sur (gamertag, xuid) + WithFriendsLoader. Retourne
// nil si s.pool est nil ou si le gamertag n'est pas dans le pool — le caller
// (syncPlayer) check ces préconditions avant.
func (s *AutoSyncScheduler) defaultRunnerFactory(_ context.Context, gamertag, xuid string) DeltaRunner {
	engine := sync.NewSyncEngine(s.cfg.RepoRoot, gamertag, xuid, &domain.HaloTokens{}, s.provider)
	// Commit 8i : en mode B-swap (LEVELUP_USE_SHARED_PROVIDER=1), router les
	// ouvertures RW de shared via Provider.AcquireWriter au lieu d'OpenSharedDB
	// direct. Coordonne avec le pool joueur via Subscribe (DETACH/REATTACH).
	if s.cfg.SharedProvider != nil {
		engine.WithSharedProvider(s.cfg.SharedProvider)
	}
	if s.settings != nil {
		engine.WithFriendsLoader(func() ([]string, error) {
			cfg, lerr := s.settings.Load()
			if lerr != nil {
				return nil, lerr
			}
			return cfg.FriendGamertags, nil
		})
	}
	if s.pool != nil {
		pooledClient := sync.NewPooledHaloClient(s.pool, gamertag, xuid)
		engine.SetCustomClient(pooledClient)
	}
	return engine
}

// Snapshot retourne un cliché thread-safe du dernier cycle de sync, incluant
// le détail par joueur (raison du skip/failure, compteurs, erreurs).
// Utilisé par /api/v1/_diag/auto-sync/snapshot.
func (s *AutoSyncScheduler) Snapshot() SchedulerSnapshot {
	s.snapshotMu.RLock()
	defer s.snapshotMu.RUnlock()

	players := make([]PlayerOutcomeDetail, 0, len(s.playerOutcomes))
	for _, d := range s.playerOutcomes {
		players = append(players, d)
	}
	poolSize := 0
	if s.pool != nil {
		poolSize = s.pool.Size()
	}
	snap := SchedulerSnapshot{
		LastCycleAt:     s.lastCycleAt,
		IntervalMinutes: int(s.CurrentInterval() / time.Minute),
		PoolSize:        poolSize,
		Players:         players,
	}
	if s.lastCycleResult != nil {
		copyRes := *s.lastCycleResult
		snap.LastCycleResult = &copyRes
	}
	return snap
}

// recordOutcome enregistre le résultat détaillé pour un joueur (thread-safe).
func (s *AutoSyncScheduler) recordOutcome(d PlayerOutcomeDetail) {
	s.snapshotMu.Lock()
	defer s.snapshotMu.Unlock()
	if s.playerOutcomes == nil {
		s.playerOutcomes = make(map[string]PlayerOutcomeDetail)
	}
	s.playerOutcomes[d.Gamertag] = d
}

// poolSizeSafe retourne s.pool.Size() ou 0 si pool nil — pour les logs.
func (s *AutoSyncScheduler) poolSizeSafe() int {
	if s.pool != nil {
		return s.pool.Size()
	}
	return 0
}

// Run démarre la boucle périodique. Doit être lancé dans une goroutine.
// Se termine proprement à l'annulation de ctx.
func (s *AutoSyncScheduler) Run(ctx context.Context) {
	interval := s.CurrentInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.InfoContext(ctx, "auto_sync: scheduler démarré",
		"interval", interval,
		"interval_hours", int(interval.Hours()),
		"pool_size", s.poolSizeSafe(),
	)

	for {
		select {
		case <-ticker.C:
			cfg, err := s.settings.Load()
			if err != nil {
				slog.WarnContext(ctx, "auto_sync: lecture settings échouée — tick ignoré", "err", err)
				continue
			}
			if !cfg.SpnkrAutoSyncEnabled {
				slog.DebugContext(ctx, "auto_sync: désactivé dans les settings, tick ignoré")
				continue
			}
			if newInterval := resolveInterval(cfg.SpnkrAutoSyncIntervalMinutes, cfg.SpnkrAutoSyncIntervalHours); newInterval != interval {
				slog.InfoContext(ctx, "auto_sync: intervalle mis à jour", "old", interval, "new", newInterval)
				interval = newInterval
				ticker.Reset(interval)
			}
			res := s.RunOnce(ctx)
			slog.InfoContext(ctx, "auto_sync: cycle terminé",
				"total", res.Total,
				"synced", res.Synced,
				"skipped", res.Skipped,
				"failed", res.Failed,
				"duration", res.Duration.Round(time.Millisecond),
			)
			if res.Failed > 0 {
				slog.WarnContext(ctx, "auto_sync: des joueurs ont échoué — consulter le snapshot diag",
					"failed_count", res.Failed,
				)
			}
			if res.Total > 0 && res.Skipped == res.Total {
				slog.WarnContext(ctx, "auto_sync: aucun joueur synchronisé — vérifier le pool",
					"pool_size", s.poolSizeSafe(),
					"hint", "voir GET /api/v1/_diag/auto-sync/snapshot pour le détail",
				)
			}
		case <-ctx.Done():
			slog.InfoContext(ctx, "auto_sync: arrêt du scheduler (contexte annulé)")
			return
		}
	}
}

// RunOnce exécute un cycle de sync pour tous les joueurs configurés.
// Peut être appelé manuellement (debug, endpoint admin) sans attendre un tick.
func (s *AutoSyncScheduler) RunOnce(ctx context.Context) *RunOnceResult {
	start := time.Now()
	res := &RunOnceResult{}

	players, err := s.cfg.LoadPlayers()
	if err != nil {
		slog.ErrorContext(ctx, "auto_sync: chargement des joueurs échoué", "err", err)
		return res
	}
	res.Total = len(players)
	slog.InfoContext(ctx, "auto_sync: démarrage du cycle",
		"player_count", res.Total,
		"pool_size", s.poolSizeSafe(),
	)

	for _, p := range players {
		outcome := s.syncPlayer(ctx, p)
		switch outcome {
		case outcomeOK:
			res.Synced++
		case outcomeSkipped:
			res.Skipped++
		case outcomeFailed:
			res.Failed++
		}
	}

	res.Duration = time.Since(start)

	s.snapshotMu.Lock()
	s.lastCycleAt = time.Now()
	copyRes := *res
	s.lastCycleResult = &copyRes
	s.snapshotMu.Unlock()

	return res
}

// syncOutcome encode le résultat d'une tentative de sync par joueur.
type syncOutcome int

const (
	outcomeOK      syncOutcome = iota // sync delta réussie
	outcomeSkipped                    // pas de token / watcher actif / DB absente → ignoré sans erreur
	outcomeFailed                     // erreur bloquante pendant RunDelta
)

// syncPlayer lance une sync delta pour un joueur via le pool de tokens.
//
// Conditions de skip silencieux (outcome=skipped) :
//   - pool nil (aucun credential découvert au boot)
//   - joueur absent du pool (pas dans .env.local et pas dans sync_meta)
//   - watcher actif sur ce joueur (céder la priorité pour éviter conflit DB)
//   - DB joueur absente (sync initiale jamais effectuée)
//
// Enregistre toujours un PlayerOutcomeDetail (via defer) pour exposition via
// /api/v1/_diag/auto-sync/snapshot.
func (s *AutoSyncScheduler) syncPlayer(ctx context.Context, p domain.PlayerSummary) syncOutcome {
	slog.DebugContext(ctx, "auto_sync: traitement joueur", "gamertag", p.Gamertag, "xuid", p.XUID)

	startedAt := time.Now()
	detail := PlayerOutcomeDetail{
		Gamertag:    p.Gamertag,
		XUID:        p.XUID,
		AttemptedAt: startedAt,
	}
	var outcome syncOutcome
	defer func() {
		switch outcome {
		case outcomeOK:
			detail.Outcome = "ok"
		case outcomeSkipped:
			detail.Outcome = "skipped"
		case outcomeFailed:
			detail.Outcome = "failed"
		}
		detail.DurationMs = time.Since(startedAt).Milliseconds()
		s.recordOutcome(detail)
	}()

	// Précondition : pool initialisé.
	if s.pool == nil {
		slog.InfoContext(ctx, "auto_sync: pool nil, joueur ignoré", "gamertag", p.Gamertag)
		detail.Reason = "pool de tokens non initialisé (aucun credential découvert au boot)"
		outcome = outcomeSkipped
		return outcome
	}

	// Précondition : ce joueur a-t-il un token dans le pool ?
	if !s.pool.HasPlayer(p.Gamertag) {
		slog.InfoContext(ctx, "auto_sync: joueur absent du pool, ignoré",
			"gamertag", p.Gamertag,
			"hint", "définir SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG> dans .env.local ou faire une sync initiale",
		)
		detail.Reason = "joueur absent du pool (pas de token discoverable via Discovery — vérifier .env.local et sync_meta)"
		outcome = outcomeSkipped
		return outcome
	}

	// Précondition : watcher actif sur ce joueur ?
	if s.ActivityChecker != nil && s.ActivityChecker.IsPlayerActive(p.Gamertag) {
		slog.InfoContext(ctx, "auto_sync: watcher actif sur ce joueur — tick cédé",
			"gamertag", p.Gamertag,
		)
		detail.Reason = "watcher actif (Watching/Syncing/Cooling) — tick cédé"
		outcome = outcomeSkipped
		return outcome
	}

	// Précondition : DB joueur présente (sinon la 1re sync doit créer le schéma —
	// on préfère skip et laisser l'utilisateur faire une sync initiale explicite).
	dbPath := titlePkg.NewPathResolver(s.cfg.RepoRoot).PlayerDBPath(titlePkg.DefaultSlug, p.Gamertag)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		slog.InfoContext(ctx, "auto_sync: DB joueur absente, joueur ignoré",
			"gamertag", p.Gamertag, "db_path", dbPath,
			"hint", "lancer la sync initiale via POST /sync/initial pour créer la DB",
		)
		detail.Reason = "DB joueur absente — sync initiale jamais effectuée"
		outcome = outcomeSkipped
		return outcome
	}

	// ──────────────────────────────────────────────────────────────────────
	// Sync delta via DeltaRunner (PooledHaloClient en production, mockable en test).
	// ──────────────────────────────────────────────────────────────────────
	slog.InfoContext(ctx, "auto_sync: démarrage sync delta", "gamertag", p.Gamertag)

	factory := s.RunnerFactory
	if factory == nil {
		factory = s.defaultRunnerFactory
	}
	runner := factory(ctx, p.Gamertag, p.XUID)
	if runner == nil {
		slog.ErrorContext(ctx, "auto_sync: RunnerFactory a retourné nil",
			"gamertag", p.Gamertag)
		detail.Reason = "RunnerFactory a retourné nil (pool absent ?)"
		outcome = outcomeFailed
		return outcome
	}

	syncResult, err := runner.RunDelta(ctx, domain.DefaultSyncOptions())
	if err != nil {
		slog.ErrorContext(ctx, "auto_sync: RunDelta échoué",
			"gamertag", p.Gamertag, "err", err)
		detail.Reason = "RunDelta échoué: " + err.Error()
		detail.FirstError = err.Error()
		outcome = outcomeFailed
		return outcome
	}

	detail.MatchesInserted = syncResult.MatchesInserted
	detail.MatchesSkipped = syncResult.MatchesSkipped
	detail.MedalsInserted = syncResult.MedalsInserted
	detail.SyncStatus = syncResult.Status()
	detail.ErrorCount = len(syncResult.Errors)

	if len(syncResult.Errors) > 0 {
		detail.FirstError = syncResult.Errors[0]
		detail.Reason = "sync terminée avec erreurs partielles"
		slog.WarnContext(ctx, "auto_sync: sync terminée avec erreurs partielles",
			"gamertag", p.Gamertag,
			"inserted", syncResult.MatchesInserted,
			"skipped", syncResult.MatchesSkipped,
			"error_count", len(syncResult.Errors),
			"first_error", syncResult.Errors[0],
			"duration_s", syncResult.DurationSeconds,
			"status", syncResult.Status())
	} else {
		if syncResult.MatchesInserted > 0 {
			detail.Reason = "sync delta réussie"
		} else {
			detail.Reason = "sync delta réussie — 0 nouveau match (déjà à jour)"
		}
		slog.InfoContext(ctx, "auto_sync: sync delta réussie",
			"gamertag", p.Gamertag,
			"inserted", syncResult.MatchesInserted,
			"skipped", syncResult.MatchesSkipped,
			"medals_inserted", syncResult.MedalsInserted,
			"duration_s", syncResult.DurationSeconds,
			"status", syncResult.Status())
	}
	outcome = outcomeOK
	return outcome
}

// CurrentInterval retourne l'intervalle courant depuis les settings.
// Utilise _minutes en priorité si > 0, sinon fallback sur _hours.
// Retourne defaultIntervalHours en cas d'erreur de lecture.
func (s *AutoSyncScheduler) CurrentInterval() time.Duration {
	cfg, err := s.settings.Load()
	if err != nil {
		return intervalFromHours(0)
	}
	return resolveInterval(cfg.SpnkrAutoSyncIntervalMinutes, cfg.SpnkrAutoSyncIntervalHours)
}

// resolveInterval retourne la durée à utiliser. minutes > 0 prime sur hours.
func resolveInterval(minutes, hours int) time.Duration {
	if minutes > 0 {
		return time.Duration(minutes) * time.Minute
	}
	return intervalFromHours(hours)
}

// intervalFromHours convertit un nombre d'heures en Duration.
// Retourne defaultIntervalHours si h <= 0.
func intervalFromHours(h int) time.Duration {
	if h <= 0 {
		return defaultIntervalHours * time.Hour
	}
	return time.Duration(h) * time.Hour
}
