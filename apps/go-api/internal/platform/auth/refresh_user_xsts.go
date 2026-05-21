// Package auth — refresh_user_xsts.go : refresh XSTS RTA pour un user multi-user.
//
// Différent de watcher_refresh.go (mono-user, basé sur tokens.RefreshToken brut) :
// ce helper utilise le cache MSAL JSON sérialisé pour rafraîchir l'access_token
// Microsoft via AcquireTokenSilent, puis acquérir un nouveau XSTS RTA.
//
// Flux :
//  1. Charger UserTokens depuis MultiUserTokenStore
//  2. Si MSALCacheJSON présent : reconstruire InMemoryCacheAccessor, AcquireTokenSilent
//     → nouvel access_token (MSAL peut tourner le RT en interne dans le cache)
//  3. Sinon mode dégradé : utiliser AccessToken stocké tel quel si encore valide
//  4. AcquireXSTSForRTA(accessToken) → nouveau XSTS RTA
//  5. Persister UserTokens à jour (nouvel access_token, MSAL cache, XSTS)
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

// RefreshUserXSTS rafraîchit le XSTS RTA d'un user en utilisant son cache MSAL persisté.
// Persiste les nouveaux tokens via store.Upsert. Retourne le nouvel auth header XBL3.0.
//
// Retourne ("", err) si :
//   - le user est introuvable dans le store
//   - le MSAL cache est absent ET l'access_token stocké est expiré
//   - AcquireTokenSilent ou AcquireXSTSForRTA échouent
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
// 1. Si MSALCacheJSON présent → AcquireTokenSilent (utilise le RT depuis le cache).
// 2. Sinon, si tokens.AccessToken encore valide → retourne tel quel.
// 3. Sinon : "" (l'appelant retournera une erreur, user doit re-login Xbox SSO).
//
// Les erreurs intermédiaires (AcquireTokenSilent) sont loguées en WARN et ignorées
// — la fonction tombe sur le fallback access_token stocké ou retourne "" pour
// signaler "pas de token, re-login requis".
func refreshAccessTokenForUser(ctx context.Context, tokens *UserTokens) string {
	if tokens.MSALCacheJSON != "" {
		accessor := NewInMemoryCacheAccessorFromJSON(tokens.MSALCacheJSON)
		token, err := AcquireTokenSilent(ctx, accessor)
		if err != nil {
			slog.WarnContext(ctx, "refresh_user_xsts: AcquireTokenSilent erreur",
				"xuid", tokens.XUID, "err", err)
		}
		if token != "" {
			// Note : on ne re-sérialise PAS le cache ici — le MSAL SDK le mutate en
			// interne via l'accessor.Export(), donc tokens.MSALCacheJSON sera mis à
			// jour si on appelle accessor.Serialize() avant de persister.
			if updatedCache, serr := accessor.Serialize(); serr == nil && updatedCache != "" {
				tokens.MSALCacheJSON = updatedCache
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
