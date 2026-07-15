// Package auth — device_token.go : obtention d'un Device Token Xbox via PoP.
//
// Endpoint : POST https://device.auth.xboxlive.com/device/authenticate
// Le Device Token est signé avec la paire PoP et nécessaire pour initier une session SISU.
package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	deviceAuthURL = "https://device.auth.xboxlive.com/device/authenticate"
	// deviceType "Android" (pas "Win32") : SISU /authorize ne fait confiance
	// qu'aux device tokens des plateformes mobiles/console (Android, iOS,
	// Nintendo) — un device token Win32 exige une attestation TPM que nous
	// n'avons pas. Aligné sur MinecraftAuth (XblDeviceAuthenticateRequest,
	// défaut "Android") et XAL.
	deviceType         = "Android"
	deviceRelyingParty = "http://auth.xboxlive.com"
)

// RequestDeviceToken obtient un Device Token Xbox signé avec la paire PoP.
// Retourne le token JWT opaque à passer à CompleteSISUFlow (/authorize).
func RequestDeviceToken(ctx context.Context, client *http.Client, kp *PoPKeyPair) (string, error) {
	return requestDeviceTokenWithURL(ctx, client, kp, deviceAuthURL)
}

// requestDeviceTokenWithURL est la version testable avec URL configurable.
func requestDeviceTokenWithURL(ctx context.Context, client *http.Client, kp *PoPKeyPair, targetURL string) (string, error) {
	slog.DebugContext(ctx, "device_token: requête Device Token Xbox")

	deviceID := "{" + strings.ToUpper(uuid.New().String()) + "}"
	body := map[string]any{
		xboxFieldRelyingParty: deviceRelyingParty,
		xboxFieldTokenType:    "JWT",
		xboxFieldProperties: map[string]any{
			"AuthMethod": "ProofOfPossession",
			"DeviceType": deviceType,
			"Id":         deviceID,
			"ProofKey":   kp.GetProofKey(),
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("device_token: marshal body: %w", err)
	}

	sig, err := kp.SignRequest(targetURL, "", string(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("device_token: signature PoP: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("device_token: création requête: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-xbl-contract-version", "2")
	req.Header.Set("Signature", sig)

	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("device_token: POST %s: %w", targetURL, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("device_token: lecture réponse: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.ErrorContext(ctx, "device_token: échec", "status", resp.StatusCode)
		return "", fmt.Errorf("device_token: HTTP %d: %s", resp.StatusCode, raw)
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return "", fmt.Errorf("device_token: décodage JSON: %w", err)
	}
	token, ok := data[xboxFieldToken].(string)
	if !ok || token == "" {
		return "", fmt.Errorf("device_token: champ Token absent ou vide")
	}

	slog.InfoContext(ctx, "device_token: Device Token obtenu")
	return token, nil
}
