// exchange_access.go : échange access_token Microsoft -> jetons Halo, et la jambe User Token XBL.
//
// Chaîne d'échange (portage de SPNKr) :
//
//	access_token (Microsoft)
//	-> User Token (XBL)  [user.auth.xboxlive.com]
//	-> XSTS du titre     [xsts.auth.xboxlive.com, audience du Descriptor]
//	-> Spartan Token     [Descriptor.SpartanTokenURL]
//	-> Clearance Token   [Descriptor.ClearanceURL]
//
// Les jambes XSTS, Spartan et Clearance et le transport JSON vivent dans exchange_xsts.go.

package xblauth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"levelup/go-api/internal/haloapi"
)

const (
	// URLs de la plateforme Xbox (communes à tous les titres). Les adresses propres au titre
	// (Spartan, Clearance, audience XSTS) viennent du Descriptor.
	xblUserAuthURL   = "https://user.auth.xboxlive.com/user/authenticate"
	xstsAuthorizeURL = "https://xsts.auth.xboxlive.com/xsts/authorize"
)

// ExchangeResult regroupe les jetons Halo ET l'identité extraite de la réponse XSTS.
type ExchangeResult struct {
	Tokens   *Tokens
	Gamertag string // DisplayClaims.xui[0].gtg
	XUID     string // DisplayClaims.xui[0].xid
}

// ExchangeAccessToken échange un access_token Microsoft contre les jetons du titre décrit par
// `d` : XBL user -> XSTS (audience du titre) -> Spartan -> Clearance. Rend aussi le gamertag et
// le xuid de la réponse XSTS. Le préfixe RpsTicket suit `ch.Famille` ; la provenance acceptée est
// rapportée à `ch.Observe`.
func ExchangeAccessToken(ctx context.Context, ch Chaine, accessToken string, d Descriptor) (*ExchangeResult, error) {
	// Étape 1 : User Token XBL (commun à tous les titres, plateforme Xbox)
	userToken, err := requestUserToken(ctx, ch, accessToken)
	if err != nil {
		return nil, fmt.Errorf("user token XBL: %w", err)
	}

	// Étape 2 : XSTS Token (audience du titre) + extraction gamertag/xuid
	xstsToken, gamertag, xuid, err := requestXSTSToken(ctx, ch.Doer, userToken, d.XSTSAudience)
	if err != nil {
		return nil, fmt.Errorf("XSTS token: %w", err)
	}

	// Étape 3 : Spartan Token (audience + endpoint du titre, + expiry réel)
	spartanToken, spartanExpiry, err := requestSpartanTokenWith(ctx, ch.Doer, xstsToken, d.SpartanAudience, d.SpartanTokenURL)
	if err != nil {
		return nil, fmt.Errorf("spartan token: %w", err)
	}

	// Étape 4 : Clearance Token (endpoint du titre)
	clearanceToken, err := requestClearanceTokenWith(ctx, ch.Doer, spartanToken, d.ClearanceURL)
	if err != nil {
		return nil, fmt.Errorf("clearance token: %w", err)
	}

	return &ExchangeResult{
		Tokens: &Tokens{
			SpartanToken:     spartanToken,
			ClearanceToken:   clearanceToken,
			SpartanExpiresAt: spartanExpiry,
		},
		Gamertag: gamertag,
		XUID:     xuid,
	}, nil
}

// ExchangeXSTSForTokens échange un XSTS du titre déjà obtenu contre Spartan + Clearance, sans
// refaire la chaîne OAuth.
func ExchangeXSTSForTokens(ctx context.Context, doer haloapi.Doer, xstsToken string, d Descriptor) (*Tokens, error) {
	spartanToken, spartanExpiry, err := requestSpartanTokenWith(ctx, doer, xstsToken, d.SpartanAudience, d.SpartanTokenURL)
	if err != nil {
		return nil, fmt.Errorf("spartan token depuis XSTS: %w", err)
	}

	clearanceToken, err := requestClearanceTokenWith(ctx, doer, spartanToken, d.ClearanceURL)
	if err != nil {
		return nil, fmt.Errorf("clearance token: %w", err)
	}

	return &Tokens{
		SpartanToken:     spartanToken,
		ClearanceToken:   clearanceToken,
		SpartanExpiresAt: spartanExpiry,
	}, nil
}

