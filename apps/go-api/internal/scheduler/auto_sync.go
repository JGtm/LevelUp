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
	"strconv"
	"strings"
	gosync "sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_5/livesync"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/pool"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
	"levelup/go-api/internal/sync"
	syncv2 "levelup/go-api/internal/sync/v2"
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

// liveTitleRunnerResolver résout le runner d'un titre live-only (Halo 5+) pour le
// scheduler : pinne un token du pool sur le joueur, le pose dans le ctx (l'adapter
// du titre lit ctxkeys.HaloTokens), et instancie le runner registry-driven
// (livesync.RunnerForTitle). Le runCtx retourné porte l'auth ; release libère le
// lease pool (à DIFFÉRER par le caller — le SpartanToken doit rester valide le
// temps du RunDelta). En prod : s.acquireLiveTitleRunner. Tests : injecter pour
// mocker sans pool ni réseau (parité avec RunnerFactory pour le path engine).
type liveTitleRunnerResolver func(ctx context.Context, slug, gamertag, xuid string) (runner DeltaRunner, runCtx context.Context, release func(), err error)

// RunOnceResult agrège les compteurs d'un cycle de sync.
type RunOnceResult struct {
	Total    int           `json:"total"`   // joueurs dans db_profiles
	Synced   int           `json:"synced"`  // sync delta réussie
	Skipped  int           `json:"skipped"` // joueur absent du pool, watcher actif, etc.
	Failed   int           `json:"failed"`  // erreur pendant RunDelta
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
	// ConsecutiveZeroInserts compte les cycles successifs où la sync delta a
	// réussi sans insérer aucun match. Reset à 0 dès qu'un cycle insère ≥1
	// match. Non incrémenté sur outcome=skipped/failed (préserve la valeur
	// précédente). Ajouté suite à l'incident 2026-05-20 : 14 jours de sync à
	// inserted=0 sans alerte, cf. fix endpoint /matches xuid(NNN).
	ConsecutiveZeroInserts int `json:"consecutive_zero_inserts,omitempty"`
	// PostSync : compteurs du pipeline post-sync du dernier RunDelta réussi
	// (dashboard monitoring admin). Limite assumée : seuls les syncs passés
	// par syncPlayer (scheduler + cycle forcé) sont capturés — les paths
	// watcher et HTTP /sync/all ne remontent pas ici (couverts par sync.log).
	PostSync *domain.PostSyncResult `json:"post_sync,omitempty"`
	// PostSyncHistoryMs : durées post-sync (ms) des derniers cycles de CE joueur
	// (ancien → récent), pour la sparkline de tendance. Rempli par Snapshot
	// depuis le ring postSyncHistory.
	PostSyncHistoryMs []int64 `json:"post_sync_history_ms,omitempty"`
}

// ConsecutiveZeroInsertWarnThreshold est le seuil au-delà duquel le scheduler
// émet un slog.WarnContext "zero-insert prolongé" pour un joueur. 6 cycles à
// 15min = 1h30 de sync delta sans aucun nouveau match — pour un joueur actif
// c'est suspect (API stale, gamertag changé, format URL incorrect, etc.).
// Pour un joueur inactif c'est normal et restera juste informatif.
const ConsecutiveZeroInsertWarnThreshold = 6

