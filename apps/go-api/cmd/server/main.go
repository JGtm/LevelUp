// cmd/server — point d'entrée du backend Go LevelUp.
//
// Sprint 0 : POC DuckDB + HTTP.
// Sprint 4 : CORS, rate-limit, slog JSON, oapi-codegen types.
// Ce binaire démarre un serveur HTTP qui :
//   - ouvre metadata.duckdb et shared_matches_v2.duckdb en read-only
//   - expose GET /health (nb de matchs + version DuckDB)
//   - expose GET /api/v1/bootstrap (réponse structurée, parité Python cible)
//   - expose GET /api/v1/players
//
// Variables d'environnement :
//
//	LEVELUP_REPO_ROOT    — racine du repo (par défaut : auto-détection)
//	LEVELUP_API_PORT     — port d'écoute (défaut : 8000)
//	LEVELUP_DEMO_MODE    — "true" pour activer le mode démo
//	LEVELUP_LOG_JSON     — "true" pour JSON logging (prod), défaut: text (dev)
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"levelup/go-api/internal/api"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/halo"
	"levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/scheduler"
	"levelup/go-api/internal/service"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/watcher"
)

// version est injectée au build via -ldflags "-X main.version=X.Y.Z".
var version = "dev"

// buildTokenProvider instancie le TokenProvider selon app_settings.json:auth_provider.
// Par défaut (valeur vide ou "msal") : MSALProvider.
// Si "sisu" : SISUProvider (authentification native Xbox, sans app Azure).
func buildTokenProvider(settingsStore *settings.Store) auth.TokenProvider {
	s, err := settingsStore.Load()
	if err != nil {
		slog.Warn("buildTokenProvider: impossible de lire les settings, utilisation MSAL", "err", err)
		return auth.NewMSALProvider()
	}
	switch s.AuthProvider {
	case "sisu":
		slog.Info("buildTokenProvider: SISU provider activé")
		return auth.NewSISUProvider()
	default:
		if s.AuthProvider != "" && s.AuthProvider != "msal" {
			slog.Warn("buildTokenProvider: valeur auth_provider inconnue, utilisation MSAL", "value", s.AuthProvider)
		}
		slog.Info("buildTokenProvider: MSAL provider activé")
		return auth.NewMSALProvider()
	}
}

