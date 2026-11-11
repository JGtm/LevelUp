// boot_config.go — première moitié du démarrage : arguments, crochets de processus, journaux,
// trace de plantage, configuration, ouverture de la porte, outillage et registre de titres.
//
// main() enchaîne les étapes (bootConfigure, bootMigrate, ...) ; ce qu'une étape laisse aux
// suivantes passe par bootState, jamais par une variable globale.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"levelup/go-api/internal/api/wire"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/adminstate"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/pool"
	"levelup/go-api/internal/platform/auth/tokencipher"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/friendstore"
	"levelup/go-api/internal/platform/gameproc"
	"levelup/go-api/internal/platform/groupstore"
	"levelup/go-api/internal/platform/instancelock"
	"levelup/go-api/internal/platform/netguard"
	"levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/replaybuild"
	"levelup/go-api/internal/resourcepack"
	"levelup/go-api/internal/scheduler"
	"levelup/go-api/internal/service"
	"levelup/go-api/internal/watcher"
	"levelup/go-api/pkg/duckdbbackup"
)

// bootState réunit ce que les étapes de démarrage se transmettent. Chaque champ est posé par
// l'étape qui le produit et lu par celles qui la suivent, dans l'ordre de main().
type bootState struct {
	ctx          context.Context
	flags        serverFlags
	cfg          *config.AppConfig
	multiHandler *logging.MultiModuleHandler // nil : journalisation console seule
	gate         *bootGate
	srv          *http.Server
	bootSrv      *bootServer
	// events : les événements du contrat coquille <-> serveur ; nil hors -mode desktop.
	events *desktopEvents
	// instanceLock : le verrou d'instance tenu par ce processus, tous profils confondus (nil si une autre
	// instance le tient : instanceLockErr). Rendu à l'arrêt (releaseInstanceLock).
	instanceLock    *instancelock.Lock
	instanceLockErr error
	pr              *title.PathResolver
	titleSlug       string
	dbPaths         bootDBPaths

	sharedReader duckdb.SharedReader
	closeShared  func() error
	metaDB       *duckdb.DB

	bootRepo      *duckdb.BootstrapRepo
	bootSvc       *service.BootstrapService
	us            *userstore.Store
	groupStore    *groupstore.GroupStore
	friendStore   *friendstore.FriendStore
	settingsStore *settings.Store
	tokenProvider auth.TokenProvider

	boot               *bootTasks
	schedulerWG        sync.WaitGroup // goroutines « suivies », attendues à l'arrêt avant duckdb.CloseAll
	autoSyncPool       pool.Pool      // nil : aucun credential découvert
	autoScheduler      *scheduler.AutoSyncScheduler
	schedulerCtx       context.Context
	cancelScheduler    context.CancelFunc
	adminActionJournal *adminstate.ActionJournal
	autoBatchQueue     *persist.BatchQueue // nil : persistance synchrone
	workerWG           sync.WaitGroup      // suit la goroutine du worker de la file de persistance

	watcherDaemon   *watcher.Daemon // nil : watcher désactivé (réglage ou démo)
	watcherCtrl     watcher.DaemonController
	gameWatcher     *gameproc.Watcher // nil : détection du jeu absente du profil
	healthScheduler *scheduler.HealthScheduler
	backupSched     *duckdbbackup.Scheduler
	reg             *wire.ServiceRegistry // nil jusqu'à la construction du routeur
	router          http.Handler

	// resources : le gestionnaire du paquet de ressources (profil desktop seulement, nil sinon) ; resourcesCatchUp :
	// le démarrage a eu lieu sur un paquet différent de celui du moteur, le bon est à télécharger (launchResources).
	resources        *resourcepack.Manager
	resourcesCatchUp bool
}

// bootConfigure enchaîne les étapes du démarrage qui précèdent l'accès aux bases : crochets
// de processus, journaux, configuration, ouverture de la porte, outillage média et registre
// de titres.
func bootConfigure(st *bootState) {
	initProcessHooks()
	setupLogging(st)
	loadBootConfig(st)
	applyBootGuards(st.cfg)
	applyBootMemoryLimit(st.cfg)
	openBootGate(st)
	refuseIfInstanceLockFailed(st)
	bootResourcePack(st)
	initToolingAndRegistry(st.cfg)
}

