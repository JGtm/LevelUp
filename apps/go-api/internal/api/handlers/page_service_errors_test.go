// Package handlers_test — page_service_errors_test.go : les handlers de pages trient les
// erreurs de service par mapServiceError (plan perf du 2026-09-23, D3.2). Trois
// garanties, toutes routes confondues :
//   - base occupée (verrou d'écriture) : 503 `db_busy` + Retry-After, que le front rejoue ;
//   - client parti (contexte de la requête annulé) : 499 `client_closed`, compté en 4xx
//     par le middleware — jamais en 5xx ;
//   - le contrat OpenAPI documente ces deux réponses pour chacune de ces opérations.
package handlers_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/port"
)

// pageRoute : une opération de page montée avec un service qui échoue sur l'erreur donnée.
type pageRoute struct {
	name    string
	method  string
	path    string // chemin appelé (joueur de test)
	docPath string // chemin documenté dans api/openapi.yaml (relatif à /api/v1)
	body    string
	mount   func(err error) http.Handler
}

// failingCareerService : toutes les méthodes échouent sur err, sauf GetHighlightMatchIDs
// quand highlight est posé (pour atteindre l'enrichissement des matchs marquants).
type failingCareerService struct {
	err       error
	highlight *domain.HighlightMatchesData
}

func (s *failingCareerService) GetCareerPage(context.Context) (domain.CareerPageResponse, error) {
	return domain.CareerPageResponse{}, s.err
}

func (s *failingCareerService) GetTopMatches(context.Context) (domain.CareerTopMatchesResponse, error) {
	return domain.CareerTopMatchesResponse{}, s.err
}

func (s *failingCareerService) GetEncounters(context.Context) (domain.CareerEncountersResponse, error) {
	return domain.CareerEncountersResponse{}, s.err
}

func (s *failingCareerService) GetHighlightMatchIDs(context.Context, domain.HighlightFilterInput) (domain.HighlightMatchesData, error) {
	if s.highlight != nil {
		return *s.highlight, nil
	}
	return domain.HighlightMatchesData{}, s.err
}

func (s *failingCareerService) GetTopEncounters(context.Context) (domain.CareerTopEncountersResponse, error) {
	return domain.CareerTopEncountersResponse{}, s.err
}

func (s *failingCareerService) GetRivals(context.Context) (domain.CareerRivalsResponse, error) {
	return domain.CareerRivalsResponse{}, s.err
}

func (s *failingCareerService) GetCareerCSRs(context.Context, string) (domain.CareerCSRResponse, error) {
	return domain.CareerCSRResponse{}, s.err
}

// newCareerRouterWithHistory monte le handler Carrière AVEC un MatchHistoryService (les
// routes highlight-matches répondent sinon 503 `match_history_unavailable` avant tout appel).
func newCareerRouterWithHistory(svc port.CareerService, mh port.MatchHistoryService) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewCareerHandler(
		func(context.Context, string) (port.CareerService, error) { return svc, nil },
		func(context.Context, string) (port.MatchHistoryService, string, string, error) {
			return mh, testXUID, testGamertag, nil
		},
	)
	r.Route("/players/{player_slug}", func(sub chi.Router) { h.Mount(sub) })
	return r
}

func careerRoute(name, suffix string, svc func(err error) *failingCareerService, mh func(err error) port.MatchHistoryService) pageRoute {
	return pageRoute{
		name: name, method: http.MethodGet,
		path:    "/players/test-player/pages/career" + suffix,
		docPath: "/players/{player_slug}/pages/career" + suffix,
		mount: func(err error) http.Handler {
			return newCareerRouterWithHistory(svc(err), mh(err))
		},
	}
}

