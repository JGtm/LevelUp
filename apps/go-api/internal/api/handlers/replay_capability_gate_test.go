// Package handlers — replay_capability_gate_test.go : LA PORTE DE TITRE DES ROUTES /replay*.
//
// Jusqu'au 2026-09-05, les quatre routes du rejeu 2D etaient montees HORS de tout sous-groupe
// de capability, sous un commentaire qui l'assumait : « la disponibilite EST la presence
// d'artefact, pas une declaration de titre ». La consequence, sur un titre sans decodeur de
// film : trois requetes 404 et un etat vide, la ou rien n'aurait du etre servi du tout
// (registre `.ai/AUDIT_V75_DEPUIS_V7.3.0_2026-09-05.md`, constats D1 et L2).
//
// La decision utilisateur du 2026-09-05 pose DEUX portes qui disent deux choses differentes :
// `CapReplay` (« ce TITRE a-t-il un rejeu ? » -> 503) et la presence d'artefact (« CE MATCH
// en a-t-il un ? » -> 404, inchangee). Ce fichier verrouille la premiere, sur le REGISTRE REEL
// du depot — un test qui fabriquerait ses descripteurs prouverait la mecanique du middleware
// (deja couverte par require_capability_test.go), pas la configuration livree.
//
// Le montage reproduit celui de server_apiv1.go, moins le garde local de transport
// (LocalOnlyReplay), qui juge l'adresse de la connexion et non le titre : il est monte APRES
// cette porte-ci et n'a rien a voir avec elle.
package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

// racineDepotReplayGate remonte a la racine du depot par runtime.Caller (jamais un chemin
// absolu : executable en CI).
func racineDepotReplayGate(t *testing.T) string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) a echoue")
	}
	// .../apps/go-api/internal/api/handlers -> racine
	root := filepath.Join(filepath.Dir(ici), "..", "..", "..", "..", "..")
	if _, err := os.Stat(filepath.Join(root, "config", "titles")); err != nil {
		t.Fatalf("racine du depot introuvable : %v", err)
	}
	return root
}

// routeurRejeuGate monte les routes /replay* derriere la porte CapReplay, avec le registre
// REEL (halo_infinite built-in + les titres decouverts sous config/titles/).
func routeurRejeuGate(t *testing.T, serviceAppele *bool) *chi.Mux {
	t.Helper()
	reg := titlePkg.NewRegistryFromConfig(racineDepotReplayGate(t),
		slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))

	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(r chi.Router) {
		r.Use(middleware.RequireCapability(reg, titlePkg.CapReplay))
		// La factory NOTE son appel puis refuse : la porte franchie, le handler rend un 404
		// player_not_found — ce qui suffit a prouver que la requete est passee, sans avoir a
		// simuler tout un ReplayService (dont ce test ne dit rien).
		handlers.NewReplayHandler(func(context.Context, string) (port.ReplayService, error) {
			*serviceAppele = true
			return nil, errors.New("stub: aucun service de rejeu dans ce test")
		}).Mount(r)
	})
	return r
}

// TestReplayRoutes_TitreSansCapability_503 : halo_5 ne declare pas `replay` — les QUATRE
// routes rendent 503 capability_unavailable, et la factory de service n'est jamais appelee.
func TestReplayRoutes_TitreSansCapability_503(t *testing.T) {
	chemins := []string{
		"/players/p/matches/abc123/replay",
		"/players/p/matches/abc123/replay/background",
		"/players/p/matches/abc123/replay/callouts",
		"/players/p/matches/abc123/replay/background.png",
	}
	for _, chemin := range chemins {
		appele := false
		req := httptest.NewRequest(http.MethodGet, chemin, nil)
		req = req.WithContext(ctxkeys.WithTitleSlug(req.Context(), "halo_5"))
		w := httptest.NewRecorder()
		routeurRejeuGate(t, &appele).ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("%s sur halo_5 : status = %d, attendu 503 (corps : %s)", chemin, w.Code, w.Body.String())
			continue
		}
		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Errorf("%s : corps illisible : %v", chemin, err)
			continue
		}
		if body["code"] != "capability_unavailable" {
			t.Errorf("%s : code = %v, attendu capability_unavailable", chemin, body["code"])
		}
		if body["capability"] != "replay" {
			t.Errorf("%s : capability = %v, attendu replay", chemin, body["capability"])
		}
		if appele {
			t.Errorf("%s : le service de rejeu a ete resolu alors que le titre n'a pas la capability", chemin)
		}
	}
}

// TestReplayRoutes_TitreAvecCapability_PasseLaPorte : halo_infinite la declare — la porte
// laisse passer (le service est resolu). Sans ce volet, une porte qui refuserait TOUT
// passerait le test precedent.
func TestReplayRoutes_TitreAvecCapability_PasseLaPorte(t *testing.T) {
	appele := false
	req := httptest.NewRequest(http.MethodGet, "/players/p/matches/abc123/replay", nil)
	req = req.WithContext(ctxkeys.WithTitleSlug(req.Context(), titlePkg.DefaultSlug))
	w := httptest.NewRecorder()
	routeurRejeuGate(t, &appele).ServeHTTP(w, req)

	if w.Code == http.StatusServiceUnavailable {
		t.Fatalf("halo_infinite declare `replay` : la porte a refuse (corps : %s)", w.Body.String())
	}
	if !appele {
		t.Error("halo_infinite : le service de rejeu n'a pas ete resolu — la porte s'est fermee a tort")
	}
}
