package handlers_test

import (
	"net/http"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

// L9-5 : un refus de la politique d'identifiants porte un code par règle et un message fixe ;
// l'écran écrit lui-même le texte (FR + EN) à partir du code.
func TestUserAuth_Register_CredentialPolicyCodes(t *testing.T) {
	cases := []struct {
		name     string
		username string
		password string
		wantCode string
	}{
		{"mot de passe trop court", "validuser", "court", "password_too_short"},
		{"mot de passe trop long", "validuser", strings.Repeat("a", 80), "password_too_long"},
		{"nom invalide", "a b", testUserAuthPass, "invalid_username"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, _ := newUserAuthRouter(t, "password")
			w := postJSON(t, r, "/auth/register", domain.RegisterRequest{Username: tc.username, Password: tc.password})
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, attendu 400. Body: %s", w.Code, w.Body.String())
			}
			body := w.Body.String()
			if !strings.Contains(body, `"code":"`+tc.wantCode+`"`) {
				t.Errorf("code attendu %q, body: %s", tc.wantCode, body)
			}
			if !strings.Contains(body, `"message":"`+domain.ErrorMessageForCode(tc.wantCode)+`"`) {
				t.Errorf("message fixe attendu, body: %s", body)
			}
			for _, leaked := range []string{"caractères", "octets", "minimum"} {
				if strings.Contains(body, leaked) {
					t.Errorf("le texte de la politique part au client (%q) : %s", leaked, body)
				}
			}
		})
	}
}
