// Package watcher — daemon.go : démon de surveillance de présence.
//
// Le Daemon orchestre tout le système de présence :
//   - Connexion RTA WebSocket (Xbox Live)
//   - Pollers Steam (fallback)
//   - PlayerWatchers (FSM + MatchPoller) par joueur
//   - MatchQueue + Coordinator pour les syncs
//
// Le Daemon est démarré par main.go et tourne en arrière-plan.
// Il expose son état via WatcherStateProvider.
package watcher

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/presence"
	syncpkg "levelup/go-api/internal/sync"
)

// stopWaitTimeout est le délai max pendant lequel Stop() attend que les
// goroutines internes (connectAndSubscribe, consumeQueue) retournent après
// l'annulation du ctx. Au-delà, on rend la main pour ne jamais bloquer le
// shutdown global — les goroutines qui ignoreraient le ctx seront tuées par
// l'OS lors de l'os.Exit().
const stopWaitTimeout = 3 * time.Second

// DaemonController est l'interface exposée à l'API HTTP pour contrôler le daemon.
// Implémenté par *Daemon. Nil autorisé dans les handlers (watcher désactivé).
type DaemonController interface {
	Start(ctx context.Context, authHeader string, playerList []domain.PlayerSummary)
	Stop()
	UpdateAuth(authHeader string)
	UpdateSubscriptions(gamertags []string)
	IsRunning() bool
	GetStatus() WatcherStatus
	// PR 2.5b — ajout dynamique d'un joueur après le démarrage (typiquement après
	// login Xbox SSO). Crée un PlayerWatcher + REST poller pour le nouveau joueur.
	AddPlayer(ctx context.Context, p domain.PlayerSummary) error
}

// DaemonConfig configure le watcher daemon.
type DaemonConfig struct {
	RepoRoot        string
	SteamAPIKey     string // vide = pas de polling Steam
	MaxParallelSync int

	// LiveRefreshFactory est une factory optionnelle pour créer un LiveRefreshTrigger
	// par joueur. Si nil, le rafraîchissement live BP/Challenges est désactivé.
	LiveRefreshFactory func(gamertag, xuid string) LiveRefreshTrigger
}

// Daemon est le démon de surveillance de présence.
//
// Architecture (post-cleanup RTA 2026-05-26) : un seul `PresenceClient` REST
// partagé `trackerRestClient` utilise l'authHeader du tracker (token JGtm,
// refresh via UpdateAuth) pour interroger la présence de chaque joueur (lui
// + amis Xbox visibles). Chaque PlayerWatcher a son propre RESTPoller qui
// dispatche les events vers le handler watcher → FSM → MatchPoller → sync.
type Daemon struct {
	cfg         DaemonConfig
	titleReg    *title.Registry
	coordinator *syncpkg.Coordinator
	queue       *MatchQueue

	// trackerRestClient : 1 client REST partagé entre les RESTPoller des
	// joueurs trackés. UpdateAuth propage le refresh XSTS à tous les pollers
	// atomiquement.
	trackerRestClient *presence.PresenceClient

	playersMu sync.RWMutex
	players   map[string]*PlayerWatcher // gamertag → watcher

	running bool
	cancel  context.CancelFunc
	rootCtx context.Context // capturé dans Start, utilisé pour les goroutines des pollers

	// wg track les goroutines internes lancées dans Start() pour que Stop()
	// puisse les attendre. Sans ce tracking, les goroutines peuvent encore
	// toucher metaDB après que main.go a fait duckdb.CloseAll() → handles
	// DuckDB orphelins lors d'un SIGKILL d'air.
	wg sync.WaitGroup
}

// NewDaemon crée un watcher daemon (non démarré).
func NewDaemon(cfg DaemonConfig, titleReg *title.Registry, syncRunner syncpkg.SyncRunner) *Daemon {
	maxParallel := cfg.MaxParallelSync
	if maxParallel < 1 {
		maxParallel = 2
	}

	return &Daemon{
		cfg:         cfg,
		titleReg:    titleReg,
		coordinator: syncpkg.NewCoordinator(syncRunner, maxParallel),
		queue:       NewMatchQueue(100),
		players:     make(map[string]*PlayerWatcher),
	}
}

