// boot_watcher.go — démarrage du daemon de présence Xbox (watcher) : activation, jetons du
// tracker, déclencheur de sync, récupérateur de matchs, porte « profil suivi », boucle de
// rafraîchissement XSTS.

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_5/livesync"
	"levelup/go-api/internal/notify"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/pool"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/scheduler"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/watcher"
)

// startWatcher démarre le daemon de présence (sauf en démo) et lui branche le scheduler.
func startWatcher(st *bootState) {
	ctx, cfg, boot := st.ctx, st.cfg, st.boot
	settingsStore, tokenProvider, autoSyncPool, autoScheduler := st.settingsStore, st.tokenProvider, st.autoSyncPool, st.autoScheduler

	// Watcher daemon (présence Xbox RTA) — démarré avant le scheduler pour câbler
	// l'ActivityChecker : quand un joueur est en état Watching/Syncing/Cooling, le scheduler
	// cède son tick pour ce joueur et évite deux syncs concurrentes sur la même stats.duckdb.
	//
	// Le notifierGetter est un getter lazy : il référence reg via une closure afin que le
	// LiveRefreshFactory puisse lier le SessionNotifier quand il est appelé (après le démarrage).
	notifierGetter := func(xuid string) port.SessionNotifier {
		if st.reg != nil {
			return st.reg.GetSessionNotifier(xuid)
		}
		return nil
	}
	// tokenRefresher est un getter lazy qui tente un refresh OAuth v2 depuis le
	// MultiUserTokenStore (ADR 0023) quand le cache process de tokens est expiré
	// (~50 min). Évite que le watcher cesse de rafraîchir le BP après une longue
	// session sans appel HTTP de l'UI.
	tokenRefresher := func(ctx context.Context, xuid string) (*domain.HaloTokens, error) {
		if st.reg != nil {
			return st.reg.RefreshTokensForXUID(ctx, xuid)
		}
		return nil, fmt.Errorf("registry non initialisé")
	}
	var watcherDaemon *watcher.Daemon
	if !boot.isCut(stepWatcher) {
		watcherDaemon = startWatcherDaemon(ctx, cfg, watcherDeps{
			settingsStore:  settingsStore,
			tokenProvider:  tokenProvider,
			getNotifier:    notifierGetter,
			tokenRefresher: tokenRefresher,
			haloPool:       autoSyncPool,
			autoScheduler:  autoScheduler,
			loopCtx:        st.schedulerCtx,
			loopWG:         &st.schedulerWG,
		})
	}
	if watcherDaemon != nil {
		autoScheduler.ActivityChecker = watcher.NewStateProvider(watcherDaemon)
		// Dédup cross-source (unification 2026-06-02) : le Coordinator du watcher
		// devient le point de dédup partagé. L'auto-sync l'interroge avant chaque
		// RunDelta (cf. syncPlayer) et NewRouter le propage au handler HTTP via
		// autoScheduler.Gate(). Si le watcher est désactivé, SyncGate reste le
		// NopSyncGate par défaut → comportement legacy (lease seul rempart).
		autoScheduler.SyncGate = watcherDaemon.SyncGate()
	}
	st.watcherDaemon = watcherDaemon
}

// watcherDeps réunit ce que main fournit au démarrage du daemon de présence.
type watcherDeps struct {
	settingsStore *settings.Store
	// tokenProvider sert à régénérer l'access_token Microsoft via OAuth v2 refresh
	// (refresh_token du MultiUserTokenStore, source unique ADR 0023).
	tokenProvider auth.TokenProvider
	// getNotifier est un getter lazy (xuid -> SessionNotifier) ; peut être nil.
	getNotifier    func(xuid string) port.SessionNotifier
	tokenRefresher func(ctx context.Context, xuid string) (*domain.HaloTokens, error)
	haloPool       pool.Pool
	// autoScheduler fournit BuildEngine pour le câblage syncTrigger.
	// Source of truth UNIQUE du wiring engine (cf. trigger.go godoc + ADR
	// incident 2026-05-26). Doit être non-nil en production.
	autoScheduler *scheduler.AutoSyncScheduler
	// loopCtx est le contexte des boucles de fond que le daemon laisse derrière lui (schedulerCtx,
	// annulé au début de l'arrêt) ; loopWG les suit : la phase scheduler de l'arrêt les attend, donc
	// un jeton obtenu avant l'annulation est écrit avant la fermeture des bases.
	loopCtx context.Context
	loopWG  *sync.WaitGroup
}

