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
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/pool"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
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
	// Trois formats console (LEVELUP_LOG_FORMAT) :
	//   - compact (défaut) : ConsoleHandler — `HH:MM:SS [INFO] sync.postSync: msg k=v`
	//     tronqué à LEVELUP_LOG_MAX_LINE (défaut 200), skip event_id/source.* pour
	//     limiter le bruit console (préservés dans logs/{module}.log JSON).
	//   - text             : slog.NewTextHandler natif (verbose, pre-sprint).
	//   - json             : slog.NewJSONHandler (prod).
	// Rétro-compat : LEVELUP_LOG_JSON=true → format=json.
	// Niveau via LEVELUP_LOG_LEVEL (défaut INFO).
	logLevelStr := strings.ToLower(os.Getenv("LEVELUP_LOG_LEVEL"))
	logLevel := slog.LevelInfo
	switch logLevelStr {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}

	preliminaryRepoRoot := os.Getenv("LEVELUP_REPO_ROOT") // peut être vide, sera résolu plus tard
	logsCfg := logging.LoadConfig(preliminaryRepoRoot)

	var logHandler slog.Handler
	switch logsCfg.ConsoleFormat {
	case "json":
		logHandler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})
	case "text":
		logHandler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})
	default: // "compact"
		logHandler = logging.NewConsoleHandler(os.Stderr, logging.ConsoleHandlerOptions{
			Level:    logLevel,
			MaxWidth: logsCfg.MaxLineWidth,
			Color:    logsCfg.ConsoleColor,
		})
	}

	// P6.4 : envelopper avec ContextHandler pour attacher automatiquement
	// request_id + event_id (depuis ctxkeys) à chaque slog.*Context(...) émis
	// par les services. Sans ça, debug prod cassé en multi-user.
	logHandler = observability.NewContextHandler(logHandler)

	// Sprint B1 commit 16 : MultiModuleHandler tri-stream — console (comportement
	// pré-sprint) + fichiers `logs/{module}.log` (un par module sync/provider/
	// pool/handlers/...). Activé par défaut, désactivable via
	// LEVELUP_LOGS_ENABLED=false. Dossier configurable via LEVELUP_LOGS_DIR.
	// Référençabilité cross-module : event_id propagé via ctx + ContextHandler.
	var multiHandler *logging.MultiModuleHandler
	if logsCfg.Enabled && logsCfg.LogsDir != "" {
		mh, err := logging.NewMultiModuleHandler(logHandler, logsCfg.LogsDir, logsCfg.FileLevel)
		if err != nil {
			slog.New(logHandler).Warn("logging: multi-module handler indisponible (console only)",
				"err", err, "logs_dir", logsCfg.LogsDir)
		} else {
			multiHandler = mh
			logHandler = mh
		}
	}
	slog.SetDefault(slog.New(logHandler))
	if multiHandler != nil {
		slog.Info("logging: multi-module actif",
			"logs_dir", logsCfg.LogsDir,
			"file_level", logsCfg.FileLevel.String(),
			"console_format", logsCfg.ConsoleFormat,
			"max_line", logsCfg.MaxLineWidth,
		)
	}

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

	// Architecture B-swap (sprint sharedprovider, ADR 0016) — ACTIVÉ PAR DÉFAUT
	// au commit 9.
	//
	// shared_matches_v2.duckdb est géré par un SharedDBProvider qui swap
	// dynamiquement RO ↔ RW autour des leases writer (sync engine). Les
	// handlers HTTP attendent la fenêtre de sync (timeout borné, mappé en
	// 503 Retry-After) — comparable à un read replica qui réplique.
	//
	// Élimine le bug historique "different configuration" qui plantait
	// auto_sync RunDelta quand le sync ouvrait shared en RW pendant qu'une
	// instance RO globale était ouverte ailleurs.
	//
	// LEVELUP_USE_SHARED_PROVIDER (kill-switch d'urgence) :
	//   - "0" : repli mode legacy LegacySharedReader(pdb.Shared) — UNIQUEMENT
	//     en cas de régression critique constatée en prod. Logger les
	//     compteurs `shared_provider_swap_failures_total` avant de basculer.
	//   - "1" / non défini (default) : Provider actif (recommandé).
	//
	// Note : `attachShared` (pool.go) reste en place pour résoudre les
	// queries cross-DB encore non-migrées (squad_repo : LoadTopTeammates,
	// LoadSquadMatches, LoadTeammateMatches, LoadSynthesisMatches,
	// LoadMapStatsForSquad). Le split+merge complet de ces 5 méthodes est
	// prévu pour un commit follow-up post-9.
	useSharedProvider := os.Getenv("LEVELUP_USE_SHARED_PROVIDER") != "0"

	var (
		sharedReader duckdb.SharedReader
		closeShared  func() error
	)
	var unsubscribeSwap func()
	if useSharedProvider {
		sharedMgr := sharedprovider.NewManager()
		provider, err := sharedMgr.For(sharedPath)
		if err != nil {
			slog.Error("ouverture shared_matches_v2 via provider échouée", "err", err)
			os.Exit(1)
		}
		sharedReader = provider

		// Commit 8g : brancher le Provider au pool joueur via Subscribe.
		// Sur PreSwap, le pool DETACH ses ATTACH RO sur shared (cf. POC
		// diagnostique commit 8f : DETACH explicite libère le file alors
		// que Reopen ne le fait pas). Sur RWToRO/ErrorToRO, le pool
		// re-ATTACH.
		//
		// L'adapter traduit sharedprovider.SwapEvent → duckdb.SwapDirection
		// (évite le cycle d'import duckdb ↔ sharedprovider).
		//
		// Sprint B1 commit 19 : ctx propagé depuis le caller du swap (sync
		// engine typiquement) — porte l'event_id pour traçabilité du
		// callback pool dans logs/duckdb.log.
		unsubscribeSwap = provider.Subscribe(func(ctx context.Context, evt sharedprovider.SwapEvent) {
			switch evt.Direction {
			case sharedprovider.DirectionPreSwapToRW:
				duckdb.OnSharedSwap(ctx, duckdb.SwapDirPreSwapToRW)
			case sharedprovider.DirectionRWToRO:
				duckdb.OnSharedSwap(ctx, duckdb.SwapDirRWToRO)
			case sharedprovider.DirectionErrorToRO:
				duckdb.OnSharedSwap(ctx, duckdb.SwapDirErrorToRO)
			}
		})

		// Injecter le Provider dans le AppConfig pour que les
		// PlayerPoolConfig créés ensuite (resolveDemoPlayer, buildPlayerPool
		// Config) reçoivent SharedReader → mode B-swap actif au niveau pool.
		// + le SyncEngine via WithSharedProvider (auto_sync.go, sync_handler.go).
		cfg.SharedProvider = provider

		closeShared = func() error {
			if unsubscribeSwap != nil {
				unsubscribeSwap()
			}
			return sharedMgr.Close()
		}
		slog.Info("shared_matches_v2: mode sharedprovider (B-swap actif, pool inscrit via Subscribe)")
	} else {
		sharedDB, err := duckdb.OpenReadOnly(sharedPath)
		if err != nil {
			slog.Error("ouverture shared_matches_v2 échouée", "err", err)
			os.Exit(1)
		}
		sharedReader = duckdb.LegacySharedReader(sharedDB)
		closeShared = sharedDB.Close
		slog.Info("shared_matches_v2: mode legacy (RO direct, trade-off différent-configuration accepté)")
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

	// --- 3.bis. Filet de garde corruption ART (Phase 1, plan stabilisation
	// 2026-05-22). Scanne shared_matches_v2 + metadata pour détecter les
	// tables dont l'index ART est corrompu (filter pushdown qui rate des
	// rows — cf. INCIDENT_2026-05-20_match_participants_index.md). Non-
	// bloquant : log WARN + métrique expvar si divergence, le serveur démarre.
	// Sample 5 par table ; suffisant pour démasquer le bug qui dépend du
	// contenu de la liste IN, pas d'une valeur unique.
	{
		bootCtx, bootCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer bootCancel()
		if sharedSQL, release, err := sharedReader.Get(bootCtx); err == nil {
			duckdb.BootARTGuard(bootCtx, sharedSQL, "shared", 5)
			release()
		} else {
			slog.WarnContext(bootCtx, "art_guard: shared reader indisponible pour probe boot",
				"err", err)
		}
		duckdb.BootARTGuard(bootCtx, metaDB.SQLDb(), "metadata", 5)
	}

	// --- 4. Repositories + services ---
	bootRepo := duckdb.NewBootstrapRepo(sharedReader, metaDB)
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

	// Discovery + Resolver + Pool : tous les appels API Halo passent par là.
	// - Discovery scanne env + sync_meta pour découvrir les credentials joueur.
	// - Resolver échange CredentialSource → ResolvedTokens (Spartan+Clearance)
	//   avec cache TTL ~3h30.
	// - Pool maintient les tokens vivants, gère 429/503 cooldown, et permet
	//   le round-robin (PolicyAnyPublic) ou pinned (PolicyPinnedPlayer).
	//
	// Le callback onRotated persiste le refresh_token rotaté par Microsoft
	// dans sync_meta.oauth_refresh_token de la player DB — sans ça, le
	// prochain refresh échouerait avec invalid_grant (Microsoft rotate
	// systématiquement le RT à chaque usage).
	autoSyncPool := buildAutoSyncPool(ctx, cfg, tokenProvider)
	if autoSyncPool != nil {
		defer autoSyncPool.Close()
		slog.Info("auto_sync: pool initialisé", "size", autoSyncPool.Size())
	} else {
		slog.Warn("auto_sync: pool non initialisé — aucun credential découvert",
			"hint", "vérifier SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG> dans .env.local",
		)
	}

	autoScheduler := scheduler.New(cfg, settingsStore, tokenProvider, autoSyncPool)
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
	var watcherDaemon *watcher.Daemon = startWatcherDaemon(ctx, cfg, settingsStore, tokenProvider, notifierGetter, tokenRefresher)
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
	// (UUIDs résiduels, bits menteurs, garbage URLs). Depuis 2026-05-20 les
	// compteurs sont uniquement loggés dans `logs/scheduler.log` (le diag
	// admin se fait via `cmd/diag_db_health` ou la lecture des logs) — plus
	// d'émission de notif `data_health_warning` (jargon dev sans intérêt
	// pour un end user lambda sur une app de stats).
	healthScheduler := scheduler.NewDataHealthScheduler(cfg.RepoRoot)
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
	router, reg = api.NewRouter(cfg, bootRepo, bootSvc, watcherCtrl, tokenProvider, autoScheduler)

	// app_release : émission asynchrone d'une notification in-app par joueur si la
	// version a changé depuis sync_meta.last_seen_app_version. Ne bloque pas le boot.
	go api.EmitAppReleaseForAllPlayers(context.Background(), cfg, reg, cfg.AppVersion)

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
	if err := closeShared(); err != nil {
		slog.Warn("fermeture shared DB", "err", err)
	}
	if err := metaDB.Close(); err != nil {
		slog.Warn("fermeture metadata DB", "err", err)
	}
	// Sprint B1 commit 16 : fermer proprement les fichiers logs/{module}.log
	// (flush + close des handles). Idempotent.
	if multiHandler != nil {
		if err := multiHandler.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "fermeture logging multi-module: %v\n", err)
		}
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

