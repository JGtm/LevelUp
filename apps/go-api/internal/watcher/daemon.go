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
	"sync/atomic"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/presence"
	syncpkg "levelup/go-api/internal/sync"
)

// stopWaitTimeout est le délai max pendant lequel Stop() attend que les
// goroutines internes (consumeQueue, REST pollers) ET les syncs Coordinator en
// vol retournent après l'annulation du ctx. Les RunDelta en cours doivent voir
// ctx.Done(), abandonner et libérer le lease KindPlayer — ce qui prend plus que
// les pollers, d'où 8s (revue COORD-1 2026-06-01 : avant, Stop() n'attendait pas
// du tout les syncs → write-after duckdb.CloseAll possible). Au-delà, on rend la
// main pour ne jamais bloquer le shutdown global (budget total 15s côté main.go).
const stopWaitTimeout = 8 * time.Second

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
	RepoRoot string
	// SteamAPIKey : NON CONSOMMÉ actuellement (W8, revue 2026-06-01). Le
	// presence.SteamPoller est implémenté + testé (presence_test.go) mais PAS
	// encore câblé dans le daemon — fallback présence Steam planifié, non activé.
	// Renseigné par STEAM_API_KEY pour le jour où il sera branché ; ne déclenche
	// aucun polling tant que le daemon ne l'instancie pas.
	SteamAPIKey     string
	MaxParallelSync int

	// LiveRefreshFactory est une factory optionnelle pour créer un LiveRefreshTrigger
	// par joueur. Si nil, le rafraîchissement live BP/Challenges est désactivé.
	// titleSlug cible la bonne arbo DB (data/titles/{slug}/...) : sans lui, le
	// refresh live (BP/challenges) écrivait dans halo_infinite quel que soit le titre.
	LiveRefreshFactory func(gamertag, xuid, titleSlug string) LiveRefreshTrigger

	// MatchFetcher est partagé entre tous les PlayerWatcher pour le polling
	// Halo API (/hi/players/xuid(N)/matches). Si nil, le MatchPoller est
	// désactivé (mode dégradé loggé une fois par joueur) — pas de panic.
	// Normalement injecté depuis main avec un HaloMatchFetcher branché sur
	// le pool de tokens auto-sync.
	MatchFetcher MatchFetcher

	// BroadcastPresenceActive (incident 2026-05-27) : quand un joueur passe
	// présent in-game (titre tracké), propager l'état Active à TOUS les
	// PlayerWatcher pour qu'ils démarrent leur propre MatchPoller. Utile pour
	// les sessions de groupe où plusieurs joueurs jouent ensemble : si la
	// présence Xbox ne signale fiablement que JGtm (token tracker), les autres
	// (Madina, Choco) restent en Idle et leur MatchPoller ne tourne pas — leurs
	// nouveaux matchs ne sont jamais détectés tant que le scheduler delta ne
	// tourne pas (15 min). Avec broadcast = true, dès que JGtm passe Watching
	// les 3 autres aussi → leurs MatchPoller tournent → leurs nouveaux matchs
	// sont détectés au prochain poll.
	//
	// Coût : 4× appels GET /matches?count=25 toutes les 30s pendant la
	// session. Pour un endpoint public (PolicyAnyPublic du pool), c'est
	// négligeable (~25 IDs × 4 KB par réponse).
	//
	// Idempotent côté PlayerWatcher : `OnPresenceActive` ne re-transitionne
	// pas la FSM si elle est déjà Watching. Pas de risque de cascade.
	BroadcastPresenceActive bool
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
	// players : clé composite playerKey(gamertag, titleSlug) → watcher. Un même
	// gamertag suivi sur 2 titres a 2 watchers DISTINCTS (multi-titre). Les
	// consommateurs orientés-gamertag (broadcast, IsPlayerActive, UpdateSubscriptions)
	// itèrent et filtrent sur pw.gamertag plutôt que d'indexer par la clé.
	players map[string]*PlayerWatcher
	// playerCancels : CancelFunc du REST poller par joueur (W2). Sans elle, un
	// joueur retiré via UpdateSubscriptions laissait tourner sa goroutine REST
	// poller (annulable seulement globalement via d.cancel) → fuite + poll fantôme.
	playerCancels map[string]context.CancelFunc

	// running : lu sans verrou depuis les handlers HTTP (IsRunning) → atomic.
	running atomic.Bool
	// cancel et rootCtx sont écrits dans Start et lus dans Stop/initPlayers/AddPlayer.
	// Tous leurs accès se font sous playersMu (revue 2026-06-02, fix data race).
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
		cfg:           cfg,
		titleReg:      titleReg,
		coordinator:   syncpkg.NewCoordinator(syncRunner, maxParallel),
		queue:         NewMatchQueue(100),
		players:       make(map[string]*PlayerWatcher),
		playerCancels: make(map[string]context.CancelFunc),
	}
}

