// Package api assemble le routeur HTTP et le serveur.
// Sprint 4 : CORS, rate-limit, slog logging, mode démo.
// Sprint 16 : Settings, Setup.
// Sprint 17 : Jobs longs persistants, sync initiale.
package api

import (
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/port"
	auth_platform "levelup/go-api/internal/platform/auth"
	jobs_platform "levelup/go-api/internal/platform/jobs"
	session_platform "levelup/go-api/internal/platform/session"
	settings_platform "levelup/go-api/internal/platform/settings"
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
	r.Use(middleware.RateLimit(cfg.DemoMode))
	r.Use(middleware.SlogLogger)
	r.Use(chimiddleware.Compress(5))
	r.Use(middleware.WithSession(sessionStore, isProduction))

	// Health check (pas de préfixe /api/v1 — sondage infrastructurel)
	r.Get("/health", handlers.NewHealthHandler(bootRepo).ServeHTTP)

	// v1 API
	r.Route("/api/v1", func(r chi.Router) {
		// Endpoints P0 : bootstrap + liste joueurs
		r.Get("/bootstrap", handlers.NewBootstrapHandler(bootSvc).ServeHTTP)
		r.Get("/players", handlers.NewPlayersHandler(bootSvc).ServeHTTP)

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

		setupHandler := handlers.NewSetupHandler(cfg, sessionStore, settingsStore, jobStore)
		r.Post("/setup/players", setupHandler.CreatePlayer)
		r.Post("/setup/smoke-test", setupHandler.SmokeTest)

		// Sprint 17 : Jobs longs persistants + sync initiale
		r.Get("/jobs/{job_id}", handlers.NewJobsHandler(jobStore).GetJob)
		r.Post("/sync/initial", handlers.NewSyncHandler(cfg, settingsStore, jobStore).StartInitialSync)

		// Endpoints P1 : pages par joueur
		r.Route("/players/{player_slug}", func(r chi.Router) {
			filters := handlers.NewFiltersHandler(cfg)
			r.Post("/filters/resolve", filters.Resolve)

			mh := handlers.NewMatchHistoryHandler(cfg)
			r.Post("/pages/match-history/query", mh.Query)

			career := handlers.NewCareerHandler(cfg)
			r.Get("/pages/career", career.GetCareer)
			r.Get("/pages/career/top-matches", career.GetTopMatches)
			r.Get("/pages/career/encounters", career.GetEncounters)

			// Sprint 8 : Match View + Explorer
			mv := handlers.NewMatchViewHandler(cfg)
			r.Get("/matches/{match_id}", mv.GetMatchView)

			explorer := handlers.NewExplorerHandler(cfg)
			r.Post("/pages/explorer/player-query", explorer.QueryPlayer)

			// Sprint 9 : Sessions
			sessions := handlers.NewSessionsHandler(cfg)
			r.Get("/pages/sessions", sessions.GetSessions)

			// Sprint 10 : Stats/Séries temporelles
			stats := handlers.NewStatsHandler(cfg)
			r.Post("/pages/stats/query", stats.GetPage)

			// Sprint 11 : Accueil/Home + Battle Pass + Challenges
			home := handlers.NewHomeHandler(cfg)
			r.Get("/pages/home", home.GetHomePage)
			r.Get("/battlepass", home.GetBattlePass)
			r.Get("/challenges", home.GetChallenges)

			// Sprint 12 : Escouade + Synthèse
			squad := handlers.NewSquadHandler(cfg)
			r.Get("/pages/squad", squad.GetSquadPage)
			r.Get("/pages/synthesis", squad.GetSynthesisPage)

			// Sprint 13 : Citations + Commendations + Médias
			citations := handlers.NewCitationsHandler(cfg)
			r.Get("/pages/citations", citations.GetCitations)
			r.Get("/pages/commendations", citations.GetCommendations)

			media := handlers.NewMediaHandler(cfg)
			r.Get("/pages/media", media.GetMediaLibrary)
		})

		// Endpoints P1 : répertoire gamertags
		r.Get("/directory/gamertags/search", handlers.NewGamertagHandler(cfg).Search)
	})

	return r
}
