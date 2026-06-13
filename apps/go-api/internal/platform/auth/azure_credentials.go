// Package auth — azure_credentials.go : SOURCE UNIQUE des credentials Azure
// (client_id / client_secret) pour les flux OAuth v2 Microsoft.
//
// Seam introduit pour centraliser des lectures d'environnement auparavant
// DUPLIQUÉES dans auth_code.go (Authorization Code Flow SSO) et oauth_refresh.go
// (refresh token). Cf. .ai/PLAN_AUTH_HARDENING_OPTIONAL.md — Phase 1.
//
// IMPORTANT : ce module ne change AUCUN comportement. Il déplace les lectures
// existantes derrière une API unique pour préparer (Phase 2) l'uniformisation
// vers l'app canonique. Le comportement actuel est figé par des golden tests
// (azure_credentials_test.go).
package auth

import "os"

// AzureOAuthClient est le client Azure utilisé pour les échanges OAuth v2
// Microsoft (Authorization Code Flow SSO web + refresh token). Construire via
// ResolveAzureOAuthClient.
type AzureOAuthClient struct {
	// ClientID : SPNKR_AZURE_CLIENT_ID si défini, sinon LevelUpClientID.
	ClientID string
	// rawSecret : valeur brute de SPNKR_AZURE_CLIENT_SECRET (éventuellement vide).
	// Non exporté : passer par SecretToSend() qui applique la garde public/confidentiel.
	rawSecret string
}

// ResolveAzureOAuthClient lit le client OAuth Azure depuis l'environnement.
// Lecteur UNIQUE de SPNKR_AZURE_CLIENT_ID / SPNKR_AZURE_CLIENT_SECRET côté prod
// (garanti par le test sentinelle Guard 4).
//
// Comportement (inchangé) : SPNKR_AZURE_CLIENT_ID si défini, sinon LevelUpClientID.
func ResolveAzureOAuthClient() AzureOAuthClient {
	clientID := os.Getenv("SPNKR_AZURE_CLIENT_ID")
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
// tokens (cmd/token-capture). Défaut : HaloToolsClientID (app publique "halo-tools"),
// override par SPNKR_AZURE_CLIENT_ID.
//
// ⚠️ COUPLAGE CRITIQUE : le client_id du token capturé DOIT matcher celui utilisé
// par le refresh serveur (ResolveAzureOAuthClient) — un refresh_token est lié à son
// client émetteur, sinon le refresh échoue (token révoqué). En prod, SPNKR_AZURE_CLIENT_ID
// est posé → les deux convergent. NB : le DÉFAUT diffère volontairement de
// ResolveAzureOAuthClient (HaloToolsClientID ici vs LevelUpClientID) — c'est l'état
// actuel à figer ; Phase 2 (uniformisation) alignera les deux défauts sur l'app canonique.
func TokenCaptureClientID() string {
	if v := os.Getenv("SPNKR_AZURE_CLIENT_ID"); v != "" {
		return v
	}
	return HaloToolsClientID
}