// playerKey est la clé d'indexation des maps du daemon : un couple (gamertag,
// titre) est tracké INDÉPENDAMMENT par titre (multi-titre live). Sans cette clé
// composite, deux profils d'un même gamertag sous deux titres s'écraseraient (un
// seul watcher survivant). titleSlug vide → DefaultSlug.
func playerKey(gamertag, titleSlug string) string {
	if titleSlug == "" {
		titleSlug = title.DefaultSlug
	}
	return gamertag + "|" + titleSlug
}

// SyncGate expose le Coordinator comme point de déduplication cross-source des
// syncs (interface syncpkg.SyncGate). main.go l'injecte dans l'auto-sync
// scheduler et le handler HTTP pour qu'ils cèdent à un sync déjà en vol (peu
// importe la source). Retourne l'INTERFACE (pas *Coordinator) ; comme le getter
// est appelé sur le type concret *Daemon (jamais sur l'interface DaemonController),
// pas de risque de nil-interface panic quand le watcher est désactivé (dans ce
// cas main.go ne dispose pas de *Daemon et câble un NopSyncGate).
func (d *Daemon) SyncGate() syncpkg.SyncGate {
	return d.coordinator
}

// Start démarre le daemon. Non bloquant — lance des goroutines internes.
func (d *Daemon) Start(ctx context.Context, authHeader string, playerList []domain.PlayerSummary) {
	// Garde double-Start : sans elle, un 2e Start écraserait d.cancel et leakerait
	// la goroutine/ctx du premier (revue 2026-06-02).
	if d.running.Load() {
		slog.WarnContext(ctx, "watcher_daemon: Start ignoré (déjà démarré)")
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	// Sprint B1 commit 17 : event_id sur le daemon (un id pour toute la vie
	// du watcher).
	ctx, daemonID := logging.WithEvent(ctx, "watcher.daemon")
	// cancel/rootCtx écrits sous playersMu (mêmes verrou que leurs lecteurs
	// initPlayers/AddPlayer). initPlayers re-locke ensuite hors de cette section.
	d.playersMu.Lock()
	d.cancel = cancel
	d.rootCtx = ctx
	d.playersMu.Unlock()
	d.running.Store(true)

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
	d.playersMu.Lock()
	cancel := d.cancel
	d.playersMu.Unlock()
	if cancel != nil {
		cancel()
	}

	// Attendre les goroutines internes ET les syncs Coordinator en vol, avec un
	// timeout dur. d.cancel() ci-dessus a annulé le ctx → les RunDelta en cours
	// abandonnent et libèrent le lease, puis leur goroutine run() décrémente le
	// wg du Coordinator (COORD-1).
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		d.coordinator.Wait()
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

	d.running.Store(false)
	slog.Info("watcher_daemon: arrêté")
}

// IsRunning retourne true si le daemon tourne.
func (d *Daemon) IsRunning() bool {
	return d.running.Load()
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

	// Arrêter les joueurs non voulus (W2 : stopper réellement leurs goroutines,
	// pas juste les retirer de la map). La souscription se fait par GAMERTAG :
	// retirer un gamertag retire TOUS ses watchers de titres (la map est keyée par
	// playerKey composite, on filtre sur pw.gamertag).
	for key, pw := range d.players {
		if _, ok := wanted[pw.gamertag]; !ok {
			slog.Info("watcher_daemon: UpdateSubscriptions: joueur retiré",
				"gamertag", pw.gamertag, "title_slug", pw.titleSlug)
			if cancel := d.playerCancels[key]; cancel != nil {
				cancel() // stoppe le REST poller du joueur
				delete(d.playerCancels, key)
			}
			pw.stopPoller() // stoppe le MatchPoller (+ live_refresh lié à pollerCtx)
			delete(d.players, key)
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

	key := playerKey(p.Gamertag, p.TitleSlug)
	d.playersMu.Lock()
	if _, exists := d.players[key]; exists {
		d.playersMu.Unlock()
		slog.DebugContext(ctx, "watcher_daemon: AddPlayer no-op, déjà présent",
			"gamertag", p.Gamertag, "xuid", p.XUID, "title_slug", p.TitleSlug)
		return nil
	}
	pw := NewPlayerWatcher(p.Gamertag, p.XUID, d.cfg.MatchFetcher, &queueSyncTrigger{
		queue:    d.queue,
		gamertag: p.Gamertag,
		xuid:     p.XUID,
	})
	pw.SetTitleSlug(p.TitleSlug) // Phase 1.9 : titre configuré → ctx poller + write-path
	if d.cfg.LiveRefreshFactory != nil {
		pw = pw.WithLiveRefresh(d.cfg.LiveRefreshFactory(p.Gamertag, p.XUID, p.TitleSlug))
	}
	d.players[key] = pw

	// W2 : ctx annulable PAR JOUEUR pour le REST poller (dérivé de rootCtx → le
	// cancel global le coupe aussi, mais on peut désormais le couper seul à la
	// suppression). Créé sous le lock, stocké, puis le spawn se fait hors lock.
	var pollerCtx context.Context
	if d.trackerRestClient != nil && d.rootCtx != nil {
		var pollerCancel context.CancelFunc
		pollerCtx, pollerCancel = context.WithCancel(d.rootCtx)
		d.playerCancels[key] = pollerCancel
	}
	d.playersMu.Unlock()

	slog.InfoContext(ctx, "watcher_daemon: joueur ajouté dynamiquement",
		"gamertag", p.Gamertag, "xuid", p.XUID)

	// Spawn REST poller pour ce joueur (utilise le client tracker partagé).
	// Skip si le daemon n'a pas encore été démarré (rootCtx nil) — le poller
	// sera créé par initPlayers au prochain Start si le joueur est encore là.
	if pollerCtx != nil {
		handler := d.makePresenceHandler(pollerCtx, pw)
		poller := NewRESTPoller(p.XUID, p.Gamertag, d.trackerRestClient, handler)
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			poller.Run(pollerCtx)
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
		pw := NewPlayerWatcher(p.Gamertag, p.XUID, d.cfg.MatchFetcher, &queueSyncTrigger{
			queue:    d.queue,
			gamertag: p.Gamertag,
			xuid:     p.XUID,
		})
		pw.SetTitleSlug(p.TitleSlug) // Phase 1.9 : titre configuré → ctx poller + write-path
		if d.cfg.LiveRefreshFactory != nil {
			pw = pw.WithLiveRefresh(d.cfg.LiveRefreshFactory(p.Gamertag, p.XUID, p.TitleSlug))
		}
		key := playerKey(p.Gamertag, p.TitleSlug)
		d.players[key] = pw

		slog.InfoContext(ctx, "watcher_daemon: joueur initialisé",
			"gamertag", p.Gamertag,
			"xuid", p.XUID,
			"title_slug", p.TitleSlug,
		)

		// REST poller par joueur — partage le client tracker (token JGtm).
		// Pas de doublon problématique avec AddUserClient (le poller dédié
		// JGtm a son propre client + son propre auth refresh) : les deux
		// pollers dispatch vers le même PlayerWatcher dont les transitions
		// FSM sont idempotentes.
		if d.trackerRestClient != nil {
			// W2 : ctx annulable par joueur (cf. AddPlayer).
			pollerCtx, pollerCancel := context.WithCancel(d.rootCtx)
			d.playerCancels[key] = pollerCancel
			handler := d.makePresenceHandler(pollerCtx, pw)
			poller := NewRESTPoller(p.XUID, p.Gamertag, d.trackerRestClient, handler)
			d.wg.Add(1)
			go func() {
				defer d.wg.Done()
				poller.Run(pollerCtx)
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

		// Témoin de vivacité : tout event reçu (même Offline sans titre) fait
		// avancer lastEventAt. Posé avant tout filtrage pour mesurer la santé
		// du flux REST/RTA, pas l'activité in-game du joueur.
		pw.RecordEvent(time.Now())

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
				// Titre tracké reconnu → mémorisé AVANT le test « titre du
				// watcher » ci-dessous. L'ordre est le fond du sujet : ce test
				// sort en OnPresenceInactive+return quand le joueur lance un
				// AUTRE titre tracké ; capter le titre après lui laisserait un
				// joueur configuré halo_5 jouant à Infinite « hors jeu » pour
				// l'UI de présence. Cf. godoc de PlayerWatcher.currentTitleSlug.
				pw.SetCurrentTitle(td.Slug, td.Name)
				// Multi-titre : ce watcher ne suit QUE pw.titleSlug. Si le joueur
				// lance un AUTRE titre tracké, ce n'est pas « son » jeu pour CE
				// watcher → inactif ici (le watcher du même gamertag sur ce titre,
				// s'il existe, prendra le relais). Évite de syncer dans le mauvais titre.
				expectedSlug := pw.titleSlug
				if expectedSlug == "" {
					expectedSlug = title.DefaultSlug
				}
				if td.Slug != expectedSlug {
					slog.InfoContext(evCtx, "watcher_daemon: titre tracké mais != titre du watcher → inactif",
						"gamertag", pw.gamertag, "watcher_title", expectedSlug,
						"detected_title", td.Slug, "event", evID)
					pw.OnPresenceInactive(evCtx)
					return
				}
				slog.InfoContext(evCtx, "watcher_daemon: présence détectée — titre tracké",
					"gamertag", pw.gamertag,
					"title", td.Name,
					"state", event.PresenceState,
					"event", evID,
				)
				pw.OnPresenceActive(evCtx)
				// Broadcast l'état Active aux autres PlayerWatcher si activé.
				// Cf. DaemonConfig.BroadcastPresenceActive godoc — incident
				// 2026-05-27 sessions de groupe non détectées. Scopé au MÊME titre.
				if d.cfg.BroadcastPresenceActive {
					d.broadcastPresenceActive(evCtx, pw.gamertag, expectedSlug)
				}
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
			pw.SetCurrentTitle("", "")
			pw.OnPresenceInactive(evCtx)
			return
		}

		// Pas de PresenceDetail (state Offline ou payload sans titre)
		slog.DebugContext(evCtx, "watcher_daemon: présence sans titre actif",
			"gamertag", pw.gamertag, "state", event.PresenceState, "event", evID)
		pw.SetCurrentTitle("", "")
		pw.OnPresenceInactive(evCtx)
	}
}

// broadcastPresenceActive propage l'état Active à tous les PlayerWatcher
// sauf celui qui a déclenché l'event. Utilisé pour les sessions de groupe
// où la présence Xbox ne signale pas tous les joueurs (cf. incident
// 2026-05-27 : Madina/Choco/XxDaemon jouent avec JGtm mais seul JGtm est
// vu in-game par le tracker → leurs MatchPoller ne tournent pas → leurs
// nouveaux matchs jamais détectés).
//
// L'appel est synchrone car `OnPresenceActive` est rapide (transition FSM
// + démarrage poller en goroutine). Idempotent : ne re-transitionne pas
// la FSM si déjà Watching, pas de cascade entre broadcasts simultanés.
//
// Le triggering player est exclu — il a déjà été activé par le handler.
func (d *Daemon) broadcastPresenceActive(ctx context.Context, triggeringGamertag, triggeringTitleSlug string) {
	d.playersMu.RLock()
	others := make([]*PlayerWatcher, 0, len(d.players))
	otherNames := make([]string, 0, len(d.players))
	for _, pw := range d.players {
		// Exclure le triggering player ET les watchers d'un AUTRE titre : la
		// détection de session de groupe est par-titre (un squad halo_infinite ne
		// doit pas réveiller un poller halo_5). La map est keyée composite, on
		// filtre donc sur les champs du watcher (pas la clé). Normalisation
		// vide→DefaultSlug des deux côtés (watchers sans titre explicite).
		pwSlug := pw.titleSlug
		if pwSlug == "" {
			pwSlug = title.DefaultSlug
		}
		if pw.gamertag == triggeringGamertag || pwSlug != triggeringTitleSlug {
			continue
		}
		others = append(others, pw)
		otherNames = append(otherNames, pw.gamertag)
	}
	d.playersMu.RUnlock()

	if len(others) == 0 {
		return
	}

	slog.InfoContext(ctx, "watcher_daemon: broadcast présence active",
		"triggered_by", triggeringGamertag,
		"broadcasted_to", otherNames,
	)

	for _, pw := range others {
		pw.OnPresenceActive(ctx)
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
				Gamertag:  req.Gamertag,
				XUID:      req.XUID,
				MatchIDs:  req.MatchIDs,
				TitleSlug: req.TitleSlug, // MT-11 / PMT-3 : "" → halo_infinite (seul titre suivi)
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

func (q *queueSyncTrigger) TriggerSync(ctx context.Context, gamertag, xuid string, matchIDs []string) error {
	q.queue.Enqueue(MatchRequest{
		Gamertag: gamertag,
		XUID:     xuid,
		MatchIDs: matchIDs,
		// Phase 1.9 : le ctx du poller porte le titre du joueur (startPoller) →
		// propagé au CoordinatorRequest (clé de dédup composite + ctx du moteur).
		// Vide ⇒ halo_infinite (byte-identique mono-titre).
		TitleSlug: ctxkeys.TitleSlug(ctx),
	})
	return nil
}
