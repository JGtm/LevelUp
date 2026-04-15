// Package auth — halo_exchange.go : échange access_token Microsoft → tokens Halo Infinite.
//
// Ce module est SANS état (stateless) : aucune persistance, aucune logique de cache.
// Reçoit un access_token Microsoft et retourne les tokens Halo prêts à l'emploi.
//
// Chaîne d'échange (portage de SPNKr) :
//
//	access_token (Microsoft)
//	→ User Token (XBL)  [user.auth.xboxlive.com]
//	→ XSTS Token Halo   [xsts.auth.xboxlive.com, audience Halo]
//	→ Spartan Token     [settings.svc.halowaypoint.com/spartan-token]
//	→ Clearance Token   [settings.svc.halowaypoint.com/oban/flight-configurations/...]
package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"levelup/go-api/internal/domain"
)

const (
	xblUserAuthURL   = "https://user.auth.xboxlive.com/user/authenticate"
	xstsAuthorizeURL = "https://xsts.auth.xboxlive.com/xsts/authorize"
	spartanTokenURL  = "https://settings.svc.halowaypoint.com/spartan-token"
	clearanceURL     = "https://settings.svc.halowaypoint.com/oban/flight-configurations/titles/hi/audiences/RETAIL/active"

	xstsHaloAudience = "https://prod.xsts.halowaypoint.com/"
)

// ExchangeAccessToken échange un access_token Microsoft contre des tokens Halo Infinite.
// Implémente la chaîne : access_token → XBL user → XSTS Halo → Spartan → Clearance.
func ExchangeAccessToken(ctx context.Context, accessToken string) (*domain.HaloTokens, error) {
	client := &http.Client{Timeout: 20 * time.Second}

	// Étape 1 : User Token XBL
	userToken, err := requestUserToken(ctx, client, accessToken)
	if err != nil {
		return nil, fmt.Errorf("user token XBL: %w", err)
	}

	// Étape 2 : XSTS Token Halo
	xstsToken, err := requestXSTSToken(ctx, client, userToken, xstsHaloAudience)
	if err != nil {
		return nil, fmt.Errorf("XSTS token Halo: %w", err)
	}

	// Étape 3 : Spartan Token
	spartanToken, err := requestSpartanToken(ctx, client, xstsToken)
	if err != nil {
		return nil, fmt.Errorf("spartan token: %w", err)
	}

	// Étape 4 : Clearance Token
	clearanceToken, err := requestClearanceToken(ctx, client, spartanToken)
	if err != nil {
		return nil, fmt.Errorf("clearance token: %w", err)
	}

	return &domain.HaloTokens{
		SpartanToken:   spartanToken,
		ClearanceToken: clearanceToken,
	}, nil
}

// =============================================================================
// Étapes de la chaîne d'échange
// =============================================================================

// requestUserToken obtient un User Token XBL depuis un access_token Microsoft.
func requestUserToken(ctx context.Context, client *http.Client, accessToken string) (string, error) {
	body := map[string]any{
		"RelyingParty": "http://auth.xboxlive.com",
		"TokenType":    "JWT",
		"Properties": map[string]string{
			"AuthMethod": "RPS",
			"SiteName":   "user.auth.xboxlive.com",
			"RpsTicket":  "d=" + accessToken,
		},
	}
	resp, err := postJSON(ctx, client, xblUserAuthURL, map[string]string{
		"x-xbl-contract-version": "1",
	}, body)
	if err != nil {
		return "", err
	}
	token, ok := resp["Token"].(string)
	if !ok || token == "" {
		return "", fmt.Errorf("Token absent dans la réponse XBL")
	}
	return token, nil
}

// requestXSTSToken échange un User Token XBL contre un XSTS Token.
func requestXSTSToken(ctx context.Context, client *http.Client, userToken, relyingParty string) (string, error) {
	body := map[string]any{
		"RelyingParty": relyingParty,
		"TokenType":    "JWT",
		"Properties": map[string]any{
			"UserTokens": []string{userToken},
			"SandboxId":  "RETAIL",
		},
	}
	resp, err := postJSON(ctx, client, xstsAuthorizeURL, map[string]string{
		"x-xbl-contract-version": "1",
	}, body)
	if err != nil {
		return "", err
	}
	token, ok := resp["Token"].(string)
	if !ok || token == "" {
		return "", fmt.Errorf("Token absent dans la réponse XSTS")
	}
	return token, nil
}

// requestSpartanToken échange un XSTS Token Halo contre un Spartan Token.
func requestSpartanToken(ctx context.Context, client *http.Client, xstsToken string) (string, error) {
	body := map[string]any{
		"Audience":   "urn:343:s3:services",
		"MinVersion": "4",
		"Proof": []map[string]string{
			{"Token": xstsToken, "TokenType": "Xbox_XSTSv3"},
		},
	}
	resp, err := postJSON(ctx, client, spartanTokenURL, map[string]string{
		"Accept": "application/json",
	}, body)
	if err != nil {
		return "", err
	}
	token, ok := resp["SpartanToken"].(string)
	if !ok || token == "" {
		return "", fmt.Errorf("SpartanToken absent dans la réponse")
	}
	return token, nil
}

// requestClearanceToken obtient le Clearance Token (FlightConfigurationId).
func requestClearanceToken(ctx context.Context, client *http.Client, spartanToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, clearanceURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("x-343-authorization-spartan", spartanToken)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET clearance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("clearance token HTTP %d", resp.StatusCode)
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", fmt.Errorf("décodage clearance: %w", err)
	}

	token, ok := data["FlightConfigurationId"].(string)
	if !ok || token == "" {
		return "", fmt.Errorf("FlightConfigurationId absent dans la réponse clearance")
	}
	return token, nil
}

// =============================================================================
// Helpers HTTP
// =============================================================================

// postJSON effectue un POST JSON et retourne le body décodé.
func postJSON(ctx context.Context, client *http.Client, url string, headers map[string]string, body any) (map[string]any, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lecture réponse %s: %w", url, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d depuis %s: %s", resp.StatusCode, url, raw)
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("JSON decode %s: %w", url, err)
	}
	return data, nil
}
