// Package api assemble le routeur HTTP et le serveur.
// Sprint 4 : CORS, rate-limit, slog logging, mode démo.
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
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
	r := chi.NewRouter()

	// Middlewares transverses (ordre important)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.CORS(cfg.CORSOrigins))
	r.Use(middleware.RateLimit(cfg.DemoMode))
	r.Use(middleware.SlogLogger)
	r.Use(chimiddleware.Compress(5))

	// Health check (pas de préfixe /api/v1 — sondage infrastructurel)
	r.Get("/health", handlers.NewHealthHandler(bootRepo).ServeHTTP)

	// v1 API
	r.Route("/api/v1", func(r chi.Router) {
		// Endpoints P0 : bootstrap + liste joueurs
		r.Get("/bootstrap", handlers.NewBootstrapHandler(bootSvc).ServeHTTP)
		r.Get("/players", handlers.NewPlayersHandler(bootSvc).ServeHTTP)

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
		})

		// Endpoints P1 : répertoire gamertags
		r.Get("/directory/gamertags/search", handlers.NewGamertagHandler(cfg).Search)
	})

	return r
}
