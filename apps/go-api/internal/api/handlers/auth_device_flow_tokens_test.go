// Package handlers_test — auth_device_flow_tokens_test.go : en mode d'authentification
// "none" (poste seul), une connexion par code d'appareil réussie persiste le jeton de
// rafraîchissement dans le magasin de jetons (ADR 0023) pour un profil suivi, et
// n'écrit rien pour un profil qui n'existe pas encore (ADR 0035 D3).
package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/session"
	"levelup/go-api/internal/platform/userstore"
	"levelup/go-api/internal/service"
)

const noneModeXUID = "2535400000000042"

// noneModeRig : stratégie du mode "none" (stratégie mot de passe décorée par la
// complétion de connexion), session sans compte, tentative autorisée portant un RT.
type noneModeRig struct {
	router     http.Handler
	store      *auth_platform.MultiUserTokenStore
	completion *service.AuthCompletion
	attempts   *auth_platform.AttemptStore
	cookie     *http.Cookie
	attemptID  string
}

func newNoneModeRig(t *testing.T, tracked bool) *noneModeRig {
	t.Helper()
	dir := t.TempDir()
	sessStore := session.NewStore(filepath.Join(dir, "sessions"), time.Hour, "test-secret-32bytesXXXXXXXXXXX")
	users := userstore.NewStore(filepath.Join(dir, "users.json"))
	attempts := auth_platform.NewAttemptStore()
	store := auth_platform.NewMultiUserTokenStore(filepath.Join(dir, "watcher_tokens"))

	gate := func(_ context.Context, _, xuid string) bool { return tracked && xuid == noneModeXUID }
	completion := service.NewAuthCompletion(store, gate)
	strategy := service.NewAuthLinkStrategy(auth_platform.NewPasswordLinkStrategy(users), completion)
	authHandler := handlers.NewAuthHandler(sessStore, attempts, &stubTokenProvider{}).WithLinkStrategy(strategy)

	r := chi.NewRouter()
	r.Use(middleware.WithSession(sessStore, middleware.SecureCookiePolicy{}))
	authHandler.MountDeviceFlow(r)

	sess := sessStore.New()
	if err := sessStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}
	attempt, _, _ := attempts.GetOrCreate(sess.SessionID, "")
	attempts.Update(attempt.AttemptID, func(a *auth_platform.Attempt) {
		a.Status = auth_platform.AttemptStatusAuthorized
		a.Gamertag = "PosteSeul"
		a.XUID = noneModeXUID
		a.SpartanToken = "spartan"
		a.MicrosoftAccessToken = "ms-access"
		a.OAuthRefreshToken = "rt-du-poste"
		a.XSTSRTAToken = "xsts-rta"
	})
	return &noneModeRig{
		router:     r,
		store:      store,
		completion: completion,
		attempts:   attempts,
		cookie:     &http.Cookie{Name: session.CookieName, Value: sessStore.SignCookie(sess.SessionID)},
		attemptID:  attempt.AttemptID,
	}
}

func (rig *noneModeRig) pollStatus(t *testing.T) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/auth/device-flow/"+rig.attemptID, nil)
	req.AddCookie(rig.cookie)
	w := httptest.NewRecorder()
	rig.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("statut = %d, corps = %s", w.Code, w.Body.String())
	}
}

func TestDeviceFlow_ModeNone_ProfilSuivi_PersisteLeJeton(t *testing.T) {
	rig := newNoneModeRig(t, true)

	rig.pollStatus(t)

	stored, err := rig.store.Load(noneModeXUID)
	if err != nil {
		t.Fatalf("le magasin doit contenir le jeton du poste : %v", err)
	}
	if stored.OAuthRefreshToken != "rt-du-poste" {
		t.Errorf("RT persisté = %q, attendu rt-du-poste", stored.OAuthRefreshToken)
	}
	if stored.PoolSharing != nil {
		t.Errorf("aucune décision de mise en commun en mode none (ADR 0038), reçu %+v", stored.PoolSharing)
	}
}

func TestDeviceFlow_ModeNone_ProfilNonSuivi_RienEcritPuisRejoueAuProfil(t *testing.T) {
	rig := newNoneModeRig(t, false)

	rig.pollStatus(t)

	if _, err := rig.store.Load(noneModeXUID); err == nil {
		t.Fatal("profil non suivi : le magasin ne doit rien contenir")
	}

	// Le wizard déclare ensuite le profil (Onboard) : le jeton est rejoué.
	rig.completion.SeedCredentials(t.Context(), noneModeXUID)
	stored, err := rig.store.Load(noneModeXUID)
	if err != nil || stored.OAuthRefreshToken != "rt-du-poste" {
		t.Fatalf("jeton non rejoué à la création du profil : %+v, %v", stored, err)
	}
}

// La connexion elle-même (session) n'est pas affectée par la persistance : identité
// Halo posée et session prête, malgré l'absence de compte local (mode "none").
func TestDeviceFlow_ModeNone_LaSessionReste_Connectee(t *testing.T) {
	rig := newNoneModeRig(t, true)

	rig.pollStatus(t)

	// La connexion aboutit (tentative remise à la session) : la persistance des jetons
	// n'a pas transformé le succès en refus.
	snap := rig.attempts.Snapshot(rig.attemptID)
	if snap == nil || snap.Status != auth_platform.AttemptStatusProvisioned {
		t.Fatalf("tentative = %+v, attendu provisioned", snap)
	}
}
