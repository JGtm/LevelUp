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
	"levelup/go-api/internal/domain/title"
)

const (
	// URLs Xbox-platform (title-agnostic) — restent en const.
	xblUserAuthURL   = "https://user.auth.xboxlive.com/user/authenticate"
	xstsAuthorizeURL = "https://xsts.auth.xboxlive.com/xsts/authorize"
	// spartanTokenURL / clearanceURL / xstsHaloAudience (Halo-specific) ont migré
	// vers title.AuthDescriptor (MT-02) — source : title.DefaultHaloAuthDescriptor().
)

// xblUserAuthMaxConcurrent borne le nombre d'appels SIMULTANÉS vers
// user.auth.xboxlive.com (endpoint XBL user-token). Xbox Live throttle cet
// endpoint sur la CONCURRENCE (429 « currentRequests/maxRequests »). Sans borne,
// le refresher du pool (jusqu'à N refresh parallèles après un cooldown) ET le
// chemin user-facing (refreshTokensFromDB) tapaient l'endpoint en même temps →
// 429 auto-infligé, sans aucune coordination entre les deux chemins. Un sémaphore
// process-wide sérialise TOUS les appelants de l'échange XBL, éliminant ce 429.
const xblUserAuthMaxConcurrent = 2

// xblUserAuthSem est le sémaphore process-wide gardant les appels XBL user-token.
var xblUserAuthSem = make(chan struct{}, xblUserAuthMaxConcurrent)