// SchedulerSnapshot est exposé par l'endpoint admin pour diagnostic.
type SchedulerSnapshot struct {
	LastCycleAt     time.Time             `json:"last_cycle_at"`
	LastCycleResult *RunOnceResult        `json:"last_cycle_result,omitempty"`
	IntervalMinutes int                   `json:"interval_minutes"`
	PoolSize        int                   `json:"pool_size"`
	Players         []PlayerOutcomeDetail `json:"players"`
	// Gate : état du gate de déduplication cross-source (claims en vol + âge +
	// compteurs). Vide si watcher désactivé (NopSyncGate).
	Gate sync.GateSnapshotData `json:"gate"`
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

	// SyncGate déduplique les syncs cross-source (cf. sync.SyncGate). Avant chaque
	// RunDelta, syncPlayer demande un claim ; si le joueur est déjà en vol (watcher
	// ou HTTP), le tick est cédé (outcomeSkipped, re-tenté au prochain cycle). Par
	// défaut NopSyncGate (aucune dédup) ; main.go injecte le Coordinator partagé du
	// watcher quand celui-ci est actif. Complémentaire de l'ActivityChecker (qui
	// cède dès l'état Watching) : le gate couvre le résidu TOCTOU + le sync HTTP.
	SyncGate sync.SyncGate

	// RunnerFactory permet d'injecter une fabrique de DeltaRunner alternative
	// (pour les tests). Si nil, defaultRunnerFactory est utilisé : il crée un
	// *sync.SyncEngine configuré avec un PooledHaloClient pinné.
	RunnerFactory DeltaRunnerFactory

	// liveRunner résout le runner des titres live-only (Halo 5+), branché
	// registry-driven (livesync.HandlesTitle) — JAMAIS l'engine Infinite. Défaut :
	// s.acquireLiveTitleRunner (lease pool → ctx auth → RunnerForTitle). Tests :
	// injecter pour mocker. Cf. liveTitleRunnerResolver.
	liveRunner liveTitleRunnerResolver

	// postSyncRunner (Phase 4 plan stabilisation 2026-05-22) — runner injecté
	// dans chaque SyncEngine créé par defaultRunnerFactory. Câblé depuis
	// cmd/server/main.go via WithPostSyncRunner après création du
	// ServiceRegistry. Nil → feature off (legacy : auto-sync sautait TOUT le
	// post-sync delta + progression V2, cf. AUDIT_ASCENSION_PIPELINE_
	// DISCONNECTED_2026-05-21 cause B).
	postSyncRunner port.PostSyncRunner

	// batchQueue (Phase 4.7 closure 2026-05-24) — BatchQueue serveur-wide
	// optionnelle. Activée via LEVELUP_PERSIST_BATCH_ASYNC=1 (+ batch mode
	// déjà actif). Injectée par cmd/server/main.go via WithBatchQueue.
	// Sans queue : path synchrone (Persister.Persist direct). Avec queue :
	// path async (queue.Submit + worker, WAL durable, recovery au boot).
	batchQueue *persist.BatchQueue

	// customRefresher (DEPRECATED, supprimé par PLAN_SPARTAN_IDENTITY_REFACTOR
	// §11 Phase 5, 2026-05-25). Le champ est retiré ; la customisation est
	// désormais rafraîchie en LIVE via CareerLiveService.kickoffBackgroundRefresh
	// (UPSERT dans `spartan_identity`).

	// cycleOrchestrator (ADR 0027) — pipeline V2 cycle-level. Câblé via
	// WithCycleOrchestrator. Activation runtime gated par
	// LEVELUP_SYNC_PIPELINE=v2 (cf. shouldUseV2). En D0 le stub renvoie
	// ErrNotImplemented → fallback V1 silencieux ; le wiring du dispatch
	// proprement dit arrive en D6 du plan ADR 0027.
	cycleOrchestrator syncv2.CycleOrchestrator

	// Snapshot par joueur du dernier cycle — pour l'endpoint admin diagnostic.
	snapshotMu      gosync.RWMutex
	lastCycleAt     time.Time
	lastCycleResult *RunOnceResult
	playerOutcomes  map[string]PlayerOutcomeDetail // keyed by gamertag
	cycleHistory    []CycleRecord                  // ring FIFO borné (cf. auto_sync_history.go)
	// postSyncHistory : durées post-sync (ms) des derniers cycles PAR joueur
	// (ring borné), pour la sparkline de tendance — repère un joueur qui
	// converge toujours plus lentement. Alimenté par storeCycleResult.
	postSyncHistory map[string][]int64 // keyed by gamertag
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
		cfg:             cfg,
		settings:        settings,
		provider:        provider,
		pool:            tokenPool,
		playerOutcomes:  make(map[string]PlayerOutcomeDetail),
		postSyncHistory: make(map[string][]int64),
		// Défaut no-op : pas de dédup cross-source tant que main.go n'injecte pas
		// le Coordinator partagé du watcher. Évite un nil-check à chaque appel.
		SyncGate: sync.NopSyncGate{},
	}
	s.RunnerFactory = s.defaultRunnerFactory
	s.liveRunner = s.acquireLiveTitleRunner
	return s
}

// Gate retourne le SyncGate du scheduler (pour câblage par main.go vers le
// handler HTTP). Jamais nil : NopSyncGate par défaut.
func (s *AutoSyncScheduler) Gate() sync.SyncGate {
	return s.SyncGate
}

