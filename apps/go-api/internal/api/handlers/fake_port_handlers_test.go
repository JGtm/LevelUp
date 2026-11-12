// Package handlers_test — fake_port_handlers_test.go : les handlers qui tiennent une interface de
// internal/port, exercés par httptest avec un faux port. Ils fixent ce que la couche HTTP fait de la
// réponse et des erreurs du port : codes, corps, session et paramètres transmis.
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dbprofiles"
	"levelup/go-api/internal/port"
)

// ─── Bootstrap et liste des joueurs ──────────────────────────────────────────

type fpBootstrap struct {
	resp    *domain.BootstrapResponse
	players *domain.PlayersListResponse
	err     error
	seen    *domain.SessionData
}

func (f *fpBootstrap) Build(_ context.Context, sess *domain.SessionData) (*domain.BootstrapResponse, error) {
	f.seen = sess
	return f.resp, f.err
}

func (f *fpBootstrap) BuildPlayersList(_ context.Context, sess *domain.SessionData) (*domain.PlayersListResponse, error) {
	f.seen = sess
	return f.players, f.err
}

func fpGet(h http.Handler, path string, sess *domain.SessionData) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if sess != nil {
		req = req.WithContext(middleware.InjectSession(req.Context(), sess))
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestBootstrapHandler_FauxPort(t *testing.T) {
	slug := "jgtm"
	f := &fpBootstrap{
		resp:    &domain.BootstrapResponse{AuthState: "ready", SetupState: "ready"},
		players: &domain.PlayersListResponse{Items: []domain.PlayerSummary{{PlayerSlug: slug}}, DefaultPlayerSlug: &slug},
	}
	r := chi.NewRouter()
	handlers.NewBootstrapHandler(f).Mount(r)
	handlers.NewPlayersHandler(f).Mount(r)
	sess := &domain.SessionData{SessionID: "s1"}

	w := fpGet(r, "/bootstrap", sess)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"auth_state":"ready"`) {
		t.Fatalf("/bootstrap : %d %s", w.Code, w.Body.String())
	}
	if f.seen != sess {
		t.Error("/bootstrap : la session de la requête n'est pas transmise au port")
	}
	f.seen = nil
	if w = fpGet(r, "/players", sess); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"default_player_slug":"jgtm"`) {
		t.Fatalf("/players : %d %s", w.Code, w.Body.String())
	}
	if f.seen != sess {
		t.Error("/players : la session de la requête n'est pas transmise au port")
	}

	f.err = errors.New("base illisible : détail interne")
	for path, code := range map[string]string{"/bootstrap": "bootstrap_error", "/players": "players_error"} {
		w = fpGet(r, path, sess)
		if w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), code) {
			t.Errorf("%s en erreur : %d %s, attendu 500 %s", path, w.Code, w.Body.String(), code)
		}
		if strings.Contains(w.Body.String(), "détail interne") {
			t.Errorf("%s : le texte de l'erreur du port part au client : %s", path, w.Body.String())
		}
	}
}

// ─── Présence ────────────────────────────────────────────────────────────────

type fpPresence struct {
	snap *domain.PresenceSnapshot
	seen *domain.SessionData
}

func (f *fpPresence) GetSnapshot(_ context.Context, sess *domain.SessionData) *domain.PresenceSnapshot {
	f.seen = sess
	return f.snap
}

func TestPresenceHandler_FauxPort(t *testing.T) {
	f := &fpPresence{snap: &domain.PresenceSnapshot{
		Players:       []domain.PlayerPresence{{PlayerSlug: "jgtm", Gamertag: "JGtm", InGame: true}},
		FriendsInGame: 2,
	}}
	r := chi.NewRouter()
	handlers.NewPresenceHandler(f).Mount(r)
	sess := &domain.SessionData{SessionID: "s1"}

	w := fpGet(r, "/presence", sess)
	if w.Code != http.StatusOK {
		t.Fatalf("statut %d : %s", w.Code, w.Body.String())
	}
	var got domain.PresenceSnapshot
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.FriendsInGame != 2 || len(got.Players) != 1 || !got.Players[0].InGame {
		t.Errorf("corps = %+v", got)
	}
	if f.seen != sess {
		t.Error("la session de la requête n'est pas transmise au port")
	}
}

