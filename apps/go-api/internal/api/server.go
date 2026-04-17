// Package api assemble le routeur HTTP et le serveur.
// Sprint 4 : CORS, rate-limit, slog logging, mode démo.
// Sprint 16 : Settings, Setup.
// Sprint 17 : Jobs longs persistants, sync initiale.
// Sprint 37 : Architecture handlers & injection DI via ServiceRegistry.
// Sprint 40 : ContractValidate (dev) + ErrorTracker (Discord 500 + alerting).
package api

import (
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	auth_platform "levelup/go-api/internal/platform/auth"
	platform_duckdb "levelup/go-api/internal/platform/duckdb"
	jobs_platform "levelup/go-api/internal/platform/jobs"
	session_platform "levelup/go-api/internal/platform/session"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// NewRouter construit le routeur chi avec tous les endpoints.
// Construction par injection de dépendances — pas d'état global.
func NewRouter(
	cfg *config.AppConfig,
	bootRepo port.BootstrapRepository,
	bootSvc *service.BootstrapService,
) http.Handler {
	// Sprint 14 : session store + Sprint 15 : attempt store auth
	isProduction := cfg.SessionSecret != "CHANGE_ME_IN_PRODUCTION" // pragma: allowlist secret
	sessionStore := session_platform.NewStore(cfg.SessionDir, session_platform.DefaultTTL, cfg.SessionSecret)
	attemptStore := auth_platform.NewAttemptStore()

	// Sprint 16 : settings store + Sprint 17 : job store
	settingsStore := settings_platform.NewStore(cfg.AppSettingsPath)
	jobsPath := filepath.Join(cfg.RepoRoot, "data", "cache", "jobs.json")
	jobStore := jobs_platform.NewStore(jobsPath)

	r := chi.NewRouter()

	// Middlewares transverses (ordre important)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.CORS(cfg.CORSOrigins))
	r.Use(middleware.CSRF(cfg.CORSOrigins))
	r.Use(middleware.RateLimit(cfg.DemoMode))
	r.Use(middleware.SlogLogger)
	r.Use(chimiddleware.Compress(5))
	r.Use(middleware.WithSession(sessionStore, isProduction))

	// Sprint 44 : TitleExtractor — injecte title_slug dans le contexte.
	titleRegistry := titlePkg.NewRegistry()
	r.Use(middleware.TitleExtractor(titleRegistry))

	// Sprint 40 T1 : validation de contrat (dev mode, no-op si LEVELUP_CONTRACT_VALIDATE != 1).
	r.Use(middleware.ContractValidate)

	// Sprint 40 T2+T3 : error tracking + alerting Discord.
	errorTracker := middleware.NewErrorTracker(middleware.ErrorTrackerConfig{
		WebhookURL: cfg.DiscordWebhookURL,
	})
	r.Use(errorTracker.Middleware)

	// Sprint 35 : shadow mode — compare Go vs Python en parallèle si activé.
	if cfg.ShadowMode == "both" {
		r.Use(middleware.Shadow(middleware.ShadowConfig{PythonURL: cfg.PythonURL}))
	}

	// Sprint 37 : ServiceRegistry — câblage par injection de dépendances.
	reg := NewServiceRegistry(cfg)

	// Gamertag search — service global (shared DB, pas de résolution joueur).
	var gamertagSvc port.GamertagSearchService
	if sharedDB, err := platform_duckdb.OpenReadOnly(config.SharedDBPath(cfg, "")); err != nil {
		slog.Warn("shared DB unavailable for gamertag search", "err", err)
	} else {
		gamertagSvc = platform_duckdb.NewGamertagRepo(sharedDB)
	}

	// Health check (pas de préfixe /api/v1 — sondage infrastructurel)
	r.Get("/health", handlers.NewHealthHandlerWithVersion(bootRepo, cfg.AppVersion).ServeHTTP)

	// v1 API
	r.Route("/api/v1", func(r chi.Router) {
		// Endpoints P0 : bootstrap + liste joueurs
		r.Get("/bootstrap", handlers.NewBootstrapHandler(bootSvc).ServeHTTP)
		r.Get("/players", handlers.NewPlayersHandler(bootSvc).ServeHTTP)

		// Sprint 43 : changelog (markdown brut)
		changelog := handlers.NewChangelogHandler(cfg.RepoRoot)
		r.Get("/changelog", changelog.GetChangelog)

		// Sprint 14 : contexte de session
		sessionHandler := handlers.NewSessionHandler(sessionStore)
		r.Post("/session/context", sessionHandler.PostContext)

		// Sprint 15 : Device Code Flow + authentification Halo
		authHandler := handlers.NewAuthHandler(sessionStore, attemptStore, cfg.DemoMode)
		r.Post("/auth/device-flow/start", authHandler.StartDeviceFlow)
		r.Get("/auth/device-flow/{attempt_id}", authHandler.GetDeviceFlowStatus)

		// Sprint 16 : Settings + Setup joueur
		settingsHandler := handlers.NewSettingsHandler(cfg, settingsStore, jobStore)
		r.Get("/settings", settingsHandler.GetSettings)
		r.Patch("/settings", settingsHandler.PatchSettings)
		r.Post("/settings/media/reset-index", settingsHandler.PostMediaResetIndex)

		setupHandler := handlers.NewSetupHandler(cfg, sessionStore, settingsStore, jobStore,
			service.NewProfileService(cfg.DBProfilesPath, cfg.RepoRoot))
		r.Post("/setup/players", setupHandler.CreatePlayer)
		r.Post("/setup/smoke-test", setupHandler.SmokeTest)

		// Sprint 17 : Jobs longs persistants + sync initiale
		r.Get("/jobs/{job_id}", handlers.NewJobsHandler(jobStore).GetJob)
		r.Post("/sync/initial", handlers.NewSyncHandler(cfg, settingsStore, jobStore).StartInitialSync)

		// Endpoints P1 : pages par joueur (Sprint 37 — DI via ServiceRegistry)
		r.Route("/players/{player_slug}", func(r chi.Router) {
			filters := handlers.NewFiltersHandler(reg.Filters)
			r.Post("/filters/resolve", filters.Resolve)

			mh := handlers.NewMatchHistoryHandler(reg.MatchHistoryCtx)
			r.Post("/pages/match-history/query", mh.Query)

			career := handlers.NewCareerHandler(reg.Career)
			r.Get("/pages/career", career.GetCareer)
			r.Get("/pages/career/top-matches", career.GetTopMatches)
			r.Get("/pages/career/encounters", career.GetEncounters)

			// Sprint 8 : Match View + Explorer
			mv := handlers.NewMatchViewHandler(reg.MatchView)
			r.Get("/matches/{match_id}", mv.GetMatchView)

			explorer := handlers.NewExplorerHandler(reg.ExplorerCtx, reg.MatchHistoryCtx)
			r.Post("/pages/explorer/player-query", explorer.QueryPlayer)

			// Sprint 9 : Sessions
			sessions := handlers.NewSessionsHandler(reg.Sessions)
			r.Get("/pages/sessions", sessions.GetSessions)

			// Sprint 10 : Stats/Séries temporelles
			stats := handlers.NewStatsHandler(reg.Stats)
			r.Post("/pages/stats/query", stats.GetPage)

			// Sprint 11 : Accueil/Home + Battle Pass + Challenges
			home := handlers.NewHomeHandler(reg.HomeCtx)
			r.Get("/pages/home", home.GetHomePage)
			r.Get("/battlepass", home.GetBattlePass)
			r.Get("/challenges", home.GetChallenges)

			// Sprint 12 : Escouade | Sprint 32 : Synthèse → POST
			squad := handlers.NewSquadHandler(reg.SquadCtx)
			r.Get("/pages/squad", squad.GetSquadPage)
			r.Post("/pages/synthesis", squad.GetSynthesisPage)

			// Sprint 13 → Sprint 32 : Citations + Commendations + Médias → POST
			citations := handlers.NewCitationsHandler(reg.CitationsCtx)
			r.Post("/pages/citations", citations.GetCitations)
			r.Post("/pages/commendations", citations.GetCommendations)

			media := handlers.NewMediaHandler(reg.Media)
			r.Post("/pages/media", media.GetMediaLibrary)

			// Sprint 32 : Explorer matches-query + Match History export
			r.Post("/pages/explorer/matches-query", explorer.QueryMatches)
			r.Get("/pages/match-history/export", mh.Export)

			// Sprint 33 : Teammates (contrat FastAPI)
			teammates := handlers.NewTeammatesHandler(reg.TeammatesCtx)
			r.Post("/pages/teammates", teammates.GetPage)

			// Sprint 33 : Timeseries (contrat FastAPI)
			timeseries := handlers.NewTimeseriesHandler(reg.Timeseries)
			r.Post("/pages/timeseries", timeseries.GetPage)

			// Sprint 33 : Session Compare
			sc := handlers.NewSessionCompareHandler(reg.SessionCompare)
			r.Post("/pages/session-compare", sc.Compare)

			// Sprint 33 : Last Match Resolve
			lm := handlers.NewLastMatchHandler(reg.LastMatch)
			r.Post("/pages/last-match/resolve", lm.Resolve)
		})

		// Endpoints P1 : répertoire gamertags
		// Sprint 49 : route inconditionnelle — retourne 503 si shared DB absente.
		r.Get("/directory/gamertags/search", handlers.NewGamertagHandler(gamertagSvc).Search)
	})

	return r
}
