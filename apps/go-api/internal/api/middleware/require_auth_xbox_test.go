package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"levelup/go-api/internal/domain"
)

// TestRequireAuth_XboxMode_SessionKinds : en mode d'auth « xbox », trois chemins
// de connexion produisent trois formes de session, et toutes sont authentifiées.
//
// Le cas « mot de passe » est la régression couverte : le login par mot de passe
// est ouvert en mode xbox (POST /auth/login, comptes ayant défini un mot de passe)
// et ne pose que Username/Role, sans AuthReady. Exiger AuthReady refusait en 401
// toutes les routes sous RequireAuth (/settings, /presence, /admin/...) alors que
// /bootstrap déclarait l'utilisateur connecté — la coquille web rechargeait en
// boucle vers /login.
func TestRequireAuth_XboxMode_SessionKinds(t *testing.T) {
	admin := "jgtm"
	role := string(domain.RoleAdmin)
	cases := []struct {
		name string
		sess *domain.SessionData
		want int
	}{
		{"login par mot de passe (Username sans AuthReady)",
			&domain.SessionData{SessionID: "s", Username: &admin, Role: &role}, http.StatusOK},
		{"SSO Xbox (Username et AuthReady)",
			&domain.SessionData{SessionID: "s", Username: &admin, Role: &role, AuthReady: true}, http.StatusOK},
		{"device-code (AuthReady sans Username)",
			&domain.SessionData{SessionID: "s", AuthReady: true}, http.StatusOK},
		{"visiteur anonyme (session vide)",
			&domain.SessionData{SessionID: "s"}, http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := RequireAuth(false, "xbox")(okHandler)
			rr := httptest.NewRecorder()
			req := withSession(httptest.NewRequest(http.MethodGet, "/settings", nil), tc.sess)
			handler.ServeHTTP(rr, req)
			if rr.Code != tc.want {
				t.Errorf("status = %d, want %d", rr.Code, tc.want)
			}
		})
	}
}

// TestRequireAuth_PasswordMode_AuthReadyAloneRefused : en mode « password », le
// login local fait foi ; une session qui n'a que l'auth Halo (device-code) reste
// refusée.
func TestRequireAuth_PasswordMode_AuthReadyAloneRefused(t *testing.T) {
	handler := RequireAuth(false, "password")(okHandler)
	rr := httptest.NewRecorder()
	req := withSession(httptest.NewRequest(http.MethodGet, "/settings", nil),
		&domain.SessionData{SessionID: "s", AuthReady: true})
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}
