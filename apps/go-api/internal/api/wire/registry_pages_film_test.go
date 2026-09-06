package wire

// registry_pages_film_test.go — LE 503 DES DEUX PROJECTIONS DE L'ARTEFACT, DE BOUT EN BOUT.
//
// Ce que ce fichier prouve, et qu'aucun test ne prouvait avant le 2026-09-05 (registre
// `.ai/AUDIT_V75_DEPUIS_V7.3.0_2026-09-05.md`, constat D2) : sur un titre qui ne declare pas
// `film.replay_artifact`, `/objective-events` et `/positions` rendent un 503
// `capability_not_supported` — la ou ils rendaient 200 `[]`, indistinguable d'un match sans
// donnees.
//
// LA CHAINE EST JOUEE ENTIERE, pas simulee : capabilities.toml LIVRES -> porte du wire
// (withFilmArtifactRepos) -> service.MatchViewService -> handler HTTP. Seuls les deux loaders
// sont factices — s'ils sont appeles, c'est que la porte s'est ouverte, et le test le dit.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/analysis/positions"
	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// racineDepotDeCeTest remonte du package a la racine du depot, pour lire les
// config/titles/{slug}/mappings/capabilities.toml LIVRES. Par runtime.Caller, jamais par un
// chemin absolu : c'est ce qui le rend executable en CI.
func racineDepotDeCeTest(t *testing.T) string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) a echoue")
	}
	// .../apps/go-api/internal/api/wire -> racine
	return filepath.Join(filepath.Dir(ici), "..", "..", "..", "..", "..")
}

// loaderTemoin : un loader factice qui note qu'on l'a appele. La porte fermee doit le laisser
// muet.
type loaderTemoin struct{ appele *bool }

func (l loaderTemoin) LoadMatch(context.Context, string) ([]domain.ObjectiveEvent, error) {
	*l.appele = true
	return nil, nil
}

type loaderPositionsTemoin struct{ appele *bool }

func (l loaderPositionsTemoin) LoadMatch(context.Context, string) ([]positions.PlayerPosition, error) {
	*l.appele = true
	return nil, nil
}

// serviceAvecPorte construit un MatchViewService et lui applique LA VRAIE porte du wire, avec
// les capabilities REELLES du titre demande.
func serviceAvecPorte(t *testing.T, slug string, objAppele, posAppele *bool) *service.MatchViewService {
	t.Helper()
	caps, err := games.LoadCapabilityMap(racineDepotDeCeTest(t), slug)
	if err != nil {
		t.Fatalf("LoadCapabilityMap(%s): %v", slug, err)
	}
	return withFilmArtifactRepos(
		service.NewMatchViewService(nil, ""),
		caps,
		loaderTemoin{appele: objAppele},
		loaderPositionsTemoin{appele: posAppele},
	)
}

// routeurMatchView monte les routes de la Match View sur un service donne.
func routeurMatchView(svc port.MatchViewService) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewMatchViewHandler(func(context.Context, string) (port.MatchViewService, error) {
		return svc, nil
	})
	r.Route("/players/{player_slug}", func(r chi.Router) { h.Mount(r) })
	return r
}

// TestFilmArtifactRepos_CleAbsente_503 : une cle ABSENTE de la CapabilityMap ferme la porte
// comme un `not_exposed` explicite.
//
// Les deux « non » se distinguent dans le MANIFESTE (halo_5 declare son refus, la plupart des
// titres n'en diraient rien) mais doivent se comporter pareil dans le CODE : `Has` est faux
// dans les deux cas. Ce test le fige sur la porte reelle, sans passer par un TOML — c'est le
// seul moyen d'exercer l'absence pure, qu'aucun titre livre ne produit aujourd'hui.
func TestFilmArtifactRepos_CleAbsente_503(t *testing.T) {
	for _, chemin := range []string{"objective-events", "positions"} {
		objAppele, posAppele := false, false
		svc := withFilmArtifactRepos(
			service.NewMatchViewService(nil, ""),
			games.CapabilityMap{games.CapMatchHistory: games.CapSupported}, // aucune cle film.*
			loaderTemoin{appele: &objAppele},
			loaderPositionsTemoin{appele: &posAppele},
		)

		req := httptest.NewRequest(http.MethodGet, "/players/p/matches/abc123/"+chemin, nil)
		w := httptest.NewRecorder()
		routeurMatchView(svc).ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("/%s sans la cle : status = %d, attendu 503 (corps : %s)", chemin, w.Code, w.Body.String())
		}
		if objAppele || posAppele {
			t.Errorf("/%s : un loader du film a ete appele alors que la cle est absente", chemin)
		}
	}
}

// TestFilmArtifactRepos_TitreSansCapability_503 : halo_5 ne declare pas
// `film.replay_artifact` -> les deux routes rendent 503 capability_not_supported, et aucun
// loader n'est appele.
func TestFilmArtifactRepos_TitreSansCapability_503(t *testing.T) {
	for _, chemin := range []string{"objective-events", "positions"} {
		objAppele, posAppele := false, false
		svc := serviceAvecPorte(t, "halo_5", &objAppele, &posAppele)

		req := httptest.NewRequest(http.MethodGet, "/players/p/matches/abc123/"+chemin, nil)
		w := httptest.NewRecorder()
		routeurMatchView(svc).ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("/%s sur halo_5 : status = %d, attendu 503 (corps : %s)", chemin, w.Code, w.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("/%s : corps illisible : %v", chemin, err)
		}
		if body["code"] != "capability_not_supported" {
			t.Errorf("/%s : code = %v, attendu capability_not_supported", chemin, body["code"])
		}
		if objAppele || posAppele {
			t.Errorf("/%s : un loader du film a ete appele alors que le titre n'a pas la cle", chemin)
		}
	}
}

// TestFilmArtifactRepos_TitreAvecCapability_200 : un titre qui declare la cle -> les deux
// loaders sont câbles et servent (200). Sans ce volet, une porte qui refuserait TOUT passerait
// le test precedent.
//
// DEUX TITRES, et le second n'est pas decoratif : `synthetic_title_b` declare
// `film.replay_artifact` = supported ET ses cinq derives `not_exposed`. Il prouve donc que
// c'est bien CETTE cle-la qui ouvre les deux loaders, et non « le titre a quelque chose du
// film » (revue C-R1, constat C3 : la fixture ne prouvait rien).
func TestFilmArtifactRepos_TitreAvecCapability_200(t *testing.T) {
	cas := []struct {
		slug   string
		chemin string
		lu     func(obj, pos bool) bool
	}{
		{"halo_infinite", "objective-events", func(obj, _ bool) bool { return obj }},
		{"halo_infinite", "positions", func(_, pos bool) bool { return pos }},
		{"synthetic_title_b", "objective-events", func(obj, _ bool) bool { return obj }},
		{"synthetic_title_b", "positions", func(_, pos bool) bool { return pos }},
	}
	for _, c := range cas {
		objAppele, posAppele := false, false
		svc := serviceAvecPorte(t, c.slug, &objAppele, &posAppele)

		req := httptest.NewRequest(http.MethodGet, "/players/p/matches/abc123/"+c.chemin, nil)
		w := httptest.NewRecorder()
		routeurMatchView(svc).ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("/%s sur %s : status = %d, attendu 200 (corps : %s)", c.chemin, c.slug, w.Code, w.Body.String())
		}
		if !c.lu(objAppele, posAppele) {
			t.Errorf("/%s sur %s : le loader n'a pas ete appele — la porte s'est fermee a tort", c.chemin, c.slug)
		}
	}
}
