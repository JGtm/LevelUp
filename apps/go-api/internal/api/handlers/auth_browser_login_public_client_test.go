package handlers_test

import (
	"os"
	"strings"
	"testing"
)

// TestBrowserLogin_ClientPublicSeulement : le Microsoft réel de la connexion par le navigateur passe
// par les variantes PUBLIQUES de platform/auth (identifiant fixe, aucun secret, ADR 0045 D-19), jamais
// par celles qui résolvent le client Azure de l'environnement (secret du `.env.local` d'un dépôt).
// Le comportement des requêtes est testé dans platform/auth (xbl_oauth_public_internal_test.go).
func TestBrowserLogin_ClientPublicSeulement(t *testing.T) {
	src, err := os.ReadFile("auth_browser_login.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, interdit := range []string{
		"auth_platform.ExchangeAuthorizationCode(",
		"auth_platform.BuildAuthorizeURL(",
		"ResolveAzureOAuthClient",
	} {
		if strings.Contains(string(src), interdit) {
			t.Errorf("auth_browser_login.go ne doit pas utiliser %s : client confidentiel de l'environnement", interdit)
		}
	}
}
