// Package handlers_test — sync_handler_test.go : SyncHandler et BackfillHandler contre un faux port.
//
// Le handler décode, appelle le port et traduit ses refus typés : ces tests posent un faux service qui
// rend un job ou un refus donné, et vérifient la réponse HTTP (statut, code, message, en-têtes) ainsi
// que ce que le handler a transmis au service.
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// fakeJobService est un port.SyncJobService qui rend le job ou le refus configuré et garde les appels.
type fakeJobService struct {
	job *domain.AsyncJobStatus
	err error
	// enabledErr : refus de CheckInitialSyncEnabled.
	enabledErr error

	mu       sync.Mutex
	calls    []string
	caller   port.SyncCaller
	initial  domain.InitialSyncStartRequest
	backfill domain.BackfillStartRequest
	slug     string
}

func newFakeJobService() *fakeJobService {
	return &fakeJobService{job: &domain.AsyncJobStatus{
		JobID: "job-1", JobType: string(domain.JobTypeInitialSync), Status: domain.JobStatusQueued,
	}}
}

func (f *fakeJobService) record(name string, c port.SyncCaller) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, name)
	f.caller = c
}

func (f *fakeJobService) CheckInitialSyncEnabled(context.Context) error {
	f.record("check", port.SyncCaller{})
	return f.enabledErr
}

func (f *fakeJobService) StartInitialSync(_ context.Context, c port.SyncCaller, req domain.InitialSyncStartRequest) (*domain.AsyncJobStatus, error) {
	f.record("initial", c)
	f.initial = req
	return f.job, f.err
}

func (f *fakeJobService) StartDeltaSync(_ context.Context, c port.SyncCaller, slug string) (*domain.AsyncJobStatus, error) {
	f.record("delta", c)
	f.slug = slug
	return f.job, f.err
}

func (f *fakeJobService) StartSyncAll(_ context.Context, c port.SyncCaller) (*domain.AsyncJobStatus, error) {
	f.record("all", c)
	return f.job, f.err
}

func (f *fakeJobService) StartBackfill(_ context.Context, c port.SyncCaller, req domain.BackfillStartRequest) (*domain.AsyncJobStatus, error) {
	f.record("backfill", c)
	f.backfill = req
	return f.job, f.err
}

func (f *fakeJobService) StartSessionsRecalc(context.Context) (*domain.AsyncJobStatus, error) {
	f.record("recalc", port.SyncCaller{})
	return f.job, f.err
}

func (f *fakeJobService) called() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}

// jobRefusal : un refus typé du service.
func jobRefusal(kind port.SyncJobErrorKind) *port.SyncJobError { return &port.SyncJobError{Kind: kind} }

// syncRouter monte les routes de sync sur le faux service, avec la session injectée (nil = aucune).
func syncRouter(svc port.SyncService, sess *domain.SessionData) *chi.Mux {
	h := handlers.NewSyncHandler(svc)
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()
			if sess != nil {
				ctx = middleware.InjectSession(ctx, sess)
			}
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	h.MountInitialAndAll(r)
	r.Route("/players/{player_slug}", func(r chi.Router) { h.MountDelta(r) })
	return r
}

func postSyncJSON(r http.Handler, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// errorBody lit {code, message} d'une réponse d'erreur.
func errorBody(t *testing.T, w *httptest.ResponseRecorder) (code, message string) {
	t.Helper()
	var b struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &b); err != nil {
		t.Fatalf("corps d'erreur illisible %q : %v", w.Body.String(), err)
	}
	return b.Code, b.Message
}

func sessionWithTokens() *domain.SessionData {
	return &domain.SessionData{HaloTokens: &domain.HaloTokens{SpartanToken: "spartan-test"}}
}