// BuildEngine construit un *sync.SyncEngine fully-configured pour un
// (gamertag, xuid) donné. C'est l'UNIQUE source of truth du wiring engine
// pour le serveur — utilisée à la fois par defaultRunnerFactory (path
// scheduler / auto_sync) et par cmd/server/main.go pour câbler le path
// watcher via syncTrigger.WithEngineFactory.
//
// Toute évolution du wiring (nouveau hook, nouveau provider, etc.) ne doit
// se faire QUE dans cette méthode — c'est ce qui garantit que watcher path
// et scheduler path restent en parité runtime. Cf. incident 2026-05-26 :
// trigger.go faisait un NewSyncEngine direct sans WithSharedProvider, le
// path watcher tombait en legacy → conflit "different configuration".
//
// Sémantique des nils :
//   - s.cfg.SharedProvider == nil → engine en mode legacy (OpenSharedDB direct)
//   - s.settings == nil → pas de FriendsLoader, pas de MediaScanHook
//   - s.pool == nil → pas de PooledHaloClient (le moteur tombera back sur le
//     client default si pas .SetCustomClient — non-recommandé en prod)
//   - s.postSyncRunner == nil → post-sync runner V1 désactivé
//   - s.batchQueue == nil → batch mode synchrone (sans WAL durable)
func (s *AutoSyncScheduler) BuildEngine(ctx context.Context, gamertag, xuid string) *sync.SyncEngine {
	// MT-11 / PMT-3 : le titre du joueur est porté par le ctx (posé par les 3
	// points d'entrée : scheduler syncPlayer, watcher/HTTP). BuildEngine reste
	// l'UNIQUE funnel de construction (utilisé comme valeur de fonction par
	// WithEngineFactory/WithEngineBuilder + appel direct) ; sa signature ne change
	// donc PAS — le slug transite par ctxkeys (carrier titre établi par PMT-10),
	// pas par un param qui rippler­ait EngineFactory/SyncRunner/Coordinator.
	// Fallback DefaultSlug si le ctx ne porte pas de titre → byte-identique halo.
	titleSlug := ctxkeys.TitleSlug(ctx)
	engine := sync.NewSyncEngineForTitle(s.cfg.RepoRoot, titleSlug, gamertag, xuid, &domain.HaloTokens{}, s.provider)
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
		pooledClient := sync.NewPooledHaloClient(s.pool, gamertag, xuid, 0) // 0 = defaultPooledRPS
		engine.SetCustomClient(pooledClient)
	}
	// Phase 4 plan stabilisation 2026-05-22 : injecter le runner post-sync
	// (notifications delta + pipeline progression V2). Sans ça, auto-sync
	// sautait TOUT le pipeline → page Ascension vide + notifications muettes.
	// gamertag = PlayerSlug (cf. config.go:284 — PlayerSlug = gamertag dans
	// db_profiles.json).
	if s.postSyncRunner != nil {
		engine.WithPostSyncRunner(s.postSyncRunner, gamertag)
	}
	// Media scan post-sync : indexe les captures présentes sur disque et les
	// associe aux matchs fraîchement synchronisés. Les 2 closures lisent les
	// settings live à chaque tick : MediaCapturesBaseDir + UserTimezone.
	// Sans timezone, parseCaptureTimeFromFilename retourne nil → 0 associations
	// (bug observé 2026-05-25, cf. thought_log).
	if s.settings != nil {
		engine.WithMediaScanHook(service.BuildMediaScanHook(s.cfg.RepoRoot, gamertag,
			func() string {
				cfg, _ := s.settings.Load()
				if cfg != nil {
					return cfg.MediaCapturesBaseDir
				}
				return ""
			},
			func() string {
				cfg, _ := s.settings.Load()
				if cfg != nil {
					return cfg.UserTimezone
				}
				return ""
			},
		))
	}
	// Le batch INSERT-only est le seul chemin d'écriture depuis D1b. Phase 4.9 :
	// le layer async reste optionnel. Set LEVELUP_PERSIST_BATCH_ASYNC=0 pour un
	// fallback synchrone direct Persister. La queue est injectée par main.go au
	// boot si != "0" (cf. autoBatchQueue init).
	if s.batchQueue != nil && os.Getenv("LEVELUP_PERSIST_BATCH_ASYNC") != "0" {
		engine.WithBatchQueue(s.batchQueue)
	}
	if s.cfg.CurrentCSRSeasonID != "" {
		engine.WithCSRSeasonID(s.cfg.CurrentCSRSeasonID)
	}
	// Résolution autonome des noms d'assets au sync (primary write) — chemin V1
	// engine.run (fallback quand V2 off + CLI). On passe le POOL UNIFIÉ (s.pool,
	// la même source que tous les syncs) ; GameCMS exige un token. Gate
	// LEVELUP_SYNC_RESOLVE_ASSETS appliqué côté sync. Le chemin prod V2 est câblé
	// séparément (persister).
	engine.WithAssetNameResolution(s.pool)
	return engine
}

// defaultRunnerFactory adapte BuildEngine vers l'interface DeltaRunner
// attendue par le scheduler (RunDelta-only). *sync.SyncEngine satisfait
// DeltaRunner via sa méthode RunDelta.
//
// Garde un nom historique pour ne pas casser l'API interne du scheduler
// (champ RunnerFactory, tests existants).
func (s *AutoSyncScheduler) defaultRunnerFactory(ctx context.Context, gamertag, xuid string) DeltaRunner {
	return s.BuildEngine(ctx, gamertag, xuid)
}

// acquireLiveTitleRunner est l'implémentation prod de liveTitleRunnerResolver pour
// les titres live-only (Halo 5+). Délègue à livesync.AcquireRunner (source unique
// pool→runner, partagée avec le path watcher) : le pool fournit le SpartanToken
// (là où le path HTTP le tient de la session), posé dans le ctx pour que l'adapter
// du titre le lise. Le lease est tenu jusqu'au release (différé par syncPlayer).
func (s *AutoSyncScheduler) acquireLiveTitleRunner(ctx context.Context, slug, gamertag, xuid string) (DeltaRunner, context.Context, func(), error) {
	// *Runner est nil-able : en cas d'erreur on retourne l'interface DeltaRunner
	// explicitement nil (évite le piège du typed-nil).
	r, runCtx, release, err := livesync.AcquireRunner(ctx, s.pool, s.cfg, slug, gamertag, xuid)
	if err != nil {
		return nil, ctx, release, err
	}
	return r, runCtx, release, nil
}

// resolveTitleSlug retourne le slug du profil joueur, avec fallback DefaultSlug
// (profils flat implicitement halo_infinite, ou TitleSlug vide). Source unique de
// la règle de fallback (MT-11 / PMT-3), partagée par syncPlayer + la précondition.
func resolveTitleSlug(p domain.PlayerSummary) string {
	if p.TitleSlug == "" {
		return titlePkg.DefaultSlug
	}
	return p.TitleSlug
}

