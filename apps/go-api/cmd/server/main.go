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
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
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
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/persist"
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

	// --- 0.5 Charger .env.local AVANT la lecture des LEVELUP_LOG_* ---
	// Sinon le logger boote toujours en INFO/compact même si .env.local force
	// LEVELUP_LOG_LEVEL=warn (l'appel à config.Load qui charge .env.local
	// arrive bien plus tard dans main, après le setup logging).
	config.BootstrapEnvLocal()

	// --- 0.6 Wirer la factory SocialPersister (ADR 0020 Phase 5) ---
	// Permet à duckdb.openPlayerDB d'instancier un SharedSocialPersister
	// sans cycle d'import (duckdb -> persist serait cyclique car persist
	// -> duckdb via combined_persister.go). Le hook factory est lu à chaque
	// openPlayerDB ; si nil les writes shared_social retombent en legacy.
	duckdb.SocialPersisterFactory = func(db *sql.DB) duckdb.SocialPersister {
		return persist.NewSharedSocialPersister(db)
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

	// Crash output : redirige les panics Go runtime + fatal errors vers un
	// fichier dédié (sinon ils partent sur stderr qui n'est capturé nulle part
	// sous air/Windows). Sans ça : crash silencieux, pas de stack trace, diag
	// impossible — symptôme de l'incident 2026-05-22 (post-sync JGtm tué à
	// 18:41:19 sans aucune trace). Best-effort : si l'ouverture échoue, on
	// continue (le crash ira sur stderr comme avant).
	//
	// Phase 4.2 (plan stabilisation 2026-05-22) : on garde aussi un handle sur
	// le crashFile pour brancher un signal handler SIGABRT/SIGSEGV — utile
	// pour capturer les FatalException C++ de DuckDB (terminate() raise
	// SIGABRT côté libc Linux) avant que le process ne meure silencieusement.
	var crashFile *os.File
	if logsCfg.LogsDir != "" {
		crashLogPath := filepath.Join(logsCfg.LogsDir, "server.crash.log")
		if f, err := os.OpenFile(crashLogPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644); err == nil {
			_, _ = fmt.Fprintf(f, "\n=== server start %s pid=%d ===\n", time.Now().Format(time.RFC3339), os.Getpid())
			if setErr := debug.SetCrashOutput(f, debug.CrashOptions{}); setErr != nil {
				slog.Warn("crash_output: SetCrashOutput échoué", "err", setErr, "path", crashLogPath)
				_ = f.Close()
			} else {
				crashFile = f
			}
		} else {
			slog.Warn("crash_output: ouverture fichier échouée — panics resteront sur stderr",
				"err", err, "path", crashLogPath)
		}
	}

	// Phase 4.2 — signal handler SIGABRT (+ SIGSEGV) pour capturer les
	// FatalException C++ levées par DuckDB depuis cgo. `recover()` Go ne capte
	// pas ces erreurs car elles bypass la stack Go via libc terminate(). Le
	// handler dump la stack de toutes les goroutines puis os.Exit(2) — sinon
	// abort() ré-émet le signal après notre handler et le process meurt
	// silencieusement.
	//
	// Note Windows : signal.Notify(SIGABRT) compile mais ne fire pas — sur
	// Windows les erreurs C++ font des structured exceptions (SEH), pas des
	// signaux POSIX. Code defensive cross-platform : on l'enregistre quand
	// même, sur Linux/macOS ça fonctionne, sur Windows c'est un no-op.
	if crashFile != nil {
		installFatalSignalHandler(crashFile)
	}

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

	// --- 3.ter. Résolution CSR season depuis metadata.duckdb ---
	// Si LEVELUP_CSR_SEASON_ID n'est pas défini (ni env, ni app_settings.json),
	// requête csr_season_calendars pour la saison active du jour. Doit être fait
	// AVANT l'initialisation des services qui utilisent cfg.CurrentCSRSeasonID.
	if cfg.CurrentCSRSeasonID == "" {
		if id := resolveCSRSeasonFromDB(metaDB.SQLDb()); id != "" {
			cfg.CurrentCSRSeasonID = id
			slog.Info("csr_season_id résolu depuis csr_season_calendars", "season_id", id)
		} else {
			slog.Warn("csr_season_id absent de csr_season_calendars — sync CSR désactivé")
		}
	}

	// --- 3.bis. Filet de garde corruption ART (Phase 1, plan stabilisation
	// 2026-05-22). Scanne shared_matches_v2 + metadata pour détecter les
	// tables dont l'index ART est corrompu (filter pushdown qui rate des
	// rows — cf. INCIDENT_2026-05-20_match_participants_index.md). Non-
	// bloquant : log WARN + métrique expvar si divergence, le serveur démarre.
	// Sample 5 par table ; suffisant pour démasquer le bug qui dépend du
	// contenu de la liste IN, pas d'une valeur unique.
	//
	// Phase 4.1 (2026-05-23) : si match_participants montre une divergence
	// sur shared, auto-heal via swap CTAS (RebuildMatchParticipantsART).
	// Requiert le SharedProvider (mode B-swap) pour AcquireWriter; en mode
	// legacy on log seulement.
	{
		// Phase 5 cleanup (2026-05-24) : auto-heal supprimé — Phase 4 batch
		// INSERT-only path élimine la corruption ART à la racine. On garde la
		// DÉTECTION (BootARTGuard) pour alerte ops, mais le rebuild auto au
		// boot est remplacé par l'outil CLI manuel `force_rebuild_art --all true`
		// (voir runbook Phase 4.5). Cf. ADR 0019.
		bootCtx, bootCancel := context.WithTimeout(context.Background(), 60*time.Second)
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

	// Phase 4.9 (2026-05-24) : BatchQueue serveur-wide default ON. Set
	// LEVELUP_PERSIST_BATCH_ASYNC=0 pour fallback path synchrone (validé Phase 4.5).
	//
	// Path async : queue.Submit + worker + WAL durable (recovery au boot via
	// RecoverPending). Bénéfice : décorrélation sync/persist + résilience crash
	// mid-persist (le batch est journalisé sur disque AVANT le push channel).
	// Cycle de vie : créée 1× au boot, drainée + fermée à shutdown via
	// autoBatchQueue.Drain() + Close() AVANT duckdb.CloseAll() (ordre critique).
	var autoBatchQueue *persist.BatchQueue
	var workerWG sync.WaitGroup // tracks the batch Worker goroutine lifecycle
	if os.Getenv("LEVELUP_PERSIST_BATCH_ASYNC") != "0" {
		walDir := pr.WALDir()
		q, qErr := persist.NewBatchQueue(persist.BatchQueueConfig{
			WALDir:      walDir,
			ChanBufSize: 1000, // cf. Q2 ADR 0019 — backpressure naturelle
		})
		if qErr != nil {
			slog.WarnContext(ctx, "persist: BatchQueue init échoué — fallback path synchrone",
				"err", qErr)
		} else {
			autoBatchQueue = q
			autoScheduler.WithBatchQueue(q)
			slog.InfoContext(ctx, "persist: BatchQueue activée (async path)",
				"wal_dir", walDir)

			// Câblage Worker — CombinedPersister écrit shared + player par batch.
			// context.Background() : le Worker doit finir le batch en cours avant
			// de s'arrêter → ne doit pas être annulé par cancelScheduler().
			// Arrêt naturel via autoBatchQueue.Close() (channel close) au shutdown.
			combinedP := persist.NewCombinedPersister(
				func(workerCtx context.Context) (*sql.DB, func(), error) {
					return syncpkg.AcquireSharedWriterStandalone(workerCtx, cfg.SharedProvider, sharedPath)
				},
				func(gamertag string) string { return pr.PlayerDBPath(titleSlug, gamertag) },
			)
			batchWorker := persist.NewWorker("combined", q, persist.TargetShared, combinedP)
			workerWG.Add(1)
			go func() {
				defer workerWG.Done()
				_ = batchWorker.Run(context.Background())
			}()
			slog.InfoContext(ctx, "persist: Worker combiné démarré")

			// Recovery boot : re-soumet les batches WAL pending d'un crash précédent.
			// Appelé APRES le démarrage du Worker pour traitement immédiat.
			if rerr := q.RecoverPending(); rerr != nil {
				slog.WarnContext(ctx, "persist: RecoverPending échoué (non-bloquant)",
					"err", rerr)
			}
		}
	}

	// Phase 4.7 closure : janitor périodique (1× / 24h) qui purge :
	//   - data/sync_cache/sync.RunDelta_* > 7 jours (fetch cache éphémère)
	//   - data/wal/*.json ACKés > 7 jours (résidus si BatchQueue active)
	// Best-effort, non-bloquant sur erreur.
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		runJanitor := func() {
			cacheRoot := pr.SyncCacheDir()
			if n, err := syncpkg.PurgeOldFetchCache(cacheRoot, 7*24*time.Hour); err != nil {
				slog.WarnContext(schedulerCtx, "janitor: PurgeOldFetchCache échoué (non-bloquant)",
					"err", err)
			} else if n > 0 {
				slog.InfoContext(schedulerCtx, "janitor: fetch_cache purgé",
					"dirs_removed", n)
			}
			if autoBatchQueue != nil {
				if n, err := autoBatchQueue.PurgeOldWAL(7 * 24 * time.Hour); err != nil {
					slog.WarnContext(schedulerCtx, "janitor: PurgeOldWAL échoué (non-bloquant)",
						"err", err)
				} else if n > 0 {
					slog.InfoContext(schedulerCtx, "janitor: WAL purgé",
						"files_removed", n)
				}
			}
		}
		// Run once at boot (in case of stale data from previous run).
		runJanitor()
		for {
			select {
			case <-schedulerCtx.Done():
				return
			case <-ticker.C:
				runJanitor()
			}
		}
	}()

	// Phase 4.9 / PLAN_AUTH E.v2 (2026-05-24) : periodic Discovery re-scan
	// pour hot-add nouveaux tokens (env vars ajoutées, watcher_tokens.json mis
	// à jour) sans reboot. Skip si pool nil (aucun credential au boot).
	if autoSyncPool != nil {
		go func() {
			rescanTicker := time.NewTicker(15 * time.Minute)
			defer rescanTicker.Stop()
			runRescan := func() {
				resolver := title.NewPathResolver(cfg.RepoRoot)
				multiUserStore := auth.NewMultiUserTokenStore(resolver.WatcherTokensDir())
				legacyStore := auth.NewTokenStore(resolver.WatcherTokensPath())
				discovery := pool.NewDiscoveryWithStores(cfg, resolver, title.DefaultSlug, multiUserStore, legacyStore)
				sources, err := discovery.Scan(schedulerCtx)
				if err != nil {
					slog.WarnContext(schedulerCtx, "pool: re-scan échoué (non-bloquant)",
						"err", err)
					return
				}
				added := 0
				for _, src := range sources {
					if err := autoSyncPool.AddOrUpdateSource(schedulerCtx, src); err != nil {
						slog.DebugContext(schedulerCtx, "pool: re-scan AddOrUpdateSource skip",
							"gamertag", src.Gamertag, "err", err)
						continue
					}
					added++
				}
				if added > 0 {
					slog.InfoContext(schedulerCtx, "pool: re-scan terminé",
						"sources_scanned", len(sources), "sources_processed", added, "pool_size", autoSyncPool.Size())
				}
			}
			for {
				select {
				case <-schedulerCtx.Done():
					return
				case <-rescanTicker.C:
					runRescan()
				}
			}
		}()
	}

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

	// Backup périodique DuckDB via restic (pkg/duckdbbackup).
	// Créé inconditionnellement pour exposer le statut dans l'UI settings.
	// Run() est appelé seulement si backup_enabled=true dans app_settings.json.
	backupSched := ops.NewLevelUpBackupScheduler(cfg.Backup, pr)
	if cfg.Backup.Enabled {
		schedulerWG.Add(1)
		go func() {
			defer schedulerWG.Done()
			backupSched.Run(schedulerCtx)
		}()
	} else {
		slog.Debug("backup: désactivé — scheduler créé mais non démarré")
	}

	// Convertir en interface (nil safe : un *Daemon nil ne doit pas devenir une interface non-nil)
	var watcherCtrl watcher.DaemonController
	if watcherDaemon != nil {
		watcherCtrl = watcherDaemon
	}

	// Routeur HTTP — le daemon peut être nil si le watcher est désactivé.
	// reg est assigné ici : la closure notifierGetter y accède de manière lazy (joueur actif après démarrage).
	var router http.Handler
	router, reg = api.NewRouter(cfg, bootRepo, bootSvc, watcherCtrl, tokenProvider, autoScheduler, backupSched)

	// Phase 4 plan stabilisation 2026-05-22 — câblage post-sync runner sur
	// l'auto-sync scheduler. Avant ce fix, l'auto-sync court-circuitait
	// systématiquement le pipeline progression V2 (streaks/records/milestones/
	// notifications delta). Maintenant les 3 entry points (HTTP, auto-sync,
	// CLI futur) invoquent le MÊME runner via SyncEngine.WithPostSyncRunner.
	// Cf. AUDIT_ASCENSION_PIPELINE_DISCONNECTED_2026-05-21 §4 cause B.
	if postSyncRunner := api.NewPostSyncRunner(reg); postSyncRunner != nil {
		autoScheduler.WithPostSyncRunner(postSyncRunner)
	}

	// ADR 0020 D6.5 — câblage pipeline V2 (dormant tant que
	// LEVELUP_SYNC_PIPELINE=v2 n'est pas positionné). En l'absence d'env
	// var, scheduler.shouldUseV2() retourne false et le flow runtime reste
	// 100% V1 — aucun changement de comportement.
	//
	// Pré-requis : autoSyncPool non-nil + autoBatchQueue non-nil. Si l'un
	// manque, on skip le câblage : V2 sera indisponible mais V1 fonctionne.
	if autoSyncPool != nil && autoBatchQueue != nil && metaDB != nil {
		// Récupère le handle shared via le cache duckdb process-wide
		// (déjà ouvert plus tôt par le serveur en RO ou RW selon mode).
		var sharedSQLDB *sql.DB
		if cached, ok := duckdb.LookupCachedDB(sharedPath); ok {
			sharedSQLDB = cached.SQLDb()
		}
		// T1 — parity-complete : passe TOUTES les dépendances que
		// defaultRunnerFactory utilise pour câbler la SyncEngine V1.
		// Garantit que V2 a EXACTEMENT le même runtime que V1 sur le
		// post-sync (sessions, achievements, progression V2, media scan).
		v2PostSyncRunner := api.NewPostSyncRunner(reg)
		v2Orch := buildSyncV2Orchestrator(SyncV2WiringDeps{
			Cfg:            cfg,
			PathResolver:   pr,
			TitleSlug:      titleSlug,
			TokenPool:      autoSyncPool,
			BatchQueue:     autoBatchQueue,
			MetaDB:         metaDB.SQLDb(),
			SharedDB:       sharedSQLDB,
			TokenProvider:  tokenProvider,
			Settings:       settingsStore,
			PostSyncRunner: v2PostSyncRunner,
		})
		if v2Orch != nil {
			autoScheduler.WithCycleOrchestrator(v2Orch)
			slog.Info("sync.v2: orchestrator câblé parity-complete (activation via LEVELUP_SYNC_PIPELINE=v2)")
		}
	}

	// PLAN_V2 Phase 8 (2026-05-26) : SpartanCustomizationCron tourne toutes
	// les 8h (DefaultSpartanCustomizationInterval) pour rafraîchir la
	// customisation Spartan de TOUS les joueurs configurés, indépendamment
	// de l'usage UI. Réutilise CareerLiveService.GetSpartanIdentity (même
	// path que la visite home) → kickoffBackgroundRefresh → persistPartial
	// field-aware. Garantit qu'un joueur qui n'ouvre jamais l'app a quand
	// même sa customisation populée en DB.
	if autoSyncPool != nil && reg != nil {
		// Provider qui adapte la signature ServiceRegistry.CareerLiveCtx vers
		// celle attendue par le cron (retourne uniquement le SpartanIdentityFetcher).
		provider := func(ctx context.Context, slug string) (scheduler.SpartanIdentityFetcher, error) {
			svc, _, err := reg.CareerLiveCtx(ctx, slug)
			if err != nil {
				return nil, err
			}
			return svc, nil
		}
		spartanCron := scheduler.NewSpartanCustomizationCron(
			cfg, autoSyncPool, provider, titleSlug, 0,
		)
		schedulerWG.Add(1)
		go func() {
			defer schedulerWG.Done()
			spartanCron.Run(schedulerCtx)
		}()
		slog.InfoContext(ctx, "spartan_cron: scheduled",
			"interval", scheduler.DefaultSpartanCustomizationInterval)
	}

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

	// Heartbeat 30s — sentinelle de vie du process. Si les logs cessent de
	// montrer "alive" mais que le binaire est toujours en mémoire → deadlock
	// ou hang. Si les logs s'arrêtent ET le process disparaît → crash (cf.
	// logs/server.crash.log + recover() dans post-sync). Diagnostic immédiat
	// au prochain incident type 2026-05-22 (silence total post 18:41:19).
	startedAt := time.Now()
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-schedulerCtx.Done():
				return
			case <-ticker.C:
				slog.InfoContext(schedulerCtx, "heartbeat: alive",
					"uptime_s", int(time.Since(startedAt).Seconds()),
					"goroutines", runtime.NumGoroutine(),
				)
			}
		}
	}()

	<-sigCh
	fmt.Fprint(os.Stderr, "\n  [..] Arret en cours...")

	cancelScheduler()

	if watcherDaemon != nil {
		watcherDaemon.Stop()
	}

	// Phase 2 plan stabilisation 2026-05-22 : mesurer la durée totale du
	// shutdown pour valider que Air SIGKILL (stop_timeout=20s) n'est jamais
	// atteint en pratique. Si shutdown_total_duration_ms > 15000, augmenter
	// le timeout Air OU identifier l'étape qui traîne.
	shutdownStart := time.Now()

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

	// Phase 4.7 closure : drain + close BatchQueue avant CloseAll DuckDB.
	// Drain attend les persists en cours ; Close stoppe les workers.
	// Ordre critique : BatchQueue avant duckdb.CloseAll sinon les workers
	// tentent d'écrire sur des DBs fermées.
	if autoBatchQueue != nil {
		drainCtx, drainCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := autoBatchQueue.Drain(drainCtx); err != nil {
			slog.WarnContext(drainCtx, "persist: BatchQueue.Drain timeout (non-bloquant)",
				"err", err)
		}
		drainCancel()
		if err := autoBatchQueue.Close(); err != nil {
			slog.Warn("persist: BatchQueue.Close échoué (non-bloquant)", "err", err)
		}
	}
	workerWG.Wait() // attend la terminaison du Worker après channel close

	duckdb.CloseAll()

	// Phase 2 plan stabilisation 2026-05-22 : fermer le ServiceRegistry AVANT
	// metaDB.Close() pour décrémenter proprement le refCount sur metadata
	// porté par PrestigeBundle (cf. INCIDENT_2026-05-21_metadata_duckdb_lock
	// _air_hot_reload.md §3.2 leak refCount). Sans ça, refCount=2 au moment
	// du metaDB.Close() → décrément à 1 → handle Windows tenu jusqu'à exit
	// process → verrou metadata au prochain hot-reload Air.
	if reg != nil {
		reg.Close()
	}

	if err := closeShared(); err != nil {
		slog.Warn("fermeture shared DB", "err", err)
	}
	if err := metaDB.Close(); err != nil {
		slog.Warn("fermeture metadata DB", "err", err)
	}

	// Phase 2 plan stabilisation 2026-05-22 : détecter les fuites de refCount
	// sur le cache openDBs. Une fuite ici = HANDLE Windows tenu jusqu'à exit
	// process → verrou metadata.duckdb au prochain hot-reload Air.
	// cf. docs/INCIDENT_2026-05-21_metadata_duckdb_lock_air_hot_reload.md
	if leaks := duckdb.DumpCachedLeaks(); len(leaks) > 0 {
		for k, refs := range leaks {
			slog.Warn("shutdown_db_leak",
				"cache_key", k,
				"refCount", refs)
		}
		slog.Warn("shutdown_db_leak aggregate",
			"leaks_count", len(leaks))
	}

	slog.Info("shutdown_total_duration_ms",
		"ms", time.Since(shutdownStart).Milliseconds())

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

	// Seed Milestones catalogue (Phase 4 plan stabilisation 2026-05-22) via
	// migration backfill multi-titres. Pattern identique à Prestige.
	// configTitlesRoot = parent du dossier titre (ex: config/titles/) — itère
	// sur tous les `<slug>/milestones/catalog.toml` présents.
	// Cf. AUDIT_ASCENSION_PIPELINE_DISCONNECTED_2026-05-21 §4 cause A.
	if prestigeConfigDir != "" {
		configTitlesRoot := filepath.Dir(prestigeConfigDir)
		migration.RegisterMilestonesSeedMigration(configTitlesRoot)
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
	// E.v1 — attacher les watcher stores au Discovery pour peupler le pool
	// avec MSAL frais (watcher daemon les rafraîchit chaque ~5min) au 1er
	// boot sans dépendre d'un sync manuel ayant écrit sync_meta DuckDB.
	multiUserStore := auth.NewMultiUserTokenStore(pr.WatcherTokensDir())
	legacyStore := auth.NewTokenStore(pr.WatcherTokensPath())
	discovery := pool.NewDiscoveryWithStores(cfg, pr, title.DefaultSlug, multiUserStore, legacyStore)
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

// resolveCSRSeasonFromDB retourne le season_id CSR actif depuis csr_season_calendars.
// Retourne "" si la table est absente ou si aucune saison couvre la date courante.
func resolveCSRSeasonFromDB(db *sql.DB) string {
	var id string
	err := db.QueryRow(`
		SELECT season_id FROM csr_season_calendars
		WHERE title_id = 'halo_infinite'
		  AND start_date <= CURRENT_DATE
		  AND (end_date IS NULL OR end_date >= CURRENT_DATE)
		ORDER BY start_date DESC
		LIMIT 1
	`).Scan(&id)
	if err != nil {
		return ""
	}
	return id
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

	// Refresh loop : met à jour les tokens XSTS et le daemon.
	// PR 2.5b (2026-05-24) : WithMultiUserMirror — chaque refresh XSTS/OAuth
	// du tracker initial (legacy store) est aussi mirroré dans le multi-user
	// store. Maintient la cohérence entre les 2 stores en vue d'une future
	// migration read-path. Read continue via legacy store (compat user).
	multiMirror := auth.NewMultiUserTokenStore(title.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
	refreshLoop := auth.NewRefreshLoop(store, func(result *auth.XSTSResult) {
		daemon.UpdateAuth(result.AuthHeader())
	}).WithMultiUserMirror(multiMirror)
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

// installFatalSignalHandler enregistre un handler SIGABRT (+ SIGSEGV) qui
// dump la stack de toutes les goroutines vers crashFile avant exit. Capture
// les FatalException C++ de DuckDB que recover() Go ne peut pas attraper.
//
// Phase 4.2 du plan stabilisation 2026-05-22. Cf. main() au boot.
//
// Pré-condition : crashFile non-nil, ouvert en append.
//
// Note : sur Windows, signal.Notify(SIGABRT) compile mais ne fire pas. Le
// handler est un no-op sur cette plateforme — pas grave, on garde le code
// cross-platform.
func installFatalSignalHandler(crashFile *os.File) {
	sigFatal := make(chan os.Signal, 1)
	signal.Notify(sigFatal, syscall.SIGABRT, syscall.SIGSEGV)
	go func() {
		s := <-sigFatal
		dumpFatalStack(crashFile, s)
		// os.Exit(2) bypass les defers, mais le process est déjà mort logiquement —
		// abort() ré-émettra le signal sinon. C'est le seul moyen d'éviter le
		// double-crash silencieux.
		os.Exit(2)
	}()
}

// dumpFatalStack écrit un en-tête (timestamp + signal + pid) puis la stack
// complète de toutes les goroutines dans le crashFile. Extrait pour
// testabilité (unit test peut passer un *bytes.Buffer + un faux signal).
func dumpFatalStack(w io.Writer, sig os.Signal) {
	_, _ = fmt.Fprintf(w, "\n=== fatal signal %s at %s pid=%d ===\n",
		sig, time.Now().Format(time.RFC3339), os.Getpid())
	// Buffer 1 MB : suffit pour ~5000 goroutines avec frames raisonnables.
	// On veut TOUTES les goroutines (all=true) pour identifier deadlocks /
	// fuites au moment du crash.
	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)
	_, _ = w.Write(buf[:n])
	_, _ = w.Write([]byte("\n=== end fatal stack ===\n"))
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