// acquireXBLUserAuthSlot prend un slot du sémaphore XBL user-token (bloquant, mais
// annulable par ctx). Retourne la fonction de libération (à defer). Tous les
// appelants de l'échange XBL passent par ici → concurrence bornée globalement.
func acquireXBLUserAuthSlot(ctx context.Context) (func(), error) {
	select {
	case xblUserAuthSem <- struct{}{}:
		return func() { <-xblUserAuthSem }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ExchangeResult regroupe les tokens Halo ET l'identité extraite de la réponse XSTS.
type ExchangeResult struct {
	Tokens   *domain.HaloTokens
	Gamertag string // DisplayClaims.xui[0].gtg
	XUID     string // DisplayClaims.xui[0].xid
}

// ExchangeAccessToken échange un access_token Microsoft contre des tokens Halo Infinite.
// Implémente la chaîne : access_token → XBL user → XSTS Halo → Spartan → Clearance.
// Retourne aussi le gamertag et le XUID extraits de la réponse XSTS.
func ExchangeAccessToken(ctx context.Context, accessToken string) (*ExchangeResult, error) {
	return ExchangeAccessTokenWithDescriptor(ctx, accessToken, title.DefaultHaloAuthDescriptor())
}

// ExchangeAccessTokenWithDescriptor échange un access_token Microsoft contre des
// tokens d'un titre donné (MT-02). Le descripteur porte l'audience XSTS, l'audience
// + endpoint spartan, et l'endpoint clearance. ExchangeAccessToken délègue avec le
// défaut Halo → byte-identique. (Le *http.Client est construit en interne ; les
// tests de parité ciblent les fonctions de leg, qui prennent le client.)
func ExchangeAccessTokenWithDescriptor(ctx context.Context, accessToken string, d title.AuthDescriptor) (*ExchangeResult, error) {
	client := &http.Client{Timeout: 20 * time.Second}

	// Étape 1 : User Token XBL (title-agnostic, Xbox platform)
	userToken, err := requestUserToken(ctx, client, accessToken)
	if err != nil {
		return nil, fmt.Errorf("user token XBL: %w", err)
	}

	// Étape 2 : XSTS Token (audience du titre) + extraction gamertag/xuid
	xstsToken, gamertag, xuid, err := requestXSTSToken(ctx, client, userToken, d.XSTSAudience)
	if err != nil {
		return nil, fmt.Errorf("XSTS token: %w", err)
	}

	// Étape 3 : Spartan Token (audience + endpoint du titre, + expiry réel)
	spartanToken, spartanExpiry, err := requestSpartanTokenWith(ctx, client, xstsToken, d.SpartanAudience, d.SpartanTokenURL)
	if err != nil {
		return nil, fmt.Errorf("spartan token: %w", err)
	}

	// Étape 4 : Clearance Token (endpoint du titre)
	clearanceToken, err := requestClearanceTokenWith(ctx, client, spartanToken, d.ClearanceURL)
	if err != nil {
		return nil, fmt.Errorf("clearance token: %w", err)
	}

	return &ExchangeResult{
		Tokens: &domain.HaloTokens{
			SpartanToken:     spartanToken,
			ClearanceToken:   clearanceToken,
			SpartanExpiresAt: spartanExpiry,
		},
		Gamertag: gamertag,
		XUID:     xuid,
	}, nil
}

// ExchangeXSTSForHaloTokens échange un XSTS Token Halo déjà obtenu contre un Spartan Token
// et un Clearance Token. Utile pour les CLIs batch qui chargent le token depuis tokens.json
// sans refaire la chaîne OAuth complète.
func ExchangeXSTSForHaloTokens(ctx context.Context, xstsToken string) (*domain.HaloTokens, error) {
	return ExchangeXSTSForHaloTokensWithDescriptor(ctx, xstsToken, title.DefaultHaloAuthDescriptor())
}

// ExchangeXSTSForHaloTokensWithDescriptor échange un XSTS Token contre Spartan +
// Clearance pour un titre donné (MT-02). ExchangeXSTSForHaloTokens délègue Halo.
func ExchangeXSTSForHaloTokensWithDescriptor(ctx context.Context, xstsToken string, d title.AuthDescriptor) (*domain.HaloTokens, error) {
	client := &http.Client{Timeout: 20 * time.Second}

	spartanToken, spartanExpiry, err := requestSpartanTokenWith(ctx, client, xstsToken, d.SpartanAudience, d.SpartanTokenURL)
	if err != nil {
		return nil, fmt.Errorf("spartan token depuis XSTS: %w", err)
	}

	clearanceToken, err := requestClearanceTokenWith(ctx, client, spartanToken, d.ClearanceURL)
	if err != nil {
		return nil, fmt.Errorf("clearance token: %w", err)
	}

	return &domain.HaloTokens{
		SpartanToken:     spartanToken,
		ClearanceToken:   clearanceToken,
		SpartanExpiresAt: spartanExpiry,
	}, nil
}

// =============================================================================
// Étapes de la chaîne d'échange
// =============================================================================

// requestUserToken obtient un User Token XBL depuis un access_token Microsoft.
func requestUserToken(ctx context.Context, client *http.Client, accessToken string) (string, error) {
	// Concurrence bornée sur l'endpoint XBL user-token (anti-429 « currentRequests »).
	// Sérialise le refresher du pool ET le chemin user-facing sur le même sémaphore.
	release, err := acquireXBLUserAuthSlot(ctx)
	if err != nil {
		return "", err
	}
	defer release()

	body := map[string]any{
		xboxFieldRelyingParty: "http://auth.xboxlive.com",
		xboxFieldTokenType:    "JWT",
		xboxFieldProperties: map[string]string{
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
	token, ok := resp[xboxFieldToken].(string)
	if !ok || token == "" {
		return "", fmt.Errorf("token absent dans la réponse XBL")
	}
	return token, nil
}

// requestXSTSToken échange un User Token XBL contre un XSTS Token.
// Retourne (token, gamertag, xuid, error) — gamertag/xuid extraits de DisplayClaims.xui[0].
func requestXSTSToken(ctx context.Context, client *http.Client, userToken, relyingParty string) (string, string, string, error) {
	body := map[string]any{
		xboxFieldRelyingParty: relyingParty,
		xboxFieldTokenType:    "JWT",
		xboxFieldProperties: map[string]any{
			"UserTokens": []string{userToken},
			"SandboxId":  "RETAIL",
		},
	}
	resp, err := postJSON(ctx, client, xstsAuthorizeURL, map[string]string{
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

// requestSpartanToken échange un XSTS Token Halo contre un Spartan Token (défaut Halo).
// Retourne aussi l'expiry RÉEL du token (cf. requestSpartanTokenWith).
func requestSpartanToken(ctx context.Context, client *http.Client, xstsToken string) (string, time.Time, error) {
	d := title.DefaultHaloAuthDescriptor()
	return requestSpartanTokenWith(ctx, client, xstsToken, d.SpartanAudience, d.SpartanTokenURL)
}

// requestSpartanTokenWith échange un XSTS Token contre un Spartan Token avec une
// audience + un endpoint paramétrés (MT-02). MinVersion="4" et le proof TokenType
// "Xbox_XSTSv3" restent en dur : ce sont des constantes du PROTOCOLE spartan, pas
// des paramètres de titre.
//
// Retourne aussi l'expiry RÉEL du token (champ `ExpiresUtc.ISO8601Date` de la réponse) ;
// expiry zéro si le champ est absent/illisible (le caller appliquera un fallback).
func requestSpartanTokenWith(ctx context.Context, client *http.Client, xstsToken, audience, tokenURL string) (string, time.Time, error) {
	body := map[string]any{
		"Audience":   audience,
		"MinVersion": "4",
		"Proof": []map[string]string{
			{xboxFieldToken: xstsToken, xboxFieldTokenType: "Xbox_XSTSv3"},
		},
	}
	resp, err := postJSON(ctx, client, tokenURL, map[string]string{
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

// parseSpartanExpiry extrait l'expiry du Spartan token depuis `ExpiresUtc.ISO8601Date`
// (format Waypoint). Retourne le zéro de time.Time si le champ est absent ou illisible —
// signal "expiry inconnu" plutôt qu'une valeur inventée.
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

// requestClearanceToken obtient le Clearance Token (FlightConfigurationId) (défaut Halo).
func requestClearanceToken(ctx context.Context, client *http.Client, spartanToken string) (string, error) {
	return requestClearanceTokenWith(ctx, client, spartanToken, title.DefaultHaloAuthDescriptor().ClearanceURL)
}

// requestClearanceTokenWith obtient le Clearance Token via un endpoint paramétré (MT-02).
//
// Titre SANS clearance (clearanceEndpoint vide, ex. Halo 5 : ClearanceAware:false
// confirmé sonde) : on saute proprement la jambe clearance et on retourne un token
// vide sans erreur — sinon http.NewRequestWithContext(GET, "") échouerait. Pour
// Halo Infinite (URL non vide), comportement inchangé.
func requestClearanceTokenWith(ctx context.Context, client *http.Client, spartanToken, clearanceEndpoint string) (string, error) {
	if clearanceEndpoint == "" {
		return "", nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, clearanceEndpoint, nil)
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