// buildAutoSyncPool construit le pool de tokens utilisé par l'AutoSyncScheduler.
// Pipeline :
//  1. Discovery scanne env + sync_meta DuckDB → []CredentialSource
//  2. Resolver échange ces sources en tokens Halo (cache TTL ~3h30) + callback
//     onRotated qui persiste le RT rotaté par Microsoft dans sync_meta de la
//     player DB. Sans cette persistance, le prochain refresh OAuth échouerait
//     avec invalid_grant (Microsoft rotate systématiquement le RT à chaque
//     usage pour des raisons de sécurité).
//  3. NewPool maintient les tokens vivants, gère cooldown 429/503, et expose
//     un Acquire round-robin + pinned.
//
// Retourne nil (et un log Warn) si aucun credential n'est trouvé — le
// scheduler tournera quand même mais tous les joueurs seront skipped avec
// raison explicite.
func buildAutoSyncPool(
	ctx context.Context,
	cfg *config.AppConfig,
	tokenProvider auth.TokenProvider,
) pool.Pool {
	pr := title.NewPathResolver(cfg.RepoRoot)
	discovery := pool.NewDiscovery(cfg, pr, title.DefaultSlug)
	sources, err := discovery.Scan(ctx)
	if err != nil {
		slog.Error("auto_sync: pool discovery échoué", "err", err)
		return nil
	}
	if len(sources) == 0 {
		return nil // log Warn fait par le caller
	}

	// Callback de persistance du RT rotaté : ouvre la player DB en
	// OpenReadWriteShared (partage l'instance du pool joueur) et UPSERT le
	// nouveau RT dans sync_meta. Best-effort : une erreur est loguée mais
	// n'interrompt pas le Resolve.
	onRotated := func(ctx context.Context, gamertag, newRT string) error {
		dbPath := pr.PlayerDBPath(title.DefaultSlug, gamertag)
		db, err := duckdb.OpenReadWriteShared(dbPath)
		if err != nil {
			return fmt.Errorf("open player db: %w", err)
		}
		defer db.Close() //nolint:errcheck // ref-count : best-effort
		return duckdb.WriteOAuthRefreshToken(ctx, db, newRT)
	}

	resolver := pool.NewResolver(tokenProvider, 0, onRotated) // 0 = default TTL ~3h30
	p, err := pool.NewPool(ctx, resolver, sources, pool.PoolOptions{
		MaxSize:     0, // 0 = tous les sources découverts
		PerTokenRPS: 5, // Option 2 audit 2026-05-21 : aligné sur le sync manuel
		//                 (validé bench-rps real-multi : 0 × 429 à 5N RPS)
	})
	if err != nil {
		slog.Error("auto_sync: pool creation échouée", "err", err)
		return nil
	}
	return p
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
	slog.Debug("watcher: tokens path", "path", tokenStorePath)
	store := auth.NewTokenStore(tokenStorePath)
	tokens, err := store.Load()
	if err != nil {
		slog.Info("watcher: pas de token store, daemon désactivé", "path", tokenStorePath)
		return nil
	}

	// PR 2.5b — Fallback : si pas de tokens legacy mono-user, scanner le store
	// multi-user (data/auth/watcher_tokens/{xuid}.json) pour trouver un user avec
	// un XSTS valide à utiliser comme tracker initial. Le 1er user trouvé devient
	// le tracker ; les autres seront subscribés via Daemon.AddPlayer (post-login
	// pour les nouveaux, ou au prochain boot pour les existants).
	if tokens.XSTSToken == "" {
		multiDir := title.NewPathResolver(cfg.RepoRoot).WatcherTokensDir()
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
			// Persister dans le legacy : permet aux prochains boots de retrouver
			// directement le tracker via le chemin habituel.
			if saveErr := store.Save(tokens); saveErr != nil {
				slog.Warn("watcher: persistance fallback tokens dans legacy échouée", "err", saveErr)
			}
			break
		}
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
	copy(playerSummaries, players)

	// Registre de titres
	titleReg := title.NewRegistry()

	// Sync trigger (in-process)
	syncTrigger := syncpkg.NewTrigger(cfg.RepoRoot, &staticTokenProvider{tokens: *tokens}, domain.SyncOptions{
		MatchType:         "matchmaking",
		MaxMatches:        25,
		WithParticipants:  true,
		WithMedals:        true,
		RequestsPerSecond: 5,
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

	// PR 2.5c — boot reload des userClients depuis MultiUserTokenStore.
	// Chaque user SSO Xbox déjà inscrit (a son XSTS persisté dans
	// data/auth/watcher_tokens/{xuid}.json) gagne sa propre connexion RTA dédiée
	// dès le démarrage du serveur. Indépendant du social graph Xbox.
	multiDir := title.NewPathResolver(cfg.RepoRoot).WatcherTokensDir()
	multiStore := auth.NewMultiUserTokenStore(multiDir)
	// Brancher le refresh on-demand par user (status=3 → MSAL cache → AcquireXSTSForRTA).
	daemon.WithPerUserAuthRefresh(func(refreshCtx context.Context, xuid string) (string, error) {
		return auth.RefreshUserXSTS(refreshCtx, multiStore, xuid)
	})
	allUsers, scanErr := multiStore.LoadAll()
	if scanErr != nil {
		slog.Warn("watcher: scan multi-user token store échoué", "err", scanErr)
	}
	for xuid, ut := range allUsers {
		if !ut.IsXSTSValid(0) {
			slog.Info("watcher: skip userClient avec XSTS expiré au boot",
				"xuid", xuid, "gamertag", ut.Gamertag, "xsts_expires_at", ut.XSTSExpiresAt)
			continue
		}
		if err := daemon.AddUserClient(ctx, ut); err != nil {
			slog.Warn("watcher: AddUserClient échoué au boot",
				"xuid", xuid, "gamertag", ut.Gamertag, "err", err)
			continue
		}
	}

	slog.Info("watcher: daemon démarré",
		"players", len(playerSummaries),
		"user_clients", len(allUsers),
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
