// Package auth — xsts.go : obtention de tokens XSTS pour Xbox Live RTA.
//
// Réutilise les fonctions internes de halo_exchange.go (XBL user token + XSTS)
// mais avec un RelyingParty différent : http://xboxlive.com (au lieu de l'audience Halo).
//
// Flow :
//
//	access_token (Microsoft)
//	  → User Token (XBL)                   [user.auth.xboxlive.com]
//	  → XSTS Token Xbox Live               [xsts.auth.xboxlive.com, RP=http://xboxlive.com]
//	  → Authorization header pour RTA WS :  XBL3.0 x=<userhash>;<xsts_token>
package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

const (
	// xboxLiveRelyingParty est le RP pour les tokens XSTS destinés à Xbox Live (RTA, People, etc.).
	xboxLiveRelyingParty = "http://xboxlive.com"
)

// XSTSResult contient le token XSTS + userhash nécessaires pour la connexion RTA.
type XSTSResult struct {
	Token    string // XSTS token
	UserHash string // userhash extraite de DisplayClaims
	Gamertag string // gamertag extrait de DisplayClaims
	XUID     string // xuid extrait de DisplayClaims
}

// AuthHeader retourne le header Authorization pour Xbox Live RTA.
func (r *XSTSResult) AuthHeader() string {
	return fmt.Sprintf("XBL3.0 x=%s;%s", r.UserHash, r.Token)
}

// AcquireXSTSForRTA obtient un token XSTS avec RelyingParty=http://xboxlive.com.
// Ce token est nécessaire pour la connexion WebSocket RTA.
func AcquireXSTSForRTA(ctx context.Context, accessToken string) (*XSTSResult, error) {
	slog.DebugContext(ctx, "xsts: début acquisition XSTS pour RTA")
	client := &http.Client{Timeout: 20 * time.Second}

	// Étape 1 : User Token XBL (identique au flow Halo)
	userToken, err := requestUserToken(ctx, client, accessToken)
	if err != nil {
		slog.ErrorContext(ctx, "xsts: échec User Token XBL", "err", err)
		return nil, fmt.Errorf("xsts rta: user token: %w", err)
	}
	slog.DebugContext(ctx, "xsts: User Token XBL obtenu")

	// Étape 2 : XSTS avec RP Xbox Live (pas Halo)
	xstsResp, err := requestXSTSTokenFull(ctx, client, userToken, xboxLiveRelyingParty)
	if err != nil {
		slog.ErrorContext(ctx, "xsts: échec XSTS Xbox Live", "err", err)
		return nil, fmt.Errorf("xsts rta: xsts token: %w", err)
	}
	slog.InfoContext(ctx, "xsts: XSTS Xbox Live obtenu",
		"gamertag", xstsResp.Gamertag,
		"xuid", xstsResp.XUID,
	)
	return xstsResp, nil
}

// requestXSTSTokenFull échange un User Token XBL contre un XSTS Token.
// Retourne le token complet avec userhash (nécessaire pour l'auth RTA).
func requestXSTSTokenFull(ctx context.Context, client *http.Client, userToken, relyingParty string) (*XSTSResult, error) {
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
		return nil, err
	}
	token, ok := resp["Token"].(string)
	if !ok || token == "" {
		return nil, fmt.Errorf("Token absent dans la réponse XSTS")
	}

	gamertag, xuid := extractDisplayClaims(resp)
	userHash := extractUserHash(resp)
	return &XSTSResult{
		Token:    token,
		UserHash: userHash,
		Gamertag: gamertag,
		XUID:     xuid,
	}, nil
}

// extractUserHash extrait le userhash de DisplayClaims.xui[0].uhs.
func extractUserHash(resp map[string]any) string {
	dc, ok := resp["DisplayClaims"].(map[string]any)
	if !ok {
		return ""
	}
	xuiRaw, ok := dc["xui"].([]any)
	if !ok || len(xuiRaw) == 0 {
		return ""
	}
	first, ok := xuiRaw[0].(map[string]any)
	if !ok {
		return ""
	}
	uhs, _ := first["uhs"].(string)
	return uhs
}
