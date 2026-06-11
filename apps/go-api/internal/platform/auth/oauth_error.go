// Package auth — oauth_error.go : erreur typée du endpoint OAuth v2 + classification.
//
// Permet aux couches amont (pool/resolver, dashboard admin) de distinguer un
// échec de configuration (déterministe, inutile de retenter à l'identique),
// un refresh_token révoqué (action user : re-capture) et un incident
// transitoire (réseau, 5xx) qui mérite un retry.
package auth

import (
	"errors"
	"fmt"
	"strings"
)

// AuthErrorClass est la classe d'un échec d'authentification OAuth.
type AuthErrorClass string

const (
	// AuthErrorConfig : requête refusée de façon déterministe (app/client mal
	// configuré, ex. AADSTS90023 client_secret envoyé à une app publique).
	AuthErrorConfig AuthErrorClass = "config"
	// AuthErrorRevoked : refresh_token mort — ré-authentification interactive requise.
	AuthErrorRevoked AuthErrorClass = "revoked"
	// AuthErrorTransient : incident réseau/serveur — un retry ultérieur peut réussir.
	AuthErrorTransient AuthErrorClass = "transient"
)

// OAuthExchangeError est l'erreur structurée renvoyée par le endpoint
// /oauth2/v2.0/token (champ JSON `error` + `error_description`).
type OAuthExchangeError struct {
	ErrorCode   string // ex. "invalid_request", "invalid_grant"
	Description string // ex. "AADSTS90023: Public clients can't send a client secret..."
}

func (e *OAuthExchangeError) Error() string {
	return fmt.Sprintf("oauth_refresh: %s — %s", e.ErrorCode, e.Description)
}

// IsSecretRejected retourne true pour AADSTS90023 : l'app Azure est déclarée
// client public et refuse le client_secret — le même échange sans secret est
// la correction attendue.
func (e *OAuthExchangeError) IsSecretRejected() bool {
	return e.ErrorCode == "invalid_request" && strings.Contains(e.Description, "AADSTS90023")
}

// Class mappe le code d'erreur OAuth vers une classe d'échec.
func (e *OAuthExchangeError) Class() AuthErrorClass {
	switch e.ErrorCode {
	case "invalid_grant", "interaction_required":
		return AuthErrorRevoked
	case "invalid_request", "invalid_client", "unauthorized_client", "unsupported_grant_type":
		return AuthErrorConfig
	default:
		return AuthErrorTransient
	}
}

// ClassifyAuthError extrait la classe d'une erreur quelconque de la chaîne
// d'authentification. Toute erreur non typée (réseau, JSON, timeout) est
// considérée transitoire.
func ClassifyAuthError(err error) AuthErrorClass {
	var oerr *OAuthExchangeError
	if errors.As(err, &oerr) {
		return oerr.Class()
	}
	return AuthErrorTransient
}