func main() {
	// --- 0. Health-check mode (Docker HEALTHCHECK) ---
	if len(os.Args) == 2 && os.Args[1] == "-health-check" {
		hcClient := &http.Client{Timeout: 5 * time.Second}
		hcDo := func(url string) (*http.Response, error) {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
			if err != nil {
				return nil, err
			}
			return hcClient.Do(req) //nolint:bodyclose // body fermé par le caller via defer resp.Body.Close()
		}
		port := os.Getenv("LEVELUP_API_PORT_OR_DEFAULT")
		resp, err := hcDo("http://127.0.0.1:" + port + "/health") //nolint:bodyclose // body fermé via defer resp.Body.Close()
		if err != nil {
			// Fallback sur le port par défaut 8000
			resp, err = hcDo("http://127.0.0.1:8000/health") //nolint:bodyclose // body fermé via defer resp.Body.Close()
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "health-check failed:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "health-check status: %d\n", resp.StatusCode)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// --- 1. Logging structuré ---
	// En production (LEVELUP_LOG_JSON=true) : JSON. En dev : texte lisible.
	// Niveau par défaut : INFO. Passer LEVELUP_LOG_LEVEL=debug pour activer les logs HTTP 2xx.
	logJSON := strings.ToLower(os.Getenv("LEVELUP_LOG_JSON")) == "true"
	logLevelStr := strings.ToLower(os.Getenv("LEVELUP_LOG_LEVEL"))
	logLevel := slog.LevelInfo
	if logLevelStr == "debug" {
		logLevel = slog.LevelDebug
	} else if logLevelStr == "warn" {
		logLevel = slog.LevelWarn
	} else if logLevelStr == "error" {
		logLevel = slog.LevelError
	}
	var logHandler slog.Handler
	if logJSON {
		logHandler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})
	} else {
		logHandler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})
	}
	// P6.4 : envelopper avec ContextHandler pour attacher automatiquement
	// request_id (depuis ctxkeys) à chaque slog.*Context(...) émis par les
	// services. Sans ça, debug prod cassé en multi-user.
	logHandler = observability.NewContextHandler(logHandler)
	slog.SetDefault(slog.New(logHandler))

	// --- 2. Configuration ---
	cfg, err := config.Load()
	if err != nil {
		slog.Error("chargement config", "err", err)
		os.Exit(1)
	}
	// Injecter la version buildée (ldflags) si LEVELUP_APP_VERSION non défini.
	if cfg.AppVersion == "dev" && version != "dev" {
		cfg.AppVersion = version
	}
	slog.Debug("config chargée",
		"repo_root", cfg.RepoRoot,
		"demo_mode", cfg.DemoMode,
		"version", cfg.AppVersion,
		"addr", cfg.ServerAddr(),
	)

	// --- 3. Connexions DuckDB ---
	pr := title.NewPathResolver(cfg.RepoRoot)
	titleSlug := title.DefaultSlug
	sharedPath := pr.SharedDBPath(titleSlug)
	metaPath := pr.MetadataDBPath(titleSlug)
	sharedSocialPath := pr.SharedSocialDBPath(titleSlug)

	// En DEMO_MODE, utiliser les fixtures de démo si les DBs prod n'existent pas.
	if cfg.DemoMode {
		demoPaths := []struct{ name, path *string }{
			{name: strPtr("shared"), path: &sharedPath},
			{name: strPtr("metadata"), path: &metaPath},
		}
		for _, dp := range demoPaths {
			if _, err := os.Stat(*dp.path); os.IsNotExist(err) {
				demo := filepath.Join(cfg.DemoFixturesDir, "warehouse", filepath.Base(*dp.path))
				if _, err2 := os.Stat(demo); err2 == nil {
					*dp.path = demo
					slog.Info("demo_mode: utilisation fixture", "db", *dp.name, "path", demo)
				}
			}
		}

		// Validation au boot — warn si la fixture player démo est absente.
		// Sans ce warning, l'erreur n'apparaît qu'à la première requête joueur
		// avec un message IO Error opaque ("cannot open file..."), et le
		// frontend affiche "DemoPlayer" comme un fantôme avec erreurs en cascade.
		// Fix après incident 2026-04-29 — process API laissé tournant en demo
		// mode sans fixture installée.
		demoStats := filepath.Join(cfg.DemoFixturesDir, "stats.duckdb")
		if _, err := os.Stat(demoStats); os.IsNotExist(err) {
			slog.Warn(
				"demo_mode: fixture joueur absente — les requêtes /pages/* échoueront avec un IO Error",
				"path", demoStats,
				"hint", "désactiver LEVELUP_DEMO_MODE ou installer la fixture dans LEVELUP_DEMO_FIXTURES_DIR",
			)
		}
	}

	slog.Debug("ouverture DuckDB", "shared", sharedPath, "metadata", metaPath, "shared_social", sharedSocialPath)

	// --- 3a. Migrations (read-write, avant l'ouverture des connexions runtime) ---
	titleConfigDir := filepath.Join(cfg.RepoRoot, "config", "titles", titleSlug)
	if err := runMigrations(metaPath, sharedPath, sharedSocialPath, pr.SharedPVEDBPath(titleSlug), titleConfigDir); err != nil {
		slog.Debug("migrations ignorées (DB verrouillée), démarrage sans migration")
	} else {
		slog.Debug("migrations appliquées")
	}

	sharedDB, err := duckdb.OpenReadOnly(sharedPath)
	if err != nil {
		slog.Error("ouverture shared_matches_v2 échouée", "err", err)
		os.Exit(1)
	}
	// Retry sur metadata : hot-reload peut créer une fenêtre où l'ancien processus
	// n'a pas encore libéré le verrou DuckDB (write-ahead lock). Sur Windows, après un
	// SIGKILL d'air, l'OS peut mettre plusieurs secondes à libérer les HANDLEs DuckDB
	// orphelins ; 6×500 ms = 3 s ne suffisait pas dans certains cas observés.
	// IMPORTANT : OpenReadWriteShared (et non OpenReadOnly) pour partager la même
	// instance DuckDB que le pool joueur (pool.go) et le DuckDBIndexStore (assets).
	// Sinon DuckDB rejette toute deuxième connexion sur le même fichier avec
	// "Can't open a connection to same database file with a different configuration".
	const metaOpenAttempts = 12
	const metaOpenDelay = 500 * time.Millisecond
	var metaDB *duckdb.DB
	for attempt := range metaOpenAttempts {
		metaDB, err = duckdb.OpenReadWriteShared(metaPath)
		if err == nil {
			break
		}
		if attempt == metaOpenAttempts-1 {
			slog.Error("ouverture metadata échouée", "attempts", metaOpenAttempts, "err", err)
			os.Exit(1)
		}
		slog.Warn("metadata verrouillée, nouvelle tentative...", "attempt", attempt+1, "max", metaOpenAttempts, "err", err)
		time.Sleep(metaOpenDelay)
	}
	slog.Debug("DuckDB ouvert")

	// --- 4. Repositories + services ---
	bootRepo := duckdb.NewBootstrapRepo(sharedDB, metaDB)
	bootSvc := service.NewBootstrapService(cfg, bootRepo).
		WithPrivacyProvider(halo.DefaultHaloProvider)

	// Auth locale : injecter le check "first launch" si mode password.
	if cfg.AuthMode == "password" {
		usersPath := filepath.Join(cfg.AuthDir, "users.json")
		us := userstore.NewStore(usersPath)
		bootSvc = bootSvc.WithUserStoreEmpty(us.IsEmpty)
	}

	// --- 5. Sprint 0 : validation des types critiques ---
	ctx := context.Background()
	if err := bootRepo.ValidateTypes(ctx); err != nil {
		slog.Error("validation types DuckDB échouée", "err", err)
		os.Exit(1)
	}
	if _, err := bootRepo.GetCareerRanksSample(ctx); err != nil {
		slog.Warn("lecture career_ranks échouée", "err", err)
	}

	// --- 6. Scheduler, watcher, puis routeur HTTP ---
	settingsStore := settings.NewStore(cfg.AppSettingsPath)
	tokenProvider := buildTokenProvider(settingsStore)
	autoScheduler := scheduler.New(cfg, settingsStore, tokenProvider)
	schedulerCtx, cancelScheduler := context.WithCancel(ctx)

	// Watcher daemon (présence Xbox RTA + Steam) — démarré avant le scheduler pour câbler
	// l'ActivityChecker : quand un joueur est en état Watching/Syncing/Cooling, le scheduler
	// cède son tick pour ce joueur et évite deux syncs concurrentes sur la même stats.duckdb.
	//
	// Le notifierGetter est un getter lazy : il référence reg via une closure afin que le
	// LiveRefreshFactory puisse lier le SessionNotifier quand il est appelé (après le démarrage).
	var reg *api.ServiceRegistry
	notifierGetter := func(xuid string) port.SessionNotifier {
		if reg != nil {
			return reg.GetSessionNotifier(xuid)
		}
		return nil
	}
	// tokenRefresher est un getter lazy qui tente un refresh MSAL/OAuth v2 depuis la DB
	// quand le cache process de tokens est expiré (~50 min). Évite que le watcher
	// cesse de rafraîchir le BP après une longue session sans appel HTTP de l'UI.
	tokenRefresher := func(ctx context.Context, xuid string) (*domain.HaloTokens, error) {
		if reg != nil {
			return reg.RefreshTokensForXUID(ctx, xuid)
		}
		return nil, fmt.Errorf("registry non initialisé")
	}
	var watcherDaemon *watcher.Daemon
	watcherDaemon = startWatcherDaemon(ctx, cfg, settingsStore, tokenProvider, notifierGetter, tokenRefresher)
	if watcherDaemon != nil {
		autoScheduler.ActivityChecker = watcher.NewStateProvider(watcherDaemon)
	}

	// schedulerWG track la goroutine du scheduler pour que le shutdown attende
	// qu'elle retourne (sur ctx.Done()) avant duckdb.CloseAll(). Sans ce wait,
	// un cycle RunOnce en cours peut encore toucher metaDB après la fermeture.
	var schedulerWG sync.WaitGroup
	schedulerWG.Add(1)
	go func() {
		defer schedulerWG.Done()
		autoScheduler.Run(schedulerCtx)
	}()

	// 2026-05-08 — Data health scheduler : audit périodique multi-DB
	// (UUIDs résiduels, bits menteurs, orphelins, garbage URLs). Émet une
	// notification admin via `data_health_warning` quand des warnings sont
	// détectés. L'emitter est wiré après NewRouter (cf. plus bas) car le
	// notifications.Service est par-joueur et créé via le ServiceRegistry.
	healthScheduler := scheduler.NewDataHealthScheduler(cfg.RepoRoot, nil)
	schedulerWG.Add(1)
	go func() {
		defer schedulerWG.Done()
		healthScheduler.Run(schedulerCtx)
	}()

	// Convertir en interface (nil safe : un *Daemon nil ne doit pas devenir une interface non-nil)
	var watcherCtrl watcher.DaemonController
	if watcherDaemon != nil {
		watcherCtrl = watcherDaemon
	}

	// Routeur HTTP — le daemon peut être nil si le watcher est désactivé.
	// reg est assigné ici : la closure notifierGetter y accède de manière lazy (joueur actif après démarrage).
	var router http.Handler
	router, reg = api.NewRouter(cfg, bootRepo, bootSvc, watcherCtrl, tokenProvider)

	// app_release : émission asynchrone d'une notification in-app par joueur si la
	// version a changé depuis sync_meta.last_seen_app_version. Ne bloque pas le boot.
	go api.EmitAppReleaseForAllPlayers(context.Background(), cfg, reg, cfg.AppVersion)

	// data_health : wirer l'emitter sur le 1er joueur configuré (cible admin
	// par défaut en setup mono-utilisateur). Si aucun joueur en
	// db_profiles.json, le scheduler tourne sans émission de notif (les
	// warnings sont uniquement dans les logs).
	if reg != nil {
		if players, err := cfg.LoadPlayers(title.DefaultSlug); err == nil && len(players) > 0 {
			adminSlug := cfg.AdminPlayer()
			target := players[0]
			for _, p := range players {
				if p.PlayerSlug == adminSlug {
					target = p
					break
				}
			}
			if emitter, err := reg.NotificationsEmitter(ctx, target.PlayerSlug); err == nil {
				healthScheduler.SetEmitter(emitter)
				slog.Info("data_health: emitter wiré", "admin_player", target.PlayerSlug)
			} else {
				slog.Warn("data_health: emitter non câblé", "err", err)
			}
		}
	}

	srv := &http.Server{
		Addr:         cfg.ServerAddr(),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// --- 7. Démarrage + graceful shutdown ---
	// On bind le port en premier pour détecter immédiatement un conflit.
	ln, err := net.Listen("tcp", cfg.ServerAddr())
	if err != nil {
		fmt.Fprintf(os.Stderr, "  [ERR] Port %s deja occupe -- fermez l'ancien processus\n", cfg.ServerAddr())
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	fmt.Fprintf(os.Stderr, "\n  [OK] LevelUp API ready -> http://%s\n\n", cfg.ServerAddr())
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-sigCh
	fmt.Fprint(os.Stderr, "\n  [..] Arret en cours...")

	cancelScheduler()

	if watcherDaemon != nil {
		watcherDaemon.Stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown", "err", err)
	}

	// Attendre la goroutine du scheduler avec un timeout dur pour que les
	// connexions DuckDB qu'elle pourrait tenir soient libérées avant CloseAll.
	schedulerDone := make(chan struct{})
	go func() {
		schedulerWG.Wait()
		close(schedulerDone)
	}()
	select {
	case <-schedulerDone:
		slog.Debug("scheduler terminé")
	case <-time.After(3 * time.Second):
		slog.Warn("scheduler: timeout sur Wait — RunOnce probablement en cours")
	}

	duckdb.CloseAll()
	if err := sharedDB.Close(); err != nil {
		slog.Warn("fermeture shared DB", "err", err)
	}
	if err := metaDB.Close(); err != nil {
		slog.Warn("fermeture metadata DB", "err", err)
	}
	fmt.Fprintln(os.Stderr, " terminé.")
}

func strPtr(s string) *string { return &s }

// runMigrations applique les migrations DuckDB dans l'ordre :
// metadata → shared → shared_pve → shared_social.
// Les migrations player sont gérées à l'ouverture de chaque player DB.
//
// prestige ConfigDir : chemin vers config/titles/{slug}/ pour le seed du catalogue
// Prestige (challenge_template + preset_arc). Si vide, le seed est ignoré.
func runMigrations(metaPath, sharedPath, sharedSocialPath, pvePath, prestigeConfigDir string) error {
	// Ensure all step init() have been registered (side-effect imports).
	_ = migration.All()

	// Seed Prestige catalogue via migration backfill (une seule fois, idempotent).
	if prestigeConfigDir != "" {
		migration.RegisterPrestigeSeedMigration(prestigeConfigDir)
	}

	// 1. metadata.duckdb
	metaDB, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		return fmt.Errorf("open metadata rw: %w", err)
	}
	if err := migration.RunForDB(metaDB.SQLDb(), migration.TargetMetadata); err != nil {
		metaDB.Close()
		return fmt.Errorf("metadata migrations: %w", err)
	}
	metaDB.Close()

	// 2. shared_matches_v2.duckdb
	sharedDB, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		return fmt.Errorf("open shared rw: %w", err)
	}
	if err := migration.RunForDB(sharedDB.SQLDb(), migration.TargetShared); err != nil {
		sharedDB.Close()
		return fmt.Errorf("shared migrations: %w", err)
	}
	sharedDB.Close()

	// 3. shared_pve.duckdb (optionnel, fichier peut ne pas exister)
	if _, err := os.Stat(pvePath); err == nil {
		pveDB, err := duckdb.OpenReadWrite(pvePath)
		if err != nil {
			return fmt.Errorf("open shared_pve rw: %w", err)
		}
		if err := migration.RunForDB(pveDB.SQLDb(), migration.TargetSharedPvE); err != nil {
			pveDB.Close()
			return fmt.Errorf("shared_pve migrations: %w", err)
		}
		pveDB.Close()
	}

	// 4. shared_social.duckdb (créé automatiquement s'il n'existe pas)
	socialDB, err := duckdb.OpenReadWrite(sharedSocialPath)
	if err != nil {
		return fmt.Errorf("open shared_social rw: %w", err)
	}
	if err := migration.RunForDB(socialDB.SQLDb(), migration.TargetSharedSocial); err != nil {
		socialDB.Close()
		return fmt.Errorf("shared_social migrations: %w", err)
	}
	socialDB.Close()

	return nil
}

