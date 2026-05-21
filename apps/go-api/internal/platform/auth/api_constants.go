// Package auth — api_constants.go : champs JSON et form-encoded partagés.
//
// Ce fichier centralise les noms de champs des payloads OAuth v2 / Xbox /
// XSTS / SISU pour réduire la duplication signalée par goconst.
// Ces noms sont définis par les protocoles externes (Microsoft, Xbox Live)
// — ne pas modifier sans coordination avec les endpoints distants.
package auth

// Champs JSON Xbox / XSTS payloads (RelyingParty, TokenType, Properties, Token).
const (
	xboxFieldRelyingParty = "RelyingParty"
	xboxFieldTokenType    = "TokenType"
	xboxFieldProperties   = "Properties"
	xboxFieldToken        = "Token"
)

// Champs form-encoded OAuth v2 / Xbox Device Code (RFC 8628).
const (
	oauthFieldClientID     = "client_id"
	oauthFieldScope        = "scope"
	oauthFieldCode         = "code"
	oauthFieldRefreshToken = "refresh_token"
	oauthFieldDeviceCode   = "device_code"
)
