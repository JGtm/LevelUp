// Package domain — auth.go : types pour le Device Code Flow et l'authentification Halo.
//
// Sprint 15 :
//
//	POST /auth/device-flow/start      → DeviceFlowStartResponse
//	GET  /auth/device-flow/{id}       → DeviceFlowStatusResponse
package domain

import "time"

// DeviceFlowStartResponse est la réponse à POST /auth/device-flow/start.
// Le frontend doit afficher user_code à l'utilisateur et ouvrir verification_uri.
type DeviceFlowStartResponse struct {
	AttemptID       string `json:"attempt_id"`
	UserCode        string `json:"user_code" doc:"Code à afficher à l'utilisateur"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in" doc:"Durée de validité en secondes"`
	PollIntervalSec int    `json:"poll_interval_seconds" default:"5"`
}

// DeviceFlowStatusResponse est la réponse à GET /auth/device-flow/{id}.
type DeviceFlowStatusResponse struct {
	AttemptID   string  `json:"attempt_id"`
	Status      string  `json:"status" enum:"pending,authorized,provisioned,failed,expired"`
	Gamertag    *string `json:"gamertag,omitempty"`
	XUID        *string `json:"xuid,omitempty"`
	ErrorCode   *string `json:"error_code,omitempty"`
	ErrorDetail *string `json:"error_detail,omitempty"`
}

// HaloTokens regroupe les tokens Halo Infinite obtenus après la chaîne d'échange.
//
// SpartanExpiresAt porte l'expiry RÉEL du Spartan token (champ `ExpiresUtc` de la
// réponse /spartan-token), et non un TTL deviné. Zéro = inconnu (sources legacy /
// imports qui ne fournissent pas l'expiry) → les consommateurs appliquent alors un
// fallback conservateur. Cf. ADR auth + player_token_cache.go.
type HaloTokens struct {
	SpartanToken     string
	ClearanceToken   string
	SpartanExpiresAt time.Time
}