// startWatcherDaemon tente de démarrer le daemon de présence.
// Retourne nil si watcher_presence_enabled est false ou si les prérequis ne sont pas remplis.
// getNotifier est un getter lazy (xuid → SessionNotifier) injecté par main ; peut être nil.
// tokenProvider sert à régénérer l'access_token Microsoft via OAuth v2 refresh
// (refresh_token du MultiUserTokenStore, source unique ADR 0023), pour que le
// watcher puisse aussi rafraîchir son XSTS RTA tout seul sans avoir besoin que
// l'utilisateur regénère manuellement les tokens.
func startWatcherDaemon(ctx context.Context, cfg *config.AppConfig, deps watcherDeps) *watcher.Daemon {
	if !watcherPresenceEnabled(deps.settingsStore) {
		return nil
	}
	store, tokens, ok := loadWatcherTokens(cfg)
	if !ok {
		return nil
	}
	tokens = refreshWatcherTokens(ctx, cfg, deps.tokenProvider, store, tokens)

	if !tokens.IsXSTSValid(0) {
		slog.Warn("watcher: tokens XSTS expirés et refresh impossible, daemon désactivé",
			"gamertag", tokens.XSTSGamertag,
			"hint", "re-authentifier le compte (SSO Xbox) ou `go run ./cmd/token-capture/ <GT>`",
		)
		return nil
	}

	playerSummaries, ok := watchedPlayerSummaries(cfg)
	if !ok {
		return nil
	}

	// Registre de titres PARTAGÉ (MT-16 : scheduler/watcher voient les titres
	// additionnels config → sync écrit dans leurs DB isolées via PMT-3).
	titleReg := title.DefaultRegistry()

	syncTrigger := newWatcherSyncTrigger(cfg, deps, tokens)
	matchFetcher := newWatcherMatchFetcher(deps.haloPool)
	daemon := newWatcherDaemon(cfg, deps, titleReg, matchFetcher, syncTrigger)

	// Le refresh XSTS proactif a déjà été fait plus haut dans la fonction (avant
	// le check IsXSTSValid). `tokens` reflète ici le state à jour : si un refresh
	// a réussi, tokens.XSTSToken / XSTSUserHash sont déjà les valeurs fraîches
	// persistées dans watcher_tokens.json.
	xstsResult := &auth.XSTSResult{
		Token:    tokens.XSTSToken,
		UserHash: tokens.XSTSUserHash,
	}

	daemon.Start(ctx, xstsResult.AuthHeader(), playerSummaries)

	runWatcherRefreshLoop(deps, cfg, daemon, store)

	// Cleanup 2026-05-26 : la boucle de boot reload des userClients (PR 2.5c)
	// est supprimée. Les joueurs sont surveillés via REST poll partagé créé
	// dans daemon.Start. Le MultiUserTokenStore reste utilisé par le SSO Xbox
	// pour persister les tokens des users connectés (cf. xbox_auth_service).

	slog.Info("watcher: daemon démarré",
		"players", len(playerSummaries),
		"presence_source", "rest_poll",
	)
	return daemon
}

// watcherPresenceEnabled dit si les réglages activent le daemon de présence.
func watcherPresenceEnabled(settingsStore *settings.Store) bool {
	// Vérifier que le watcher est activé dans les settings
	appSettings, err := settingsStore.Load()
	if err != nil {
		slog.Info("watcher: lecture settings échouée, daemon désactivé", "err", err)
		return false
	}
	if !appSettings.WatcherPresenceEnabled {
		slog.Info("watcher: watcher_presence_enabled=false, daemon désactivé")
		return false
	}
	return true
}

