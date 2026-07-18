//go:build cgo

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/wire"
)

// TestPrestigePlayerSlugCtx_StampsPathSlug : le middleware place le {player_slug}
// du chemin dans le contexte de résolution Prestige. C'est LE mécanisme qui ancre
// le LazyPrestigeService sur le player DB du chemin (ownership-gardé par
// ownershipMW en amont) — donc qui clôt le BOLA objet-level des défis/arcs perso
// (isolation par DB : un {id} d'un autre joueur n'existe pas dans ce DB → 404) et
// répare les routes {id} qui, sans slug en contexte, tombaient en
// ErrPlayerNotResolved. Garde-rail anti-régression : retirer ce câblage
// rouvrirait les deux problèmes.
func TestPrestigePlayerSlugCtx_StampsPathSlug(t *testing.T) {
	var seen string
	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Use(prestigePlayerSlugCtx)
		r.Get("/probe", func(w http.ResponseWriter, req *http.Request) {
			seen = wire.PlayerSlugFromContext(req.Context())
			w.WriteHeader(http.StatusOK)
		})
	})
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/players/alice/probe", nil))
	if seen != "alice" {
		t.Errorf("PlayerSlugFromContext=%q, want alice (résolution Prestige ancrée sur le slug du chemin)", seen)
	}
}