// ─── Import depuis le site ───────────────────────────────────────────────────

type fpSiteImport struct {
	startErr error
	status   domain.SiteImportStatus
	got      port.SiteImportRequest
}

func (f *fpSiteImport) Start(req port.SiteImportRequest) error { f.got = req; return f.startErr }
func (f *fpSiteImport) Status() domain.SiteImportStatus        { return f.status }

func fpPost(h http.Handler, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestSiteImportHandler_FauxPort(t *testing.T) {
	token := strings.Repeat("x", 20) // le port ne lit pas le jeton : seule sa longueur compte pour la validation
	body := `{"url":"https://example.org","token":"` + token + `","gamertag":"Spartan"}`
	f := &fpSiteImport{status: domain.SiteImportStatus{Phase: "downloading", Matches: 7}}
	r := chi.NewRouter()
	handlers.NewSiteImportHandler(f).Mount(r)

	w := fpPost(r, "/import/site", body)
	if w.Code != http.StatusAccepted || !strings.Contains(w.Body.String(), `"phase":"downloading"`) {
		t.Fatalf("démarrage : %d %s", w.Code, w.Body.String())
	}
	if f.got.URL != "https://example.org" || f.got.Token != token || f.got.Gamertag != "Spartan" {
		t.Errorf("demande transmise au port : %+v", f.got)
	}
	if w = fpGet(r, "/import/site", nil); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"matches":7`) {
		t.Errorf("état : %d %s", w.Code, w.Body.String())
	}

	cases := []struct {
		err    error
		status int
		code   string
	}{
		{fmt.Errorf("déjà: %w", port.ErrSiteImportRunning), http.StatusConflict, "import_already_running"},
		{port.ErrSiteImportNoPlayer, http.StatusConflict, "import_no_tracked_player"},
		{port.ErrSiteImportAmbiguous, http.StatusBadRequest, "import_player_ambiguous"},
		{port.ErrSiteImportPlayerNoXID, http.StatusConflict, "import_player_without_xuid"},
		{errors.New("service: site import not bound"), http.StatusServiceUnavailable, "import_unavailable"},
	}
	for _, c := range cases {
		f.startErr = c.err
		w = fpPost(r, "/import/site", body)
		if w.Code != c.status || !strings.Contains(w.Body.String(), c.code) {
			t.Errorf("%v : %d %s, attendu %d %s", c.err, w.Code, w.Body.String(), c.status, c.code)
		}
	}
}

// ─── Titre d'un joueur : pause, réactivation, purge ──────────────────────────

type fpTitleProfiles struct {
	mu      sync.Mutex
	err     error
	removed bool
	calls   []string
}

func (f *fpTitleProfiles) SetTitleSyncEnabled(titleSlug, gamertag string, enabled bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fmt.Sprintf("sync %s %s %v", titleSlug, gamertag, enabled))
	return f.err
}

func (f *fpTitleProfiles) PurgeTitleData(titleSlug, gamertag string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fmt.Sprintf("purge %s %s", titleSlug, gamertag))
	return f.removed, f.err
}

func TestTitleSyncHandler_FauxPort(t *testing.T) {
	f := &fpTitleProfiles{removed: true}
	r := chi.NewRouter()
	r.Route("/players/{player_slug}/titles/{slug}", func(sub chi.Router) { handlers.NewTitleSyncHandler(f).Mount(sub) })
	do := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	if w := do(http.MethodPatch, "/players/Spartan/titles/halo_5/sync", `{"enabled":false}`); w.Code != http.StatusOK ||
		!strings.Contains(w.Body.String(), `"sync_enabled":false`) {
		t.Fatalf("pause : %d %s", w.Code, w.Body.String())
	}
	if w := do(http.MethodDelete, "/players/Spartan/titles/halo_5/data", ""); w.Code != http.StatusOK ||
		!strings.Contains(w.Body.String(), `"data_removed":true`) {
		t.Fatalf("purge : %d %s", w.Code, w.Body.String())
	}
	if got := strings.Join(f.calls, ";"); got != "sync halo_5 Spartan false;purge halo_5 Spartan" {
		t.Errorf("appels du port : %s", got)
	}

	for _, c := range []struct {
		err    error
		status int
		code   string
	}{
		{fmt.Errorf("x: %w", dbprofiles.ErrEntryNotFound), http.StatusNotFound, "title_profile_not_found"},
		{dbprofiles.ErrLastActiveTitle, http.StatusConflict, "last_active_title"},
		{errors.New("disque"), http.StatusInternalServerError, "toggle_error"},
	} {
		f.err = c.err
		if w := do(http.MethodPatch, "/players/Spartan/titles/halo_5/sync", `{"enabled":true}`); w.Code != c.status ||
			!strings.Contains(w.Body.String(), c.code) {
			t.Errorf("%v : %d %s, attendu %d %s", c.err, w.Code, w.Body.String(), c.status, c.code)
		}
	}
}

// ─── Engagement ──────────────────────────────────────────────────────────────

type fpEngagement struct {
	matchErr error
}

func (f *fpEngagement) GetMatchEngagement(context.Context, string) (*domain.EngagementScoreResult, error) {
	return &domain.EngagementScoreResult{}, f.matchErr
}
func (f *fpEngagement) GetEngagementProfile(context.Context) ([]domain.EngagementProfile, error) {
	return nil, nil
}
func (f *fpEngagement) GetTimeseries(context.Context, domain.FilterContextInput, int) (*domain.EngagementTimeseriesResponse, error) {
	return &domain.EngagementTimeseriesResponse{}, nil
}
func (f *fpEngagement) GetSquadSession(context.Context, []string, []domain.EngagementCoefficient) (*domain.SquadEngagementSession, error) {
	return &domain.SquadEngagementSession{}, nil
}
func (f *fpEngagement) RecomputeCoefficients(context.Context) (*domain.RecomputeReport, error) {
	return &domain.RecomputeReport{}, nil
}

func TestEngagementHandler_FauxPort_ErreursDuMatch(t *testing.T) {
	f := &fpEngagement{}
	factory := func(_ context.Context, slug string) (port.PlayerEngagementService, error) {
		if slug != "spartan" {
			return nil, errors.New("joueur inconnu")
		}
		return f, nil
	}
	r := chi.NewRouter()
	r.Route("/players/{player_slug}", func(sub chi.Router) { handlers.NewEngagementHandler(factory).Mount(sub) })

	if w := fpGet(r, "/players/spartan/matches/m1/engagement", nil); w.Code != http.StatusOK {
		t.Fatalf("nominal : %d %s", w.Code, w.Body.String())
	}
	if w := fpGet(r, "/players/inconnu/matches/m1/engagement", nil); w.Code != http.StatusNotFound ||
		!strings.Contains(w.Body.String(), "player_not_found") {
		t.Errorf("joueur inconnu : %d %s", w.Code, w.Body.String())
	}
	for _, c := range []struct {
		err    error
		status int
		code   string
	}{
		{port.ErrEngagementMatchNotFound, http.StatusNotFound, "match_not_found"},
		{fmt.Errorf("x: %w", port.ErrEngagementPvENotSupported), http.StatusUnprocessableEntity, "pve_not_supported"},
		{port.ErrEngagementInsufficient, http.StatusUnprocessableEntity, "engagement_insufficient"},
		{port.ErrEngagementUnavailable, http.StatusServiceUnavailable, "engagement_unavailable"},
		{errors.New("boum"), http.StatusInternalServerError, "engagement_error"},
	} {
		f.matchErr = c.err
		if w := fpGet(r, "/players/spartan/matches/m1/engagement", nil); w.Code != c.status || !strings.Contains(w.Body.String(), c.code) {
			t.Errorf("%v : %d %s, attendu %d %s", c.err, w.Code, w.Body.String(), c.status, c.code)
		}
	}
}