// WithBatchQueue branche la BatchQueue serveur-wide (Phase 4.7 closure
// 2026-05-24). Injectée par cmd/server/main.go après NewBatchQueue.
// Le batch INSERT-only est le seul chemin d'écriture (D1b) ; la queue async est
// activée dans defaultRunnerFactory sous LEVELUP_PERSIST_BATCH_ASYNC != "0"
// (queue async + WAL durable).
//
// Nil queue → path synchrone direct Persister (déjà validé Phase 4.5).
func (s *AutoSyncScheduler) WithBatchQueue(q *persist.BatchQueue) *AutoSyncScheduler {
	s.batchQueue = q
	return s
}

// WithPostSyncRunner branche le runner post-sync (Phase 4 plan stabilisation
// 2026-05-22). Le runner sera injecté dans chaque SyncEngine créé par
// defaultRunnerFactory. À appeler depuis cmd/server/main.go après création
// du ServiceRegistry, AVANT autoSyncScheduler.Run.
//
// Nil runner → no-op (legacy : feature désactivée).
func (s *AutoSyncScheduler) WithPostSyncRunner(runner port.PostSyncRunner) *AutoSyncScheduler {
	s.postSyncRunner = runner
	return s
}

// WithCycleOrchestrator branche le pipeline V2 (ADR 0027). L'activation
// runtime reste gated par LEVELUP_SYNC_PIPELINE=v2 ; câbler un orchestrator
// non-nil sans l'env var ne change rien au comportement. Nil → V1 toujours.
//
// Doit être appelé depuis cmd/server/main.go au boot, avant Run.
func (s *AutoSyncScheduler) WithCycleOrchestrator(o syncv2.CycleOrchestrator) *AutoSyncScheduler {
	s.cycleOrchestrator = o
	return s
}

// shouldUseV2 retourne true si le pipeline V2 (ADR 0027) doit être tenté pour le
// prochain cycle. V2 est désormais le pipeline PAR DÉFAUT : il isole la fenêtre
// d'écriture (Discovery/Fetch en RO, Persist writer court) et supprime la
// contention qui gelait les lectures user-facing pendant un sync V1 (le writer RW
// shared était tenu pendant tout le fetch réseau — cf. .ai/PLAN_CONTENTION_SYNC_SERVICE.md).
//
// Conditions :
//   - cycleOrchestrator non-nil (câblé par main.go, parity-complete)
//   - LEVELUP_SYNC_PIPELINE != "v1" (échappatoire de rollback : forcer le legacy V1)
//
// Le dispatch conserve un fallback V1 automatique sur ErrNotImplemented ou échec
// global (runOnceV2 → V1), donc l'activation par défaut ne peut pas casser un cycle.
func (s *AutoSyncScheduler) shouldUseV2() bool {
	if s.cycleOrchestrator == nil {
		return false
	}
	// V2 par défaut ; seul un opt-out explicite "v1" repasse en legacy.
	v := strings.ToLower(strings.TrimSpace(os.Getenv("LEVELUP_SYNC_PIPELINE")))
	return v != "v1"
}