// Préfixes RpsTicket de l'échange XBL user-token, et leur correspondance avec les provenances.
// Source UNIQUE de cette correspondance (le choix du préfixe et la lecture de la mesure la
// partagent).
const (
	rpsPrefixAzure      = "d="
	rpsPrefixXboxNative = "t="
)

// rpsPrefixesForFamily rend (préfixe à essayer d'abord, préfixe de repli) pour une provenance
// connue. Provenance inconnue -> ordre historique « d= » puis « t= ».
func rpsPrefixesForFamily(family string) (primary, fallback string) {
	if family == FamilleXboxNative {
		return rpsPrefixXboxNative, rpsPrefixAzure
	}
	return rpsPrefixAzure, rpsPrefixXboxNative
}

// familyForRpsPrefix rend la provenance correspondant à un préfixe accepté.
func familyForRpsPrefix(prefix string) string {
	if prefix == rpsPrefixXboxNative {
		return FamilleXboxNative
	}
	return FamilleAzure
}

// observe rapporte la provenance mesurée à l'appelant (s'il écoute).
func (c Chaine) observe(family string) {
	if c.Observe != nil && family != "" {
		c.Observe(family)
	}
}

// prendreCreneau prend un créneau de concurrence XBL si l'appelant en fournit un.
func (c Chaine) prendreCreneau(ctx context.Context) (func(), error) {
	if c.Creneau == nil {
		return func() {}, nil
	}
	return c.Creneau(ctx)
}

// requestUserToken obtient un User Token XBL depuis un access_token Microsoft.
//
// Le préfixe RpsTicket dépend du CLIENT OAuth qui a émis l'access_token — « d= » pour l'app Azure
// (SSO web, refresh v2), « t= » pour le client Xbox natif (device-flow, refresh MSA) — PAS de son
// format : les deux familles émettent des compact tickets « EwA… » indistinguables. Quand la
// provenance est connue (`ch.Famille`), on utilise directement le bon préfixe ; sinon « d= »
// d'abord (ordre historique). Le retry sur l'AUTRE préfixe reste un FILET (401), logué
// WarnContext pour rendre visible un 401 qui n'est PAS un simple mauvais préfixe.
func requestUserToken(ctx context.Context, ch Chaine, accessToken string) (string, error) {
	primary, fallback := rpsPrefixesForFamily(ch.Famille)
	token, err := requestUserTokenPrefixed(ctx, ch, accessToken, primary)
	if err == nil {
		// Provenance MESURÉE : l'appelant la persiste pour commencer par là au prochain échange.
		ch.observe(familyForRpsPrefix(primary))
		return token, nil
	}
	var herr *xblHTTPError
	if errors.As(err, &herr) && herr.Status == http.StatusUnauthorized {
		slog.WarnContext(ctx, "halo_exchange: RpsTicket refusé (401) — retry sur l'autre préfixe (filet provenance)",
			"primary", primary, "fallback", fallback, "family", ch.Famille)
		token, err = requestUserTokenPrefixed(ctx, ch, accessToken, fallback)
		if err == nil {
			ch.observe(familyForRpsPrefix(fallback))
		}
		return token, err
	}
	return token, err
}

// requestUserTokenPrefixed exécute l'échange XBL user-token avec un préfixe RpsTicket explicite
// (« d= » ou « t= »), sous le créneau de concurrence de l'appelant.
func requestUserTokenPrefixed(ctx context.Context, ch Chaine, accessToken, prefix string) (string, error) {
	release, err := ch.prendreCreneau(ctx)
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
			"RpsTicket":  prefix + accessToken,
		},
	}
	resp, err := postJSON(ctx, ch.Doer, xblUserAuthURL, map[string]string{
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