// Start démarre le daemon. Non bloquant — lance des goroutines internes.
func (d *Daemon) Start(ctx context.Context, authHeader string, playerList []domain.PlayerSummary) {
	ctx, d.cancel = context.WithCancel(ctx)
	// Sprint B1 commit 17 : event_id sur le daemon (un id pour toute la vie
	// du watcher).
	ctx, daemonID := logging.WithEvent(ctx, "watcher.daemon")
	d.rootCtx = ctx
	d.running = true

	slog.InfoContext(ctx, "watcher_daemon: démarrage",
		"players", len(playerList),
		"max_parallel_sync", d.cfg.MaxParallelSync,
		"event", daemonID,
	)

	// Client REST partagé pour les pollers tracker (token JGtm, propagé via
	// UpdateAuth atomic à tous les pollers actifs).
	d.trackerRestClient = presence.NewPresenceClient(authHeader)

	// Initialiser les PlayerWatchers + lancer un REST poller par joueur
	d.initPlayers(ctx, playerList)

	// Consommer la queue de matchs
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		d.consumeQueue(ctx)
	}()

	slog.InfoContext(ctx, "watcher_daemon: démarré")
}

// Stop arrête le daemon proprement et attend que ses goroutines internes
// retournent (avec un timeout dur de stopWaitTimeout pour ne jamais bloquer
// le shutdown global).
func (d *Daemon) Stop() {
	if d.cancel != nil {
		d.cancel()
	}

	// Attendre les goroutines internes avec un timeout dur.
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		slog.Info("watcher_daemon: goroutines internes terminées")
	case <-time.After(stopWaitTimeout):
		slog.Warn("watcher_daemon: timeout sur Wait — goroutines internes non terminées",
			"timeout", stopWaitTimeout,
		)
	}

	d.running = false
	slog.Info("watcher_daemon: arrêté")
}

// IsRunning retourne true si le daemon tourne.
func (d *Daemon) IsRunning() bool {
	return d.running
}

// UpdateAuth met à jour le header d'auth du client REST tracker (après
// refresh XSTS). Les pollers REST par joueur partagent ce client donc
// voient le nouveau header atomiquement.
func (d *Daemon) UpdateAuth(authHeader string) {
	if d.trackerRestClient != nil {
		d.trackerRestClient.UpdateAuth(authHeader)
	}
	slog.Info("watcher_daemon: auth tracker REST mis à jour")
}

// GetStatus retourne l'état courant du daemon via StateProvider.
func (d *Daemon) GetStatus() WatcherStatus {
	return NewStateProvider(d).GetStatus()
}

// UpdateSubscriptions remplace la liste des joueurs surveillés.
// gamertags vide ou ["all"] signifie tous les joueurs configurés.
// Les joueurs retirés sont stoppés ; les nouveaux joueurs sont ajoutés.
func (d *Daemon) UpdateSubscriptions(gamertags []string) {
	d.playersMu.Lock()
	defer d.playersMu.Unlock()

	// ["all"] ou vide → rien à filtrer, garder tous
	if len(gamertags) == 0 || (len(gamertags) == 1 && gamertags[0] == "all") {
		slog.Info("watcher_daemon: UpdateSubscriptions: mode 'all' — pas de changement de filtrage")
		return
	}

	// Construire un set des gamertags souhaités
	wanted := make(map[string]struct{}, len(gamertags))
	for _, gt := range gamertags {
		wanted[gt] = struct{}{}
	}

	// Arrêter les joueurs non voulus
	for gt := range d.players {
		if _, ok := wanted[gt]; !ok {
			slog.Info("watcher_daemon: UpdateSubscriptions: joueur retiré", "gamertag", gt)
			delete(d.players, gt)
		}
	}

	slog.Info("watcher_daemon: UpdateSubscriptions appliqué",
		"subscribed", gamertags,
		"active_players", len(d.players),
	)
}

