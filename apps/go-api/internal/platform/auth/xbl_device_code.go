// Package auth — xbl_device_code.go : façade d'exploitation du Device Code Flow natif Xbox
// (protocole dans `internal/haloapi/xblauth/xbox_device_code.go`, plan D3 lot 2). La façade
// fournit le client HTTP (et son délai) sous la garde `netguard` de chaque surface
// (« xbox_device_code.start », « xbox_device_code.poll »), comme avant le déplacement.
package auth

import (
	"context"
	"net/http"

	"levelup/go-api/internal/haloapi/xblauth"
)

// XboxDeviceCodeResult contient les données retournées par StartXboxDeviceCode.
type XboxDeviceCodeResult = xblauth.XboxDeviceCodeResult

// StartXboxDeviceCode initie un Device Code Flow sur login.live.com.
// clientID : appID Xbox (ex: "000000004c20a908").
func StartXboxDeviceCode(ctx context.Context, clientID string) (*XboxDeviceCodeResult, error) {
	return startXboxDeviceCodeWithURL(ctx, http.DefaultClient, clientID, xblauth.URLConnectMSA)
}

// startXboxDeviceCodeWithURL est la version testable avec URL configurable. client nil = client
// à délai xblExchangeTimeout.
func startXboxDeviceCodeWithURL(ctx context.Context, client *http.Client, clientID, targetURL string) (*XboxDeviceCodeResult, error) {
	if client == nil {
		client = &http.Client{Timeout: xblExchangeTimeout}
	}
	// Mode démo : aucune sortie tierce (le flux est démarrable depuis le web, pas seulement
	// depuis la CLI). Défense en profondeur : la garde démo en écriture refuse déjà les routes.
	flux := xblauth.FluxAppareil{Doer: netguardDoer{surface: "xbox_device_code.start", next: client}, URLConnect: targetURL}
	return flux.Start(ctx, clientID)
}

// PollXboxDeviceCode attend la complétion du Device Code Flow.
// Bloquant — à appeler dans une goroutine.
// Gère : authorization_pending (continuer), slow_down (augmenter interval),
// succès (access_token + refresh_token), erreur fatale.
func PollXboxDeviceCode(ctx context.Context, clientID, deviceCode string, interval int) (accessToken, refreshToken string, err error) {
	return pollXboxDeviceCodeWithURL(ctx, http.DefaultClient, clientID, deviceCode, interval, xblauth.URLTokenMSA)
}

// pollXboxDeviceCodeWithURL est la version testable avec URL configurable. client nil = client à
// délai xblExchangeTimeout.
func pollXboxDeviceCodeWithURL(
	ctx context.Context,
	client *http.Client,
	clientID, deviceCode string,
	interval int,
	targetURL string,
) (string, string, error) {
	if client == nil {
		client = &http.Client{Timeout: xblExchangeTimeout}
	}
	flux := xblauth.FluxAppareil{Doer: netguardDoer{surface: "xbox_device_code.poll", next: client}, URLToken: targetURL}
	return flux.Poll(ctx, clientID, deviceCode, interval)
}