// RunPlayerMigrations applique les migrations player pour une DB individuelle.
// Appelé lors de l'ouverture d'une player DB.
func RunPlayerMigrations(playerDBPath string) error {
	db, err := duckdb.OpenReadWrite(playerDBPath)
	if err != nil {
		return fmt.Errorf("open player rw: %w", err)
	}
	defer db.Close()
	return migration.RunForDB(db.SQLDb(), migration.TargetPlayer)
}

// startWatcherDaemon tente de démarrer le daemon de présence.
// Retourne nil si watcher_presence_enabled est false ou si les prérequis ne sont pas remplis.
// getNotifier est un getter lazy (xuid → SessionNotifier) injecté par main ; peut être nil.
// tokenProvider sert à régénérer l'access_token Microsoft via OAuth v2 refresh
// (depuis watcher_tokens.json ou SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG>), pour
// que le watcher puisse aussi rafraîchir son XSTS RTA tout seul sans avoir
// besoin que l'utilisateur regénère manuellement les tokens.
func startWatcherDaemon(
	ctx context.Context,
	cfg *config.AppConfig,
	settingsStore *settings.Store,
	tokenProvider auth.TokenProvider,
	getNotifier func(xuid string) port.SessionNotifier,
	tokenRefresher func(ctx context.Context, xuid string) (*domain.HaloTokens, error),
) *watcher.Daemon {
	// Vérifier que le watcher est activé dans les settings
	appSettings, err := settingsStore.Load()
	if err != nil {
		slog.Info("watcher: lecture settings échouée, daemon désactivé", "err", err)
		return nil
	}
	if !appSettings.WatcherPresenceEnabled {
		slog.Info("watcher: watcher_presence_enabled=false, daemon désactivé")
		return nil
	}

	tokenStorePath := title.NewPathResolver(cfg.RepoRoot).WatcherTokensPath()
	slog.Info("watcher: tokens path", "path", tokenStorePath)
	store := auth.NewTokenStore(tokenStorePath)
	tokens, err := store.Load()
	if err != nil {
		slog.Info("watcher: pas de token store, daemon désactivé", "path", tokenStorePath)
		return nil
	}

	// 1) S'assurer qu'on a un access_token Microsoft frais.
	//    EnsureWatcherAccessToken réutilise l'access_token courant s'il est
	//    valide, sinon tente un OAuth v2 refresh depuis (a) tokens.RefreshToken
	//    ou (b) SPNKR_OAUTH_REFRESH_TOKEN_<XSTSGamertag> (.env.local).
	//    Persiste le nouveau access_token dans watcher_tokens.json.
	freshAccessToken, err := auth.EnsureWatcherAccessToken(ctx, store, tokenProvider, tokens.XSTSGamertag)
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
		if freshResult, xerr := auth.AcquireXSTSForRTA(ctx, freshAccessToken); xerr == nil {
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

	if !tokens.IsXSTSValid(0) {
		slog.Warn("watcher: tokens XSTS expirés et refresh impossible, daemon désactivé",
			"hint", "vérifier SPNKR_OAUTH_REFRESH_TOKEN_"+strings.ToUpper(tokens.XSTSGamertag)+" dans .env.local",
		)
		return nil
	}

	// Charger les joueurs depuis db_profiles.json
	players, err := cfg.LoadPlayers()
	if err != nil || len(players) == 0 {
		slog.Info("watcher: aucun joueur configuré, daemon désactivé")
		return nil
	}

	// Convertir en domain.PlayerSummary
	playerSummaries := make([]domain.PlayerSummary, len(players))
	for i, p := range players {
		playerSummaries[i] = p
	}

	// Registre de titres
	titleReg := title.NewRegistry()

	// Sync trigger (in-process)
	syncTrigger := syncpkg.NewTrigger(cfg.RepoRoot, &staticTokenProvider{tokens: *tokens}, domain.SyncOptions{
		MatchType:         "matchmaking",
		MaxMatches:        25,
		WithParticipants:  true,
		WithMedals:        true,
		RequestsPerSecond: 1,
	})

	// daemon est déclaré ici pour permettre à la closure RefreshRTAAuth d'y référer
	// avant que NewDaemon retourne (pattern forward-reference via pointeur).
	var daemon *watcher.Daemon
	watcherPR := title.NewPathResolver(cfg.RepoRoot)
	watcherSlug := title.DefaultSlug
	daemon = watcher.NewDaemon(watcher.DaemonConfig{
		RepoRoot:        cfg.RepoRoot,
		SteamAPIKey:     os.Getenv("STEAM_API_KEY"),
		MaxParallelSync: 2,
		LiveRefreshFactory: func(gamertag, xuid string) watcher.LiveRefreshTrigger {
			wMetaPath := watcherPR.MetadataDBPath(watcherSlug)
			wPlayerPath := watcherPR.PlayerDBPath(watcherSlug, gamertag)
			sink := duckdb.NewPersistSink(wMetaPath, wPlayerPath, xuid)
			// resolver nil : le watcher ne pré-chauffe pas les définitions BP.
			// Les définitions sont chargées à la demande via l'endpoint HTTP (resolver HTTP).
			refresher := watcher.NewPlayerLiveRefresher(gamertag, xuid, sink, nil).
				WithTokenRefresher(tokenRefresher)
			if getNotifier != nil {
				if n := getNotifier(xuid); n != nil {
					refresher = refresher.WithSessionNotifier(n)
				}
			}
			return refresher
		},
		// RefreshRTAAuth est appelé on-demand par RunWithReconnect quand status=3 est reçu.
		// Tente d'abord d'obtenir un access_token frais via OAuth v2 refresh (env
		// var SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG> ou tokens.RefreshToken), puis
		// acquiert un XSTS RTA frais. Pousse le nouveau header dans le daemon.
		RefreshRTAAuth: func(ctx context.Context) error {
			currentTokens, err := store.Load()
			if err != nil {
				return fmt.Errorf("refresh RTA auth: lecture token store: %w", err)
			}
			accessToken, eatErr := auth.EnsureWatcherAccessToken(ctx, store, tokenProvider, currentTokens.XSTSGamertag)
			if eatErr != nil {
				slog.WarnContext(ctx, "refresh RTA auth: EnsureWatcherAccessToken erreur structurelle", "err", eatErr)
			}
			if accessToken == "" {
				accessToken = currentTokens.AccessToken
			}
			if accessToken == "" {
				return fmt.Errorf("refresh RTA auth: aucun access_token disponible (refresh_token absent ou révoqué)")
			}
			result, err := auth.AcquireXSTSForRTA(ctx, accessToken)
			if err != nil {
				return fmt.Errorf("refresh RTA auth: AcquireXSTSForRTA: %w", err)
			}
			if storeErr := store.UpdateXSTS(result, 55*time.Minute); storeErr != nil {
				slog.WarnContext(ctx, "watcher: refresh RTA auth: impossible de persister XSTS", "err", storeErr)
			}
			daemon.UpdateAuth(result.AuthHeader())
			slog.InfoContext(ctx, "watcher: refresh RTA auth on-demand OK",
				"gamertag", result.Gamertag,
				"not_after", result.NotAfter,
			)
			return nil
		},
	}, titleReg, syncTrigger)

	// Le refresh XSTS proactif a déjà été fait plus haut dans la fonction (avant
	// le check IsXSTSValid). `tokens` reflète ici le state à jour : si un refresh
	// a réussi, tokens.XSTSToken / XSTSUserHash sont déjà les valeurs fraîches
	// persistées dans watcher_tokens.json.
	xstsResult := &auth.XSTSResult{
		Token:    tokens.XSTSToken,
		UserHash: tokens.XSTSUserHash,
	}

	daemon.Start(ctx, xstsResult.AuthHeader(), playerSummaries)

	// Refresh loop : met à jour les tokens XSTS et le daemon
	refreshLoop := auth.NewRefreshLoop(store, func(result *auth.XSTSResult) {
		daemon.UpdateAuth(result.AuthHeader())
	})
	go refreshLoop.Run(ctx)

	slog.Info("watcher: daemon démarré",
		"players", len(playerSummaries),
		"rta_auth", "ok",
	)
	return daemon
}

// staticTokenProvider fournit les tokens Halo depuis le token store.
type staticTokenProvider struct {
	tokens auth.StoredTokens
}

func (s *staticTokenProvider) GetTokens(_ context.Context) (*domain.HaloTokens, error) {
	return &domain.HaloTokens{
		SpartanToken:   s.tokens.AccessToken,
		ClearanceToken: "",
	}, nil
}