// AddPlayer ajoute un joueur au tracking dynamiquement (typiquement après
// login SSO Xbox). Crée un PlayerWatcher + REST poller pour le joueur en
// utilisant le client REST partagé (token tracker).
//
// Erreur si XUID vide ou player démo. No-op si le joueur est déjà présent.
// Si le daemon n'a pas encore été démarré (rootCtx nil), le PlayerWatcher
// est créé mais sans poller — il sera spawned au prochain Start.
func (d *Daemon) AddPlayer(ctx context.Context, p domain.PlayerSummary) error {
	if p.IsDemo {
		return fmt.Errorf("watcher_daemon: AddPlayer ignore player démo")
	}
	if p.XUID == "" {
		return fmt.Errorf("watcher_daemon: AddPlayer requires non-empty XUID (gamertag=%q)", p.Gamertag)
	}

	ctx, evID := logging.WithEvent(ctx, "watcher.add_player:"+p.Gamertag)
	slog.InfoContext(ctx, "watcher_daemon: AddPlayer démarré",
		"gamertag", p.Gamertag, "xuid", p.XUID, "event", evID)

	d.playersMu.Lock()
	if _, exists := d.players[p.Gamertag]; exists {
		d.playersMu.Unlock()
		slog.DebugContext(ctx, "watcher_daemon: AddPlayer no-op, déjà présent",
			"gamertag", p.Gamertag, "xuid", p.XUID)
		return nil
	}
	pw := NewPlayerWatcher(p.Gamertag, p.XUID, nil, &queueSyncTrigger{
		queue:    d.queue,
		gamertag: p.Gamertag,
		xuid:     p.XUID,
	})
	if d.cfg.LiveRefreshFactory != nil {
		pw = pw.WithLiveRefresh(d.cfg.LiveRefreshFactory(p.Gamertag, p.XUID))
	}
	d.players[p.Gamertag] = pw
	d.playersMu.Unlock()

	slog.InfoContext(ctx, "watcher_daemon: joueur ajouté dynamiquement",
		"gamertag", p.Gamertag, "xuid", p.XUID)

	// Spawn REST poller pour ce joueur (utilise le client tracker partagé).
	// Skip si le daemon n'a pas encore été démarré (rootCtx nil) — le poller
	// sera créé par initPlayers au prochain Start si le joueur est encore là.
	if d.trackerRestClient != nil && d.rootCtx != nil {
		handler := d.makePresenceHandler(d.rootCtx, pw)
		poller := NewRESTPoller(p.XUID, p.Gamertag, d.trackerRestClient, handler)
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			poller.Run(d.rootCtx)
		}()
	}
	return nil
}

// initPlayers crée un PlayerWatcher par joueur + lance son REST poller
// (qui utilise l'authHeader du tracker pour interroger la présence des
// amis via l'API Xbox standard).
func (d *Daemon) initPlayers(ctx context.Context, playerList []domain.PlayerSummary) {
	d.playersMu.Lock()
	defer d.playersMu.Unlock()

	for _, p := range playerList {
		if p.IsDemo || p.XUID == "" {
			continue
		}
		pw := NewPlayerWatcher(p.Gamertag, p.XUID, nil, &queueSyncTrigger{
			queue:    d.queue,
			gamertag: p.Gamertag,
			xuid:     p.XUID,
		})
		if d.cfg.LiveRefreshFactory != nil {
			pw = pw.WithLiveRefresh(d.cfg.LiveRefreshFactory(p.Gamertag, p.XUID))
		}
		d.players[p.Gamertag] = pw

		slog.InfoContext(ctx, "watcher_daemon: joueur initialisé",
			"gamertag", p.Gamertag,
			"xuid", p.XUID,
		)

		// REST poller par joueur — partage le client tracker (token JGtm).
		// Pas de doublon problématique avec AddUserClient (le poller dédié
		// JGtm a son propre client + son propre auth refresh) : les deux
		// pollers dispatch vers le même PlayerWatcher dont les transitions
		// FSM sont idempotentes.
		if d.trackerRestClient != nil {
			handler := d.makePresenceHandler(ctx, pw)
			poller := NewRESTPoller(p.XUID, p.Gamertag, d.trackerRestClient, handler)
			d.wg.Add(1)
			go func() {
				defer d.wg.Done()
				poller.Run(d.rootCtx)
			}()
		}
	}
}

