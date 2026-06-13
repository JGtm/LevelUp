// Package auth — azure_credentials_test.go : golden / caractérisation du seam des
// credentials Azure. Ces tests FIGENT le comportement ACTUEL (Phase 1). Ils doivent
// rester verts après le refactor seam (preuve : aucun changement). En Phase 2
// (uniformisation vers l'app canonique), le diff de ces assertions = le changement assumé.
package auth

import (
	"net/url"
	"testing"
)

func TestResolveAzureOAuthClient_DefaultIsLevelUpHalo(t *testing.T) {
	t.Setenv("SPNKR_AZURE_CLIENT_ID", "")
	t.Setenv("SPNKR_AZURE_CLIENT_SECRET", "")

	c := ResolveAzureOAuthClient()
	if c.ClientID != LevelUpClientID {
		t.Errorf("ClientID défaut = %q, want %q (LevelUp Halo)", c.ClientID, LevelUpClientID)
	}
	if s := c.SecretToSend(); s != "" {
		t.Errorf("SecretToSend défaut = %q, want \"\" (pas de secret)", s)
	}
}

func TestResolveAzureOAuthClient_EnvClientIDOverride(t *testing.T) {
	t.Setenv("SPNKR_AZURE_CLIENT_ID", "custom-app-id")
	t.Setenv("SPNKR_AZURE_CLIENT_SECRET", "")

	if c := ResolveAzureOAuthClient(); c.ClientID != "custom-app-id" {
		t.Errorf("ClientID = %q, want custom-app-id", c.ClientID)
	}
}

func TestSecretToSend_ConfidentialClientSendsSecret(t *testing.T) {
	// Client != LevelUpClientID + secret défini → secret envoyé (flux confidentiel).
	t.Setenv("SPNKR_AZURE_CLIENT_ID", "39829f7a-confidential")
	t.Setenv("SPNKR_AZURE_CLIENT_SECRET", "s3cret")

	if got := ResolveAzureOAuthClient().SecretToSend(); got != "s3cret" {
		t.Errorf("SecretToSend = %q, want s3cret (client confidentiel)", got)
	}
}

func TestSecretToSend_PublicClientNeverSendsSecret(t *testing.T) {
	// Garde anti-AADSTS90023 : LevelUpClientID (public) + secret → JAMAIS de secret.
	t.Setenv("SPNKR_AZURE_CLIENT_ID", LevelUpClientID)
	t.Setenv("SPNKR_AZURE_CLIENT_SECRET", "s3cret")

	if got := ResolveAzureOAuthClient().SecretToSend(); got != "" {
		t.Errorf("SecretToSend (client public) = %q, want \"\" (anti-AADSTS90023)", got)
	}
}

func TestSecretToSend_NoSecretEnv(t *testing.T) {
	t.Setenv("SPNKR_AZURE_CLIENT_ID", "39829f7a-confidential")
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
	// Défaut (pas d'env) : HaloToolsClientID — DIFFÈRE volontairement du défaut
	// serveur (LevelUpClientID), état actuel à figer.
	t.Setenv("SPNKR_AZURE_CLIENT_ID", "")
	if got := TokenCaptureClientID(); got != HaloToolsClientID {
		t.Errorf("TokenCaptureClientID défaut = %q, want %q (halo-tools)", got, HaloToolsClientID)
	}
	// Override env (convergence avec le refresh serveur en prod).
	t.Setenv("SPNKR_AZURE_CLIENT_ID", "39829f7a-shared")
	if got := TokenCaptureClientID(); got != "39829f7a-shared" {
		t.Errorf("TokenCaptureClientID override = %q, want 39829f7a-shared", got)
	}
}

// TestBuildAuthorizeURL_ClientIDGolden caractérise le client_id réellement émis
// dans l'URL /authorize (golden). Le SSO web passe par le seam.
func TestBuildAuthorizeURL_ClientIDGolden(t *testing.T) {
	redirect := "https://lvelup.info/auth/xbox/callback"

	t.Setenv("SPNKR_AZURE_CLIENT_ID", "")
	def, _ := url.Parse(BuildAuthorizeURL(redirect, "state123", ""))
	if got := def.Query().Get("client_id"); got != LevelUpClientID {
		t.Errorf("client_id défaut = %q, want %q (LevelUp Halo)", got, LevelUpClientID)
	}

	t.Setenv("SPNKR_AZURE_CLIENT_ID", "39829f7a-prod")
	ovr, _ := url.Parse(BuildAuthorizeURL(redirect, "state123", ""))
	if got := ovr.Query().Get("client_id"); got != "39829f7a-prod" {
		t.Errorf("client_id override = %q, want 39829f7a-prod", got)
	}
}
