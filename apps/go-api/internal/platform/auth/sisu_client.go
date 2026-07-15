// Package auth — sisu_client.go : client SISU pour l'authentification Xbox native.
//
// Variante DEVICE-CODE du flux SISU (celle de LevelUp) : une seule étape côté
// SISU — POST https://sisu.xboxlive.com/authorize avec le ticket MSA du
// device-code flow ("t=<access_token>") + DeviceToken + ProofKey + RelyingParty.
// PAS de /authenticate ni de SessionId : ce leg appartient au flux par
// REDIRECTION (PKCE) des apps natives ; mélanger les deux (compléter une
// session PKCE avec un token device-code) est rejeté 401 corps vide par le
// serveur (constaté 2026-07-15, cross-référencé sur MinecraftAuth/RaphiMC —
// XblSisuAuthorizeRequest — et MCXboxBroadcast).
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
	"sort"
	"strings"
	"time"
)

const (
	sisuAuthorizeURL = "https://sisu.xboxlive.com/authorize"
	sisuSandbox      = "RETAIL"
	// sisuMSAScope est le scope MSA natif de la chaîne SISU : le device-flow
	// login.live.com doit le demander pour que l'access_token présenté à
	// /authorize ("t=<ticket>", famille MSA) soit accepté. Cross-référencé sur
	// XAL/OpenXbox et MinecraftAuth : les scopes Azure AD (Xboxlive.signin …)
	// produisent un JWT AAD que SISU rejette en 401. Sert aussi au refresh MSA
	// natif (postMSATokenExchange).
	sisuMSAScope = "service::user.auth.xboxlive.com::MBI_SSL"
	// sisuLogBodyMax borne la taille du corps d'erreur serveur relogué (diagnostic).
	sisuLogBodyMax = 512
)

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

// CompleteSISUFlow échange le ticket MSA du device-code flow contre un XSTS du
// titre via SISU /authorize. relyingParty = audience XSTS du titre (source :
// title.AuthDescriptor.XSTSAudience) — l'AuthorizationToken retourné est
// directement le XSTS de cette audience, pas besoin de l'étape User Token →
// XSTS séparée. Retourne un XSTSResult (même struct qu'AcquireXSTSForRTA).
func CompleteSISUFlow(
	ctx context.Context,
	client *http.Client,
	kp *PoPKeyPair,
	deviceToken, accessToken string,
	appID, relyingParty string,
) (*XSTSResult, error) {
	return completeSISUFlowWithURL(ctx, client, kp, deviceToken, accessToken, appID, relyingParty, sisuAuthorizeURL)
}

// completeSISUFlowWithURL est la version testable avec URL configurable.
//
// Corps aligné sur la référence MinecraftAuth (XblSisuAuthorizeRequest) : PAS de
// SessionId ni de SiteName — ces champs appartiennent au flux par redirection
// (PKCE) ; les envoyer avec un token device-code fait rejeter la requête en 401
// corps vide (constaté 2026-07-15).
func completeSISUFlowWithURL(
	ctx context.Context,
	client *http.Client,
	kp *PoPKeyPair,
	deviceToken, accessToken string,
	appID, relyingParty string,
	targetURL string,
) (*XSTSResult, error) {
	slog.DebugContext(ctx, "sisu: complétion flow",
		"relying_party", relyingParty,
		"access_token_format", accessTokenFormat(accessToken),
		"access_token_len", len(accessToken),
	)

	body := map[string]any{
		"AppId":             appID,
		"DeviceToken":       deviceToken,
		"ProofKey":          kp.GetProofKey(),
		"Sandbox":           sisuSandbox,
		"AccessToken":       "t=" + accessToken,
		"UseModernGamertag": true,
		"RelyingParty":      relyingParty,
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
		// Corps + header d'erreur du serveur Xbox (XErr/Message) : diagnostiques,
		// sans secret — aucun token n'y figure, seul le code d'erreur serveur.
		slog.ErrorContext(ctx, "sisu: échec complétion",
			"status", resp.StatusCode,
			"relying_party", relyingParty,
			"www_authenticate", resp.Header.Get("WWW-Authenticate"),
			"raw", truncateForLog(string(raw), sisuLogBodyMax),
			"access_token_format", accessTokenFormat(accessToken),
			"access_token_len", len(accessToken),
		)
		return nil, fmt.Errorf("sisu: HTTP %d complete: %s", resp.StatusCode, raw)
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("sisu: décodage JSON complete: %w", err)
	}

	// Diagnostic itératif (noms de clés uniquement, jamais les valeurs) : la
	// réponse /authorize porte plusieurs tokens (AuthorizationToken, UserToken,
	// TitleToken…) — savoir lesquels sont présents guide la suite de la chaîne.
	slog.InfoContext(ctx, "sisu: complétion HTTP OK",
		"relying_party", relyingParty, "response_keys", sortedKeys(data))

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

// accessTokenFormat classe un access_token Microsoft par famille pour le
// diagnostic (jamais le token lui-même) : un JWT Azure AD commence par "eyJ"
// (header JSON en base64), un compact ticket MSA natif par "Ew". SISU
// (/authorize, préfixe "t=") n'accepte que la famille MSA.
func accessTokenFormat(token string) string {
	switch {
	case strings.HasPrefix(token, "eyJ"):
		return "jwt_aad"
	case strings.HasPrefix(token, "Ew"):
		return "msa_compact"
	default:
		return "inconnu"
	}
}

// truncateForLog borne une chaîne destinée aux logs de diagnostic.
func truncateForLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…(tronqué)"
}

// sortedKeys retourne les clés d'une map triées — pour logger la FORME d'une
// réponse serveur sans jamais en logger les valeurs.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