// makePresenceHandler crée le callback de présence pour un joueur.
func (d *Daemon) makePresenceHandler(ctx context.Context, pw *PlayerWatcher) presence.EventHandler {
	return func(event presence.PresenceEvent) {
		// Sprint B1 commit 19 : event_id par event de présence. Trace le
		// maillon manquant entre `watcher.rta` (RTA WebSocket event reçu)
		// et `watcher.trigger` (sync déclenché par dequeue). Permet de
		// répondre "pourquoi ce user a/n'a pas déclenché un sync ?"
		evCtx, evID := logging.WithEvent(ctx, "watcher.presence:"+pw.gamertag)

		// Capture `lastSeen` si présent dans le payload, sans filtrage côté
		// backend : tout est stocké tel que renvoyé par Xbox (jeu, ou
		// Dashboard "Online" id=1022622766). Le frontend re-mappe les noms
		// spéciaux ("Online" → "Accueil Xbox") pour une UI lisible —
		// décision UX, pas backend.
		if event.LastSeen != nil {
			pw.RecordLastSeen(evCtx, event.LastSeen)
		}

		// Capture le state Xbox brut (Online/Away/Offline) pour affichage UI
		// précis. Différencie "Hors-ligne" (Offline) vs "Absent" (Away) vs
		// "En ligne" (Online + pas en jeu tracké) — plus parlant que le state
		// FSM générique "Idle".
		pw.RecordPresenceState(event.PresenceState)

		// Vérifier si le titre correspond à un jeu tracké
		if event.PresenceDetail != nil {
			td := d.titleReg.MatchPresence(event.PresenceDetail.TitleID)
			if td != nil {
				slog.InfoContext(evCtx, "watcher_daemon: présence détectée — titre tracké",
					"gamertag", pw.gamertag,
					"title", td.Name,
					"state", event.PresenceState,
					"event", evID,
				)
				pw.OnPresenceActive(evCtx)
				return
			}
			// Titre présent mais pas dans le registre (ex: Xbox Dashboard
			// `Online` id=1022622766) → sémantiquement le user est sorti du
			// jeu tracké, on bascule donc en Inactive. Avant le fix 2026-05-26
			// ce cas était un no-op silencieux qui laissait la FSM bloquée en
			// Watching après extinction Halo.
			slog.InfoContext(evCtx, "watcher_daemon: titre non tracké → traité comme inactif",
				"gamertag", pw.gamertag,
				"title_id", event.PresenceDetail.TitleID,
				"title_name", event.PresenceDetail.TitleName,
				"state", event.PresenceState,
				"event", evID,
			)
			pw.OnPresenceInactive(evCtx)
			return
		}

		// Pas de PresenceDetail (state Offline ou payload sans titre)
		slog.DebugContext(evCtx, "watcher_daemon: présence sans titre actif",
			"gamertag", pw.gamertag, "state", event.PresenceState, "event", evID)
		pw.OnPresenceInactive(evCtx)
	}
}

// consumeQueue consomme la MatchQueue et soumet au Coordinator.
func (d *Daemon) consumeQueue(ctx context.Context) {
	// Sprint B1 commit 17 : event_id sur la loop globale. Chaque match
	// dequeue génère son propre sous-event (par gamertag) pour tracer
	// le déclenchement du sync.
	baseCtx, qID := logging.WithEvent(ctx, "watcher.queue")
	slog.InfoContext(baseCtx, "watcher_daemon: queue consumer démarré", "event", qID)

	for {
		select {
		case <-baseCtx.Done():
			return
		case req := <-d.queue.Dequeue():
			// Sous-event par dequeue : identifie le déclenchement d'un sync
			// par le watcher (vs un déclenchement périodique scheduler).
			reqCtx, reqID := logging.WithEvent(baseCtx, "watcher.trigger:"+req.Gamertag)
			slog.InfoContext(reqCtx, "watcher_daemon: match dequeued → sync trigger",
				"gamertag", req.Gamertag, "xuid", req.XUID, "match_count", len(req.MatchIDs), "event", reqID)
			d.coordinator.Submit(reqCtx, syncpkg.CoordinatorRequest{
				Gamertag: req.Gamertag,
				XUID:     req.XUID,
				MatchIDs: req.MatchIDs,
			})
		}
	}
}

// queueSyncTrigger adapte la MatchQueue en SyncTrigger pour le PlayerWatcher.
type queueSyncTrigger struct {
	queue    *MatchQueue
	gamertag string
	xuid     string
}

func (q *queueSyncTrigger) TriggerSync(_ context.Context, gamertag, xuid string, matchIDs []string) error {
	q.queue.Enqueue(MatchRequest{
		Gamertag: gamertag,
		XUID:     xuid,
		MatchIDs: matchIDs,
	})
	return nil
}
