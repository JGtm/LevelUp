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
	"net/http"
	"os"
	"time"
	// La base de fuseaux est embarquée : sans elle time.LoadLocation ne trouve rien sur un poste
	// sans installation de Go (application de bureau installée), et chaque lecture du fuseau de
	// l'utilisateur retombe sur son repli.
	_ "time/tzdata"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/replaychild"
)

// version est injectée au build via -ldflags "-X main.version=X.Y.Z".
var version = "dev"

// serverWriteTimeout borne l'écriture d'une réponse HTTP (http.Server.WriteTimeout).
//
// 2026-09-23 : 30 s -> 120 s. Cause : les pages lourdes (Escouade, 25 à 55 s de calcul
// mesurées ce jour-là) étaient coupées à 30 s — 4 096 octets écrits, 502 renvoyé par le
// proxy, puis rejeu du front qui triplait le travail serveur (état des lieux perf du
// 2026-09-23, constat C2). Compatible avec la prod : nginx `proxy_read_timeout 300s`
// (packaging/nginx/levelup.conf). ReadTimeout et IdleTimeout inchangés. Le garde-rail
// internal/api/wire/build_queue_writer_budget_test.go lit cette constante.
const serverWriteTimeout = 120 * time.Second

func main() {
	interceptReplayChild()

	// --- 0. Arguments du serveur (flags.go), puis health-check (Docker HEALTHCHECK) ---
	st := &bootState{ctx: context.Background(), flags: parseServerFlagsOrExit()}
	if st.flags.HealthCheck {
		runHealthCheck()
	}
	if st.flags.Mode == serverModeRefreshOnly {
		os.Exit(runRefreshOnly(st)) // rafraîchit les jetons puis sort : ni base, ni port, ni événements
	}
	if st.flags.Mode == serverModeRestore {
		os.Exit(runRestoreMode(st.ctx, st.flags)) // restaure un jeu de sauvegarde puis sort : ni configuration serveur, ni port
	}
	if st.flags.Mode == serverModeUninstallCleanup {
		os.Exit(runUninstallCleanupMode(st.ctx)) // retire l'entrée de démarrage et la tâche planifiée puis sort : ni base, ni port, ni verrou
	}
	bootConfigure(st)
	bootMigrate(st)
	bootOpenDatabases(st)
	bootBuildServices(st)
	validateBootTypes(st)
	initSettingsAndAuth(st)
	initBootTasks(st)
	initAccountsPool(st)
	initScheduler(st)
	openPersistQueue(st)
	armGameWatch(st) // AVANT toute tâche lourde : premier relevé du jeu synchrone
	launchCoreTasks(st)
	startWatcher(st)
	launchSchedulers(st)
	buildRouter(st)
	launchMonitoring(st)
	launchCatalogCrons(st)
	wireSyncPipeline(st)
	launchSpartanCron(st)
	launchWorldLeaderboardCron(st)
	launchReplayPurgeCron(st)
	wireTitleHooks(st)
	openRouterAndHeartbeat(st)
	launchWakeSequence(st)
	launchGameWatch(st)
	launchShellIntegration(st)
	launchScheduledRefresh(st)
	launchParquetBackup(st)
	bindSiteImport(st)
	bindReplayRegen(st)
	awaitShutdown(st)
}

// interceptReplayChild détecte le mode enfant de cuisson et lui cède le processus.
func interceptReplayChild() {
	// --- 0 bis. Mode ENFANT DE CUISSON (lot BUILDALL, 2026-08-26) ---
	//
	// LE SERVEUR SE RÉ-EXÉCUTE LUI-MÊME pour décoder un film hors de son propre processus : le
	// décodage est un amplificateur mémoire, et l'enchaîner in-process a tué la machine trois
	// fois. L'interception est ICI, AVANT TOUT LE RESTE — un enfant qui démarrerait le serveur
	// ouvrirait un second écrivain sur les mêmes bases, ce que le modèle mono-processus
	// interdit (ADR 0013/0016). Même patron que le health-check ci-dessous.
	if replaychild.IsChild(os.Args) {
		// Les racines du parent (ResourcesRoot, DataRoot, LogsRoot) arrivent par l'environnement :
		// sans Load, c'est le seul moyen pour l'enfant de résoudre les mêmes chemins que lui.
		config.InstallProcessRootsFromEnv()
		os.Exit(replaychild.RunChild(context.Background(), os.Args))
	}
}

// newBootHTTPServer construit le http.Server qui sert la porte de démarrage puis le routeur.
func newBootHTTPServer(cfg *config.AppConfig, gate *bootGate) *http.Server {
	return &http.Server{
		Addr:         cfg.ServerAddr(),
		Handler:      gate,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: serverWriteTimeout,
		IdleTimeout:  60 * time.Second,
	}
}
