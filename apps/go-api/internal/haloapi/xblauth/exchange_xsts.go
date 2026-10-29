// exchange_xsts.go : jambes XSTS, Spartan et Clearance de la chaîne d'échange, et le transport
// JSON (POST) qu'elles partagent avec la jambe User Token (exchange_access.go) et l'XSTS RTA
// (xsts_rta.go).

package xblauth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"levelup/go-api/internal/haloapi"
)

// requestXSTSToken échange un User Token XBL contre un XSTS Token.
// Retourne (token, gamertag, xuid, error) — gamertag/xuid extraits de DisplayClaims.xui[0].
func requestXSTSToken(ctx context.Context, doer haloapi.Doer, userToken, relyingParty string) (string, string, string, error) {
	body := map[string]any{
		xboxFieldRelyingParty: relyingParty,
		xboxFieldTokenType:    "JWT",
		xboxFieldProperties: map[string]any{
			"UserTokens": []string{userToken},
			"SandboxId":  "RETAIL",
		},
	}
	resp, err := postJSON(ctx, doer, xstsAuthorizeURL, map[string]string{
		"x-xbl-contract-version": "1",
	}, body)
	if err != nil {
		return "", "", "", err
	}
	token, ok := resp[xboxFieldToken].(string)
	if !ok || token == "" {
		return "", "", "", fmt.Errorf("token absent dans la réponse XSTS")
	}

	// Extraire gamertag + xuid depuis DisplayClaims.xui[0]
	gamertag, xuid := extractDisplayClaims(resp)
	return token, gamertag, xuid, nil
}

// extractDisplayClaims extrait gamertag (gtg) et xuid (xid) de la réponse XSTS.
func extractDisplayClaims(resp map[string]any) (string, string) {
	dc, ok := resp["DisplayClaims"].(map[string]any)
	if !ok {
		return "", ""
	}
	xuiRaw, ok := dc["xui"].([]any)
	if !ok || len(xuiRaw) == 0 {
		return "", ""
	}
	first, ok := xuiRaw[0].(map[string]any)
	if !ok {
		return "", ""
	}
	gamertag, _ := first["gtg"].(string)
	xuid, _ := first["xid"].(string)
	return gamertag, xuid
}

// requestSpartanTokenWith échange un XSTS Token contre un Spartan Token avec une audience et un
// endpoint paramétrés. MinVersion="4" et le proof TokenType "Xbox_XSTSv3" restent en dur : ce
// sont des constantes du PROTOCOLE spartan, pas des paramètres de titre.
//
// Retourne aussi l'expiry RÉEL du token (champ `ExpiresUtc.ISO8601Date` de la réponse) ; expiry
// zéro si le champ est absent/illisible (l'appelant appliquera un repli).
func requestSpartanTokenWith(ctx context.Context, doer haloapi.Doer, xstsToken, audience, tokenURL string) (string, time.Time, error) {
	body := map[string]any{
		"Audience":   audience,
		"MinVersion": "4",
		"Proof": []map[string]string{
			{xboxFieldToken: xstsToken, xboxFieldTokenType: "Xbox_XSTSv3"},
		},
	}
	resp, err := postJSON(ctx, doer, tokenURL, map[string]string{
		"Accept": "application/json",
	}, body)
	if err != nil {
		return "", time.Time{}, err
	}
	token, ok := resp["SpartanToken"].(string)
	if !ok || token == "" {
		return "", time.Time{}, fmt.Errorf("SpartanToken absent dans la réponse")
	}
	return token, parseSpartanExpiry(resp), nil
}

// parseSpartanExpiry extrait l'expiry du Spartan token depuis `ExpiresUtc.ISO8601Date` (format
// Waypoint). Retourne le zéro de time.Time si le champ est absent ou illisible — signal « expiry
// inconnu » plutôt qu'une valeur inventée.
func parseSpartanExpiry(resp map[string]any) time.Time {
	expiresUtc, ok := resp["ExpiresUtc"].(map[string]any)
	if !ok {
		return time.Time{}
	}
	iso, ok := expiresUtc["ISO8601Date"].(string)
	if !ok || iso == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

// requestClearanceTokenWith obtient le Clearance Token via un endpoint paramétré.
//
// Titre SANS clearance (clearanceEndpoint vide, ex. Halo 5 : ClearanceAware:false) : on saute
// proprement la jambe clearance et on retourne un token vide sans erreur.
func requestClearanceTokenWith(ctx context.Context, doer haloapi.Doer, spartanToken, clearanceEndpoint string) (string, error) {
	if clearanceEndpoint == "" {
		return "", nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, clearanceEndpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("x-343-authorization-spartan", spartanToken)
	req.Header.Set("Accept", "application/json")

	resp, err := doer.Do(req)
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

// xblHTTPError porte le statut d'une réponse non-2xx d'un endpoint Xbox/Halo (postJSON). Le type
// permet de brancher sur le statut (retry d=/t= de requestUserToken).
type xblHTTPError struct {
	Status int
	URL    string
	Body   string
}

func (e *xblHTTPError) Error() string {
	return fmt.Sprintf("HTTP %d depuis %s: %s", e.Status, e.URL, e.Body)
}

// postJSON effectue un POST JSON et retourne le body décodé.
func postJSON(ctx context.Context, doer haloapi.Doer, url string, headers map[string]string, body any) (map[string]any, error) {
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

	resp, err := doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lecture réponse %s: %w", url, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &xblHTTPError{Status: resp.StatusCode, URL: url, Body: string(raw)}
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("JSON decode %s: %w", url, err)
	}
	return data, nil
}
