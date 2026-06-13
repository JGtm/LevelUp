// Package auth — azure_credentials.go : SOURCE UNIQUE des credentials Azure
// (client_id / client_secret) pour les flux OAuth v2 Microsoft.
//
// Seam introduit pour centraliser des lectures d'environnement auparavant
// DUPLIQUÉES dans auth_code.go (Authorization Code Flow SSO) et oauth_refresh.go
// (refresh token). Cf. .ai/PLAN_AUTH_HARDENING_OPTIONAL.md.
//
// Phase 3 (uniformisation) : SSO + refresh + token-capture sont consolidés sur
// l'app canonique e1cb35ab (« LevelUp Halo », publique) via LEVELUP_OAUTH_CLIENT_ID.
// L'ancienne app « Spartan Graph » 39829f7a n'est plus utilisée. Le changement est
// documenté par le diff des golden tests (azure_credentials_test.go).
package auth

import "os"

// AzureOAuthClient est le client Azure utilisé pour les échanges OAuth v2
// Microsoft (Authorization Code Flow SSO web + refresh token). Construire via
// ResolveAzureOAuthClient.
type AzureOAuthClient struct {
	// ClientID : LEVELUP_OAUTH_CLIENT_ID si défini, sinon LevelUpClientID (e1cb35ab).
	ClientID string
	// rawSecret : valeur brute de SPNKR_AZURE_CLIENT_SECRET (éventuellement vide).
	// Non exporté : passer par SecretToSend() qui applique la garde public/confidentiel.
	rawSecret string
}

// ResolveAzureOAuthClient lit le client OAuth Azure (SSO web + refresh) depuis
// l'environnement. Lecteur UNIQUE de LEVELUP_OAUTH_CLIENT_ID / SPNKR_AZURE_CLIENT_SECRET
// côté prod (garanti par le test sentinelle Guard 4).
//
// Phase 3 (uniformisation) : LEVELUP_OAUTH_CLIENT_ID si défini, sinon LevelUpClientID
// (= e1cb35ab, app canonique « LevelUp Halo », PUBLIQUE). On NE lit PLUS
// SPNKR_AZURE_CLIENT_ID (ancienne app « Spartan Graph » 39829f7a) : SSO + refresh
// + token-capture sont désormais sur l'app canonique. e1cb35ab étant publique,
// SecretToSend() renvoie "" — la sécurité repose sur PKCE (RFC 7636).
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

// SecretToSend retourne le client_secret à inclure dans l'échange OAuth, ou ""
// pour un flux client public. Garde anti-AADSTS90023 : on n'envoie un secret que
// s'il est défini ET que le client n'est pas LevelUpClientID (app publique, qui
// rejette tout secret). Le retry sans secret sur AADSTS90023 reste géré par
// l'appelant (oauth_refresh) pour les RT émis par un flux public.
func (c AzureOAuthClient) SecretToSend() string {
	if c.rawSecret != "" && c.ClientID != LevelUpClientID {
		return c.rawSecret
	}
	return ""
}

// DeviceFlowClientID est le client_id du Device Code Flow (app publique
// "LevelUp Halo"). Exposé ici pour que TOUS les client_id Azure transitent par
// ce module (source unique de vérité).
func DeviceFlowClientID() string {
	return LevelUpClientID
}

// TokenCaptureClientID retourne le client_id du CLI de génération manuelle de
// tokens (cmd/token-capture). Phase 3 : aligné sur l'app canonique —
// LEVELUP_OAUTH_CLIENT_ID si défini, sinon LevelUpClientID (= e1cb35ab).
//
// ⚠️ COUPLAGE CRITIQUE : le client_id du token capturé DOIT matcher celui du refresh
// serveur (ResolveAzureOAuthClient) — un refresh_token est lié à son client émetteur,
// sinon le refresh échoue (token révoqué). Les deux lisent désormais la MÊME source
// (LEVELUP_OAUTH_CLIENT_ID, même défaut e1cb35ab) → convergence garantie.
func TokenCaptureClientID() string {
	if v := os.Getenv("LEVELUP_OAUTH_CLIENT_ID"); v != "" {
		return v
	}
	return LevelUpClientID
}