// loadWatcherTokens charge l'état du tracker (mono-user) ; s'il est vide, adopte un compte du
// store multi-user dont le XSTS est valide.
func loadWatcherTokens(cfg *config.AppConfig) (*auth.TokenStore, *auth.StoredTokens, bool) {
	tokenStorePath := cfg.PathResolver().WatcherTokensPath()
	slog.Debug("watcher: tokens path", "path", tokenStorePath)
	store := auth.NewTokenStore(tokenStorePath)
	tokens, err := store.Load()
	if err != nil {
		slog.Info("watcher: pas de token store, daemon désactivé", "path", tokenStorePath, "err", err)
		return nil, nil, false
	}

	// PR 2.5b — Fallback : si l'état watcher mono-user est vide, scanner le store
	// multi-user (data/auth/watcher_tokens/{xuid}.json) pour trouver un user avec
	// un XSTS valide à utiliser comme tracker initial. Le 1er user trouvé devient
	// le tracker ; les autres seront subscribés via Daemon.AddPlayer (post-login
	// pour les nouveaux, ou au prochain boot pour les existants).
	if tokens.XSTSToken == "" {
		multiDir := cfg.PathResolver().WatcherTokensDir()
		multi := auth.NewMultiUserTokenStore(multiDir)
		all, scanErr := multi.LoadAll()
		if scanErr != nil {
			slog.Warn("watcher: scan multi-user token store échoué", "err", scanErr)
		}
		for xuid, ut := range all {
			if !ut.IsXSTSValid(0) {
				continue
			}
			slog.Info("watcher: fallback sur tokens multi-user pour le tracker initial",
				"xuid", xuid, "gamertag", ut.Gamertag, "xsts_expires_at", ut.XSTSExpiresAt)
			tokens = &auth.StoredTokens{
				XSTSToken:      ut.XSTSToken,
				XSTSUserHash:   ut.XSTSUserHash,
				XSTSGamertag:   ut.Gamertag,
				XSTSXUID:       ut.XUID,
				XSTSExpiresAt:  ut.XSTSExpiresAt,
				AccessToken:    ut.AccessToken,
				OAuthExpiresAt: ut.OAuthExpiresAt,
			}
			// Persister dans l'état watcher mono-user : permet aux prochains boots
			// de retrouver directement le tracker via le chemin habituel.
			if saveErr := store.Save(tokens); saveErr != nil {
				slog.Warn("watcher: persistance fallback tokens dans legacy échouée", "err", saveErr)
			}
			break
		}
	}
	return store, tokens, true
}

// refreshWatcherTokens rafraîchit l'access_token puis le XSTS RTA du tracker avant le test de
// validité ; rend l'état rechargé si le refresh a abouti.
func refreshWatcherTokens(ctx context.Context, cfg *config.AppConfig, tokenProvider auth.TokenProvider,
	store *auth.TokenStore, tokens *auth.StoredTokens) *auth.StoredTokens {
	// 1) S'assurer qu'on a un access_token Microsoft frais.
	//    EnsureWatcherAccessToken réutilise l'access_token courant s'il est
	//    valide, sinon tente un OAuth v2 refresh depuis le MultiUserTokenStore
	//    (source unique ADR 0023) et persiste la rotation dans ce store.
	multiStore := auth.NewMultiUserTokenStore(cfg.PathResolver().WatcherTokensDir())
	freshAccessToken, err := auth.EnsureWatcherAccessToken(ctx, multiStore, store, tokenProvider, tokens.XSTSGamertag)
	if err != nil {
		slog.Warn("watcher: EnsureWatcherAccessToken erreur structurelle", "err", err)
	}
	if freshAccessToken == "" {
		freshAccessToken = tokens.AccessToken // mode dégradé
	}

	// 2) Si on a un access_token frais, on tente un refresh XSTS RTA proactif
	//    AVANT le check IsXSTSValid : ça permet de récupérer un watcher sain
	//    même si watcher_tokens.json contient un XSTS périmé tant qu'on a
	//    encore un refresh_token utilisable côté env var.
	if freshAccessToken != "" {
		slog.Info("watcher: refresh XSTS proactif avant démarrage…")
		// Provenance du RpsTicket posée ET mesurée : ce chemin est l'un des 401
		// « RpsTicket refusé » du boot (le compte du tracker n'accepte pas « d= »).
		if freshResult, xerr := auth.AcquireXSTSForRTAWithProvenance(
			ctx, multiStore, tokens.XSTSXUID, freshAccessToken); xerr == nil {
			if storeErr := store.UpdateXSTS(freshResult, 55*time.Minute); storeErr == nil {
				slog.Info("watcher: XSTS frais obtenu",
					"gamertag", freshResult.Gamertag,
					"not_after", freshResult.NotAfter,
				)
				// Recharger pour la suite (IsXSTSValid et xstsResult plus bas).
				if reloaded, lerr := store.Load(); lerr == nil {
					tokens = reloaded
				}
			} else {
				slog.Warn("watcher: persistance XSTS frais échouée", "err", storeErr)
			}
		} else {
			slog.Warn("watcher: refresh XSTS proactif échoué, utilisation du token stocké", "err", xerr)
		}
	}
	return tokens
}

