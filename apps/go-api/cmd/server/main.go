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

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/api"
	"levelup/go-api/internal/assetnames"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/games/halo_5/livesync"
	halo5migrations "levelup/go-api/internal/games/halo_5/migrations"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/games/halo_infinite/skillchain"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/notify"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/capturecli"
	"levelup/go-api/internal/platform/auth/pool"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/platform/groupstore"
	"levelup/go-api/internal/platform/halo"
	"levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/scheduler"
	"levelup/go-api/internal/service"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/watcher"
	"levelup/go-api/internal/worldenrich"
)

// version est injectée au build via -ldflags "-X main.version=X.Y.Z".
var version = "dev"

// buildTokenProvider instancie le TokenProvider selon app_settings.json:auth_provider.
//
// PR-D : SISU est le défaut (authentification native Xbox, ZÉRO app Azure) — car
// LevelUp est distribué à des self-hosters qui ne peuvent pas tous enregistrer
// une app Azure. MSAL reste conservé en code et activable explicitement via
// auth_provider="msal" (déprécié, sans entrée UI) : fallback si SISU casse
// (client_id Xbox natif non officiel — risque que Microsoft le modifie).
//
//	"" (défaut) | "sisu" → SISUProvider
//	"msal"               → MSALProvider (fallback config-only)
func buildTokenProvider(settingsStore *settings.Store, authDesc title.AuthDescriptor) auth.TokenProvider {
	// MT-02 (PMT-2 leg 3) : les SISU app/title id viennent du descripteur du titre
	// (byte-identique au défaut Halo). Descripteur incomplet → garde NewSISUProvider().
	newSISU := func() auth.TokenProvider {
		if authDesc.SISUAppID != "" && authDesc.SISUTitleID != "" {
			return auth.NewSISUProviderWithIDs(authDesc.SISUAppID, authDesc.SISUTitleID)
		}
		return auth.NewSISUProvider()
	}
	s, err := settingsStore.Load()
	if err != nil {
		slog.Warn("buildTokenProvider: lecture settings échouée, défaut SISU", "err", err)
		return newSISU()
	}
	if s.AuthProvider == "msal" {
		slog.Info("buildTokenProvider: MSAL provider activé (fallback config)")
		return auth.NewMSALProvider()
	}
	if s.AuthProvider != "" && s.AuthProvider != "sisu" {
		slog.Warn("buildTokenProvider: valeur auth_provider inconnue, défaut SISU", "value", s.AuthProvider)
	}
	slog.Info("buildTokenProvider: SISU provider activé (défaut)",
		"sisu_title_id", authDesc.SISUTitleID)
	return newSISU()
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

	// MT-05 (PMT-10 PR-4) : namespacer les fichiers logs par titre UNIQUEMENT
	// quand plusieurs titres sont servis (déploiement multi-titre). En mono-titre
	// Halo (cas actuel, len==1), no-op → LogsDir nu, byte-identique. S'active
	// automatiquement dès qu'un 2e titre est enregistré dans le registre.
	if len(title.NewRegistry().All()) > 1 {
		logsCfg = logsCfg.WithTitleNamespace(title.DefaultSlug)
	}

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

	// Garde-fou de démarrage (revue P0 2026-06-02) : en production
	// (LEVELUP_ENV=production), refuser de booter avec une configuration non sûre
	// — secret de session par défaut (cookies forgeables), AuthMode=none
	// (ownership multi-user désactivé), ou origines CORS limitées à localhost.
	// Hors production, on ne bloque pas mais on émet un avertissement visible.
	if err := cfg.Validate(); err != nil {
		slog.Error("démarrage refusé : configuration non sûre pour la production", "err", err,
			"hint", "définir LEVELUP_SESSION_SECRET (>=32 octets), LEVELUP_AUTH_MODE=xbox|password et LEVELUP_CORS_ORIGINS, ou retirer LEVELUP_ENV=production")
		os.Exit(1)
	}
	if warnings := cfg.SecurityWarnings(); len(warnings) > 0 {
		slog.Warn("configuration non sûre pour un déploiement multi-user exposé",
			"issues_count", len(warnings),
			"issues", strings.Join(warnings, " | "),
			"prod_guard", "LEVELUP_ENV=production refuserait de démarrer dans cet état")
	}

	// Foot-gun rate-limit (incident "Too Many Requests" prod) : le limiter applicatif
	// (httprate) clé sur RemoteAddr. En production derrière un reverse proxy SANS
	// LEVELUP_TRUST_PROXY_HEADERS=true, chi RealIP n'est pas câblé → RemoteAddr reste
	// l'IP du proxy (127.0.0.1) pour TOUS les clients → un unique bucket partagé →
	// 429 en masse sous trafic public. Non-fatal (une expo prod directe sans proxy est
	// un setup légitime où TrustProxyHeaders=false est correct), mais on alerte.
	if cfg.IsProduction() && !cfg.TrustProxyHeaders {
		slog.Warn("rate-limit keyé sur RemoteAddr et LEVELUP_TRUST_PROXY_HEADERS non activé : derrière un reverse proxy, tous les clients partagent un seul bucket (429 en masse)",
			"rate_limit_rpm", cfg.RateLimitRPM,
			"recommendation", "si le serveur est derrière nginx/Caddy/Traefik, définir LEVELUP_TRUST_PROXY_HEADERS=true")
	}

	// Foot-gun ART (revue P1 2026-06-02) : LEVELUP_PERSIST_BATCH=0 désactive le
	// chemin d'écriture batch INSERT-only et réactive le chemin legacy
	// (ON CONFLICT DO UPDATE sur les tables shared match_registry/match_participants)
	// qui peut rouvrir le bug ART DuckDB ("Failed to delete all rows from index")
	// sous concurrence multi-user. Le défaut (batch ON) est sûr ; on alerte
	// bruyamment si l'opérateur a explicitement désactivé le batch — ce fallback
	// de rollback ne doit jamais rester posé en prod/multi-user.
	if os.Getenv("LEVELUP_PERSIST_BATCH") == "0" {
		slog.Warn("LEVELUP_PERSIST_BATCH=0 : chemin d'écriture legacy ART-unsafe activé (UPSERT concurrent sur tables shared) — risque de corruption d'index DuckDB en multi-user",
			"recommendation", "retirer LEVELUP_PERSIST_BATCH (ou le mettre à 1) hors situation de rollback ponctuel")
	}

	// --- Registre de titres PILOTÉ PAR CONFIG (MT-16 / day-one 2e titre) ---
	// Built-in halo_infinite + titres additionnels découverts sous
	// config/titles/<slug>/title.toml. Posé en registre partagé AVANT toute
	// ouverture DB / démarrage serveur → tous les call-sites DefaultRegistry()
	// (front switcher, résolution titre, XboxTitleIDFor) voient les titres
	// additionnels. Mono-titre = byte-identique (seul halo_infinite, built-in).
	title.SetDefaultRegistry(title.NewRegistryFromConfig(cfg.RepoRoot, slog.Default()))

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
	// Sur un clone frais, le dossier warehouse n'existe pas encore : sans lui,
	// runMigrations puis l'ouverture du provider shared échouent et le serveur
	// fait os.Exit(1) avant même d'afficher /setup.
	if err := ensureWarehouseDir(pr, titleSlug); err != nil {
		slog.Error("création warehouse dir échouée", "err", err)
		os.Exit(1)
	}
	// Clone frais : extraire la base metadata pré-construite (référentiels prêts)
	// si metadata.duckdb n'existe pas encore. Non-fatal — sinon les migrations
	// créeront une base vide à repeupler via fetch live.
	if err := extractPrebuiltMetadataIfAbsent(pr, titleSlug); err != nil {
		slog.Warn("prebuilt metadata: extraction échouée (non-fatal)", "err", err)
	}
	titleConfigDir := filepath.Join(cfg.RepoRoot, "config", "titles", titleSlug)
	if err := runMigrations(metaPath, sharedPath, sharedSocialPath, pr.SharedPVEDBPath(titleSlug), titleConfigDir); err != nil {
		slog.Debug("migrations ignorées (DB verrouillée), démarrage sans migration")
	} else {
		slog.Debug("migrations appliquées")
	}

	// --- 3a-bis. Provisionner les DB des titres ADDITIONNELS actifs (MT-16 /
	// day-one 2e titre). No-op en mono-titre (seul halo_infinite actif, déjà
	// provisionné ci-dessus). Un 2e titre déclaré en config (status="active") voit
	// ici ses warehouses créées + migrées (RunForTitleDB → jeu de migrations du
	// titre, fallback set Halo), isolées sous data/titles/<slug>/. Non-fatal : un
	// titre additionnel cassé ne doit jamais empêcher Halo de démarrer.
	if !cfg.DemoMode {
		provisionAdditionalActiveTitles(pr, title.DefaultRegistry())
	}

	// --- 3b. Migrations player DB (TargetPlayer) ---
	// RW-EXCLUSIF, AVANT que le provider/scheduler/watcher n'ouvrent les player
	// DBs. Sans cet appel, les player DBs ne recevaient QUE EnsurePlayerSchema
	// (CREATE TABLE IF NOT EXISTS) — no-op sur une table legacy préexistante →
	// la PRIMARY KEY n'était jamais ajoutée et les writes ON CONFLICT /
	// INSERT OR IGNORE échouaient en Binder Error (citations, enrichment rows).
	// `RunForDB(TargetPlayer)` n'était jusqu'ici câblé qu'en CLI ; on le câble
	// au boot par profil. Idempotent (migrations tracées dans schema_migrations).
	// Non-fatal : une player DB verrouillée/absente ne doit pas bloquer le boot.
	// Cf. repair_*_primary_key + .ai/thought_log 2026-06-04.
	if !cfg.DemoMode {
		if players, perr := cfg.LoadPlayers(); perr != nil {
			slog.Warn("migrations player: chargement profils échoué (non-fatal)", "err", perr)
		} else {
			for _, p := range players {
				if p.Gamertag == "" || p.IsDemo {
					continue
				}
				dbPath := pr.PlayerDBPath(titleSlug, p.Gamertag)
				// Comptes token-only (watchers, db_path vide dans db_profiles.json) :
				// pas de player DB → rien à migrer. La création de la DB appartient
				// au chemin sync/onboarding, pas au boot.
				if _, statErr := os.Stat(dbPath); statErr != nil {
					slog.Debug("migrations player ignorées — player DB absente (compte token-only ?)",
						"gamertag", p.Gamertag, "db", dbPath)
					continue
				}
				if err := RunPlayerMigrations(dbPath); err != nil {
					slog.Warn("migrations player échouées (non-fatal)",
						"gamertag", p.Gamertag, "err", err)
				}
			}
			slog.Debug("migrations player appliquées")
		}
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
		// Injecter aussi le Manager : la lecture per-titre (player_resolver) et
		// l'écriture per-titre (livesync Halo 5+) résolvent le provider du shared
		// d'un titre via For(SharedDBPath(slug)). Pour DefaultSlug, For() retourne
		// CE provider (caché par path) → byte-identique. closeShared ferme TOUS
		// les providers via sharedMgr.Close().
		cfg.SharedManager = sharedMgr

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

	// Auth locale : user store partagé — filtrage ownership des joueurs (ADR 0029,
	// modes password + xbox) et check "first launch" (mode password).
	usersPath := filepath.Join(cfg.AuthDir, "users.json")
	us := userstore.NewStore(usersPath)
	bootSvc = bootSvc.WithUserLookup(us)
	if cfg.AuthMode == "password" {
		bootSvc = bootSvc.WithUserStoreEmpty(us.IsEmpty)
	}
	// Groupes/familles (accès mutuel aux données, ADR 0029 multi-groupes). Le set
	// co-membres pilote le filtrage ownership (available_players) et le switch de BDD.
	groupStore := groupstore.NewGroupStore(filepath.Join(cfg.AuthDir, "groups.json"))
	bootSvc = bootSvc.WithCoMemberResolver(func(xuid string) map[string]bool {
		co, _ := groupStore.CoMemberXUIDs(xuid)
		return co
	})

	// PR-B : expose reauth_required (refresh_token mort) du joueur courant au front.
	// Lecture par-xuid dans le MultiUserTokenStore (data/auth/watcher_tokens/{xuid}.json).
	reauthStore := auth.NewMultiUserTokenStore(title.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
	bootSvc = bootSvc.WithReauthChecker(reauthStore.IsReauthRequired)

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
	// Propage le réglage global "rendement sans assistances" au package analysis
	// dès le boot (sinon le défaut false s'applique jusqu'au premier PATCH).
	if s, lerr := settingsStore.Load(); lerr == nil {
		analysis.SetExcludeAssistsFromYield(s.RendementExcludeAssists)
		if s.RendementExcludeAssists {
			slog.Info("rendement combat : assistances EXCLUES (réglage rendement_exclude_assists actif)")
		}
	}
	tokenProvider := buildTokenProvider(settingsStore, title.DefaultHaloAuthDescriptor())

	// Migration boot-time : crée un groupe par défaut "Mon foyer" depuis l'ancienne
	// liste friend_gamertags (continuité d'accès au passage multi-groupes). Idempotent.
	migrateDefaultGroupAtBoot(ctx, cfg, settingsStore, groupStore)

	// ADR 0023 Phase 2 — Migration boot-time des tokens legacy vers MultiUserTokenStore.
	// Copie SPNKR_OAUTH_REFRESH_TOKEN_<GT> (env) + sync_meta.oauth_refresh_token (DuckDB)
	// vers le store si les entrées correspondantes n'existent pas. Idempotent, best-effort.
	// S'exécute AVANT buildAutoSyncPool pour que le Pool trouve déjà les tokens dans le store.
	migrateLegacyAuthTokensAtBoot(ctx, cfg)

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
			// Métrique : chaque WAL purgé sans avoir été persisté (perte
			// potentielle) incrémente persist_wal_purged_total (le log ERROR
			// est émis par PurgeOldWAL). Cf. PLAN_PERSIST_ROBUSTNESS Phase 1.
			q.OnWALPurged = func(info persist.PurgedWALInfo) {
				observability.IncCounter("persist_wal_purged_total")
			}
			slog.InfoContext(ctx, "persist: BatchQueue activée (async path)",
				"wal_dir", walDir)

			// Dashboard monitoring P4 : hook de chronométrage des phases
			// d'écriture (acquire/lease/write par DB) → expvar, sans coupler
			// persist à observability. Posé une fois avant le 1er batch.
			persist.OnPersistPhase = func(phase string, d time.Duration, ok bool) {
				observability.RecordDurationMS("persist_"+phase+"_ms", d.Milliseconds())
				if !ok {
					observability.IncCounter("persist_" + phase + "_err_total")
				}
			}

			// Câblage Worker — CombinedPersister écrit shared + player par batch.
			// context.Background() : le Worker doit finir le batch en cours avant
			// de s'arrêter → ne doit pas être annulé par cancelScheduler().
			// Arrêt naturel via autoBatchQueue.Close() (channel close) au shutdown.
			combinedP := persist.NewCombinedPersister(
				func(workerCtx context.Context) (*sql.DB, func(), error) {
					return syncpkg.AcquireSharedWriterStandalone(workerCtx, cfg.SharedProvider, sharedPath)
				},
				// titleSlug vient du batch (batch.TitleSlug) — chaque batch route vers
				// la player DB de SON titre (multi-titres). En mono-titre = identique.
				func(batchTitleSlug, gamertag string) string { return pr.PlayerDBPath(batchTitleSlug, gamertag) },
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
				// Garde-fou anti-perte (PLAN_PERSIST_ROBUSTNESS Phase 1) :
				// re-tenter les WAL pending AVANT de purger, sinon on pourrait
				// effacer un batch qu'on aurait pu rejouer.
				if rerr := autoBatchQueue.RecoverPending(); rerr != nil {
					slog.WarnContext(schedulerCtx, "janitor: RecoverPending échoué (non-bloquant)",
						"module", logging.ModulePersist, "err", rerr)
				}
				if n, err := autoBatchQueue.PurgeOldWAL(7 * 24 * time.Hour); err != nil {
					slog.WarnContext(schedulerCtx, "janitor: PurgeOldWAL échoué (non-bloquant)",
						"module", logging.ModulePersist, "err", err)
				} else if n > 0 {
					slog.InfoContext(schedulerCtx, "janitor: WAL purgé",
						"module", logging.ModulePersist, "files_removed", n)
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

	// Recovery périodique des WAL pending (PLAN_PERSIST_ROBUSTNESS Phase 1).
	// Avant : RecoverPending n'était appelé qu'au boot → un batch échoué
	// (DB busy au moment du persist) restait bloqué jusqu'au prochain reboot,
	// sur une webapp qui tourne des semaines. Désormais re-soumis toutes les
	// 10 min. Le dédup inFlight de la queue évite de re-pousser un batch déjà
	// en vol (seuls les WAL réellement bloqués sont rejoués).
	if autoBatchQueue != nil {
		go func() {
			recTicker := time.NewTicker(10 * time.Minute)
			defer recTicker.Stop()
			for {
				select {
				case <-schedulerCtx.Done():
					return
				case <-recTicker.C:
					if rerr := autoBatchQueue.RecoverPending(); rerr != nil {
						slog.WarnContext(schedulerCtx, "persist: recovery périodique échouée (non-bloquant)",
							"module", logging.ModulePersist, "err", rerr)
					}
				}
			}
		}()
	}

	// CHECKPOINT périodique shared_social — vide le WAL toutes les 5 min sans
	// bloquer les writes per-opération.
	//
	// Pourquoi pas un CHECKPOINT systématique dans Persist() ? La connexion
	// shared_social est ouverte MaxOpenConns(4) (OpenReadWriteShared) — donc PAS
	// "une seule connexion" (revue 2026-06-01 SS-3 : l'ancien commentaire
	// MaxOpenConns(1) était faux). Le choix est de borner la fenêtre WAL à 5 min
	// via ce scheduler + le CHECKPOINT shutdown, plutôt que de checkpointer après
	// chaque commit (coût 100–500 ms par write user-facing). Les écritures
	// critiques (likes/favoris/associations média) peuvent flush immédiat via
	// CommitWithCheckpoint.
	//
	// Sérialisation : le commit-lock single-writer de DuckDB + le lease
	// KindSharedSocial sérialisent déjà les écritures ; ce tick réutilise le même
	// handle caché (LookupCachedDB) sans contention notable.
	//
	// LookupCachedDB : pas d'ouverture propre — on réutilise la connexion du
	// pool process-wide (même *sql.DB que SharedSocialPersister). Si le premier
	// joueur n'a pas encore été chargé, on skip le tick silencieusement.
	go func() {
		ckptTicker := time.NewTicker(5 * time.Minute)
		defer ckptTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ckptTicker.C:
				socialDB, ok := duckdb.LookupCachedDB(sharedSocialPath)
				if !ok {
					continue // DB pas encore ouverte, skip
				}
				ckptStart := time.Now()
				ckptCtx, ckptCancel := context.WithTimeout(context.Background(), 10*time.Second)
				if _, err := socialDB.SQLDb().ExecContext(ckptCtx, "CHECKPOINT"); err != nil {
					slog.WarnContext(ckptCtx, "shared_social: periodic checkpoint failed", "err", err)
				} else {
					slog.DebugContext(ckptCtx, "shared_social: periodic checkpoint",
						"duration_ms", time.Since(ckptStart).Milliseconds())
				}
				ckptCancel()
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
	var watcherDaemon *watcher.Daemon = startWatcherDaemon(ctx, cfg, settingsStore, tokenProvider, notifierGetter, tokenRefresher, autoSyncPool, autoScheduler)
	if watcherDaemon != nil {
		autoScheduler.ActivityChecker = watcher.NewStateProvider(watcherDaemon)
		// Dédup cross-source (unification 2026-06-02) : le Coordinator du watcher
		// devient le point de dédup partagé. L'auto-sync l'interroge avant chaque
		// RunDelta (cf. syncPlayer) et NewRouter le propage au handler HTTP via
		// autoScheduler.Gate(). Si le watcher est désactivé, SyncGate reste le
		// NopSyncGate par défaut → comportement legacy (lease seul rempart).
		autoScheduler.SyncGate = watcherDaemon.SyncGate()
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
	//
	// routerCtx : on ne dérive les bgCtx des syncs HTTP de schedulerCtx (annulable
	// au shutdown) QUE si un vrai gate est branché (watcher actif) — alors le drain
	// gate.WaitInFlight() au shutdown les attend. Watcher off → context.Background()
	// (comportement legacy exact : NopSyncGate ne tracke rien, pas de WaitInFlight).
	routerCtx := context.Background()
	if watcherDaemon != nil {
		routerCtx = schedulerCtx
	}
	var router http.Handler
	router, reg = api.NewRouter(routerCtx, cfg, bootRepo, bootSvc, watcherCtrl, tokenProvider, autoScheduler, backupSched, groupStore)

	// Câble le hook global de re-dérivation des tokens per-player : le cache
	// expiry-aware (halo.ResolveFreshPlayerTokens) re-minte via le registry quand un
	// token est périmé. Sans ce câblage, le re-mint singleflighté est inopérant.
	reg.WireGlobalTokenRefresher()

	// Dashboard monitoring admin — câblage du HealthScheduler (créé plus haut,
	// avant NewRouter). Les runners monitoring le lisent lazily à chaque
	// requête : l'ordre boot est sûr, et l'overview expose le dernier audit
	// data health + l'action POST /admin/actions/data-health/run.
	reg.WithHealthScheduler(healthScheduler)

	// Cron catalogue (hebdomadaire) : rafraîchit le catalogue (playlists / couples
	// map-mode / maps / modes) via le drain DiscoveryUGC testé (même chemin que l'action
	// admin catalog/ugc-drain). TOUJOURS actif (autonome, plus de flag). La
	// régularité vient du ticker hebdo, PAS d'un redémarrage.
	//
	// ART-safety (2026-06-19) : l'UPSERT vers les tables catalogue est SELECT-then-write
	// (upsertRowNoConflict), MAIS la queue catalog_fetch_queue était elle-même DELETE +
	// UPDATE per-row sur index ART (PK + idx secondaire) → corrompait metadata.duckdb à
	// chaque boot. Corrigé : la queue n'a plus aucune surface ART (cf. migration
	// rebuild_catalog_fetch_queue_drop_art_indexes + dédup NOT EXISTS à l'enqueue).
	catalogCron := scheduler.NewCatalogRefreshCron(func(cctx context.Context, ts string) (domain.CatalogUGCDrainResult, error) {
		// V2 — découverte « A à Z » : avant le drain, lire la config de chaque
		// playlist (discovery-infiniteugc) pour enfiler ses couples map-mode enfants
		// (même jamais joués) + stocker les poids. Best-effort : un échec d'expansion
		// n'empêche pas le drain de nommer ce qui a été joué.
		if n, eerr := reg.ExpandPlaylistChildren(cctx, ts); eerr != nil {
			slog.WarnContext(cctx, "catalog_refresh_cron: expansion playlists échouée (best-effort)", "module", logging.ModuleCatalog, "err", eerr)
		} else {
			slog.InfoContext(cctx, "catalog_refresh_cron: playlists expansées", "module", logging.ModuleCatalog, "children_enqueued", n)
		}
		// NB : le balayage des NOMS d'assets (asset_translations) vit dans
		// asset_name_sweep_cron (découplé, gaté LEVELUP_SYNC_RESOLVE_ASSETS).
		return reg.RunCatalogUGCDrain(cctx, ts)
	}, "", 0).
		// Gate RÉEL (remplace le proxy CapForge) : ne draine QUE les titres dont le
		// catalog adapter discovery-infiniteugc est résolvable (rules TOML +
		// halo_infinite.NewCatalogAdapter). HINF → adapter présent → run ; Halo 5 →
		// pas d'experience_rules.toml → skip propre. Comportement prod identique au
		// proxy, signal précis.
		WithCatalogAdapterCheck(reg.HasCatalogAdapter)
	schedulerWG.Add(1)
	go func() {
		defer schedulerWG.Done()
		catalogCron.Run(schedulerCtx)
	}()
	slog.InfoContext(ctx, "catalog_refresh_cron: scheduled", "module", logging.ModuleCatalog, "interval", scheduler.DefaultCatalogRefreshInterval)

	// Balayage de noms d'assets (filet de rattrapage de la traîne), distinct du cron
	// catalogue : ART-safe (asset_translations via ops.UpsertAssetTranslation), gaté par
	// LEVELUP_SYNC_RESOLVE_ASSETS (ON par défaut, kill-switch). 1er passage ~60s après le
	// boot (le temps que le pool de tokens se réchauffe), puis hebdomadaire. La résolution
	// PRIMAIRE des noms reste in-sync ; ce cron ne rattrape que les assets jamais rejoués.
	if halo.AssetNameResolutionEnabled() {
		sweepCron := scheduler.NewAssetNameSweepCron(func(cctx context.Context, ts string) (assetnames.Result, error) {
			return reg.ResolveUnresolvedAssetNames(cctx, ts, autoSyncPool)
		}, "", 0).
			// Même gate RÉEL que le drain catalogue : le sweep de noms passe par le
			// même fetcher /hi/ hardcodé, donc même critère (catalog adapter résolvable).
			WithCatalogAdapterCheck(reg.HasCatalogAdapter)
		schedulerWG.Add(1)
		go func() {
			defer schedulerWG.Done()
			sweepCron.Run(schedulerCtx)
		}()
		slog.InfoContext(ctx, "asset_name_sweep_cron: scheduled", "module", logging.ModuleSync, "interval", scheduler.DefaultAssetNameSweepInterval)
	}

	// Phase 4 plan stabilisation 2026-05-22 — câblage post-sync runner sur
	// l'auto-sync scheduler. Avant ce fix, l'auto-sync court-circuitait
	// systématiquement le pipeline progression V2 (streaks/records/milestones/
	// notifications delta). Maintenant les 3 entry points (HTTP, auto-sync,
	// CLI futur) invoquent le MÊME runner via SyncEngine.WithPostSyncRunner.
	// Cf. AUDIT_ASCENSION_PIPELINE_DISCONNECTED_2026-05-21 §4 cause B.
	if postSyncRunner := api.NewPostSyncRunner(reg); postSyncRunner != nil {
		autoScheduler.WithPostSyncRunner(postSyncRunner)
	}

	// ADR 0027 D6.5 — câblage pipeline V2 (dormant tant que
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

	// v2 canonical = défaut serveur (ADR 0024) si les flags sont absents du
	// process — survit à un reset .air.toml/.env.local. Opt-out explicite :
	// LEVELUP_LUSR_CANONICAL=LUSR. AVANT LogLUSRModeAtBoot (log = état effectif).
	syncpkg.DefaultLUSRModeIfUnset(context.Background())
	// Confirme au boot le mode LUSR actif (v1 / v2 shadow / v2 canonical) dans
	// logs/sync.log + alerte sur la misconfig canonical-sans-enabled.
	syncpkg.LogLUSRModeAtBoot(context.Background())

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
		// Title-aware (refactor h5-capability-unification) : enregistre le refresher
		// de customisation des AUTRES titres (Halo 5+). Le scheduler n'importe AUCUN
		// package de titre — c'est ICI (boot, qui importe déjà halo5/livesync) que la
		// closure title-spécifique est injectée. halo_5 → livesync.PersistAppearance
		// (fetch /h5/profiles/{gt}/{appearance,spartan,emblem} + persist service tag /
		// rendu Spartan / emblème dans career_progression h5, append-only). Le ctx
		// porte déjà l'auth du joueur (posée par le cron) → NewAppearanceSource la lit.
		// Best-effort : un échec source/fetch est remonté en err (loggé par le cron).
		spartanCron.WithRefresher(halo5.TitleSlug, func(rctx context.Context, p domain.PlayerSummary) error {
			src, err := halo5.NewAppearanceSource(rctx)
			if err != nil {
				return err
			}
			playerDBPath := pr.PlayerDBPath(halo5.TitleSlug, p.Gamertag)
			cacheRoot := filepath.Join(pr.RepoRoot(), "data", "cache")
			_, err = livesync.PersistAppearance(rctx, src, playerDBPath, cacheRoot, p.Gamertag, p.XUID)
			return err
		})
		schedulerWG.Add(1)
		go func() {
			defer schedulerWG.Done()
			spartanCron.Run(schedulerCtx)
		}()
		slog.InfoContext(ctx, "spartan_cron: scheduled",
			"interval", scheduler.DefaultSpartanCustomizationInterval)
	}

	// Cron classement CSR mondial : capture quotidienne du leaderboard scrapé
	// depuis Halo Waypoint (saison active découverte automatiquement), persistée
	// en append-only via le SharedDBProvider (AcquireWriter). Premier tick au boot
	// → peuple la prod sans manip manuelle ; garde-fou fraîcheur 20h → pas de
	// re-scrape à chaque hot-reload. Gardé par cfg.SharedProvider != nil : en mode
	// legacy (RO direct), une 2e connexion RW sur shared entrerait en conflit
	// "different configuration" — on désactive donc le cron in-process (le CLI
	// snapshot-world-leaderboard reste la voie manuelle, serveur arrêté).
	if cfg.SharedProvider != nil {
		lbScraper := halo.NewLeaderboardScraper(800 * time.Millisecond)
		worldLbCron := scheduler.NewWorldLeaderboardCron(cfg.SharedProvider, lbScraper, 0)
		schedulerWG.Add(1)
		go func() {
			defer schedulerWG.Done()
			// Enrichissement des stats joueur (ranked-only, dédup match-centric,
			// append-only) : TOUJOURS actif, via le POOL multi-token (comptes db_profiles
			// round-robin — même chemin d'auth que le reste de l'app et que le CLI
			// -all-tokens, store-first ADR 0023). Construit ICI, dans la goroutine (hors
			// chemin de démarrage → boot non bloqué), en Eager:true : résout les tokens et
			// ÉLIMINE les comptes au RT mort (invalid_grant). Indispensable — sinon le
			// round-robin taperait des sources mortes et raterait des fetchs. Un build en
			// échec (aucun compte résolu) dégrade en scrape-only sans paniquer.
			if enr, gts, eerr := worldenrich.BuildEnricher(cfg, worldenrich.EnricherOptions{
				RPS:   5,
				Eager: true, // filtre les comptes db_profiles dont le RT ne résout pas
			}); eerr != nil {
				slog.WarnContext(ctx, "world_leaderboard_cron: enricher non construit — cron en scrape-only",
					"err", eerr)
			} else {
				worldLbCron.WithStatsEnricher(enr)
				slog.InfoContext(ctx, "world_leaderboard_cron: enrichissement actif (pool multi-token)",
					"token_accounts", gts)
			}
			worldLbCron.Run(schedulerCtx)
		}()
		slog.InfoContext(ctx, "world_leaderboard_cron: scheduled",
			"interval", scheduler.DefaultWorldLeaderboardInterval)
	}

	// MT-19 / axe E : notifier « titre prêt » injecté dans cfg (lu au runtime par le
	// Runner live h5 via cfg.TitleReadyNotifier). Posé APRÈS NewRouter (reg dispo) et
	// AVANT que les syncs scheduler/watcher tournent (runtime post-boot). Émet une
	// notif quand un titre live a des matchs, dans le flux du titre par défaut, hors
	// pipeline progression/prestige. Même pattern d'injection que cfg.SharedManager.
	cfg.TitleReadyNotifier = api.BuildTitleReadyNotifier(reg, cfg)
	// Progression V2 title-agnostic : le Runner live d'un titre (Halo 5+) déclenche le
	// pipeline streaks/records/milestones/coach via ce hook après un cycle qui insère
	// des matchs (deps de base, SANS le PrestigeBundle mono-titre). Même pattern d'injection.
	cfg.ProgressionAfterSync = api.BuildProgressionAfterSyncHook(reg, cfg)

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

	// Dédup cross-source (unification 2026-06-02) : attendre que les syncs HTTP
	// gate-claimés finissent avant duckdb.CloseAll(). cancelScheduler() ci-dessus
	// a annulé schedulerCtx, dont les bgCtx HTTP dérivent → les RunDelta en cours
	// abandonnent, libèrent le lease + le claim, puis gateWG se vide. Sans cette
	// attente, un sync HTTP détaché pourrait écrire une player DB après CloseAll
	// (handle orphelin / WAL #7659). Les claims watcher ont déjà été drainés par
	// watcherDaemon.Stop(), les claims auto par schedulerWG.Wait() ci-dessus.
	//
	// BeginShutdown() AVANT WaitInFlight() : fige le gate pour qu'aucun TryClaim
	// (donc gateWG.Add) ne puisse survenir concurremment au drain — sinon data race
	// WaitGroup + retour prématuré de Wait. Un sync HTTP arrivé après ce point est
	// refusé (409) sans poser de claim.
	if watcherDaemon != nil {
		gate := autoScheduler.Gate()
		gate.BeginShutdown()
		gateDone := make(chan struct{})
		go func() {
			gate.WaitInFlight()
			close(gateDone)
		}()
		select {
		case <-gateDone:
			slog.Debug("gate: tous les claims cross-source libérés")
		case <-time.After(5 * time.Second):
			slog.Warn("gate: timeout WaitInFlight — un sync HTTP est peut-être encore en vol")
		}
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

	// CHECKPOINT synchrone final sur shared_social.duckdb — vide le WAL avant
	// que duckdb.CloseAll ferme les connexions. Essentiel pour le hot-reload Air
	// (bug DuckDB #7659 : replay WAL au reopen peut échouer si le WAL contient
	// des ops non-checkpointées). Sans risque ici : tous les workers sont arrêtés
	// (workerWG.Wait()), aucune écriture concurrente possible.
	if socialDBForCheckpoint, ok := duckdb.LookupCachedDB(sharedSocialPath); ok {
		ckptCtx, ckptCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if _, ckptErr := socialDBForCheckpoint.SQLDb().ExecContext(ckptCtx, "CHECKPOINT"); ckptErr != nil {
			slog.Warn("shared_social: CHECKPOINT final au shutdown échoué (non-fatal — WAL sera rejoué au prochain boot)",
				"err", ckptErr)
		} else {
			slog.Info("shared_social: CHECKPOINT final au shutdown terminé")
		}
		ckptCancel()
	}

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

// ensureWarehouseDir crée le répertoire warehouse du titre s'il n'existe pas.
//
// metadata/shared/shared_pve/shared_social vivent tous dans ce dossier.
// runMigrations les ouvre via OpenReadWrite, qui crée le FICHIER .duckdb mais
// PAS le dossier parent (cf. initGlobalXuidAliasesSchema qui MkdirAll
// explicitement avant OpenReadWrite). Sur un clone frais le dossier est absent :
// les migrations échouent puis l'ouverture du provider shared fait os.Exit(1).
//
// Les autres dossiers runtime (players, sessions, cache, WAL, watcher_tokens)
// sont créés à la demande par leurs consommateurs respectifs (profile_service,
// session.NewStore, jobs.Store, persist.BatchQueue, MultiUserTokenStore).
func ensureWarehouseDir(pr *title.PathResolver, titleSlug string) error {
	dir := pr.WarehouseDir(titleSlug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("ensureWarehouseDir %s: %w", dir, err)
	}
	return nil
}

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
		halomigrations.RegisterPrestigeSeedMigration(prestigeConfigDir)
	}

	// Seed Milestones catalogue (Phase 4 plan stabilisation 2026-05-22) via
	// migration backfill multi-titres. Pattern identique à Prestige.
	// configTitlesRoot = parent du dossier titre (ex: config/titles/) — itère
	// sur tous les `<slug>/milestones/catalog.toml` présents.
	// Cf. AUDIT_ASCENSION_PIPELINE_DISCONNECTED_2026-05-21 §4 cause A.
	if prestigeConfigDir != "" {
		configTitlesRoot := filepath.Dir(prestigeConfigDir)
		halomigrations.RegisterMilestonesSeedMigration(configTitlesRoot)
	}

	// Phase 1.5.1 B (ADR 0025) : enregistre les migrations title-owned (Halo
	// Infinite) auprès du runner, avant tout RunForDB. Vide tant qu'aucun step
	// n'a été déplacé hors du package migration (no-op) ; se remplit en b3.
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	// ROOT FIX assets Halo 5 : enregistre le set de migrations h5 (metadata ISOLÉE
	// — référentiels h5 propres, zéro pollution HINF ; shared/player/… hérités du
	// fallback HINF via OwnsTarget). DOIT précéder provisionAdditionalTitle(halo_5).
	// Le set h5 possède SON milestone_catalog (schéma + seed) — il ne retombe pas
	// sur le seed global multi-titres ; on injecte la racine config/titles/ AVANT
	// Register pour que le seed h5 trouve config/titles/halo_5/milestones/catalog.toml.
	if prestigeConfigDir != "" {
		halo5migrations.SetMilestonesSeedRoot(filepath.Dir(prestigeConfigDir))
	}
	halo5migrations.Register()
	// MT-07 : source title-owned des libellés de rangs de carrière (seed offline).
	migration.SetCareerRankTranslationsProvider(halomigrations.CareerRankTranslations)
	// MT-15 : classifier LUSR title-owned (pair_name → chaîne TrueSkill). GetLUSRChain
	// panique si non posé (fail-loud) — protège le chemin de scoring live.
	syncpkg.SetLUSRChainClassifier(skillchain.ClassifyLUSRChain)
	// MT-15+ : classifier LUSR title-aware pour Halo 5 (pas de pair_name → chaîne
	// unique h5_arena). Le seam GetLUSRChainForTitle route h5 vers ce classifier ;
	// les autres titres gardent le défaut Infinite. Sans ça, h5 collapserait tous
	// ses modes dans arena_slayer (classifier Infinite sur pair_name vide).
	syncpkg.SetLUSRChainClassifierForTitle(halo5.TitleSlug, halo5.ClassifyLUSRChain)

	// 1. metadata.duckdb
	metaDB, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		return fmt.Errorf("open metadata rw: %w", err)
	}
	if err := migration.RunForDB(metaDB.SQLDb(), migration.TargetMetadata); err != nil {
		metaDB.Close()
		return fmt.Errorf("metadata migrations: %w", err)
	}
	// Réconciliation idempotente des seeds de traduction (mode_name_tr + playlists
	// FR) : converge les bases dont la migration de seed est "done" mais incomplète
	// (sous-modes / "Quick Play" restés en anglais). Cf. ReconcileMetadataSeeds.
	if err := halomigrations.ReconcileMetadataSeeds(metaDB.SQLDb()); err != nil {
		metaDB.Close()
		return fmt.Errorf("metadata seed reconcile: %w", err)
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

// provisionAdditionalActiveTitles crée + migre les warehouses des titres
// ADDITIONNELS actifs (slug != DefaultSlug) découverts dans le registre piloté par
// config (MT-16 / day-one 2e titre). Le titre par défaut (halo_infinite) est
// provisionné séparément (chemin byte-identique). Chaque échec est loggé sans
// interrompre le boot — un titre additionnel cassé ne bloque jamais Halo.
func provisionAdditionalActiveTitles(pr *title.PathResolver, reg *title.Registry) {
	for _, td := range reg.Active() {
		// Le titre par défaut (built-in) est provisionné par le chemin Halo
		// byte-identique ci-dessus → on saute son descripteur ici (flag sémantique,
		// pas de comparaison de slug — archlint no_slug_comparison).
		if td.IsDefault {
			continue
		}
		if err := provisionAdditionalTitle(pr, td); err != nil {
			slog.Error("provisioning titre additionnel échoué (non-fatal)", "title", td.Slug, "err", err.Error())
			continue
		}
		slog.Info("titre additionnel provisionné", "title", td.Slug, "status", string(td.Status))
	}
}

// provisionAdditionalTitle crée le warehouse + applique les migrations des DB
// partagées d'un titre additionnel via RunForTitleDB (jeu de migrations du titre
// si enregistré via RegisterMigrationSet, sinon set Halo en fallback). Toutes les
// DB sont isolées par chemin sous data/titles/<slug>/. La DB PvE n'est provisionnée
// que si le titre déclare la capability Firefight (gating par capability, jamais
// par comparaison de slug — archlint no_slug_comparison).
func provisionAdditionalTitle(pr *title.PathResolver, td *title.TitleDescriptor) error {
	slug := td.Slug
	if err := ensureWarehouseDir(pr, slug); err != nil {
		return fmt.Errorf("warehouse dir: %w", err)
	}
	type target struct {
		path string
		kind migration.TargetDB
	}
	targets := []target{
		{pr.MetadataDBPath(slug), migration.TargetMetadata},
		{pr.SharedDBPath(slug), migration.TargetShared},
		{pr.SharedSocialDBPath(slug), migration.TargetSharedSocial},
	}
	if td.HasCapability(title.CapFirefight) {
		targets = append(targets, target{pr.SharedPVEDBPath(slug), migration.TargetSharedPvE})
	}
	for _, t := range targets {
		db, err := duckdb.OpenReadWrite(t.path)
		if err != nil {
			return fmt.Errorf("open %s: %w", t.path, err)
		}
		if err := migration.RunForTitleDB(db.SQLDb(), slug, t.kind); err != nil {
			db.Close()
			return fmt.Errorf("migrate %s (%s): %w", t.kind, t.path, err)
		}
		db.Close()
	}
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

	// Callback de persistance du RT rotaté (ADR 0023) :
	//  1. PRIORITÉ — écriture dans MultiUserTokenStore (source canonique)
	//  2. Compat — écriture aussi dans sync_meta DuckDB (legacy, retiré Phase 5)
	//
	// xuid résolu via store.LoadByGamertag (l'entrée a été créée par migration
	// Phase 2 ou par Discovery) — fallback config.LoadPlayers si pas en store.
	// Best-effort : une erreur sur l'une des écritures n'interrompt pas l'autre.
	authStoreForCallback := auth.NewMultiUserTokenStore(pr.WatcherTokensDir())
	onRotated := func(ctx context.Context, gamertag, newRT string) error {
		xuid := resolveXUIDForRotation(ctx, cfg, authStoreForCallback, gamertag)
		if xuid != "" {
			if err := authStoreForCallback.UpdateOAuthRefreshToken(xuid, newRT); err != nil {
				slog.WarnContext(ctx, "onRotated: écriture store échouée",
					"gamertag", gamertag, "xuid", xuid, "err", err)
			}
			// Le RT a tourné → la chaîne d'auth a changé : le cache process des
			// HaloTokens (Spartan/Clearance, TTL 50min) peut désormais servir des
			// tokens dérivés de l'ANCIENNE chaîne → 401 sur les fetchs live
			// token-gated (career rank/XP, challenges, battle pass, CSR, identité
			// Explorer). On le purge pour forcer une re-dérivation fraîche au
			// prochain enrichWithHaloTokens. Incident 2026-06-14 (post-consolidation
			// client ID : seuls les CLI token-capture/import invalidaient ce cache).
			halo.InvalidateCachedPlayerTokens(xuid)
		} else {
			slog.WarnContext(ctx, "onRotated: xuid introuvable, store non mis à jour",
				"gamertag", gamertag)
		}

		// Compat DuckDB (sera retiré Phase 5 quand Phase 4 sera stabilisée).
		// Comptes token-only (watchers) : pas de player DB → le store suffit.
		dbPath := pr.PlayerDBPath(title.DefaultSlug, gamertag)
		if _, statErr := os.Stat(dbPath); statErr != nil {
			slog.DebugContext(ctx, "onRotated: pas de player DB, écriture compat sync_meta ignorée",
				"gamertag", gamertag)
			return nil
		}
		db, err := duckdb.OpenReadWriteShared(dbPath)
		if err != nil {
			return fmt.Errorf("open player db: %w", err)
		}
		defer db.Close() //nolint:errcheck // ref-count : best-effort
		return duckdb.WriteOAuthRefreshToken(ctx, db, newRT)
	}

	// Persistance des transitions reauth + du dernier échec OAuth permanent
	// (dashboard admin « Santé des tokens », plan anti-bruit 2026-06-11).
	// Best-effort : un échec d'écriture store ne casse pas le resolve.
	onReauth := func(ctx context.Context, gamertag, xuid string, required bool) {
		if xuid == "" {
			return
		}
		if required {
			if _, err := authStoreForCallback.MarkReauthRequired(xuid, gamertag); err != nil {
				slog.WarnContext(ctx, "onReauth: écriture store échouée",
					"gamertag", gamertag, "err", err)
			}
			return
		}
		if err := authStoreForCallback.ClearReauthRequired(xuid); err != nil {
			slog.WarnContext(ctx, "onReauth: clear store échoué",
				"gamertag", gamertag, "err", err)
		}
	}
	onAuthError := func(ctx context.Context, gamertag, xuid, class, msg string) {
		if xuid == "" {
			return
		}
		var err error
		if class == "" {
			err = authStoreForCallback.ClearAuthError(xuid)
		} else {
			err = authStoreForCallback.RecordAuthError(xuid, gamertag, class, msg)
		}
		if err != nil {
			slog.WarnContext(ctx, "onAuthError: écriture store échouée",
				"gamertag", gamertag, "err", err)
		}
	}

	resolver := pool.NewResolverWithCallbacks(tokenProvider, 0, pool.ResolverCallbacks{ // 0 = default TTL ~3h30
		OnRotated:   onRotated,
		OnReauth:    onReauth,
		OnAuthError: onAuthError,
	})
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
	haloPool pool.Pool,
	// autoScheduler fournit BuildEngine pour le câblage syncTrigger.
	// Source of truth UNIQUE du wiring engine (cf. trigger.go godoc + ADR
	// incident 2026-05-26). Doit être non-nil en production.
	autoScheduler *scheduler.AutoSyncScheduler,
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
	//    valide, sinon tente un OAuth v2 refresh depuis :
	//    (a) MultiUserTokenStore (canonique, ADR 0023)
	//    (b) tokens.RefreshToken (legacy watcher_tokens.json)
	//    (c) SPNKR_OAUTH_REFRESH_TOKEN_<XSTSGamertag> (.env.local DEPRECATED)
	//    Persiste la rotation dans le multi-store en priorité, puis le legacy.
	multiStore := auth.NewMultiUserTokenStore(title.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
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

	// Exclure les couples (joueur, titre) en pause (sync_enabled=false) : pas de
	// tracking live. Les données restent sur disque, réactivables via les réglages.
	playerSummaries := domain.SyncablePlayers(players)
	if len(playerSummaries) == 0 {
		slog.Info("watcher: tous les joueurs en pause, daemon désactivé")
		return nil
	}

	// Registre de titres PARTAGÉ (MT-16 : scheduler/watcher voient les titres
	// additionnels config → sync écrit dans leurs DB isolées via PMT-3).
	titleReg := title.DefaultRegistry()

	// Sync trigger (in-process). On câble explicitement l'engineFactory sur
	// scheduler.BuildEngine — c'est ce qui garantit la parité runtime entre
	// le path watcher (Coordinator → Trigger.RunSync) et le path scheduler
	// (auto_sync.defaultRunnerFactory). Sans ça, le SyncEngine créé par le
	// watcher tombe en mode legacy : pas de SharedProvider → conflit
	// "different configuration" sur shared_matches_v2.duckdb (incident
	// 2026-05-26 23:05+).
	syncTrigger := syncpkg.NewTrigger(cfg.RepoRoot, &staticTokenProvider{tokens: *tokens}, domain.SyncOptions{
		MatchType:        "matchmaking",
		MaxMatches:       25,
		WithParticipants: true,
		WithMedals:       true,
		// Le chemin live (watcher) DOIT récupérer les highlight events au 1er
		// passage : ils alimentent highlight_events → killer_victim_pairs →
		// weapon_kills (frags par arme). Sans ce flag (zéro-value false), le
		// watcher insérait registry+participants SANS events ; le scheduler
		// (DefaultSyncOptions, true) ne pouvait pas rattraper car son delta
		// s'arrête sur le match déjà "connu". Le heal events qui masquait ce
		// trou a été décommissionné le 2026-06-01 → events_loaded=false figé
		// sur tous les matchs depuis. Cf. .ai/thought_log 2026-06-04.
		WithHighlightEvents: true,
		RequestsPerSecond:   5,
	}).WithEngineFactory(autoScheduler.BuildEngine).
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

	// MatchFetcher pour le polling Halo API (/hi/players/xuid(N)/matches) du
	// MatchPoller. Réutilise le pool de tokens auto-sync : endpoint public
	// (PolicyAnyPublic), quota par-token Microsoft → partage le rate-limit
	// avec le scheduler évite les 429 inutiles. Si le pool est absent (mode
	// dégradé sans credential), le fetcher reste nil et le garde-fou dans
	// PlayerWatcher.startPoller logge un Warn-once sans paniquer.
	var matchFetcher watcher.MatchFetcher
	if haloPool != nil {
		pooled := syncpkg.NewPooledHaloClient(haloPool, "", "", 5)
		matchFetcher = watcher.NewHaloMatchFetcher(pooled)
		slog.Info("watcher: MatchFetcher branché sur le pool auto-sync")
	} else {
		slog.Warn("watcher: pool auto-sync absent — MatchPoller désactivé (mode dégradé)")
	}

	// daemon est déclaré ici pour permettre à la closure RefreshRTAAuth d'y référer
	// avant que NewDaemon retourne (pattern forward-reference via pointeur).
	var daemon *watcher.Daemon
	watcherPR := title.NewPathResolver(cfg.RepoRoot)
	daemon = watcher.NewDaemon(watcher.DaemonConfig{
		RepoRoot:        cfg.RepoRoot,
		SteamAPIKey:     os.Getenv("STEAM_API_KEY"),
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
	}).WithMultiUserMirror(multiMirror).
		// PR-B : ping Discord opt-in quand le refresh_token meurt (reconnexion requise).
		WithReauthNotify(func(_, gamertag string) {
			notify.NotifyReauthRequired(notify.LoadNotifyConfig(cfg.AppSettingsPath), gamertag)
		})
	go refreshLoop.Run(ctx)

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

// resolveXUIDForRotation retourne le xuid associé à un gamertag pour l'écriture
// du RT rotaté dans MultiUserTokenStore. Délègue à capturecli.ResolveXUIDForRotation
// (helper testable sans cgo). Best-effort : LoadPlayers échoué = "" silencieux.
func resolveXUIDForRotation(ctx context.Context, cfg *config.AppConfig, store *auth.MultiUserTokenStore, gamertag string) string {
	players, err := cfg.LoadPlayers()
	if err != nil {
		slog.DebugContext(ctx, "resolveXUIDForRotation: LoadPlayers erreur", "err", err)
		players = nil
	}
	return capturecli.ResolveXUIDForRotation(ctx, store, players, gamertag)
}

// migrateLegacyAuthTokensAtBoot copie les tokens legacy (env var
// SPNKR_OAUTH_REFRESH_TOKEN_* + sync_meta.oauth_refresh_token / msal_token_cache
// dans la player DB) vers le MultiUserTokenStore unique. Voir ADR 0023.
//
// Idempotent, best-effort. Une erreur sur un joueur (ex. DB inexistante) ne
// bloque pas les autres. Aucun appel HTTP — purement copie de strings entre
// stores. S'exécute AVANT buildAutoSyncPool pour que le Pool trouve le store
// déjà peuplé.
//
// Pour le caller production, la lecture des sources legacy est branchée sur
// auth.EnvRefreshTokenForGamertag (env) + duckdb.OpenReadOnly + Read*JSON.
// La fonction pure de migration vit dans internal/platform/auth/migration.go
// (testable sans dépendance DuckDB).
// migrateDefaultGroupAtBoot crée un groupe par défaut "Mon foyer" depuis l'ancienne
// liste globale friend_gamertags, pour préserver la continuité d'accès au passage au
// modèle multi-groupes. Best-effort + idempotent (no-op si un groupe existe déjà).
// Le propriétaire est l'admin de db_profiles.json ; les amis résolus en xuid via les
// profils connus deviennent membres.
func migrateDefaultGroupAtBoot(ctx context.Context, cfg *config.AppConfig, settingsStore *settings.Store, gs *groupstore.GroupStore) {
	adminGT := cfg.AdminPlayer()
	if adminGT == "" {
		return // aucun admin désigné → migration impossible
	}
	players, err := cfg.LoadPlayers()
	if err != nil {
		return
	}
	byGamertag := make(map[string]string, len(players))
	for _, p := range players {
		if p.XUID != "" {
			byGamertag[strings.ToLower(p.Gamertag)] = p.XUID
		}
	}
	ownerXUID := byGamertag[strings.ToLower(adminGT)]
	if ownerXUID == "" {
		slog.WarnContext(ctx, "groups: migration ignorée — xuid admin introuvable dans db_profiles", "admin", adminGT)
		return
	}

	s, err := settingsStore.Load()
	if err != nil || s == nil {
		return
	}
	var members []domain.GroupMember
	for _, gt := range s.FriendGamertags {
		if xuid := byGamertag[strings.ToLower(gt)]; xuid != "" {
			members = append(members, domain.GroupMember{XUID: xuid, Gamertag: gt})
		}
	}

	created, err := gs.MigrateDefault("Mon foyer", ownerXUID, adminGT, members)
	if err != nil {
		slog.WarnContext(ctx, "groups: migration groupe par défaut échouée (non bloquant)", "err", err)
		return
	}
	if created {
		slog.InfoContext(ctx, "groups: groupe par défaut créé depuis friend_gamertags",
			"owner", adminGT, "members", len(members)+1)
	}
}

func migrateLegacyAuthTokensAtBoot(ctx context.Context, cfg *config.AppConfig) {
	pr := title.NewPathResolver(cfg.RepoRoot)
	store := auth.NewMultiUserTokenStore(pr.WatcherTokensDir())

	players, err := cfg.LoadPlayers()
	if err != nil {
		slog.WarnContext(ctx, "auth_migration: LoadPlayers échoué — migration skipped", "err", err)
		return
	}
	if len(players) == 0 {
		slog.DebugContext(ctx, "auth_migration: aucun joueur configuré, rien à migrer")
		return
	}

	// Convertir players → auth.LegacyPlayer (sans dépendance domain/title dans le package auth).
	legacyPlayers := make([]auth.LegacyPlayer, 0, len(players))
	for _, p := range players {
		legacyPlayers = append(legacyPlayers, auth.LegacyPlayer{
			XUID:         p.XUID,
			Gamertag:     p.Gamertag,
			PlayerDBPath: pr.PlayerDBPath(title.DefaultSlug, p.Gamertag),
		})
	}

	reader := func(rctx context.Context, p auth.LegacyPlayer) (auth.LegacySources, error) {
		out := auth.LegacySources{
			EnvRT: auth.EnvRefreshTokenForGamertag(p.Gamertag),
		}
		// Lecture DuckDB best-effort : DB inexistante = nouveau joueur, on skip.
		if p.PlayerDBPath != "" {
			db, dbErr := duckdb.OpenReadOnly(p.PlayerDBPath)
			if dbErr == nil {
				out.DuckDBRT, _ = duckdb.ReadOAuthRefreshToken(rctx, db)
				out.DuckDBMSAL, _ = duckdb.ReadMSALCacheJSON(rctx, db)
				_ = db.Close()
			}
		}
		return out, nil
	}

	if _, err := auth.MigrateLegacyTokens(ctx, store, legacyPlayers, reader); err != nil {
		slog.WarnContext(ctx, "auth_migration: échec global", "err", err)
	}
}