// runHealthCheck est le mode -health-check (Docker HEALTHCHECK) : il sonde /health du serveur
// local et sort du processus (os.Exit), il ne revient jamais.
func runHealthCheck() {
	hcClient := &http.Client{Timeout: 5 * time.Second}
	hcDo := func(url string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		return hcClient.Do(req) //nolint:bodyclose // body fermé par le caller via defer resp.Body.Close()
	}
	// Lit la MÊME variable que le serveur bind (config.Load: LEVELUP_API_PORT),
	// pas un nom fantôme — sinon le healthcheck ne suit pas le port configuré et
	// ne marche que sur 8000 par accident (revue P0 2026-06-02).
	port := os.Getenv("LEVELUP_API_PORT")
	if port == "" {
		port = "8000"
	}
	resp, err := hcDo("http://127.0.0.1:" + port + "/health") //nolint:bodyclose // body fermé via defer resp.Body.Close()
	if err != nil && port != "8000" {
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

// initProcessHooks charge .env.local puis pose les crochets de persistance sociale.
func initProcessHooks() {
	// --- 0.5 Charger .env.local AVANT la lecture des LEVELUP_LOG_* ---
	// Sinon le logger boote toujours en INFO/compact même si .env.local force
	// LEVELUP_LOG_LEVEL=warn (l'appel à config.Load qui charge .env.local
	// arrive bien plus tard dans main, après le setup logging).
	config.BootstrapEnvLocal()

	// --- 0.6 Wirer la factory SocialPersister (ADR 0022 Phase 5) ---
	// Permet à duckdb.openPlayerDB d'instancier un SharedSocialPersister
	// sans cycle d'import (duckdb -> persist serait cyclique car persist
	// -> duckdb via combined_persister.go). Le hook factory est lu à chaque
	// openPlayerDB ; si nil les writes shared_social retombent en legacy.
	duckdb.SocialPersisterFactory = func(db *sql.DB) duckdb.SocialPersister {
		return persist.NewSharedSocialPersister(db)
	}
	// ADR 0021 Gap 1 : refuser silencieusement les écritures legacy en prod.
	// Tout call site qui détecte SocialPersister == nil retourne désormais
	// ErrSocialPersisterNotWired au lieu de fallback vers Exec direct. Permet
	// de détecter immédiatement un bug de wiring boot.
	duckdb.RequireSocialPersister = true
}

// setupLogging installe le journal par défaut (console, fichiers par module, collecteur
// d'erreurs) et la trace de plantage, puis retient le gestionnaire fichier dans l'état.
func setupLogging(st *bootState) {
	// --- 1. Logging structuré ---
	// Trois formats console (LEVELUP_LOG_FORMAT) :
	//   - compact (défaut) : ConsoleHandler — `HH:MM:SS [INFO] sync.postSync: msg k=v`
	//     tronqué à LEVELUP_LOG_MAX_LINE (défaut 200), skip event_id/source.* pour
	//     limiter le bruit console (préservés dans logs/{module}.log JSON).
	//   - text             : slog.NewTextHandler natif (verbose, pre-sprint).
	//   - json             : slog.NewJSONHandler (prod).
	// Rétro-compat : LEVELUP_LOG_JSON=true → format=json.
	// Niveau via LEVELUP_LOG_LEVEL (défaut INFO).
	logLevel := config.ConsoleLogLevel()

	// Les racines se lisent dans l'environnement seul (config.ResolveRoots) : la même résolution
	// que config.LoadServer, donc le même dossier de journaux que celui de la configuration.
	// Démo : logs et crash log sous `<démo>/runtime/logs` (lot B5, D-7), cf. bootLogsConfig.
	logsCfg := bootLogsConfig(config.ResolveRoots().Resources)

	// MT-05 (PMT-10 PR-4) : namespacer les fichiers logs par titre UNIQUEMENT
	// quand plusieurs titres sont servis (déploiement multi-titre). En mono-titre
	// Halo (cas actuel, len==1), no-op → LogsDir nu, byte-identique. S'active
	// automatiquement dès qu'un 2e titre est enregistré dans le registre.
	if len(title.NewRegistry().All()) > 1 {
		logsCfg = logsCfg.WithTitleNamespace(title.DefaultSlug)
	}

	ensureLogsRoot(logsCfg)
	installCrashOutput(logsCfg)
	logHandler := newConsoleLogHandler(logsCfg, logLevel)
	st.multiHandler = installLogHandlers(logHandler, logsCfg)
}

// ensureLogsRoot crée la racine des journaux AVANT tout écrivain : sur une racine neuve, la
// capture des plantages (server.crash.log) s'ouvre avant le gestionnaire multi-modules et
// n'aurait aucun dossier où naître. Un échec est journalisé (sur la sortie d'erreur, le journal
// par défaut n'étant pas encore posé) et le démarrage continue : les écrivains dégradent seuls.
func ensureLogsRoot(logsCfg logging.Config) {
	if logsCfg.LogsDir == "" {
		return // journaux désactivés (LEVELUP_LOGS_ENABLED=false)
	}
	if err := os.MkdirAll(logsCfg.LogsDir, 0o755); err != nil {
		slog.Warn("logs_root: création impossible", "err", err, "path", logsCfg.LogsDir)
	}
}

// installCrashOutput redirige les panics et erreurs fatales du runtime vers le fichier de
// plantage et branche le handler de signaux fatals.
func installCrashOutput(logsCfg logging.Config) {
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
}

// newConsoleLogHandler construit le gestionnaire console selon le format demandé.
func newConsoleLogHandler(logsCfg logging.Config, logLevel slog.Level) slog.Handler {
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
	return logHandler
}

// installLogHandlers enveloppe le gestionnaire console (contexte, fichiers par module,
// collecteur d'erreurs), l'installe par défaut et rend le gestionnaire fichier (nil si
// absent).
func installLogHandlers(logHandler slog.Handler, logsCfg logging.Config) *logging.MultiModuleHandler {
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
		mh, err := logging.NewMultiModuleHandler(logHandler, logsCfg.LogsDir, logsCfg.FileLevel, logsCfg.Rotation)
		if err != nil {
			slog.New(logHandler).Warn("logging: multi-module handler indisponible (console only)",
				"err", err, "logs_dir", logsCfg.LogsDir)
		} else {
			multiHandler = mh
			logHandler = mh
		}
	}
	// Tee error collector : agrège les WARN/ERROR en mémoire (panneau monitoring
	// "erreurs récurrentes") sans altérer la sortie. Wrapper le plus externe →
	// vu une fois par record ; Enabled délègue (permissif), seul le collecteur
	// filtre sur >= WARN, donc le fan-out fichiers/console reste intact.
	logHandler = observability.NewErrorCollectorHandler(logHandler)
	slog.SetDefault(slog.New(logHandler))
	if multiHandler != nil {
		slog.Info("logging: multi-module actif",
			"logs_dir", logsCfg.LogsDir,
			"file_level", logsCfg.FileLevel.String(),
			"console_format", logsCfg.ConsoleFormat,
			"max_line", logsCfg.MaxLineWidth,
			"rotation_max_size_mb", logsCfg.Rotation.MaxSizeBytes/(1024*1024),
			"rotation_max_backups", logsCfg.Rotation.MaxBackups,
		)
	}
	return multiHandler
}

// loadBootConfig charge la configuration serveur ; un échec arrête le processus.
func loadBootConfig(st *bootState) {
	// --- 2. Configuration ---
	cfg, err := config.LoadServer(st.flags.Mode) // seul processus à suivre la disposition démo (B-C10)
	if err != nil {
		slog.Error("chargement config", "err", err)
		os.Exit(1)
	}
	if err := tokenVaultRefusal(cfg, tokencipher.ProcessError()); err != nil {
		slog.Error("démarrage refusé : coffre des jetons indisponible", "err", err)
		os.Exit(1)
	}
	// Injecter la version buildée (ldflags) si LEVELUP_APP_VERSION non défini.
	if cfg.AppVersion == "dev" && version != "dev" {
		cfg.AppVersion = version
	}
	// Les fichiers d'en-tete d'artefact de rejeu portent la version qui les a ecrits (replaybuild/artifact_meta.go).
	replaybuild.SetAppVersion(cfg.AppVersion)
	slog.Debug("config chargée",
		"resources_root", cfg.RepoRoot,
		"data_root", cfg.Roots().Data,
		"logs_root", cfg.Roots().Logs,
		"demo_mode", cfg.DemoMode,
		"version", cfg.AppVersion,
		"addr", cfg.ServerAddr(),
	)
	st.cfg = cfg
}

// tokenVaultRefusal refuse le démarrage d'un profil qui exige le coffre des jetons (capacité
// TokenVault, ADR 0045 D-11) quand le système ne l'offre pas : le profil desktop est Windows seul
// (D-2), et écrire les jetons en clair en silence trahirait le choix de l'utilisateur (D-8).
// vaultErr : tokencipher.ProcessError() du chargement de la configuration.
func tokenVaultRefusal(cfg *config.AppConfig, vaultErr error) error {
	if cfg.Capabilities.TokenVault && vaultErr != nil {
		return vaultErr
	}
	return nil
}

// applyBootGuards pose le coupe-circuit réseau de la démo et refuse une configuration invalide ou non sûre.
func applyBootGuards(cfg *config.AppConfig) {

	// Coupe-circuit réseau sortant. AVANT toute construction de client/cron : le
	// mode démo sert des données seedées et ne doit émettre AUCUN appel tiers.
	// Sans ce garde, une démo lancée sur un poste porteur de vrais tokens
	// s'authentifie et martèle l'API Halo pour les xuid factices de la fixture,
	// jusqu'à affamer le rendu des pages (cf. internal/platform/netguard).
	netguard.SetOffline(cfg.DemoMode)
	if cfg.DemoMode {
		slog.Info("demo_mode: appels réseau tiers coupés (données seedées, aucune synchronisation)")
	}

	// Garde-fou de démarrage (config.Validate) : partout, un mode d'authentification ou
	// d'inscription inconnu ; en production (LEVELUP_ENV=production), une configuration
	// non sûre — secret de session par défaut (cookies forgeables), AuthMode=none
	// (ownership multi-user désactivé), ou origines CORS limitées à localhost.
	// Hors production, les réglages non sûrs ne bloquent pas mais sont signalés.
	if err := cfg.Validate(); err != nil {
		slog.Error("démarrage refusé : configuration invalide ou non sûre", "err", err,
			"hint", "LEVELUP_AUTH_MODE=none|password|xbox, LEVELUP_REGISTRATION=invite|open|closed ; en production : LEVELUP_SESSION_SECRET (>=32 octets), LEVELUP_AUTH_MODE=xbox|password et LEVELUP_CORS_ORIGINS")
		os.Exit(1)
	}
	logSecurityWarnings(cfg)

	// Foot-gun rate-limit (incident "Too Many Requests" prod) : le limiter applicatif
	// (httprate) clé sur l'IP cliente. En production derrière un reverse proxy SANS
	// LEVELUP_TRUST_PROXY_HEADERS=true, X-Real-IP n'est pas lu → l'IP cliente est le
	// pair TCP, c'est-à-dire le proxy, pour TOUS les clients → un unique bucket
	// partagé → 429 en masse sous trafic public. Non-fatal (une expo prod directe sans
	// proxy est un setup légitime où TrustProxyHeaders=false est correct), mais on alerte.
	if cfg.IsProduction() && !cfg.TrustProxyHeaders {
		slog.Warn("rate-limit keyé sur le pair TCP et LEVELUP_TRUST_PROXY_HEADERS non activé : derrière un reverse proxy, tous les clients partagent un seul bucket (429 en masse)",
			"rate_limit_rpm", cfg.RateLimitRPM,
			"recommendation", "si le serveur est derrière nginx/Caddy/Traefik, définir LEVELUP_TRUST_PROXY_HEADERS=true")
	}
}

// initToolingAndRegistry vérifie l'outillage média et pose le registre de titres piloté par config.
func initToolingAndRegistry(cfg *config.AppConfig) {
	// --- Outillage média (ffmpeg/ffprobe) — check observable NON-bloquant ---
	// L'app exécute ffmpeg/ffprobe pour les miniatures WebP, le transcodage HLS
	// et le remux live des clips. Leur absence — ou un ffmpeg compilé sans
	// encodeur H.264 (libx264, h264_mf, libopenh264)/libwebp/aac ou sans le muxer hls — n'empêche pas le serveur de
	// démarrer mais casse ces surfaces au premier upload. On rend le diagnostic
	// visible ici (slog Info si OK avec la version, Warn actionnable sinon)
	// plutôt qu'en échec silencieux. Borné dans le temps (4 lancements de ffmpeg) pour
	// ne jamais retarder le boot.
	mediaCtx, mediaCancel := context.WithTimeout(context.Background(), 5*time.Second)
	ops.LogMediaToolingStatus(mediaCtx)
	mediaCancel()

	// --- Registre de titres PILOTÉ PAR CONFIG (MT-16 / day-one 2e titre) ---
	// Built-in halo_infinite + titres additionnels découverts sous
	// config/titles/<slug>/title.toml. Posé en registre partagé AVANT toute
	// ouverture DB / démarrage serveur → tous les call-sites DefaultRegistry()
	// (front switcher, résolution titre, XboxTitleIDFor) voient les titres
	// additionnels. Mono-titre = byte-identique (seul halo_infinite, built-in).
	title.SetDefaultRegistry(title.NewRegistryFromConfig(cfg.RepoRoot, slog.Default()))
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
