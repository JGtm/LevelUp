// Package auth — azure_credentials_test.go : golden / caractérisation du seam des
// credentials Azure. Phase 3 (uniformisation) : SSO + refresh + token-capture sont
// sur l'app canonique e1cb35ab via LEVELUP_OAUTH_CLIENT_ID. e1cb35ab a un redirect
// « Web » → client CONFIDENTIEL → SecretToSend envoie TOUJOURS le secret configuré
// (AADSTS70002 sinon). Le diff vs Phase 1 (défaut TokenCapture, env var, garde
// SecretToSend supprimée) = le changement assumé.
package auth

import (
	"net/url"
	"testing"
)

func TestResolveAzureOAuthClient_DefaultIsLevelUpHalo(t *testing.T) {
	t.Setenv("LEVELUP_OAUTH_CLIENT_ID", "")
	t.Setenv("SPNKR_AZURE_CLIENT_SECRET", "")

	c := ResolveAzureOAuthClient()
	if c.ClientID != LevelUpClientID {
		t.Errorf("ClientID défaut = %q, want %q (LevelUp Halo)", c.ClientID, LevelUpClientID)
	}
	if s := c.SecretToSend(); s != "" {
		t.Errorf("SecretToSend sans secret env = %q, want \"\"", s)
	}
}

func TestResolveAzureOAuthClient_EnvClientIDOverride(t *testing.T) {
	t.Setenv("LEVELUP_OAUTH_CLIENT_ID", "custom-app-id")
	t.Setenv("SPNKR_AZURE_CLIENT_SECRET", "")

	if c := ResolveAzureOAuthClient(); c.ClientID != "custom-app-id" {
		t.Errorf("ClientID = %q, want custom-app-id", c.ClientID)
	}
}

func TestSecretToSend_CanonicalClientSendsSecret(t *testing.T) {
	// Phase 3 : e1cb35ab (LevelUpClientID) en plateforme Web = confidentiel →
	// le secret EST envoyé (AVANT, la garde l'excluait → AADSTS70002 en prod).
	t.Setenv("LEVELUP_OAUTH_CLIENT_ID", LevelUpClientID)
	t.Setenv("SPNKR_AZURE_CLIENT_SECRET", "s3cret")

	if got := ResolveAzureOAuthClient().SecretToSend(); got != "s3cret" {
		t.Errorf("SecretToSend (e1cb35ab confidentiel) = %q, want s3cret", got)
	}
}

func TestSecretToSend_OtherClientSendsSecret(t *testing.T) {
	t.Setenv("LEVELUP_OAUTH_CLIENT_ID", "39829f7a-confidential")
	t.Setenv("SPNKR_AZURE_CLIENT_SECRET", "s3cret")

	if got := ResolveAzureOAuthClient().SecretToSend(); got != "s3cret" {
		t.Errorf("SecretToSend = %q, want s3cret", got)
	}
}

func TestSecretToSend_NoSecretEnv(t *testing.T) {
	t.Setenv("LEVELUP_OAUTH_CLIENT_ID", LevelUpClientID)
	t.Setenv("SPNKR_AZURE_CLIENT_SECRET", "")

	if got := ResolveAzureOAuthClient().SecretToSend(); got != "" {
		t.Errorf("SecretToSend sans secret env = %q, want \"\"", got)
	}
}

func TestDeviceFlowClientID_IsLevelUpHalo(t *testing.T) {
	if DeviceFlowClientID() != LevelUpClientID {
		t.Errorf("DeviceFlowClientID = %q, want %q", DeviceFlowClientID(), LevelUpClientID)
	}
}

func TestTokenCaptureClientID_DefaultAndOverride(t *testing.T) {
	// Phase 3 : défaut aligné sur l'app canonique LevelUpClientID (AVANT : HaloToolsClientID).
	t.Setenv("LEVELUP_OAUTH_CLIENT_ID", "")
	if got := TokenCaptureClientID(); got != LevelUpClientID {
		t.Errorf("TokenCaptureClientID défaut = %q, want %q (LevelUp Halo)", got, LevelUpClientID)
	}
	t.Setenv("LEVELUP_OAUTH_CLIENT_ID", "39829f7a-shared")
	if got := TokenCaptureClientID(); got != "39829f7a-shared" {
		t.Errorf("TokenCaptureClientID override = %q, want 39829f7a-shared", got)
	}
}

// TestBuildAuthorizeURL_ClientIDGolden caractérise le client_id réellement émis
// dans l'URL /authorize (golden). Le SSO web passe par le seam.
func TestBuildAuthorizeURL_ClientIDGolden(t *testing.T) {
	redirect := "https://lvelup.info/auth/xbox/callback"

	t.Setenv("LEVELUP_OAUTH_CLIENT_ID", "")
	def, _ := url.Parse(BuildAuthorizeURL(redirect, "state123", ""))
	if got := def.Query().Get("client_id"); got != LevelUpClientID {
		t.Errorf("client_id défaut = %q, want %q (LevelUp Halo)", got, LevelUpClientID)
	}

	t.Setenv("LEVELUP_OAUTH_CLIENT_ID", "39829f7a-prod")
	ovr, _ := url.Parse(BuildAuthorizeURL(redirect, "state123", ""))
	if got := ovr.Query().Get("client_id"); got != "39829f7a-prod" {
		t.Errorf("client_id override = %q, want 39829f7a-prod", got)
	}
}