// soloPageRoutes : teammates, filtres, synthèse, sessions, séries temporelles, accueil.
func soloPageRoutes() []pageRoute {
	filtersRouter := func(err error) http.Handler {
		return newFiltersRouter(func(context.Context, string) (port.FiltersService, error) {
			return &mockFiltersService{err: err}, nil
		})
	}
	return []pageRoute{
		{
			name: "teammates", method: http.MethodPost, body: `{}`,
			path: "/players/test-player/pages/teammates", docPath: "/players/{player_slug}/pages/teammates",
			mount: func(err error) http.Handler {
				return newTeammatesRouter(func(context.Context, string) (port.TeammatesService, string, string, error) {
					return &mockTeammatesService{pageErr: err}, testXUID, testGamertag, nil
				})
			},
		},
		{
			name: "filters resolve", method: http.MethodPost, body: `{}`, mount: filtersRouter,
			path: "/players/test-player/filters/resolve", docPath: "/players/{player_slug}/filters/resolve",
		},
		{
			name: "filters match-ids", method: http.MethodPost, body: `{}`, mount: filtersRouter,
			path: "/players/test-player/filters/match-ids", docPath: "/players/{player_slug}/filters/match-ids",
		},
		{
			name: "synthesis", method: http.MethodPost, body: `{}`,
			path: "/players/test-player/pages/synthesis", docPath: "/players/{player_slug}/pages/synthesis",
			mount: func(err error) http.Handler {
				return newSynthesisTestRouter(synthesisContextFactory(&mockSynthesisService{err: err}, nil))
			},
		},
		{
			name: "sessions", method: http.MethodGet,
			path: "/players/test-player/pages/sessions", docPath: "/players/{player_slug}/pages/sessions",
			mount: func(err error) http.Handler {
				return newSessionsRouter(func(context.Context, string) (port.SessionsService, error) {
					return &mockSessionsService{err: err}, nil
				})
			},
		},
		{
			name: "session detail", method: http.MethodPost, body: `{}`,
			path: "/players/test-player/pages/sessions/detail", docPath: "/players/{player_slug}/pages/sessions/detail",
			mount: func(err error) http.Handler {
				return newSessionPageRouter(func(context.Context, string) (port.SessionPageService, error) {
					return &mockSessionPageService{err: err}, nil
				})
			},
		},
		{
			name: "timeseries", method: http.MethodPost, body: `{}`,
			path: "/players/test-player/pages/timeseries", docPath: "/players/{player_slug}/pages/timeseries",
			mount: func(err error) http.Handler {
				return newTimeseriesRouter(func(context.Context, string) (port.TimeseriesService, error) {
					return &mockTimeseriesService{pageErr: err}, nil
				})
			},
		},
		{
			name: "home", method: http.MethodGet,
			path: "/players/test-player/pages/home", docPath: "/players/{player_slug}/pages/home",
			mount: func(err error) http.Handler {
				return newHomeRouter(func(ctx context.Context, _ string) (port.HomeService, context.Context, string, string, error) {
					return &mockHomeService{pageErr: err}, ctx, testXUID1, testGamertag, nil
				}, nil)
			},
		},
	}
}

// explorerCareerRoutes : explorer (profil cible, matchs) et les sept routes Carrière.
func explorerCareerRoutes() []pageRoute {
	failing := func(err error) *failingCareerService { return &failingCareerService{err: err} }
	okHistory := func(error) port.MatchHistoryService { return &mockMatchHistoryForExplorer{} }
	highlightRows := func(err error) *failingCareerService {
		return &failingCareerService{err: err, highlight: &domain.HighlightMatchesData{
			Rows: []domain.HighlightMatchIDRow{{MatchID: "m1", Section: 1}},
		}}
	}
	failingHistory := func(err error) port.MatchHistoryService { return &mockMatchHistoryForExplorer{pageErr: err} }
	return []pageRoute{
		{
			name: "explorer player-query", method: http.MethodPost, body: `{"target_gamertag":"opponent"}`,
			path: "/players/test-player/pages/explorer/player-query", docPath: "/players/{player_slug}/pages/explorer/player-query",
			mount: func(err error) http.Handler {
				return newExplorerRouter(
					func(ctx context.Context, _ string) (port.ExplorerService, context.Context, string, string, error) {
						return &mockExplorerService{err: err}, ctx, testXUID, testGamertag, nil
					},
					func(context.Context, string) (port.MatchHistoryService, string, string, error) {
						return &mockMatchHistoryForExplorer{}, testXUID, testGamertag, nil
					})
			},
		},
		{
			name: "explorer matches-query", method: http.MethodPost, body: `{}`,
			path: "/players/test-player/pages/explorer/matches-query", docPath: "/players/{player_slug}/pages/explorer/matches-query",
			mount: func(err error) http.Handler {
				return newExplorerRouter(
					func(ctx context.Context, _ string) (port.ExplorerService, context.Context, string, string, error) {
						return &mockExplorerService{}, ctx, testXUID, testGamertag, nil
					},
					func(context.Context, string) (port.MatchHistoryService, string, string, error) {
						return &mockMatchHistoryForExplorer{pageErr: err}, testXUID, testGamertag, nil
					})
			},
		},
		careerRoute("career", "", failing, okHistory),
		careerRoute("career top-matches", "/top-matches", failing, okHistory),
		careerRoute("career encounters", "/encounters", failing, okHistory),
		careerRoute("career highlight-matches", "/highlight-matches", failing, okHistory),
		careerRoute("career highlight-matches (enrichissement)", "/highlight-matches", highlightRows, failingHistory),
		careerRoute("career top-encounters", "/top-encounters", failing, okHistory),
		careerRoute("career rivals", "/rivals", failing, okHistory),
		careerRoute("career csrs", "/csrs", failing, okHistory),
	}
}

