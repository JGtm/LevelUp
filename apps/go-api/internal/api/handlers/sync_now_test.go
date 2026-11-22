package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/authz"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// fakeSyncNow est un port.SyncNowService qui rend le refus configuré et compte les lancements.
type fakeSyncNow struct {
	err         error
	calls       int
	running     bool
	last        time.Time
	availableAt time.Time
}

func (f *fakeSyncNow) CycleRunning() bool { return f.running }

func (f *fakeSyncNow) ManualWindow() (time.Time, time.Time) { return f.last, f.availableAt }

func (f *fakeSyncNow) StartSyncNow() error {
	f.calls++
	return f.err
}

// syncNowRouter monte POST /sync/now derrière la garde d'authentification réelle, avec la session
// injectée (nil = aucune) et la langue de la requête.
func syncNowRouter(svc port.SyncNowService, demo bool, authMode string, sess *domain.SessionData, locale string) *chi.Mux {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()
			if sess != nil {
				ctx = middleware.InjectSession(ctx, sess)
			}
			if locale != "" {
				ctx = ctxkeys.WithLocale(ctx, locale)
			}
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(demo, authMode))
		handlers.NewSyncNowHandler(svc).Mount(r)
	})
	return r
}

func postSyncNow(r http.Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/sync/now", strings.NewReader(""))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestSyncNow_Lance_202(t *testing.T) {
	svc := &fakeSyncNow{}
	w := postSyncNow(syncNowRouter(svc, false, authz.AuthModeNone, nil, ""))
	if w.Code != http.StatusAccepted {
		t.Fatalf("statut = %d, attendu 202 (corps %s)", w.Code, w.Body.String())
	}
	var body domain.SyncNowResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Status != "started" {
		t.Errorf("corps = %s (err %v), attendu status=started", w.Body.String(), err)
	}
	if svc.calls != 1 {
		t.Errorf("lancements = %d, attendu 1", svc.calls)
	}
}

// Un cycle en cours est refusé en 409 avec le code stable et le message du catalogue, dans la langue
// de la requête.
func TestSyncNow_CycleEnCours_409(t *testing.T) {
	cases := []struct {
		locale  string
		message string
	}{
		{"fr", "Une synchronisation est déjà en cours."},
		{"en", "A sync is already running."},
	}
	for _, c := range cases {
		t.Run(c.locale, func(t *testing.T) {
			svc := &fakeSyncNow{err: port.ErrSyncCycleBusy}
			w := postSyncNow(syncNowRouter(svc, false, authz.AuthModeNone, nil, c.locale))
			if w.Code != http.StatusConflict {
				t.Fatalf("statut = %d, attendu 409", w.Code)
			}
			code, message := errorBody(t, w)
			if code != "sync_cycle_busy" || message != c.message {
				t.Errorf("(code, message) = (%q, %q), attendu (sync_cycle_busy, %q)", code, message, c.message)
			}
		})
	}
}

func TestSyncNow_OrdonnanceurIndisponible_503(t *testing.T) {
	cases := map[string]port.SyncNowService{
		"aucun ordonnanceur câblé":   nil,
		"ordonnanceur à l'arrêt":     &fakeSyncNow{err: port.ErrSyncCycleUnavailable},
		"erreur imprévue du service": &fakeSyncNow{err: errors.New("boom")},
	}
	for name, svc := range cases {
		t.Run(name, func(t *testing.T) {
			w := postSyncNow(syncNowRouter(svc, false, authz.AuthModeNone, nil, ""))
			if w.Code != http.StatusServiceUnavailable {
				t.Fatalf("statut = %d, attendu 503", w.Code)
			}
			if code, _ := errorBody(t, w); code != "scheduler_unavailable" {
				t.Errorf("code = %q, attendu scheduler_unavailable", code)
			}
		})
	}
}

// La démo refuse le clic avant le service : aucun cycle n'est lancé.
func TestSyncNow_DemoRefusee_403(t *testing.T) {
	svc := &fakeSyncNow{}
	w := postSyncNow(syncNowRouter(svc, true, authz.AuthModeNone, nil, ""))
	if w.Code != http.StatusForbidden {
		t.Fatalf("statut = %d, attendu 403", w.Code)
	}
	if code, _ := errorBody(t, w); code != "demo_mode_forbidden" {
		t.Errorf("code = %q, attendu demo_mode_forbidden", code)
	}
	if svc.calls != 0 {
		t.Errorf("lancements = %d, attendu 0", svc.calls)
	}
}