// watchedPlayerSummaries rend les couples (joueur, titre) à surveiller ; false si aucun.
func watchedPlayerSummaries(cfg *config.AppConfig) ([]domain.PlayerSummary, bool) {
	// Charger les joueurs depuis db_profiles.json
	players, err := cfg.LoadPlayers()
	if err != nil || len(players) == 0 {
		slog.Info("watcher: aucun joueur configuré, daemon désactivé")
		return nil, false
	}

	// Exclure les couples (joueur, titre) en pause (sync_enabled=false) : pas de
	// tracking live. Les données restent sur disque, réactivables via les réglages.
	playerSummaries := domain.SyncablePlayers(players)
	if len(playerSummaries) == 0 {
		slog.Info("watcher: tous les joueurs en pause, daemon désactivé")
		return nil, false
	}
	return playerSummaries, true
}

// newWatcherSyncTrigger construit le déclencheur de sync in-process du watcher.
func newWatcherSyncTrigger(cfg *config.AppConfig, deps watcherDeps, tokens *auth.StoredTokens) *syncpkg.Trigger {
	autoScheduler, haloPool := deps.autoScheduler, deps.haloPool

	// Sync trigger (in-process). On câble explicitement l'engineFactory sur
	// scheduler.BuildEngine — c'est ce qui garantit la parité runtime entre
	// le path watcher (Coordinator → Trigger.RunSync) et le path scheduler
	// (auto_sync.defaultRunnerFactory). Sans ça, le SyncEngine créé par le
	// watcher tombe en mode legacy : pas de SharedProvider → conflit
	// "different configuration" sur shared_matches_v2.duckdb (incident
	// 2026-05-26 23:05+).
	syncTrigger := syncpkg.NewTrigger(cfg.RepoRoot, &staticTokenProvider{tokens: *tokens}, watcherSyncOptions()).
		WithEngineFactory(autoScheduler.BuildEngine).
		// Titres live-only (Halo 5+) : router le watcher vers leur pipeline dédié,
		// auth pinnée depuis le pool auto-sync. Sans ça, un joueur h5 détecté en
		// présence ferait fetcher des matchs Infinite dans le store h5 (corruption).
		// handled=false → le slug n'est pas live → path engine (Infinite) inchangé.
		WithLiveRunnerFactory(func(fctx context.Context, slug, gamertag, xuid string) (syncpkg.LiveTitleRunner, context.Context, func(), bool, error) {
			if !livesync.HandlesTitle(slug) {
				return nil, fctx, func() {}, false, nil
			}
			r, runCtx, release, err := livesync.AcquireRunner(fctx, haloPool, cfg, slug, gamertag, xuid)
			if err != nil {
				return nil, fctx, func() {}, true, err
			}
			return r, runCtx, release, true, nil
		})
	return syncTrigger
}

// newWatcherMatchFetcher branche le polling des matchs sur le pool de tokens (nil sans pool).
func newWatcherMatchFetcher(haloPool pool.Pool) watcher.MatchFetcher {
	// MatchFetcher pour le polling Halo API (/hi/players/xuid(N)/matches) du
	// MatchPoller. Réutilise le pool de tokens auto-sync : endpoint public
	// (PolicyAnyPublic), quota par-token Microsoft → partage le rate-limit
	// avec le scheduler évite les 429 inutiles. Si le pool est absent (mode
	// dégradé sans credential), le fetcher reste nil et le garde-fou dans
	// PlayerWatcher.startPoller logge un Warn-once sans paniquer.
	var matchFetcher watcher.MatchFetcher
	if haloPool != nil {
		pooled := syncpkg.NewPooledHaloClient(haloPool, 5)
		matchFetcher = watcher.NewHaloMatchFetcher(pooled)
		slog.Info("watcher: MatchFetcher branché sur le pool auto-sync")
	} else {
		slog.Warn("watcher: pool auto-sync absent — MatchPoller désactivé (mode dégradé)")
	}
	return matchFetcher
}

