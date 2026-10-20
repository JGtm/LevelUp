// Package handlers_test — user_auth_set_password_current_test.go : changer un mot de
// passe exige le mot de passe actuel (registre sécurité C5-6). Un compte Xbox qui n'en
// a pas encore définit le premier sans l'ancien. Un mot de passe actuel faux compte
// comme un échec de connexion (limiteur partagé avec POST /auth/login).
package handlers_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// nouveauMotDePasse : le mot de passe que les scénarios tentent de poser.
const nouveauMotDePasse = "Nouveau-Mot-De-Passe-2"

// setPasswordBody : corps de POST /auth/password (champ current_password compris).
type setPasswordBody struct {
	Password        string `json:"password"`
	CurrentPassword string `json:"current_password,omitempty"`
}

func loggedInCookies(t *testing.T, r http.Handler, ip string) []*http.Cookie {
	t.Helper()
	ok := loginFrom(t, r, ip, loginLimitUser, loginLimitPassword)
	if ok.Code != http.StatusOK {
		t.Fatalf("connexion initiale : %d (%s)", ok.Code, ok.Body.String())
	}
	return ok.Result().Cookies()
}

func TestSetPassword_RequiresCurrentPassword(t *testing.T) {
	r := newLoginLimitRouter(t)
	const ip = "192.0.2.70"
	cookies := loggedInCookies(t, r, ip)

	w := postFromIP(t, r, "/auth/password", ip, setPasswordBody{Password: nouveauMotDePasse}, cookies...)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "current_password_required") {
		t.Errorf("sans mot de passe actuel : %d %s, attendu 400 current_password_required", w.Code, w.Body.String())
	}
	w = postFromIP(t, r, "/auth/password", ip,
		setPasswordBody{Password: nouveauMotDePasse, CurrentPassword: loginLimitWrongPass}, cookies...)
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "invalid_current_password") {
		t.Errorf("mot de passe actuel faux : %d %s, attendu 403 invalid_current_password", w.Code, w.Body.String())
	}
	w = postFromIP(t, r, "/auth/password", ip,
		setPasswordBody{Password: nouveauMotDePasse, CurrentPassword: loginLimitPassword}, cookies...)
	if w.Code != http.StatusNoContent {
		t.Fatalf("mot de passe actuel juste : %d %s, attendu 204", w.Code, w.Body.String())
	}
	if w := loginFrom(t, r, ip, loginLimitUser, nouveauMotDePasse); w.Code != http.StatusOK {
		t.Errorf("connexion avec le nouveau mot de passe : %d", w.Code)
	}
}

// TestSetPassword_SixthWrongCurrentPasswordIs429 : cinq mots de passe actuels faux
// bloquent l'identifiant ; la sixième tentative répond 429, depuis une autre IP aussi.
func TestSetPassword_SixthWrongCurrentPasswordIs429(t *testing.T) {
	r := newLoginLimitRouter(t)
	cookies := loggedInCookies(t, r, "192.0.2.80")
	for i := 1; i <= 5; i++ {
		w := postFromIP(t, r, "/auth/password", fmt.Sprintf("203.0.113.%d", 150+i),
			setPasswordBody{Password: nouveauMotDePasse, CurrentPassword: loginLimitWrongPass}, cookies...)
		if w.Code != http.StatusForbidden {
			t.Fatalf("échec %d : %d (%s), attendu 403", i, w.Code, w.Body.String())
		}
	}
	w := postFromIP(t, r, "/auth/password", "203.0.113.200",
		setPasswordBody{Password: nouveauMotDePasse, CurrentPassword: loginLimitPassword}, cookies...)
	assertTooManyAttempts(t, w, "sixième changement après cinq mots de passe actuels faux")
	if w := loginFrom(t, r, "198.51.100.9", loginLimitUser, loginLimitPassword); w.Code != http.StatusTooManyRequests {
		t.Errorf("connexion de l'identifiant bloqué : %d, attendu 429", w.Code)
	}
}