// Sur une instance à comptes, tout joueur connecté peut lancer le cycle (pas de garde administrateur) ;
// sans session, 401.
func TestSyncNow_ToutJoueurConnecte(t *testing.T) {
	user := "joueur"
	svc := &fakeSyncNow{}
	connected := &domain.SessionData{Username: &user}

	if w := postSyncNow(syncNowRouter(svc, false, authz.AuthModePassword, connected, "")); w.Code != http.StatusAccepted {
		t.Errorf("joueur connecté : statut = %d, attendu 202", w.Code)
	}
	w := postSyncNow(syncNowRouter(svc, false, authz.AuthModePassword, nil, ""))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("sans session : statut = %d, attendu 401", w.Code)
	}
	if svc.calls != 1 {
		t.Errorf("lancements = %d, attendu 1 (le visiteur anonyme ne lance rien)", svc.calls)
	}
}

func getSyncCycle(r http.Handler) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/sync/cycle", nil))
	return w
}

// L'état du cycle suit celui de l'ordonnanceur ; la démo le refuse comme le lancement.
func TestSyncCycle_Etat(t *testing.T) {
	for _, running := range []bool{false, true} {
		w := getSyncCycle(syncNowRouter(&fakeSyncNow{running: running}, false, authz.AuthModeNone, nil, ""))
		if w.Code != http.StatusOK {
			t.Fatalf("statut = %d, attendu 200", w.Code)
		}
		var body domain.SyncCycleState
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Running != running {
			t.Errorf("corps = %s (err %v), attendu running=%v", w.Body.String(), err, running)
		}
	}
	if w := getSyncCycle(syncNowRouter(nil, false, authz.AuthModeNone, nil, "")); w.Code != http.StatusServiceUnavailable {
		t.Errorf("sans ordonnanceur : statut = %d, attendu 503", w.Code)
	}
	if w := getSyncCycle(syncNowRouter(&fakeSyncNow{}, true, authz.AuthModeNone, nil, "")); w.Code != http.StatusForbidden {
		t.Errorf("démo : statut = %d, attendu 403", w.Code)
	}
}

// Un cycle demandé trop tôt : 429 au code stable, message du catalogue avec le délai en secondes dans la
// langue de la requête et Retry-After (arrondis à la seconde supérieure).
func TestSyncNow_DelaiMinimal_429(t *testing.T) {
	cases := []struct {
		locale  string
		message string
	}{
		{"fr", "Une synchronisation vient de se terminer : réessayer dans 90 s."},
		{"en", "A sync has just finished: try again in 90 s."},
	}
	for _, c := range cases {
		t.Run(c.locale, func(t *testing.T) {
			svc := &fakeSyncNow{err: &port.SyncCooldownError{RetryAfter: 89*time.Second + 200*time.Millisecond}}
			w := postSyncNow(syncNowRouter(svc, false, authz.AuthModeNone, nil, c.locale))
			if w.Code != http.StatusTooManyRequests {
				t.Fatalf("statut = %d, attendu 429 (corps %s)", w.Code, w.Body.String())
			}
			if got := w.Header().Get("Retry-After"); got != "90" {
				t.Errorf("Retry-After = %q, attendu 90", got)
			}
			var body struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("corps illisible %q : %v", w.Body.String(), err)
			}
			if body.Code != "sync_cycle_cooldown" || body.Message != c.message {
				t.Errorf("corps = %+v, attendu code sync_cycle_cooldown, message %q", body, c.message)
			}
		})
	}
}

// L'état du cycle porte la fin du dernier cycle et l'heure de disponibilité du lancement manuel, pour que
// le bouton soit juste dès l'arrivée sur l'écran ; les deux sont absentes quand rien ne les justifie.
func TestSyncCycle_PorteLaDisponibiliteDuLancement(t *testing.T) {
	last := time.Date(2026, 11, 22, 14, 2, 0, 0, time.UTC)
	avail := last.Add(2 * time.Minute)
	w := getSyncCycle(syncNowRouter(&fakeSyncNow{last: last, availableAt: avail}, false, authz.AuthModeNone, nil, ""))
	var body domain.SyncCycleState
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Running ||
		body.LastCycleAt == nil || !body.LastCycleAt.Equal(last) ||
		body.ManualAvailableAt == nil || !body.ManualAvailableAt.Equal(avail) {
		t.Errorf("corps = %s (err %v), attendu last_cycle_at %v et manual_available_at %v", w.Body.String(), err, last, avail)
	}

	w = getSyncCycle(syncNowRouter(&fakeSyncNow{last: last}, false, authz.AuthModeNone, nil, ""))
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(w.Body.Bytes(), &raw)
	if _, ok := raw["manual_available_at"]; ok {
		t.Errorf("corps = %s : manual_available_at doit être absent quand le lancement est permis", w.Body.String())
	}
	if _, ok := raw["last_cycle_at"]; !ok {
		t.Errorf("corps = %s : last_cycle_at doit être servi", w.Body.String())
	}
}
