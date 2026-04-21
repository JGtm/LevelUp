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
	"log/slog"
	"sync"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/presence"
	syncpkg "levelup/go-api/internal/sync"
)

// DaemonController est l'interface exposée à l'API HTTP pour contrôler le daemon.
// Implémenté par *Daemon. Nil autorisé dans les handlers (watcher désactivé).
type DaemonController interface {
	Start(ctx context.Context, authHeader string, playerList []domain.PlayerSummary)
	Stop()
	UpdateAuth(authHeader string)
	UpdateSubscriptions(gamertags []string)
	IsRunning() bool
	GetStatus() WatcherStatus
}

// DaemonConfig configure le watcher daemon.
type DaemonConfig struct {
	RepoRoot        string
	SteamAPIKey     string // vide = pas de polling Steam
	MaxParallelSync int

	// LiveRefreshFactory est une factory optionnelle pour créer un LiveRefreshTrigger
	// par joueur. Si nil, le rafraîchissement live BP/Challenges est désactivé.
	LiveRefreshFactory func(gamertag, xuid string) LiveRefreshTrigger

	// RefreshRTAAuth est appelé par le ReconnectManager quand un subscribe RTA est
	// refusé avec status=3 (token XSTS expiré). Doit acquérir un XSTS frais et
	// appeler daemon.UpdateAuth avec le nouveau header. Si nil, la reconnexion
	// attend authRefreshRetryDelay (30s) avant de retenter.
	RefreshRTAAuth func(ctx context.Context) error
}

// Daemon est le démon de surveillance de présence.
type Daemon struct {
	cfg         DaemonConfig
	rtaClient   *presence.RTAClient
	titleReg    *title.Registry
	coordinator *syncpkg.Coordinator
	queue       *MatchQueue

	playersMu sync.RWMutex
	players   map[string]*PlayerWatcher // gamertag → watcher

	running bool
	cancel  context.CancelFunc
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
	d.running = true

	slog.InfoContext(ctx, "watcher_daemon: démarrage",
		"players", len(playerList),
		"max_parallel_sync", d.cfg.MaxParallelSync,
	)

	// Créer le client RTA
	d.rtaClient = presence.NewRTAClient(authHeader)

	// Initialiser les PlayerWatchers
	d.initPlayers(ctx, playerList)

	// Connecter RTA + souscrire les présences
	go d.connectAndSubscribe(ctx)

	// Consommer la queue de matchs
	go d.consumeQueue(ctx)

	slog.InfoContext(ctx, "watcher_daemon: démarré")
}

// Stop arrête le daemon proprement.
func (d *Daemon) Stop() {
	if d.cancel != nil {
		d.cancel()
	}
	if d.rtaClient != nil {
		_ = d.rtaClient.Close()
	}
	d.running = false
	slog.Info("watcher_daemon: arrêté")
}

// IsRunning retourne true si le daemon tourne.
func (d *Daemon) IsRunning() bool {
	return d.running
}

// UpdateAuth met à jour le header d'auth RTA (après refresh XSTS).
func (d *Daemon) UpdateAuth(authHeader string) {
	if d.rtaClient != nil {
		d.rtaClient.UpdateAuth(authHeader)
		slog.Info("watcher_daemon: auth RTA mis à jour")
	}
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

// initPlayers crée un PlayerWatcher par joueur.
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
	}
}

// connectAndSubscribe gère la connexion RTA et l'abonnement aux présences.
func (d *Daemon) connectAndSubscribe(ctx context.Context) {
	reconnectMgr := presence.NewReconnectManager(
		d.rtaClient,
		presence.DefaultReconnectPolicy(),
		func(connectCtx context.Context) error {
			if err := d.rtaClient.Connect(connectCtx); err != nil {
				return err
			}
			// Re-souscrire tous les joueurs pour chaque titre tracké.
			// Les titres trackés sont ceux enregistrés dans le TitleRegistry avec
			// un XboxTitleID (ex: Halo Infinite, et tout futur titre Halo ajouté).
			trackedTitles := d.titleReg.All()
			d.playersMu.RLock()
			defer d.playersMu.RUnlock()
			for _, pw := range d.players {
				handler := d.makePresenceHandler(ctx, pw)
				var lastErr error
				for _, td := range trackedTitles {
					if td.XboxTitleID == "" {
						continue
					}
					if err := d.rtaClient.Subscribe(connectCtx, pw.xuid, td.XboxTitleID, handler); err != nil {
						slog.WarnContext(connectCtx, "watcher_daemon: échec subscribe",
							"gamertag", pw.gamertag, "title", td.Name, "err", err)
						lastErr = err
					}
				}
				pw.SetSubscribeError(lastErr)
			}
			return nil
		},
	)
	// Brancher le callback de refresh on-demand fourni par la config.
	reconnectMgr.OnAuthExpired = d.cfg.RefreshRTAAuth
	reconnectMgr.RunWithReconnect(ctx)
}

// makePresenceHandler crée le callback de présence pour un joueur.
func (d *Daemon) makePresenceHandler(ctx context.Context, pw *PlayerWatcher) presence.EventHandler {
	return func(event presence.PresenceEvent) {
		// Vérifier si le titre correspond à un jeu tracké
		if event.PresenceDetail != nil {
			td := d.titleReg.MatchPresence(event.PresenceDetail.TitleID)
			if td != nil {
				slog.InfoContext(ctx, "watcher_daemon: présence détectée — titre tracké",
					"gamertag", pw.gamertag,
					"title", td.Name,
					"state", event.PresenceState,
				)
				pw.OnPresenceActive(ctx)
				return
			}
		}

		// Offline ou titre non tracké
		if event.PresenceState == "Offline" || event.PresenceDetail == nil {
			pw.OnPresenceInactive(ctx)
		}
	}
}

// consumeQueue consomme la MatchQueue et soumet au Coordinator.
func (d *Daemon) consumeQueue(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-d.queue.Dequeue():
			d.coordinator.Submit(ctx, syncpkg.CoordinatorRequest{
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