// pageRoutes : toutes les opérations de pages de D3.2.
func pageRoutes() []pageRoute {
	return append(soloPageRoutes(), explorerCareerRoutes()...)
}

// servePage appelle la route ; canceled simule un client parti (contexte annulé).
func servePage(h http.Handler, rt pageRoute, canceled bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(rt.method, rt.path, bytes.NewReader([]byte(rt.body)))
	if rt.body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if canceled {
		ctx, cancel := context.WithCancel(req.Context())
		cancel()
		req = req.WithContext(ctx)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestPageHandlers_DBBusy_503Retryable(t *testing.T) {
	busy := fmt.Errorf("service: %w", dblease.ErrDBLocked)
	for _, rt := range pageRoutes() {
		t.Run(rt.name, func(t *testing.T) {
			rec := servePage(rt.mount(busy), rt, false)
			if rec.Code != http.StatusServiceUnavailable {
				t.Fatalf("statut = %d, attendu 503 : %s", rec.Code, rec.Body.String())
			}
			if code := errorCode(t, rec); code != "db_busy" {
				t.Errorf("code = %q, attendu db_busy", code)
			}
			if ra := rec.Header().Get("Retry-After"); ra != "5" {
				t.Errorf("Retry-After = %q, attendu 5", ra)
			}
		})
	}
}

func TestPageHandlers_ClientClosed_499(t *testing.T) {
	for _, rt := range pageRoutes() {
		t.Run(rt.name, func(t *testing.T) {
			rec := servePage(rt.mount(context.Canceled), rt, true)
			if rec.Code != 499 {
				t.Fatalf("statut = %d, attendu 499 : %s", rec.Code, rec.Body.String())
			}
			if code := errorCode(t, rec); code != "client_closed" {
				t.Errorf("code = %q, attendu client_closed", code)
			}
		})
	}
}

// TestPageClientClosed_CompteEn4xxJamaisEn5xx : derrière le middleware d'accès, un client
// parti incrémente le compteur 4xx, pas le 5xx (countHTTPStatusClass).
func TestPageClientClosed_CompteEn4xxJamaisEn5xx(t *testing.T) {
	rt := pageRoutes()[0]
	h := middleware.SlogLogger(rt.mount(context.Canceled))
	before4xx := observability.LoadCounter("http_status_4xx_total")
	before5xx := observability.LoadCounter("http_status_5xx_total")

	if rec := servePage(h, rt, true); rec.Code != 499 {
		t.Fatalf("statut = %d, attendu 499", rec.Code)
	}
	if d := observability.LoadCounter("http_status_4xx_total") - before4xx; d != 1 {
		t.Errorf("compteur 4xx : +%d, attendu +1", d)
	}
	if d := observability.LoadCounter("http_status_5xx_total") - before5xx; d != 0 {
		t.Errorf("compteur 5xx : +%d, attendu +0 (un client parti n'est pas une panne serveur)", d)
	}
}

// TestPageOperations_DocumentBusyAndClientClosed : le contrat publié (api/openapi.yaml,
// généré depuis le fragment manuel) déclare 499 et 503 pour chaque opération de page qui
// passe par mapServiceError, et les deux réponses partagées existent.
func TestPageOperations_DocumentBusyAndClientClosed(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "api", "openapi.yaml"))
	if err != nil {
		t.Fatalf("lecture du contrat : %v", err)
	}
	var doc struct {
		// any : un élément de chemin peut porter une liste `parameters` à côté des méthodes.
		Paths      map[string]map[string]any `yaml:"paths"`
		Components struct {
			Responses map[string]struct {
				Headers map[string]any `yaml:"headers"`
			} `yaml:"responses"`
		} `yaml:"components"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("contrat illisible : %v", err)
	}
	if _, ok := doc.Components.Responses["ClientClosed"]; !ok {
		t.Error("components.responses.ClientClosed absent")
	}
	if busy, ok := doc.Components.Responses["DbBusy"]; !ok || busy.Headers["Retry-After"] == nil {
		t.Error("components.responses.DbBusy absent ou sans en-tête Retry-After")
	}
	for _, rt := range pageRoutes() {
		op, ok := doc.Paths[rt.docPath][strings.ToLower(rt.method)].(map[string]any)
		if !ok {
			t.Errorf("%s %s : opération absente du contrat", rt.method, rt.docPath)
			continue
		}
		responses, _ := op["responses"].(map[string]any)
		for _, status := range []string{"499", "503"} {
			if _, ok := responses[status]; !ok {
				t.Errorf("%s %s : réponse %s non documentée", rt.method, rt.docPath, status)
			}
		}
	}
}
