// Package handlers_test — citations_capability_gate_test.go : LA PORTE DE TITRE DE
// POST /pages/citations.
//
// La page Citations est alimentée par le moteur de citations CALCULÉES (halo_infinite).
// Halo 5 sert à la place des commendations NATIVES (/pages/commendations) : sans porte, un
// client qui interroge /pages/citations sur halo_5 recevait la page du moteur d'un autre
// titre. La porte est la capability title-level `citations` (503 capability_unavailable) ;
// /pages/commendations, elle, reste ouverte à tous les titres.
//
// Le test s'exécute sur le REGISTRE RÉEL du dépôt (halo_infinite built-in + title.toml
// découverts), pas sur des descripteurs fabriqués : il verrouille la configuration livrée.
package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

// routeurCitationsGate monte le handler avec la porte construite exactement comme
// server_apiv1.go la construit, sur le registre réel.
func routeurCitationsGate(t *testing.T, serviceAppele *bool) *chi.Mux {
	t.Helper()
	reg := titlePkg.NewRegistryFromConfig(racineDepotReplayGate(t),
		slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))

	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(r chi.Router) {
		// La factory note son appel puis refuse : la porte franchie, le handler rend un 404
		// player_not_found, ce qui prouve que la requête est passée sans simuler un service.
		handlers.NewCitationsHandler(func(context.Context, string) (port.CitationsService, string, string, error) {
			*serviceAppele = true
			return nil, "", "", errors.New("stub: aucun service de citations dans ce test")
		}, middleware.RequireCapability(reg, titlePkg.CapCitations)).Mount(r)
	})
	return r
}

func postSurTitre(t *testing.T, chemin, titre string, serviceAppele *bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, chemin, nil)
	req = req.WithContext(ctxkeys.WithTitleSlug(req.Context(), titre))
	w := httptest.NewRecorder()
	routeurCitationsGate(t, serviceAppele).ServeHTTP(w, req)
	return w
}

// TestCitationsRoute_TitreSansCapability_503 : halo_5 ne déclare pas `citations` — la route
// rend 503 capability_unavailable et la factory de service n'est jamais appelée.
func TestCitationsRoute_TitreSansCapability_503(t *testing.T) {
	appele := false
	w := postSurTitre(t, "/players/p/pages/citations", "halo_5", &appele)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("halo_5 : status = %d, attendu 503 (corps : %s)", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("corps illisible : %v", err)
	}
	if body["code"] != "capability_unavailable" {
		t.Errorf("code = %v, attendu capability_unavailable", body["code"])
	}
	if body["capability"] != "citations" {
		t.Errorf("capability = %v, attendu citations", body["capability"])
	}
	if appele {
		t.Error("le service de citations a été résolu alors que le titre n'a pas la capability")
	}
}

// TestCitationsRoute_TitreAvecCapability_PasseLaPorte : halo_infinite la déclare — la porte
// laisse passer (le service est résolu). Sans ce volet, une porte qui refuserait tout
// passerait le test précédent.
func TestCitationsRoute_TitreAvecCapability_PasseLaPorte(t *testing.T) {
	appele := false
	w := postSurTitre(t, "/players/p/pages/citations", titlePkg.DefaultSlug, &appele)

	if w.Code == http.StatusServiceUnavailable {
		t.Fatalf("halo_infinite déclare `citations` : la porte a refusé (corps : %s)", w.Body.String())
	}
	if !appele {
		t.Error("halo_infinite : le service de citations n'a pas été résolu — la porte s'est fermée à tort")
	}
}

// TestCommendationsRoute_RestePubliquePourHalo5 : la porte de /pages/citations ne doit pas
// déborder sur /pages/commendations, que Halo 5 consomme (commendations natives).
func TestCommendationsRoute_RestePubliquePourHalo5(t *testing.T) {
	appele := false
	w := postSurTitre(t, "/players/p/pages/commendations", "halo_5", &appele)

	if w.Code == http.StatusServiceUnavailable {
		t.Fatalf("halo_5 : /pages/commendations a été refusée par la porte de citations (corps : %s)", w.Body.String())
	}
	if !appele {
		t.Error("halo_5 : le service de commendations n'a pas été résolu")
	}
}

// TestNewCitationsHandler_PorteRequise : un montage sans porte de titre est refusé à la
// construction, jamais servi ouvert en silence.
func TestNewCitationsHandler_PorteRequise(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("NewCitationsHandler(…, nil) aurait dû paniquer")
		}
	}()
	handlers.NewCitationsHandler(nil, nil)
}
