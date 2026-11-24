// Package scheduler — auto_sync.go : synchronisation delta automatique et périodique.
//
// AutoSyncScheduler lit app_settings.json au démarrage et à chaque tick pour :
//   - vérifier que spnkr_auto_sync_enabled est vrai avant d'agir
//   - adapter l'intervalle si spnkr_auto_sync_interval_(minutes|hours) a changé
//
// Pour chaque joueur configuré dans db_profiles.json, le cycle :
//  1. Vérifie que le pool de tokens existe (aucun token propre exigé : D1, plan 2026-09-16 —
//     les endpoints du sync sont publics, servis par n importe quel token du parc).
//  2. Crée un PooledHaloClient pinné sur ce joueur (le pin ne sert qu aux endpoints privés).
//  3. Lance SyncEngine.RunDelta avec ce client (fetches parallèles internes).
//
// L'auth est entièrement déléguée au Pool/Resolver, qui :
//   - tente un OAuth v2 refresh sur le RT découvert par Discovery (MultiUserTokenStore)
//   - cache les tokens Halo pour ~3h30 (Spartan token lifetime)
//   - persiste les RT rotatés par Microsoft via le callback OnTokenRotated injecté
//     à NewResolver
//
// Le scheduler n'accède plus directement à os.Getenv ni à sync_meta DuckDB pour
// les tokens : c'est entièrement encapsulé dans le couple Discovery+Resolver+Pool.
package scheduler

import (
	"context"
	gosync "sync"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/adminstate"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/pool"
	"levelup/go-api/internal/platform/friendstore"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/replayartifacts"
	syncv2 "levelup/go-api/internal/sync/v2"
)

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

// AutoSyncScheduler orchestre la sync delta périodique de tous les joueurs.
//
// Tous les appels API Halo passent par le Pool : la rotation des refresh_tokens
// est gérée par le Resolver (via callback OnTokenRotated injecté à NewResolver),
// les 429/503 déclenchent un cooldown global, et les fetches sont parallélisés
// au sein de chaque RunDelta via le PooledHaloClient.
type AutoSyncScheduler struct {
	cfg      *config.AppConfig
	settings *settings_platform.Store

	// friends résout la liste d'amis DU JOUEUR synchronisé (par xuid), pour le
	// hook post-sync is_with_friends. Injecté par main.go via WithFriendStore ;
	// nil → pas de FriendsLoader sur les moteurs construits ici.
	friends *friendstore.FriendStore

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
	// optionnelle. Active par défaut (LEVELUP_PERSIST_BATCH_ASYNC=0 la coupe ;
	// coupée aussi en démo). Injectée par le câblage de cmd/server via WithBatchQueue.
	// Sans queue : path synchrone (Persister.Persist direct). Avec queue :
	// path async (queue.Submit + worker, WAL durable, recovery au boot).
	batchQueue *persist.BatchQueue

	// customRefresher (DEPRECATED, supprimé par PLAN_SPARTAN_IDENTITY_REFACTOR
	// §11 Phase 5, 2026-05-25). Le champ est retiré ; la customisation est
	// désormais rafraîchie en LIVE via CareerLiveService.kickoffBackgroundRefresh
	// (UPSERT dans `spartan_identity`).

	// prestigeHook (optionnel) ré-évalue les défis Prestige actifs après ingestion.
	// Câblé sur chaque SyncEngine construit par BuildEngine → fire à engine.run()
	// (chemin V1 : auto-sync + HTTP delta + watcher, tous via BuildEngine). Injecté
	// depuis cmd/server/main.go = PrestigeBundle.RunPostSync. Nil → feature off
	// (no-op). L'identifiant passé est le gamertag (== PlayerSlug pour les joueurs
	// réels, cf. WithPostSyncRunner ci-dessus).
	prestigeHook func(ctx context.Context, playerSlug, titleSlug string)

	// cycleOrchestrator (ADR 0027) — pipeline V2 cycle-level, UNIQUE moteur de sync
	// des joueurs moteur depuis la suppression du pipeline V1 (lot D1c). Câblé via
	// WithCycleOrchestrator (main.go, si prérequis pool/queue/metaDB présents). Nil
	// → le cycle bascule sur le filet structurel syncPlayer de boot (cf. shouldUseV2).
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
	// cycleRanSinceBoot : true dès qu'un cycle a tourné depuis ce boot. False
	// tant que le snapshot est purement réhydraté du disque (C1) — exposé via
	// SchedulerSnapshot.SinceBoot. Protégé par snapshotMu.
	cycleRanSinceBoot bool

	// snapshotStore persiste le snapshot post-sync (JSON hors DuckDB) en fin de
	// cycle et le réhydrate au boot (C1). Nil → persistance désactivée (tests,
	// CLI) : le dashboard reste amnésique au reboot, sans jamais paniquer.
	snapshotStore *adminstate.FileStore
	// actionJournal enregistre la dernière exécution du cycle de sync (C2 —
	// action « sync_cycle », déclencheur tick/manual). Nil → non journalisé.
	actionJournal *adminstate.ActionJournal
	// catchUpCh porte les demandes de rattrapage (RequestCatchUp) jusqu'à la boucle de Run.
	catchUpCh chan string

	cycleState // cadence de Run, cycles demandés, dernière passe par joueur (auto_sync_loop.go)

	// replayEnqueue met la construction d'un rejeu dans la file durable (chemin
	// « ouvrier »). Injecté par main.go après le ServiceRegistry ; nil → le
	// placement « worker » dégrade en « aucune construction ».
	replayEnqueue replayartifacts.EnqueueFunc
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
		SyncGate:   sync.NopSyncGate{},
		catchUpCh:  make(chan string, catchUpQueue),
		cycleState: cycleState{replanCh: make(chan cycleDebt, 1)},
	}
	s.RunnerFactory = s.defaultRunnerFactory
	s.liveRunner = s.acquireLiveTitleRunner
	return s
}

// WithFriendStore attache le store des amis par joueur (data/global/player_friends.json).
// Sans lui, les moteurs construits par BuildEngine n'ont pas de FriendsLoader et les
// nouveaux matchs restent is_with_friends=FALSE jusqu'au prochain recompute.
func (s *AutoSyncScheduler) WithFriendStore(store *friendstore.FriendStore) *AutoSyncScheduler {
	s.friends = store
	return s
}

// Gate retourne le SyncGate du scheduler (pour câblage par main.go vers le
// handler HTTP). Jamais nil : NopSyncGate par défaut.
func (s *AutoSyncScheduler) Gate() sync.SyncGate {
	return s.SyncGate
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

// WithPrestigeHook branche le hook Prestige post-sync sur chaque SyncEngine
// construit par BuildEngine (chemin V1 : auto-sync + HTTP delta via engineBuilder +
// watcher). À appeler depuis cmd/server/main.go = PrestigeBundle.RunPostSync. Nil →
// no-op (feature Prestige off côté RunPostSync). Le chemin V2 (cycle orchestrator)
// est câblé séparément (le hook y est appelé après RunPostSync par PlayerSlug).
func (s *AutoSyncScheduler) WithPrestigeHook(hook func(ctx context.Context, playerSlug, titleSlug string)) *AutoSyncScheduler {
	s.prestigeHook = hook
	return s
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
		SinceBoot:       s.cycleRanSinceBoot,
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
