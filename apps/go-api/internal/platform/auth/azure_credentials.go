// Package auth — azure_credentials.go : SOURCE UNIQUE des credentials Azure
// (client_id / client_secret) pour les flux OAuth v2 Microsoft.
//
// Seam introduit pour centraliser des lectures d'environnement auparavant
// DUPLIQUÉES dans auth_code.go (Authorization Code Flow SSO) et oauth_refresh.go
// (refresh token). Cf. .ai/PLAN_AUTH_HARDENING_OPTIONAL.md.
//
// Phase 3 (uniformisation) : SSO + refresh + token-capture sont consolidés sur
// l'app canonique e1cb35ab (« LevelUp Halo ») via LEVELUP_OAUTH_CLIENT_ID. Son
// redirect est en plateforme « Web » → client CONFIDENTIEL → le client_secret est
// REQUIS (PKCE est additif). L'ancienne app « Spartan Graph » 39829f7a n'est plus
// utilisée. Le changement est documenté par le diff des golden tests.
package auth

import (
	"os"

	"levelup/go-api/internal/domain/title"
)

// Constantes de l'app Azure (rapatriées de msal_client.go lors du retrait de
// MSAL, 2026-07-15 — elles servent les flux OAuth v2 Azure encore actifs :
// SSO web par code d'autorisation + refresh des RT Azure existants).
const (
	// LevelUpClientID est le client_id de l'app Azure "LevelUp Halo".
	LevelUpClientID = "e1cb35ab-c41a-4ee5-a7a1-22ea4e94cdca" // pragma: allowlist secret

	// MSALAuthority pour les comptes personnels Microsoft (Xbox Live) —
	// endpoint v2 « consumers », utilisé par l'URL d'authorize du SSO web.
	MSALAuthority = "https://login.microsoftonline.com/consumers"
)

// XboxScopes — scopes Xbox Live de l'app Azure, dérivés du descripteur
// (MT-02, source unique).
var XboxScopes = title.DefaultHaloAuthDescriptor().OAuthScopes

// AzureOAuthClient est le client Azure utilisé pour les échanges OAuth v2
// Microsoft (Authorization Code Flow SSO web + refresh token). Construire via
// ResolveAzureOAuthClient.
type AzureOAuthClient struct {
	// ClientID : LEVELUP_OAUTH_CLIENT_ID si défini, sinon LevelUpClientID (e1cb35ab).
	ClientID string
	// rawSecret : valeur brute de SPNKR_AZURE_CLIENT_SECRET (éventuellement vide).
	// Non exporté : passer par SecretToSend().
	rawSecret string
}

// ResolveAzureOAuthClient lit le client OAuth Azure (SSO web + refresh) depuis
// l'environnement. Lecteur UNIQUE de SPNKR_AZURE_CLIENT_SECRET côté prod (garanti
// par le test sentinelle Guard 4).
//
// Phase 3 : LEVELUP_OAUTH_CLIENT_ID si défini, sinon LevelUpClientID (= e1cb35ab,
// app canonique). On NE lit PLUS SPNKR_AZURE_CLIENT_ID (ancienne app 39829f7a).
func ResolveAzureOAuthClient() AzureOAuthClient {
	clientID := os.Getenv("LEVELUP_OAUTH_CLIENT_ID")
	if clientID == "" {
		clientID = LevelUpClientID
	}
	return AzureOAuthClient{
		ClientID:  clientID,
		rawSecret: os.Getenv("SPNKR_AZURE_CLIENT_SECRET"),
	}
}

// SecretToSend retourne le client_secret à inclure dans l'échange OAuth (ou "" si
// non configuré). e1cb35ab (app canonique) a son redirect en plateforme « Web » →
// client CONFIDENTIEL → Microsoft EXIGE le secret pour l'Authorization Code Flow
// (AADSTS70002 sinon ; PKCE est additif, pas un substitut).
//
// Cas limite : un refresh_token émis par un flux PUBLIC (device code) refuse le
// secret (AADSTS90023) — l'appelant oauth_refresh retente alors sans secret.
func (c AzureOAuthClient) SecretToSend() string {
	return c.rawSecret
}

// DeviceFlowClientID est le client_id du Device Code Flow (app « LevelUp Halo »,
// utilisée en client PUBLIC côté MSAL — sans secret). Source unique de vérité.
func DeviceFlowClientID() string {
	return LevelUpClientID
}

// TokenCaptureClientID retourne le client_id du CLI de génération manuelle de
// tokens (cmd/token-capture). Phase 3 : aligné sur l'app canonique —
// LEVELUP_OAUTH_CLIENT_ID si défini, sinon LevelUpClientID (= e1cb35ab).
//
// ⚠️ COUPLAGE CRITIQUE : le client_id du token capturé DOIT matcher celui du refresh
// serveur (ResolveAzureOAuthClient) — un refresh_token est lié à son client émetteur.
// Les deux lisent désormais la MÊME source (LEVELUP_OAUTH_CLIENT_ID) → convergence.
func TokenCaptureClientID() string {
	if v := os.Getenv("LEVELUP_OAUTH_CLIENT_ID"); v != "" {
		return v
	}
	return LevelUpClientID
}
