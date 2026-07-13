// Package auth — xbox_device_code.go : Device Code Flow natif Xbox sur login.live.com.
//
// RFC 8628 pur — aucune dépendance externe (pas de MSAL).
// Utilisé par SISUProvider pour obtenir l'access_token + refresh_token Microsoft
// avant de compléter le flow SISU.
//
// Endpoints :
//   - POST https://login.live.com/oauth20_connect.srf  (start)
//   - POST https://login.live.com/oauth20_token.srf    (poll)
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// xboxDeviceCodeURL : endpoint MSA de démarrage du Device Code Flow natif Xbox.
	// ATTENTION : c'est bien oauth20_connect.srf — l'URL "/oauth20_connect/device"
	// (présente depuis l'introduction du SISU provider) renvoie HTTP 404 chez
	// Microsoft : le login SSO Xbox ne s'amorçait JAMAIS (constaté en local
	// 2026-07-13, vérifié par POST direct : .srf → 200 + device_code, /device → 404).
	xboxDeviceCodeURL = "https://login.live.com/oauth20_connect.srf"
	xboxTokenURL      = "https://login.live.com/oauth20_token.srf"
	// pollSlowDownIncrement est l'augmentation d'intervalle sur slow_down (RFC 8628 §3.5).
	pollSlowDownIncrement = 5
	// pollMaxInterval est le plafond d'intervalle de polling en secondes.
	pollMaxInterval = 60
)

// XboxDeviceCodeResult contient les données retournées par StartXboxDeviceCode.
type XboxDeviceCodeResult struct {
	// UserCode est le code court à saisir sur la page de vérification.
	UserCode string
	// VerificationURL est l'URL à présenter à l'utilisateur.
	VerificationURL string
	// DeviceCode est le code opaque utilisé pour le polling — ne pas exposer.
	DeviceCode string
	// ExpiresIn est la durée de validité en secondes.
	ExpiresIn int
	// Interval est le délai minimum entre deux polls en secondes.
	Interval int
}

// StartXboxDeviceCode initie un Device Code Flow sur login.live.com.
// clientID : appID Xbox (ex: "000000004c20a908").
func StartXboxDeviceCode(ctx context.Context, clientID string) (*XboxDeviceCodeResult, error) {
	return startXboxDeviceCodeWithURL(ctx, http.DefaultClient, clientID, xboxDeviceCodeURL)
}

// startXboxDeviceCodeWithURL est la version testable avec URL configurable.
func startXboxDeviceCodeWithURL(ctx context.Context, client *http.Client, clientID, targetURL string) (*XboxDeviceCodeResult, error) {
	slog.DebugContext(ctx, "xbox_device_code: démarrage Device Code Flow", "client_id", clientID)

	form := url.Values{
		oauthFieldClientID: {clientID},
		oauthFieldScope:    {xboxScopes},
		"response_type":    {oauthFieldDeviceCode},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("xbox_device_code: création requête start: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("xbox_device_code: POST start: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("xbox_device_code: lecture réponse start: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("xbox_device_code: HTTP %d start: %s", resp.StatusCode, raw)
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("xbox_device_code: JSON start: %w", err)
	}

	deviceCode, _ := data[oauthFieldDeviceCode].(string)
	userCode, _ := data["user_code"].(string)
	verificationURI, _ := data["verification_uri"].(string)
	if verificationURI == "" {
		verificationURI, _ = data["verification_url"].(string) // certaines réponses utilisent _url
	}
	expiresIn := int(jsonFloat(data, "expires_in"))
	interval := int(jsonFloat(data, "interval"))
	if interval <= 0 {
		interval = 5 // RFC 8628 §3.4 : valeur par défaut
	}

	if deviceCode == "" {
		return nil, fmt.Errorf("xbox_device_code: device_code absent dans la réponse")
	}

	result := &XboxDeviceCodeResult{
		UserCode:        userCode,
		VerificationURL: verificationURI,
		DeviceCode:      deviceCode,
		ExpiresIn:       expiresIn,
		Interval:        interval,
	}
	slog.InfoContext(ctx, "xbox_device_code: Device Code Flow initialisé", "expires_in", result.ExpiresIn)
	return result, nil
}

// PollXboxDeviceCode attend la complétion du Device Code Flow.
// Bloquant — à appeler dans une goroutine.
// Gère : authorization_pending (continuer), slow_down (augmenter interval),
// succès (access_token + refresh_token), erreur fatale.
func PollXboxDeviceCode(ctx context.Context, clientID, deviceCode string, interval int) (accessToken, refreshToken string, err error) {
	return pollXboxDeviceCodeWithURL(ctx, http.DefaultClient, clientID, deviceCode, interval, xboxTokenURL)
}

// pollXboxDeviceCodeWithURL est la version testable avec URL configurable.
func pollXboxDeviceCodeWithURL(
	ctx context.Context,
	client *http.Client,
	clientID, deviceCode string,
	interval int,
	targetURL string,
) (string, string, error) {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	slog.DebugContext(ctx, "xbox_device_code: début polling")

	for {
		select {
		case <-ctx.Done():
			return "", "", ctx.Err()
		case <-time.After(time.Duration(interval) * time.Second):
		}

		form := url.Values{
			"grant_type":         {"urn:ietf:params:oauth:grant-type:device_code"},
			oauthFieldClientID:   {clientID},
			oauthFieldDeviceCode: {deviceCode},
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, strings.NewReader(form.Encode()))
		if err != nil {
			return "", "", fmt.Errorf("xbox_device_code: création requête poll: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := client.Do(req)
		if err != nil {
			return "", "", fmt.Errorf("xbox_device_code: POST poll: %w", err)
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return "", "", fmt.Errorf("xbox_device_code: JSON poll: %w", err)
		}

		// Succès
		if at, ok := data["access_token"].(string); ok && at != "" {
			rt, _ := data[oauthFieldRefreshToken].(string)
			slog.InfoContext(ctx, "xbox_device_code: Device Code Flow complété")
			return at, rt, nil
		}

		errorCode, _ := data["error"].(string)
		switch errorCode {
		case "authorization_pending":
			slog.DebugContext(ctx, "xbox_device_code: poll en attente")
			continue
		case "slow_down":
			interval += pollSlowDownIncrement
			if interval > pollMaxInterval {
				interval = pollMaxInterval
			}
			slog.WarnContext(ctx, "xbox_device_code: slow_down reçu", "new_interval", interval)
			continue
		case "":
			return "", "", fmt.Errorf("xbox_device_code: réponse inattendue (HTTP %d): %s", resp.StatusCode, raw)
		default:
			errDesc, _ := data["error_description"].(string)
			slog.ErrorContext(ctx, "xbox_device_code: erreur fatale", "error_code", errorCode)
			return "", "", fmt.Errorf("xbox_device_code: erreur fatale %q: %s", errorCode, errDesc)
		}
	}
}

// jsonFloat extrait un nombre JSON en float64 depuis une map.
func jsonFloat(data map[string]any, key string) float64 {
	v, _ := data[key].(float64)
	return v
}
