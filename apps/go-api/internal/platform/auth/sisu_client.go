// Package auth — sisu_client.go : client SISU pour l'authentification Xbox native.
//
// Deux étapes :
//  1. InitSISUSession  → ouvre une session SISU et retourne l'URL OAuth à présenter
//  2. CompleteSISUFlow → échange l'access_token OAuth contre un XSTS direct
//
// Endpoints :
//   - POST https://sisu.xboxlive.com/authenticate  (init)
//   - POST https://sisu.xboxlive.com/authorize     (complete)
package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	sisuAuthenticateURL = "https://sisu.xboxlive.com/authenticate"
	sisuAuthorizeURL    = "https://sisu.xboxlive.com/authorize"
	sisuRedirectURI     = "https://login.live.com/oauth20_desktop.srf"
	sisuSandbox         = "RETAIL"
)

// SISUSession contient le résultat de l'initialisation d'une session SISU.
type SISUSession struct {
	// SessionID est l'identifiant de session retourné dans le header X-SessionId.
	SessionID string
	// MsaOauthRedirect est l'URL OAuth à présenter à l'utilisateur pour s'authentifier.
	MsaOauthRedirect string
}

// GeneratePKCE génère un code_verifier et son code_challenge S256 pour PKCE.
// Retourne (codeVerifier, codeChallenge, error).
func GeneratePKCE() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("sisu: génération PKCE: %w", err)
	}
	verifier := base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])
	return verifier, challenge, nil
}

// InitSISUSession ouvre une session SISU et retourne l'URL OAuth à afficher à l'utilisateur.
// appID   : ex "000000004c20a908" (Halo Waypoint mobile)
// titleID : ex "144209987" (Halo Infinite)
func InitSISUSession(
	ctx context.Context,
	client *http.Client,
	kp *PoPKeyPair,
	deviceToken string,
	appID, titleID string,
	codeChallenge, codeChallengeState string,
) (*SISUSession, error) {
	return initSISUSessionWithURL(ctx, client, kp, deviceToken, appID, titleID, codeChallenge, codeChallengeState, sisuAuthenticateURL)
}

// initSISUSessionWithURL est la version testable avec URL configurable.
func initSISUSessionWithURL(
	ctx context.Context,
	client *http.Client,
	kp *PoPKeyPair,
	deviceToken string,
	appID, titleID string,
	codeChallenge, codeChallengeState string,
	targetURL string,
) (*SISUSession, error) {
	slog.DebugContext(ctx, "sisu: initialisation session", "app_id", appID)

	body := map[string]any{
		"AppId":            appID,
		"TitleId":          titleID,
		"DeviceToken":      deviceToken,
		"Offers":           []string{"service::user.auth.xboxlive.com::MBI_SSL"},
		"ProofKey":         kp.GetProofKey(),
		"RedirectUri":      sisuRedirectURI,
		"Sandbox":          sisuSandbox,
		xboxFieldTokenType: "code",
		"Query": map[string]string{
			"display":               "touch",
			"code_challenge":        codeChallenge,
			"code_challenge_method": "S256",
			"state":                 codeChallengeState,
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("sisu: marshal body init: %w", err)
	}

	sig, err := kp.SignRequest(targetURL, "", string(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("sisu: signature PoP init: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("sisu: création requête init: %w", err)
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
		return nil, fmt.Errorf("sisu: POST init %s: %w", targetURL, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("sisu: lecture réponse init: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.ErrorContext(ctx, "sisu: échec init session", "status", resp.StatusCode)
		return nil, fmt.Errorf("sisu: HTTP %d init: %s", resp.StatusCode, raw)
	}

	sessionID := resp.Header.Get("X-SessionId")
	if sessionID == "" {
		return nil, fmt.Errorf("sisu: header X-SessionId absent dans la réponse")
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("sisu: décodage JSON init: %w", err)
	}
	redirect, _ := data["MsaOauthRedirect"].(string)

	slog.InfoContext(ctx, "sisu: session initiée", "session_id", sessionID)
	return &SISUSession{
		SessionID:        sessionID,
		MsaOauthRedirect: redirect,
	}, nil
}

// CompleteSISUFlow échange l'access_token OAuth contre un XSTS via SISU.
// Retourne un XSTSResult (même struct qu'AcquireXSTSForRTA).
// Note : SISU retourne le XSTS directement dans AuthorizationToken —
// pas besoin de l'étape User Token → XSTS séparée.
func CompleteSISUFlow(
	ctx context.Context,
	client *http.Client,
	kp *PoPKeyPair,
	deviceToken, accessToken string,
	appID, sessionID string,
	_ string, // codeVerifier — réservé pour future utilisation côté SISU si nécessaire
) (*XSTSResult, error) {
	return completeSISUFlowWithURL(ctx, client, kp, deviceToken, accessToken, appID, sessionID, sisuAuthorizeURL)
}

// completeSISUFlowWithURL est la version testable avec URL configurable.
func completeSISUFlowWithURL(
	ctx context.Context,
	client *http.Client,
	kp *PoPKeyPair,
	deviceToken, accessToken string,
	appID, sessionID string,
	targetURL string,
) (*XSTSResult, error) {
	slog.DebugContext(ctx, "sisu: complétion flow", "session_id", sessionID)

	body := map[string]any{
		"AppId":             appID,
		"DeviceToken":       deviceToken,
		"ProofKey":          kp.GetProofKey(),
		"Sandbox":           sisuSandbox,
		"AccessToken":       "t=" + accessToken,
		"UseModernGamertag": true,
		"SiteName":          "user.auth.xboxlive.com",
		"SessionId":         sessionID,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("sisu: marshal body complete: %w", err)
	}

	sig, err := kp.SignRequest(targetURL, "", string(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("sisu: signature PoP complete: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("sisu: création requête complete: %w", err)
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
		return nil, fmt.Errorf("sisu: POST complete %s: %w", targetURL, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("sisu: lecture réponse complete: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.ErrorContext(ctx, "sisu: échec complétion", "status", resp.StatusCode)
		return nil, fmt.Errorf("sisu: HTTP %d complete: %s", resp.StatusCode, raw)
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("sisu: décodage JSON complete: %w", err)
	}

	// SISU retourne le XSTS dans "AuthorizationToken"
	authToken, ok := data["AuthorizationToken"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("sisu: champ AuthorizationToken absent ou mauvais type")
	}

	token, ok := authToken["Token"].(string)
	if !ok || token == "" {
		return nil, fmt.Errorf("sisu: Token absent dans AuthorizationToken")
	}

	gamertag, xuid := extractDisplayClaims(authToken)
	userHash := extractUserHash(authToken)
	notAfter := extractNotAfter(authToken)

	result := &XSTSResult{
		Token:    token,
		UserHash: userHash,
		Gamertag: gamertag,
		XUID:     xuid,
		NotAfter: notAfter,
	}

	slog.InfoContext(ctx, "sisu: XSTS obtenu",
		"not_after", result.NotAfter,
		"gamertag", result.Gamertag,
	)
	return result, nil
}
