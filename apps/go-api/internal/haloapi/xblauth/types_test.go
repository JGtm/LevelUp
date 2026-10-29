package xblauth

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

// TestClientAzure_SecretMasque : ni `%v`, ni `%+v`, ni un journal `slog` ne révèlent le secret.
func TestClientAzure_SecretMasque(t *testing.T) {
	const secret = "valeur-secrete-de-test"
	c := ClientAzure{ID: "id", Secret: secret, Autorite: "https://login.example.test/consumers"}
	for _, rendu := range []string{fmt.Sprintf("%v", c), fmt.Sprintf("%+v", c), fmt.Sprint(c)} {
		if strings.Contains(rendu, secret) {
			t.Errorf("rendu %q contient le secret", rendu)
		}
	}
	var buf bytes.Buffer
	slog.New(slog.NewTextHandler(&buf, nil)).Info("x", "client", c)
	if strings.Contains(buf.String(), secret) {
		t.Errorf("journal %q contient le secret", buf.String())
	}
	if !strings.Contains(buf.String(), "***") {
		t.Errorf("journal %q : secret masqué attendu", buf.String())
	}
}

// TestClientAzure_Adresses : adresse d'autorisation dérivée de l'autorité et scopes joints par un espace.
func TestClientAzure_Adresses(t *testing.T) {
	c := ClientAzure{Autorite: "https://login.microsoftonline.com/consumers", Scopes: []string{"Xboxlive.signin", "Xboxlive.offline_access"}}
	if got := c.URLAutorisation(); got != "https://login.microsoftonline.com/consumers/oauth2/v2.0/authorize" {
		t.Errorf("URLAutorisation = %q", got)
	}
	if got := c.ScopesParam(); got != "Xboxlive.signin Xboxlive.offline_access" {
		t.Errorf("ScopesParam = %q", got)
	}
}
