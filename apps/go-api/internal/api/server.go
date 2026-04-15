// Package api assemble le routeur HTTP et le serveur.
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// NewRouter construit le routeur chi avec tous les endpoints.
// Construction par injection de dépendances — pas d'état global.
func NewRouter(
	bootRepo port.BootstrapRepository,
	bootSvc *service.BootstrapService,
) http.Handler {
	r := chi.NewRouter()

	// Middlewares transverses
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Compress(5))

	// Health check (pas de préfixe /api/v1 — sondage infrastructurel)
	r.Get("/health", handlers.NewHealthHandler(bootRepo).ServeHTTP)

	// v1 API
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/bootstrap", handlers.NewBootstrapHandler(bootSvc).ServeHTTP)
		r.Get("/players", handlers.NewPlayersHandler(bootSvc).ServeHTTP)
	})

	return r
}