// Chaque refus du service devient la réponse HTTP d'avant le déplacement : mêmes statut, code et message.
func TestSyncHandler_RefusDuService(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		status  int
		code    string
		message string
	}{
		{"sync initiale désactivée", jobRefusal(port.SyncErrInitialDisabled), 403, "initial_sync_disabled",
			"Le lancement d'une sync initiale est désactivé sur cette instance."},
		{"max_matches invalide", jobRefusal(port.SyncErrInvalidMaxMatches), 400, "invalid_max_matches",
			"max_matches doit etre entre 1 et 2000."},
		{"sync initiale déjà active", jobRefusal(port.SyncErrInitialAlreadyActive), 409, "sync_already_active",
			"Une sync initiale est deja en cours pour ce joueur."},
		{"sync déjà active", jobRefusal(port.SyncErrDeltaAlreadyActive), 409, "sync_already_active",
			"Une synchronisation est déjà en cours pour ce joueur."},
		{"joueur déjà en vol", jobRefusal(port.SyncErrAlreadyRunning), 409, "sync_already_active",
			"Une synchronisation de ce joueur est déjà en cours (watcher ou auto-sync)."},
		{"sans session", jobRefusal(port.SyncErrAuthRequired), 401, "auth_required", "Connexion requise."},
		{"session sans jetons Halo", jobRefusal(port.SyncErrHaloTokensMissing), 403, "halo_tokens_missing",
			"Aucun jeton Halo pour cette session : connecte le compte Xbox."},
		{"joueur introuvable", &port.SyncJobError{Kind: port.SyncErrPlayerNotFound, PlayerSlug: "ghost"}, 404, "player_not_found",
			`Joueur "ghost" introuvable dans db_profiles.json.`},
		{"joueur introuvable pour le titre",
			&port.SyncJobError{Kind: port.SyncErrPlayerNotFoundForTitle, PlayerSlug: "ghost", TitleSlug: "halo_5"}, 404, "player_not_found",
			`Joueur "ghost" introuvable pour le titre "halo_5" dans db_profiles.json.`},
		{"aucun joueur", jobRefusal(port.SyncErrNoPlayers), 404, "no_players", "Aucun joueur configuré dans db_profiles.json."},
		// Les pannes : la cause ne part jamais au client.
		{"profils illisibles", &port.SyncJobError{Kind: port.SyncErrProfilesLoad, Err: errors.New("secret interne")}, 500,
			"profiles_load_error", "internal error"},
		{"configuration illisible", &port.SyncJobError{Kind: port.SyncErrSettingsLoad, Err: errors.New("secret interne")}, 500,
			"settings_load_error", "internal error"},
		{"erreur inattendue", errors.New("secret interne"), 500, "sync_failed", "internal error"},
	}
	routes := []struct{ name, path, body string }{
		{"initial", "/sync/initial", `{"player_slug":"jgtm","max_matches":100}`},
		{"delta", "/players/jgtm/sync", ``},
		{"all", "/sync/all", ``},
	}
	for _, c := range cases {
		for _, route := range routes {
			t.Run(c.name+" / "+route.name, func(t *testing.T) {
				svc := newFakeJobService()
				svc.err = c.err
				w := postSyncJSON(syncRouter(svc, sessionWithTokens()), route.path, route.body)
				if w.Code != c.status {
					t.Fatalf("statut %d, attendu %d : %s", w.Code, c.status, w.Body.String())
				}
				code, message := errorBody(t, w)
				if code != c.code || message != c.message {
					t.Errorf("(code, message) = (%q, %q), attendu (%q, %q)", code, message, c.code, c.message)
				}
				if strings.Contains(w.Body.String(), "secret interne") {
					t.Error("la cause interne fuit dans la réponse")
				}
			})
		}
	}
}

// Le refus de cooldown porte son délai restant : en-tête Retry-After en secondes, arrondi au plafond.
func TestSyncHandler_Cooldown429(t *testing.T) {
	for _, path := range []string{"/sync/all", "/players/jgtm/sync"} {
		svc := newFakeJobService()
		svc.err = &port.SyncJobError{Kind: port.SyncErrCooldown, RetryAfter: 41500 * time.Millisecond}
		w := postSyncJSON(syncRouter(svc, sessionWithTokens()), path, "")
		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("%s : statut %d, attendu 429 : %s", path, w.Code, w.Body.String())
		}
		if got := w.Header().Get("Retry-After"); got != "42" {
			t.Errorf("%s : Retry-After %q, attendu 42", path, got)
		}
		code, message := errorBody(t, w)
		if code != "sync_cooldown" || message != "Synchronisation déjà déclenchée récemment — réessayez dans 42 s." {
			t.Errorf("%s : (code, message) = (%q, %q)", path, code, message)
		}
	}
}