// Snapshot retourne un cliché thread-safe du dernier cycle de sync, incluant
// le détail par joueur (raison du skip/failure, compteurs, erreurs).
// Utilisé par /api/v1/_diag/auto-sync/snapshot.
func (s *AutoSyncScheduler) Snapshot() SchedulerSnapshot {
	s.snapshotMu.RLock()
	defer s.snapshotMu.RUnlock()

	players := make([]PlayerOutcomeDetail, 0, len(s.playerOutcomes))
	for _, d := range s.playerOutcomes {
		if hist := s.postSyncHistory[d.Gamertag]; len(hist) > 0 {
			d.PostSyncHistoryMs = append([]int64(nil), hist...) // copie défensive
		}
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
	if s.SyncGate != nil {
		snap.Gate = s.SyncGate.GateSnapshot()
	}
	if s.lastCycleResult != nil {
		copyRes := *s.lastCycleResult
		snap.LastCycleResult = &copyRes
	}
	return snap
}

// recordOutcome enregistre le résultat détaillé pour un joueur (thread-safe).
// Quand le joueur a effectué un post-sync (PostSync non nil → outcome=ok),
// sa durée est ajoutée au ring postSyncHistory pour la sparkline de tendance.
func (s *AutoSyncScheduler) recordOutcome(d PlayerOutcomeDetail) {
	s.snapshotMu.Lock()
	defer s.snapshotMu.Unlock()
	if s.playerOutcomes == nil {
		s.playerOutcomes = make(map[string]PlayerOutcomeDetail)
	}
	s.playerOutcomes[d.Gamertag] = d
	if d.PostSync != nil {
		s.appendPostSyncSample(d.Gamertag, d.PostSync.DurationMs)
	}
}

// appendPostSyncSample ajoute une durée au ring borné du joueur (appelé sous
// snapshotMu).
func (s *AutoSyncScheduler) appendPostSyncSample(gamertag string, ms int64) {
	if s.postSyncHistory == nil {
		s.postSyncHistory = make(map[string][]int64)
	}
	h := append(s.postSyncHistory[gamertag], ms)
	if len(h) > postSyncHistorySize {
		h = append([]int64(nil), h[len(h)-postSyncHistorySize:]...)
	}
	s.postSyncHistory[gamertag] = h
}

// previousZeroInsertCount retourne le compteur ConsecutiveZeroInserts du
// précédent cycle pour ce joueur. 0 si aucun cycle précédent enregistré.
// Thread-safe.
func (s *AutoSyncScheduler) previousZeroInsertCount(gamertag string) int {
	s.snapshotMu.RLock()
	defer s.snapshotMu.RUnlock()
	if s.playerOutcomes == nil {
		return 0
	}
	return s.playerOutcomes[gamertag].ConsecutiveZeroInserts
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
			// Sprint B1 commit 17 : event_id sur chaque tick pour tracer le
			// cycle scheduler complet à travers les modules (scheduler →
			// auth → sync → provider → pool → handlers post-sync).
			tickCtx, tickID := logging.WithEvent(ctx, "scheduler.tick")
			cfg, err := s.settings.Load()
			if err != nil {
				slog.WarnContext(tickCtx, "auto_sync: lecture settings échouée — tick ignoré", "err", err)
				continue
			}
			if !cfg.SpnkrAutoSyncEnabled {
				slog.DebugContext(tickCtx, "auto_sync: désactivé dans les settings, tick ignoré")
				continue
			}
			if newInterval := resolveInterval(cfg.SpnkrAutoSyncIntervalMinutes, cfg.SpnkrAutoSyncIntervalHours); newInterval != interval {
				slog.InfoContext(tickCtx, "auto_sync: intervalle mis à jour", "old", interval, "new", newInterval)
				interval = newInterval
				ticker.Reset(interval)
			}
			slog.InfoContext(tickCtx, "auto_sync: tick démarré", "event", tickID)
			res := s.RunOnceTrigger(tickCtx, "tick")
			slog.InfoContext(tickCtx, "auto_sync: cycle terminé",
				"total", res.Total,
				"synced", res.Synced,
				"skipped", res.Skipped,
				"failed", res.Failed,
				"duration", res.Duration.Round(time.Millisecond),
			)
			if res.Failed > 0 {
				slog.WarnContext(tickCtx, "auto_sync: des joueurs ont échoué — consulter le snapshot diag",
					"failed_count", res.Failed,
				)
			}
			if res.Total > 0 && res.Skipped == res.Total {
				slog.WarnContext(tickCtx, "auto_sync: aucun joueur synchronisé — vérifier le pool",
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
	return s.RunOnceTrigger(ctx, "manual")
}

// RunOnceTrigger est RunOnce avec la provenance du déclenchement ("tick" pour
// la boucle périodique, "manual" pour HTTP/diag/job) — tracée dans
// l'historique des cycles du dashboard monitoring.
func (s *AutoSyncScheduler) RunOnceTrigger(ctx context.Context, trigger string) *RunOnceResult {
	start := time.Now()
	// Cumuls process-wide capturés AVANT le cycle : les deltas après cycle
	// attribuent au cycle sa fenêtre d'indispo lectures, ses 503, son temps
	// API et son temps d'écriture (corrélation dashboard monitoring P4).
	loadBefore := captureCycleLoad()
	res := &RunOnceResult{}

	players, err := s.cfg.LoadPlayers()
	if err != nil {
		slog.ErrorContext(ctx, "auto_sync: chargement des joueurs échoué", "err", err)
		return res
	}
	// Exclure les couples (joueur, titre) en pause (sync_enabled=false) : ils ne
	// sont plus rafraîchis. Filtre à la SOURCE → couvre V1 (boucle syncPlayer) ET
	// V2 (runOnceV2 reçoit la slice déjà filtrée).
	players = domain.SyncablePlayers(players)
	res.Total = len(players)

	// Détection de claim fuité : le cycle auto-sync sert de heartbeat. Un claim
	// du gate tenu anormalement longtemps signale un release jamais appelé (le
	// joueur ne serait plus jamais synchronisé). Cf. jauge expvar sync_gate_inflight
	// pour le signal temps-réel.
	s.warnStaleGateClaims(ctx)

	// ADR 0027 dispatch : si V2 est activé via env var ET orchestrator
	// non-nil, déléguer au pipeline V2. Fallback silencieux à V1 si
	// l'orchestrator retourne ErrNotImplemented (cas D0-D6 transitoire)
	// ou en cas d'échec global (best-effort, V1 reprend).
	if s.shouldUseV2() {
		if v2Res, v2Err := s.runOnceV2(ctx, players); v2Err == nil {
			s.storeCycleResult(v2Res, trigger, captureCycleLoad().deltaSince(loadBefore))
			return v2Res
		} else if v2Err == syncv2.ErrNotImplemented {
			slog.DebugContext(ctx, "auto_sync: V2 stub → fallback V1 silencieux")
		} else {
			slog.WarnContext(ctx, "auto_sync: V2 échec — fallback V1", "err", v2Err)
		}
		// Fallthrough vers V1.
	}

	// Phase 3.4 (plan stabilisation 2026-05-22) — paralléliser le cycle :
	// chaque syncPlayer tourne dans une goroutine, errgroup.SetLimit borne à
	// la taille du pool de tokens. Gain estimé 15min → 5-8min sur 3 joueurs.
	//
	// Safety : syncPlayer met à jour s.playerOutcomes via recordOutcome qui est
	// protégé par s.snapshotMu. Les writes shared sont déjà sérialisés par
	// dblease.leaseMutex + singleflight match_participants (phase 2.3). Les
	// compteurs locaux (Synced/Skipped/Failed) sont protégés via atomic.Int32.
	parallelism := s.poolSizeSafe()
	if parallelism < 1 {
		parallelism = 1
	}
	slog.InfoContext(ctx, "auto_sync: démarrage du cycle",
		"player_count", res.Total,
		"pool_size", parallelism,
		"parallel", parallelism > 1,
	)

	var synced, skipped, failed atomic.Int32
	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(parallelism)
	for _, p := range players {
		p := p
		eg.Go(func() error {
			outcome := s.syncPlayer(egCtx, p)
			switch outcome {
			case outcomeOK:
				synced.Add(1)
			case outcomeSkipped:
				skipped.Add(1)
			case outcomeFailed:
				failed.Add(1)
			}
			// Best-effort : un échec syncPlayer n'annule pas les autres goroutines.
			// L'erreur est déjà loggée + reflétée dans res.Failed++.
			return nil
		})
	}
	_ = eg.Wait()
	res.Synced = int(synced.Load())
	res.Skipped = int(skipped.Load())
	res.Failed = int(failed.Load())

	// PLAN_SPARTAN_IDENTITY_REFACTOR §11 Phase 5 (2026-05-25) :
	// customRefresher.RefreshAll(ctx) supprimé. La customisation est désormais
	// rafraîchie en LIVE à chaque visite home (CareerLiveService.kickoff).

	res.Duration = time.Since(start)

	s.storeCycleResult(res, trigger, captureCycleLoad().deltaSince(loadBefore))

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
//
// présente) → exécution sync delta → enregistrement outcome. Splitter forcerait à
// dupliquer le state PlayerOutcomeDetail dans les sous-fonctions.
//
//nolint:funlen // Orchestrateur sync-per-player : guards (cooldown, watcher, DB
func (s *AutoSyncScheduler) syncPlayer(ctx context.Context, p domain.PlayerSummary) syncOutcome {
	// Sprint B1 commit 17 : event_id par joueur (sous-événement du tick).
	// Permet de filtrer logs/*.log pour reconstituer le timeline d'un user
	// donné indépendamment des autres en cours de sync.
	ctx, evID := logging.WithEvent(ctx, "scheduler.sync:"+p.Gamertag)
	slog.InfoContext(ctx, "auto_sync: traitement joueur démarré",
		"gamertag", p.Gamertag, "xuid", p.XUID, "event", evID)

	startedAt := time.Now()
	// Lecture du compteur zero-insert précédent AVANT toute exécution. La défer
	// finale appelle recordOutcome qui écrasera la valeur ; on garde la
	// précédente pour pouvoir l'incrémenter ou la conserver selon l'outcome.
	prevZeroInserts := s.previousZeroInsertCount(p.Gamertag)
	detail := PlayerOutcomeDetail{
		Gamertag:               p.Gamertag,
		XUID:                   p.XUID,
		AttemptedAt:            startedAt,
		ConsecutiveZeroInserts: prevZeroInserts, // défaut : préserver (cas skipped/failed)
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

	if skipReason, ok := s.checkSyncPreconditions(ctx, p); !ok {
		detail.Reason = skipReason
		outcome = outcomeSkipped
		return outcome
	}

	// Gate cross-source : céder si un sync du même joueur est déjà en vol (watcher
	// ou HTTP). Posé APRÈS checkSyncPreconditions (skip économe sans claim) et
	// AVANT le RunDelta ; `defer release()` est la 1re ligne post-claim → couvre
	// tous les retours faillibles (runner nil, RunDelta err, succès). Un skip ne
	// marque JAMAIS le joueur à jour → re-tenté au prochain tick quand le claim se
	// libère. Complète l'ActivityChecker (déjà appliqué) pour le résidu TOCTOU.
	if s.SyncGate != nil {
		release, ok := s.SyncGate.TryClaimT(resolveTitleSlug(p), p.Gamertag)
		if !ok {
			slog.InfoContext(ctx, "auto_sync: sync déjà en vol (autre source) — tick différé",
				"gamertag", p.Gamertag)
			detail.Reason = "différé: sync en cours via autre source (watcher/HTTP)"
			outcome = outcomeSkipped
			return outcome
		}
		defer release()
	}

	// ──────────────────────────────────────────────────────────────────────
	// Sync delta via DeltaRunner (PooledHaloClient en production, mockable en test).
	// ──────────────────────────────────────────────────────────────────────
	// MT-11 / PMT-3 : pose le titre du joueur dans le ctx AVANT la factory, pour
	// que BuildEngine (→ NewSyncEngineForTitle) écrive dans les DB du bon titre
	// et que les sous-modules/logs héritent du slug.
	slug := resolveTitleSlug(p)
	ctx = ctxkeys.WithTitleSlug(ctx, slug)
	slog.InfoContext(ctx, "auto_sync: démarrage sync delta", "gamertag", p.Gamertag, "title_slug", slug)

	// Sélection registry-driven (jamais slug==) du runner. Les titres live-only
	// (Halo 5+) ont leur propre pipeline (fetch cryptum → persist shared) et leur
	// auth transite par le ctx (pas de PooledHaloClient interne) → branche dédiée.
	var runner DeltaRunner
	if livesync.HandlesTitle(slug) {
		r, runCtx, release, lerr := s.liveRunner(ctx, slug, p.Gamertag, p.XUID)
		if lerr != nil {
			slog.ErrorContext(ctx, "auto_sync: runner live indisponible",
				"gamertag", p.Gamertag, "title_slug", slug, "err", lerr)
			detail.Reason = "runner live indisponible: " + lerr.Error()
			detail.FirstError = lerr.Error()
			outcome = outcomeFailed
			return outcome
		}
		defer release()
		runner, ctx = r, runCtx
	} else {
		factory := s.RunnerFactory
		if factory == nil {
			factory = s.defaultRunnerFactory
		}
		runner = factory(ctx, p.Gamertag, p.XUID)
		if runner == nil {
			slog.ErrorContext(ctx, "auto_sync: RunnerFactory a retourné nil",
				"gamertag", p.Gamertag)
			detail.Reason = "RunnerFactory a retourné nil (pool absent ?)"
			outcome = outcomeFailed
			return outcome
		}
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
	// Compteurs post-sync exposés au dashboard monitoring (copie défensive :
	// le snapshot survit au SyncResult du cycle).
	if syncResult.PostSync != nil {
		ps := *syncResult.PostSync
		detail.PostSync = &ps
	}

	// Counter zero-insert : reset si on insère ≥1 match, sinon incrément.
	// Sentinelle d'API stale / format URL incorrect / gamertag changé.
	if syncResult.MatchesInserted > 0 {
		detail.ConsecutiveZeroInserts = 0
	} else {
		detail.ConsecutiveZeroInserts = prevZeroInserts + 1
		if detail.ConsecutiveZeroInserts >= ConsecutiveZeroInsertWarnThreshold {
			slog.WarnContext(ctx, "auto_sync: zero-insert prolongé — sync delta réussie mais 0 nouveau match sur N cycles consécutifs",
				"gamertag", p.Gamertag,
				"xuid", p.XUID,
				"consecutive_zero_inserts", detail.ConsecutiveZeroInserts,
				"threshold", ConsecutiveZeroInsertWarnThreshold,
				"hint", "vérifier endpoint Halo + format URL + token resolved (probe /api/v1/_diag/auto-sync/probe)",
			)
		}
	}

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

	// Convergence backfill events (bornée) : rattrape les highlight_events
	// manquants (matchs importés OpenSpartan, films retardés / 404 transitoires).
	// S'exécute SOUS le claim déjà tenu (gate=nil → pas de re-claim) après la sync
	// delta réussie. Best-effort : n'affecte jamais l'outcome du tick.
	s.runEventsConvergencePass(ctx, p.Gamertag, p.XUID)

	outcome = outcomeOK
	return outcome
}

// convergencePerCycleMax borne le nombre de matchs traités par la passe de
// convergence events à chaque tick (le reste est repris au tick suivant).
// Override via LEVELUP_EVENTS_CONVERGENCE_MAX.
const convergencePerCycleMax = 50

// eventsConvergenceEnabled : kill-switch (défaut ON). LEVELUP_EVENTS_CONVERGENCE=0
// désactive la passe scheduler ET le trigger immédiat.
func eventsConvergenceEnabled() bool {
	return os.Getenv("LEVELUP_EVENTS_CONVERGENCE") != "0"
}

func convergencePerCycleLimit() int {
	if v := os.Getenv("LEVELUP_EVENTS_CONVERGENCE_MAX"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return convergencePerCycleMax
}

// runEventsConvergencePass lance une passe bornée de convergence events pour un
// joueur, SOUS le claim déjà tenu par syncPlayer (gate=nil → pas de re-claim).
// Gate sur le pool (tests sans pool → skip) + kill-switch. Best-effort.
func (s *AutoSyncScheduler) runEventsConvergencePass(ctx context.Context, gamertag, xuid string) {
	if s.pool == nil || !eventsConvergenceEnabled() {
		return
	}
	engine := s.BuildEngine(ctx, gamertag, xuid)
	res, err := engine.RunEventsConvergence(ctx, nil, convergencePerCycleLimit())
	if err != nil {
		slog.WarnContext(ctx, "auto_sync: convergence events échouée", "gamertag", gamertag, "err", err)
		return
	}
	if res.EventsWritten > 0 || res.NoFilmFinal > 0 || res.Skipped > 0 {
		slog.InfoContext(ctx, "auto_sync: convergence events",
			"gamertag", gamertag, "detected", res.Detected,
			"events_written", res.EventsWritten, "no_film_final", res.NoFilmFinal,
			"skipped", res.Skipped, "dominance_updated", res.DominanceUpdated)
	}
}

// TriggerEventsConvergence lance une passe de convergence events IMMÉDIATE pour un
// joueur (ex. juste après un import OpenSpartan), en réutilisant le pool d'auth.
// Cède au sync live par lot (gate=SyncGate). Conçue pour un appel en goroutine
// (ne tient pas de claim externe). No-op si pool absent ou kill-switch off.
func (s *AutoSyncScheduler) TriggerEventsConvergence(ctx context.Context, gamertag, xuid string) {
	if s == nil || s.pool == nil || !eventsConvergenceEnabled() {
		return
	}
	engine := s.BuildEngine(ctx, gamertag, xuid)
	res, err := engine.RunEventsConvergence(ctx, s.SyncGate, 0) // tous les incomplets, cède au live
	if err != nil {
		slog.WarnContext(ctx, "trigger convergence events échouée", "gamertag", gamertag, "err", err)
		return
	}
	slog.InfoContext(ctx, "trigger convergence events terminée",
		"gamertag", gamertag, "detected", res.Detected,
		"events_written", res.EventsWritten, "no_film_final", res.NoFilmFinal,
		"skipped", res.Skipped, "ceded", res.Ceded)
}

// warnStaleGateClaims émet un WARN par claim du gate anormalement ancien
// (potentiellement fuité : release jamais appelé → joueur jamais re-synchronisé).
// No-op si aucun gate (NopSyncGate renvoie un cliché vide).
func (s *AutoSyncScheduler) warnStaleGateClaims(ctx context.Context) {
	if s.SyncGate == nil {
		return
	}
	for _, cl := range s.SyncGate.GateSnapshot().Claims {
		if cl.Stale {
			slog.WarnContext(ctx, "sync_gate: claim potentiellement fuité (tenu anormalement longtemps)",
				"gamertag", cl.Gamertag, "source", cl.Source, "age_ms", cl.AgeMs)
		}
	}
}

// checkSyncPreconditions vérifie les 4 préconditions de sync (pool initialisé,
// joueur dans pool, watcher inactif, DB présente). Retourne (raison_skip, false)
// si une précondition échoue, sinon ("", true).
func (s *AutoSyncScheduler) checkSyncPreconditions(ctx context.Context, p domain.PlayerSummary) (string, bool) {
	if s.pool == nil {
		slog.InfoContext(ctx, "auto_sync: pool nil, joueur ignoré", "gamertag", p.Gamertag)
		return "pool de tokens non initialisé (aucun credential découvert au boot)", false
	}
	if !s.pool.HasPlayer(p.Gamertag) {
		slog.InfoContext(ctx, "auto_sync: joueur absent du pool, ignoré",
			"gamertag", p.Gamertag,
			"hint", "définir SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG> dans .env.local ou faire une sync initiale",
		)
		return "joueur absent du pool (pas de token discoverable via Discovery — vérifier .env.local et sync_meta)", false
	}
	if s.ActivityChecker != nil && s.ActivityChecker.IsPlayerActive(p.Gamertag) {
		slog.InfoContext(ctx, "auto_sync: watcher actif sur ce joueur — tick cédé",
			"gamertag", p.Gamertag,
		)
		return "watcher actif (Watching/Syncing/Cooling) — tick cédé", false
	}
	// Slug porté par le profil (db_profiles.json title-scoped), fallback DefaultSlug.
	// MT-11 / PMT-3 : la dette « le moteur écrit encore en DefaultSlug » est RÉSORBÉE —
	// syncPlayer pose le slug dans le ctx → BuildEngine construit via
	// NewSyncEngineForTitle. La précondition os.Stat utilise le même helper.
	slug := resolveTitleSlug(p)
	// Les titres live-only (Halo 5+) écrivent en shared-only (pas de player DB ;
	// le shared est provisionné au boot via provisionAdditionalActiveTitles). La
	// précondition « player DB présente » ne s'applique donc pas — sinon ces
	// joueurs seraient skippés à jamais (os.Stat échoue toujours).
	if livesync.HandlesTitle(slug) {
		return "", true
	}
	dbPath := titlePkg.NewPathResolver(s.cfg.RepoRoot).PlayerDBPath(slug, p.Gamertag)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		slog.InfoContext(ctx, "auto_sync: DB joueur absente, joueur ignoré",
			"gamertag", p.Gamertag, "title_slug", slug, "db_path", dbPath,
			"hint", "lancer la sync initiale via POST /sync/initial pour créer la DB",
		)
		return "DB joueur absente — sync initiale jamais effectuée", false
	}
	return "", true
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

// runOnceV2 exécute un cycle complet via le pipeline V2 (ADR 0027).
// Convertit []PlayerSummary → []syncv2.PlayerProfile, appelle l'orchestrator,
// mappe le CycleResult → RunOnceResult.
//
// Retourne (nil, syncv2.ErrNotImplemented) si l'orchestrator est encore le
// stub D0 — le caller doit alors fallback vers V1.
func (s *AutoSyncScheduler) runOnceV2(ctx context.Context, players []domain.PlayerSummary) (*RunOnceResult, error) {
	profiles := make([]syncv2.PlayerProfile, 0, len(players))
	for _, p := range players {
		profiles = append(profiles, syncv2.PlayerProfile{
			Gamertag:   p.Gamertag,
			XUID:       p.XUID,
			PlayerSlug: p.PlayerSlug,
			TitleSlug:  resolveTitleSlug(p), // MT-11 / PMT-3 : porte le titre au pipeline V2
		})
	}

	start := time.Now()
	cycleRes, err := s.cycleOrchestrator.Run(ctx, profiles)
	if err != nil {
		return nil, err
	}

	res := &RunOnceResult{
		Total:    len(players),
		Duration: time.Since(start),
	}
	for _, outcome := range cycleRes.PerPlayer {
		switch outcome.Status {
		case "ok", "partial":
			res.Synced++
		case "failed":
			res.Failed++
		default:
			res.Skipped++
		}
	}
	slog.InfoContext(ctx, "auto_sync: cycle V2 terminé",
		"total", res.Total,
		"synced", res.Synced,
		"failed", res.Failed,
		"unique_matches", cycleRes.UniqueMatches,
		"duration", res.Duration.Round(time.Millisecond),
	)
	return res, nil
}
