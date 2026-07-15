// Package auth — refresh_user_xsts.go : refresh XSTS RTA pour un user multi-user.
//
// Flux (post-retrait MSAL 2026-07-15 : la voie cache MSAL/AcquireTokenSilent a
// été remplacée par le refresh_token brut, qui couvre RTs Azure ET MSA natifs
// via ExchangeRefreshTokenWithRotation) :
//  1. Charger UserTokens depuis MultiUserTokenStore
//  2. Si OAuthRefreshToken présent : refresh → nouvel access_token (+ RT rotaté persisté)
//  3. Sinon mode dégradé : utiliser AccessToken stocké tel quel si encore valide
//  4. AcquireXSTSForRTA(accessToken) → nouveau XSTS RTA
//  5. Persister UserTokens à jour (nouvel access_token, RT rotaté, XSTS)
//  6. Retourner le nouvel auth header
package auth

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// xstsRefreshTTL est utilisé comme expiresAt fallback si NotAfter absent.
const xstsRefreshTTL = 55 * time.Minute

// RefreshUserXSTS rafraîchit le XSTS RTA d'un user depuis son refresh_token persisté.
// Persiste les nouveaux tokens via store.Upsert. Retourne le nouvel auth header XBL3.0.
//
// Retourne ("", err) si :
//   - le user est introuvable dans le store
//   - le refresh_token est absent ET l'access_token stocké est expiré
//   - le refresh OAuth ou AcquireXSTSForRTA échouent
//
// Cette fonction est conçue pour être branchée comme callback OnAuthExpired d'un
// ReconnectManager (cf. PR 2.5c).
func RefreshUserXSTS(ctx context.Context, store *MultiUserTokenStore, xuid string) (string, error) {
	tokens, err := store.Load(xuid)
	if err != nil {
		return "", fmt.Errorf("refresh_user_xsts: load %s: %w", xuid, err)
	}

	// Étape 1 : obtenir un access_token Microsoft frais.
	accessToken := refreshAccessTokenForUser(ctx, tokens)
	if accessToken == "" {
		return "", fmt.Errorf("refresh_user_xsts: aucun access_token obtenu (cache vide ou expiré)")
	}

	// Étape 2 : acquérir un nouveau XSTS RTA.
	xstsResult, err := AcquireXSTSForRTA(ctx, accessToken)
	if err != nil {
		return "", fmt.Errorf("refresh_user_xsts: AcquireXSTSForRTA: %w", err)
	}

	// Étape 3 : persister les nouveaux tokens.
	expiresAt := xstsResult.NotAfter
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(xstsRefreshTTL)
	}
	tokens.XSTSToken = xstsResult.Token
	tokens.XSTSUserHash = xstsResult.UserHash
	tokens.XSTSExpiresAt = expiresAt
	tokens.AccessToken = accessToken
	tokens.OAuthExpiresAt = time.Now().Add(50 * time.Minute) // conservateur
	if err := store.Upsert(tokens); err != nil {
		// Échec persistance non bloquant — on retourne quand même le header en mémoire.
		slog.WarnContext(ctx, "refresh_user_xsts: persistance échouée (non bloquant)",
			"xuid", xuid, "err", err)
	}

	slog.InfoContext(ctx, "refresh_user_xsts: XSTS rafraîchi",
		"xuid", xuid, "gamertag", tokens.Gamertag, "expires_at", expiresAt)
	return tokens.AuthHeader(), nil
}

// refreshAccessTokenForUser obtient un access_token Microsoft frais pour ce user.
//  1. Si OAuthRefreshToken présent → refresh OAuth (Azure ou MSA natif, avec rotation :
//     le RT rotaté est écrit dans tokens pour que le caller le persiste).
//  2. Sinon, si tokens.AccessToken encore valide → retourne tel quel.
//  3. Sinon : "" (l'appelant retournera une erreur, user doit re-login Xbox SSO).
//
// Les erreurs intermédiaires (refresh) sont loguées en WARN et ignorées — la
// fonction tombe sur le fallback access_token stocké ou retourne "" pour
// signaler "pas de token, re-login requis".
func refreshAccessTokenForUser(ctx context.Context, tokens *UserTokens) string {
	if tokens.OAuthRefreshToken != "" {
		token, rotatedRT, err := ExchangeRefreshTokenWithRotation(ctx, tokens.OAuthRefreshToken)
		if err != nil {
			slog.WarnContext(ctx, "refresh_user_xsts: refresh OAuth erreur",
				"xuid", tokens.XUID, "err", err)
		}
		if token != "" {
			// RT à usage unique : écrire la rotation dans tokens AVANT le Upsert
			// du caller, sinon le prochain refresh relit un RT révoqué.
			if rotatedRT != "" && rotatedRT != tokens.OAuthRefreshToken {
				tokens.OAuthRefreshToken = rotatedRT
			}
			return token
		}
	}

	// Fallback : access_token stocké encore valide ?
	const safetyMargin = 5 * time.Minute
	if tokens.AccessToken != "" && time.Now().Add(safetyMargin).Before(tokens.OAuthExpiresAt) {
		slog.DebugContext(ctx, "refresh_user_xsts: réutilisation access_token stocké encore valide",
			"xuid", tokens.XUID)
		return tokens.AccessToken
	}

	return ""
}