func TestSyncHandler_InitialSync_Succes(t *testing.T) {
	svc := newFakeJobService()
	w := postSyncJSON(syncRouter(svc, sessionWithTokens()), "/sync/initial",
		`{"player_slug":"jgtm","max_matches":150,"title_slug":"halo_5"}`)
	if w.Code != http.StatusAccepted {
		t.Fatalf("attendu 202, obtenu %d : %s", w.Code, w.Body.String())
	}
	var got domain.AsyncJobStatus
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || got.JobID != "job-1" || got.Status != domain.JobStatusQueued {
		t.Errorf("corps %s : le job du service doit être rendu tel quel", w.Body.String())
	}
	if svc.initial.PlayerSlug != "jgtm" || svc.initial.MaxMatches != 150 || svc.initial.TitleSlug != "halo_5" {
		t.Errorf("requête transmise au service : %+v", svc.initial)
	}
	if !svc.caller.Authenticated || svc.caller.HaloTokens == nil || svc.caller.HaloTokens.SpartanToken != "spartan-test" {
		t.Errorf("appelant transmis : %+v, attendu la session et ses jetons Halo", svc.caller)
	}
}

// Sans session, l'appelant n'est pas authentifié ; session sans jetons : authentifié, sans jetons.
func TestSyncHandler_AppelantTransmis(t *testing.T) {
	svc := newFakeJobService()
	postSyncJSON(syncRouter(svc, nil), "/sync/all", "")
	if svc.caller.Authenticated || svc.caller.HaloTokens != nil {
		t.Errorf("sans session : %+v", svc.caller)
	}
	postSyncJSON(syncRouter(svc, &domain.SessionData{}), "/sync/all", "")
	if !svc.caller.Authenticated || svc.caller.HaloTokens != nil {
		t.Errorf("session sans jetons Halo : %+v", svc.caller)
	}
}

// Un refus d'instance prime sur un corps invalide ; sans refus, un corps invalide ou un slug vide ou trop
// long est refusé en 400 sans appeler le service.
func TestSyncHandler_InitialSync_Validation(t *testing.T) {
	disabled := newFakeJobService()
	disabled.enabledErr = jobRefusal(port.SyncErrInitialDisabled)
	if w := postSyncJSON(syncRouter(disabled, sessionWithTokens()), "/sync/initial", "{bad"); w.Code != http.StatusForbidden {
		t.Errorf("sync initiale désactivée + corps invalide : attendu 403, obtenu %d", w.Code)
	}

	cases := []struct {
		name, body, code, message string
	}{
		{"corps invalide", "{bad", "invalid_body", "Corps de requete JSON invalide."},
		{"slug vide", `{"player_slug":"","max_matches":100}`, "invalid_player_slug", "player_slug vide ou trop long."},
		{"slug trop long", `{"player_slug":"` + strings.Repeat("a", 51) + `","max_matches":100}`, "invalid_player_slug",
			"player_slug vide ou trop long."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := newFakeJobService()
			w := postSyncJSON(syncRouter(svc, sessionWithTokens()), "/sync/initial", c.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("attendu 400, obtenu %d : %s", w.Code, w.Body.String())
			}
			if code, message := errorBody(t, w); code != c.code || message != c.message {
				t.Errorf("(code, message) = (%q, %q), attendu (%q, %q)", code, message, c.code, c.message)
			}
			for _, call := range svc.called() {
				if call == "initial" {
					t.Error("le service ne doit pas être appelé sur une requête invalide")
				}
			}
		})
	}
}

func TestSyncHandler_DeltaEtTous_Succes(t *testing.T) {
	svc := newFakeJobService()
	r := syncRouter(svc, sessionWithTokens())

	if w := postSyncJSON(r, "/players/jgtm/sync", ""); w.Code != http.StatusAccepted || svc.slug != "jgtm" {
		t.Errorf("sync delta : statut %d, slug transmis %q", w.Code, svc.slug)
	}
	if w := postSyncJSON(r, "/sync/all", ""); w.Code != http.StatusAccepted {
		t.Errorf("sync de tous : statut %d : %s", w.Code, w.Body.String())
	}
	if got := strings.Join(svc.called(), ","); got != "delta,all" {
		t.Errorf("appels %s, attendu delta,all", got)
	}
}