// newWatcherDaemon crée le daemon de présence et lui pose la porte « profil suivi ».
func newWatcherDaemon(cfg *config.AppConfig, deps watcherDeps, titleReg *title.Registry,
	matchFetcher watcher.MatchFetcher, syncTrigger *syncpkg.Trigger) *watcher.Daemon {
	getNotifier, tokenRefresher := deps.getNotifier, deps.tokenRefresher

	// daemon est déclaré ici pour permettre à la closure RefreshRTAAuth d'y référer
	// avant que NewDaemon retourne (pattern forward-reference via pointeur).
	var daemon *watcher.Daemon
	watcherPR := cfg.PathResolver()
	daemon = watcher.NewDaemon(watcher.DaemonConfig{
		RepoRoot:        cfg.RepoRoot,
		MaxParallelSync: 2,
		MatchFetcher:    matchFetcher,
		// Broadcast présence active : quand un joueur passe in-game (titre
		// tracké), tous les autres PlayerWatcher passent aussi en Watching.
		// Évite que les sessions de groupe soient invisibles côté tracker
		// (incident 2026-05-27 — cf. ensure_enrichment_rows.go + thought_log).
		// Désactivable via LEVELUP_WATCHER_BROADCAST=0.
		BroadcastPresenceActive: os.Getenv("LEVELUP_WATCHER_BROADCAST") != "0",
		LiveRefreshFactory: func(gamertag, xuid, titleSlug string) watcher.LiveRefreshTrigger {
			if titleSlug == "" {
				titleSlug = title.DefaultSlug
			}
			wMetaPath := watcherPR.MetadataDBPath(titleSlug)
			wPlayerPath := watcherPR.PlayerDBPath(titleSlug, gamertag)
			sink := duckdb.NewPersistSink(wMetaPath, wPlayerPath, xuid, titleSlug)
			// resolver nil : le watcher ne pré-chauffe pas les définitions BP.
			// Les définitions sont chargées à la demande via l'endpoint HTTP (resolver HTTP).
			// WithTitleSlug : gate capability des surfaces live-service (BP/Challenges).
			// Un titre qui ne les expose pas (Halo 5) ⇒ ticker no-op non démarré,
			// plus de sondes economy/decks 404 toutes les 5 min.
			refresher := watcher.NewPlayerLiveRefresher(gamertag, xuid, sink, nil).
				WithTitleSlug(titleSlug).
				WithTokenRefresher(tokenRefresher)
			if getNotifier != nil {
				if n := getNotifier(xuid); n != nil {
					refresher = refresher.WithSessionNotifier(n)
				}
			}
			return refresher
		},
	}, titleReg, syncTrigger)

	// Porte « profil suivi » (ADR 0035 D3) : un couple (titre, xuid) qui n'est pas
	// déclaré dans db_profiles.json — ou qui y est en pause / auth_only — n'entre
	// ni dans le tracking live (Daemon.AddPlayer) ni dans le sync
	// (Coordinator.Submit ; le daemon possède le coordinateur, WithProfileGate
	// pose les deux). Erreur de lecture de db_profiles.json ⇒ on REFUSE, après
	// l'avoir journalisée : on ne synchronise pas « dans le doute » (c'est
	// exactement comme ça qu'un compte inconnu a écrit 25 matchs le 2026-07-23).
	profileGate := func(ctx context.Context, titleSlug, xuid string) bool {
		ok, err := cfg.HasTrackedProfile(titleSlug, xuid)
		if err != nil {
			slog.ErrorContext(ctx, "profil suivi : lecture de db_profiles.json échouée — requête refusée",
				"err", err, "title_slug", titleSlug, "xuid", xuid)
			return false
		}
		return ok
	}
	daemon.WithProfileGate(profileGate)
	return daemon
}

// runWatcherRefreshLoop lance la boucle qui rafraîchit les tokens XSTS du tracker et du daemon,
// sous le contexte d'arrêt des boucles de fond (deps.loopCtx) et suivie par leur groupe d'attente :
// ses échanges Microsoft s'interrompent à l'annulation, et un jeton déjà obtenu est écrit avant que
// l'arrêt ne ferme les bases (les écritures de jeton ne dépendent pas du contexte).
func runWatcherRefreshLoop(deps watcherDeps, cfg *config.AppConfig, daemon *watcher.Daemon, store *auth.TokenStore) {
	// Refresh loop : met à jour les tokens XSTS et le daemon.
	// WithMultiUserMirror est OBLIGATOIRE depuis ADR 0023 Phase 5 (2026-08-25) :
	// le MultiUserTokenStore est à la fois la SOURCE du refresh_token du tracker
	// et la destination miroir de ses écritures XSTS/access_token. Sans lui, la
	// boucle n'a plus aucune credential et skippe silencieusement (log Debug).
	multiMirror := auth.NewMultiUserTokenStore(cfg.PathResolver().WatcherTokensDir())
	refreshLoop := auth.NewRefreshLoop(store, func(result *auth.XSTSResult) {
		daemon.UpdateAuth(result.AuthHeader())
	}).WithMultiUserMirror(multiMirror).
		// PR-B : ping Discord opt-in quand le refresh_token meurt (reconnexion requise).
		WithReauthNotify(func(_, gamertag string) {
			notify.NotifyReauthRequired(notify.LoadNotifyConfig(cfg.AppSettingsPath), gamertag)
		})
	deps.loopWG.Add(1)
	go func() {
		defer deps.loopWG.Done()
		refreshLoop.Run(deps.loopCtx)
	}()
}
