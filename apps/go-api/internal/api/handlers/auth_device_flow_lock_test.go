// Package handlers_test — auth_device_flow_lock_test.go : un device-flow dont la
// stratégie post-flux refuse l'identité (instance verrouillée) ne laisse AUCUNE
// trace d'authentification dans la session.
package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/session"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/service"
)

// lockedDeviceFlowRig : stratégie SSO Xbox sur instance verrouillée, une session
// persistée et une tentative déjà autorisée par Microsoft pour un xuid inconnu.
type lockedDeviceFlowRig struct {
	router    *chi.Mux
	sessStore *session.Store
	users     *userstore.Store
	attempts  *auth_platform.AttemptStore
	cookie    *http.Cookie
	sessionID string
	attemptID string
}

func newLockedDeviceFlowRig(t *testing.T) *lockedDeviceFlowRig {
	t.Helper()
	dir := t.TempDir()
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	attempts := auth_platform.NewAttemptStore()

	strategy := service.NewXboxSSOLinkStrategy(users).WithInstanceLock(func() bool { return true })
	authHandler := handlers.NewAuthHandler(sessStore, attempts, &stubTokenProvider{}).WithLinkStrategy(strategy)

	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessStore, middleware.SecureCookiePolicy{}))
	authHandler.MountDeviceFlow(r)
	r.With(middleware.RequireAuth(false, "xbox")).Get("/protected", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	sess := sessStore.New()
	if err := sessStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}
	attempt, _, _ := attempts.GetOrCreate(sess.SessionID, "")
	attempts.Update(attempt.AttemptID, func(a *auth_platform.Attempt) {
		a.Status = auth_platform.AttemptStatusAuthorized
		a.Gamertag = "Intruder"
		a.XUID = "2535400000000001"
		a.SpartanToken = "spartan-of-intruder"
		a.ClearanceToken = "clearance-of-intruder"
	})
	return &lockedDeviceFlowRig{
		router:    r,
		sessStore: sessStore,
		users:     users,
		attempts:  attempts,
		cookie:    &http.Cookie{Name: session.CookieName, Value: sessStore.SignCookie(sess.SessionID)},
		sessionID: sess.SessionID,
		attemptID: attempt.AttemptID,
	}
}

func (rig *lockedDeviceFlowRig) get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.AddCookie(rig.cookie)
	w := httptest.NewRecorder()
	rig.router.ServeHTTP(w, req)
	return w
}

// TestDeviceFlow_InstanceLockedLeavesSessionUnauthenticated : la tentative passe en
// échec (code instance_locked), la session ne porte ni AuthReady, ni identité, ni
// jetons, aucun compte n'est créé, et une route gardée répond 401.
func TestDeviceFlow_InstanceLockedLeavesSessionUnauthenticated(t *testing.T) {
	rig := newLockedDeviceFlowRig(t)

	w := rig.get(t, "/auth/device-flow/"+rig.attemptID)
	if w.Code != http.StatusOK {
		t.Fatalf("statut = %d, corps = %s", w.Code, w.Body.String())
	}
	var resp domain.DeviceFlowStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("décodage : %v", err)
	}
	if resp.Status != auth_platform.AttemptStatusFailed {
		t.Errorf("statut de la tentative = %q, attendu failed", resp.Status)
	}
	if resp.ErrorCode == nil || *resp.ErrorCode != "instance_locked" {
		t.Errorf("error_code = %v, attendu instance_locked", resp.ErrorCode)
	}
	if snap := rig.attempts.Snapshot(rig.attemptID); snap == nil || snap.Status != auth_platform.AttemptStatusFailed {
		t.Error("tentative en mémoire absente ou non failed")
	} else if snap.SpartanToken != "" || snap.ClearanceToken != "" {
		t.Error("les jetons de la tentative refusée restent en mémoire")
	}

	sess := rig.sessStore.Load(t.Context(), rig.sessionID)
	if sess == nil {
		t.Fatal("session introuvable après le refus")
	}
	if sess.AuthReady {
		t.Error("session AuthReady=true malgré le refus du verrou")
	}
	if sess.LinkedHaloIdentity != nil {
		t.Errorf("identité liée en session malgré le refus : %+v", sess.LinkedHaloIdentity)
	}
	if sess.HaloTokens != nil {
		t.Error("jetons Halo en session malgré le refus")
	}
	if sess.Username != nil {
		t.Errorf("username en session malgré le refus : %q", *sess.Username)
	}
	if _, err := rig.users.GetByXUID("2535400000000001"); err == nil {
		t.Error("compte créé malgré le verrou")
	}

	if got := rig.get(t, "/protected").Code; got != http.StatusUnauthorized {
		t.Errorf("route gardée après le refus : %d, attendu 401", got)
	}
}