// --- BackfillHandler ---------------------------------------------------------

func backfillRouter(svc port.BackfillService, sess *domain.SessionData) *chi.Mux {
	h := handlers.NewBackfillHandler(svc)
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()
			if sess != nil {
				ctx = middleware.InjectSession(ctx, sess)
			}
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	h.Mount(r)
	return r
}

func TestBackfillHandler_Succes(t *testing.T) {
	svc := newFakeJobService()
	w := postSyncJSON(backfillRouter(svc, sessionWithTokens()), "/backfill/start",
		`{"player_slug":"jgtm","title":"halo_5","medals":true,"dry_run":true}`)
	if w.Code != http.StatusAccepted {
		t.Fatalf("attendu 202, obtenu %d : %s", w.Code, w.Body.String())
	}
	if svc.backfill.PlayerSlug != "jgtm" || svc.backfill.Title != "halo_5" || !svc.backfill.Medals || !svc.backfill.DryRun {
		t.Errorf("requête transmise au service : %+v", svc.backfill)
	}
	if svc.caller.HaloTokens == nil || svc.caller.HaloTokens.SpartanToken != "spartan-test" {
		t.Errorf("les jetons Halo de la session doivent être transmis : %+v", svc.caller)
	}

	// Sans session : le rattrapage est lancé, sans jetons.
	postSyncJSON(backfillRouter(svc, nil), "/backfill/start", `{"player_slug":"jgtm"}`)
	if svc.caller.Authenticated || svc.caller.HaloTokens != nil {
		t.Errorf("sans session : %+v", svc.caller)
	}
}

func TestBackfillHandler_Validation(t *testing.T) {
	cases := []struct {
		name, body, code, message string
	}{
		{"corps invalide", "{bad", "invalid_body", "Corps de requête JSON invalide."},
		{"slug vide", `{"player_slug":""}`, "invalid_player_slug", "player_slug vide ou trop long."},
		{"slug trop long", `{"player_slug":"` + strings.Repeat("a", 51) + `"}`, "invalid_player_slug", "player_slug vide ou trop long."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := newFakeJobService()
			w := postSyncJSON(backfillRouter(svc, sessionWithTokens()), "/backfill/start", c.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("attendu 400, obtenu %d : %s", w.Code, w.Body.String())
			}
			if code, message := errorBody(t, w); code != c.code || message != c.message {
				t.Errorf("(code, message) = (%q, %q), attendu (%q, %q)", code, message, c.code, c.message)
			}
			if len(svc.called()) != 0 {
				t.Error("le service ne doit pas être appelé sur une requête invalide")
			}
		})
	}
}

func TestBackfillHandler_RefusDuService(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		status  int
		code    string
		message string
	}{
		{"joueur introuvable", &port.SyncJobError{Kind: port.SyncErrPlayerNotFound, PlayerSlug: "ghost"}, 404, "player_not_found",
			`Joueur "ghost" introuvable dans db_profiles.json.`},
		{"rattrapage déjà actif", jobRefusal(port.SyncErrBackfillAlreadyActive), 409, "backfill_already_active",
			"Un backfill est déjà en cours pour ce joueur."},
		{"profils illisibles", &port.SyncJobError{Kind: port.SyncErrProfilesLoad, Err: errors.New("secret interne")}, 500,
			"profiles_load_error", "internal error"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := newFakeJobService()
			svc.err = c.err
			w := postSyncJSON(backfillRouter(svc, sessionWithTokens()), "/backfill/start", `{"player_slug":"ghost"}`)
			if w.Code != c.status {
				t.Fatalf("statut %d, attendu %d : %s", w.Code, c.status, w.Body.String())
			}
			if code, message := errorBody(t, w); code != c.code || message != c.message {
				t.Errorf("(code, message) = (%q, %q), attendu (%q, %q)", code, message, c.code, c.message)
			}
		})
	}
}
