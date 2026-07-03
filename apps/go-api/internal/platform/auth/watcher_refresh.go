// Package auth — watcher_refresh.go : refresh OAuth v2 pour le watcher RTA.
//
// Le watcher RTA s'authentifie via un XSTS specifique (audience rta.xboxlive.com)
// acquis par AcquireXSTSForRTA(accessToken). Or l'access_token Microsoft expire
// au bout d'1 h, et le XSTS RTA ~16 h. Sans refresh automatique, le watcher
// cesse de fonctionner des qu'un de ces deux tokens expire — et il faut
// regenerer manuellement le watcher_tokens.json.
//
// Ce module fournit EnsureWatcherAccessToken qui :
//  1. Lit l'access_token courant depuis le TokenStore (watcher_tokens.json).
//  2. S'il est encore valide (avec marge de securite), le retourne tel quel.
//  3. Sinon, cherche un refresh_token OAuth v2 dans l'ordre :
//     a. tokens.RefreshToken (dans watcher_tokens.json)
//     b. variable d'environnement SPNKR_OAUTH_REFRESH_TOKEN_<XSTS_GAMERTAG>
//     (meme convention que internal/api/registry.go et auto_sync.go).
//  4. Echange ce refresh_token via provider.TryOAuthRefresh pour obtenir un
//     access_token frais.
//  5. Persiste l'access_token (et le refresh_token s'il provenait de l'env)
//     dans watcher_tokens.json via store.UpdateOAuth.
//
// Convention de cle env : SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG_UPPER>, ou les
// caracteres ' ', '-' et '.' du gamertag sont remplaces par '_'. Identique a la
// convention deja en place dans scheduler/auto_sync.go et api/registry.go.
package auth

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/observability/logging"
	"strings"
	"time"
)

// accessTokenSafetyMargin est la marge de securite avant l'expiration de
// l'access_token. Si l'access_token expire dans moins que cette marge, on
// tente un refresh proactif. 5 minutes est un bon compromis entre eviter
// les refresh inutiles et garantir qu'on n'envoie pas un token sur le point
// d'expirer a Xbox Live.
const accessTokenSafetyMargin = 5 * time.Minute

// oauthAccessTokenTTL est la duree presumee de validite d'un access_token
// frechement emis par Microsoft. Microsoft retourne expires_in dans la reponse
// /token (typiquement 3600s = 1h), mais ExchangeRefreshToken ne propage pas
// cette valeur. On utilise un TTL conservateur de 50 min pour la marge.
const oauthAccessTokenTTL = 50 * time.Minute

// RefreshTokenFromEnv retourne la valeur de SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG>
// pour un gamertag donne. Retourne "" si la variable n'est pas definie.
//
// DEPRECATED (ADR 0023) : l'env var est un mécanisme legacy. La source canonique
// des refresh tokens est MultiUserTokenStore (data/auth/watcher_tokens/{xuid}.json).
// La migration boot-time (`auth.MigrateLegacyTokens`) copie automatiquement les
// SPNKR_OAUTH_REFRESH_TOKEN_* dans le store au démarrage. Cette fonction sera
// supprimée Phase 5 ; en attendant, son comportement est partagé avec
// `auth.EnvRefreshTokenForGamertag` (migration.go) et
// `pool.readOAuthRefreshTokenFromEnv` (discovery.go).
func RefreshTokenFromEnv(gamertag string) string {
	if gamertag == "" {
		return ""
	}
	key := strings.ToUpper(strings.TrimSpace(gamertag))
	key = strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' || r == '.' {
			return '_'
		}
		return r
	}, key)
	return os.Getenv("SPNKR_OAUTH_REFRESH_TOKEN_" + key)
}

// EnsureWatcherAccessToken garantit qu'un access_token Microsoft frais est
// disponible pour le watcher RTA. Voir doc du fichier pour la sequence.
//
// Parametres :
//   - multiStore : MultiUserTokenStore (ADR 0023, peut être nil). Source canonique
//     du refresh_token — consulté en premier via LoadByGamertag.
//   - store      : TokenStore legacy mono-user (watcher_tokens.json). Fallback.
//   - provider   : TokenProvider pour TryOAuthRefresh (MSAL ou SISU).
//   - gamertag   : gamertag pour résoudre la clé SPNKR_OAUTH_REFRESH_TOKEN_* et
//     pour LoadByGamertag dans le multi-store.
//
// Retours :
//   - access_token : "" si aucune source ne donne un RT utilisable.
//   - error : seulement pour erreurs structurelles (store legacy illisible).
//     Absence de RT ou refresh raté → ("", nil), pour permettre au caller de
//     retomber sur un mode dégradé (XSTS déjà stocké encore valide).
//
// Persistance rotation : Microsoft rotate le RT à chaque usage. Si le RT est
// rotaté, le nouveau RT est persisté en priorité dans le multi-store (canonique),
// puis dans le store legacy (compat). Sans persistance, le prochain refresh
// échoue avec invalid_grant.
func EnsureWatcherAccessToken(
	ctx context.Context,
	multiStore *MultiUserTokenStore,
	store *TokenStore,
	provider TokenProvider,
	gamertag string,
) (string, error) {
	if store == nil {
		return "", fmt.Errorf("EnsureWatcherAccessToken: store nil")
	}
	if provider == nil {
		return "", fmt.Errorf("EnsureWatcherAccessToken: provider nil")
	}

	ctx, evID := logging.WithEvent(ctx, "auth.watcher_refresh:"+gamertag)
	slog.DebugContext(ctx, "watcher_refresh: appel ensure_access_token",
		"gamertag", gamertag, "event", evID)

	tokens, err := store.Load()
	if err != nil {
		return "", fmt.Errorf("EnsureWatcherAccessToken: load tokens: %w", err)
	}

	// 1. Si access_token courant est encore valide, le reutiliser — pas besoin
	//    de bruler un refresh_token.
	if tokens.IsOAuthValid(accessTokenSafetyMargin) {
		slog.DebugContext(ctx, "watcher_refresh: access_token courant encore valide",
			"expires_at", tokens.OAuthExpiresAt,
		)
		return tokens.AccessToken, nil
	}

	// 2. Chercher un refresh_token. Priorité : multi-store (canonique) →
	//    legacy store → env var (DEPRECATED).
	refreshToken, xuid, source := lookupRefreshToken(ctx, multiStore, tokens, gamertag)
	if refreshToken == "" {
		slog.InfoContext(ctx, "watcher_refresh: aucun refresh_token disponible",
			"gamertag", gamertag,
			"hint", "lancer `go run ./cmd/token-capture/ <gamertag>` pour seeder le store",
		)
		return "", nil
	}

	// 3. Echange refresh_token → access_token (avec rotation).
	newAccessToken, rotatedRT, err := provider.TryOAuthRefreshWithRotation(ctx, refreshToken)
	if err != nil {
		slog.WarnContext(ctx, "watcher_refresh: TryOAuthRefreshWithRotation erreur",
			"source", source, "gamertag", gamertag, "err", err,
		)
		return "", nil //nolint:nilerr // erreur de refresh = non fatale, caller decide
	}
	if newAccessToken == "" {
		slog.InfoContext(ctx, "watcher_refresh: refresh retourne vide (refresh_token revoke ?)",
			"source", source, "gamertag", gamertag,
		)
		return "", nil
	}

	// 4. Persister le RT rotaté dans le multi-store (canonique) si on l'a.
	rtToStore := refreshToken
	if rotatedRT != "" && rotatedRT != refreshToken {
		rtToStore = rotatedRT
		if multiStore != nil && xuid != "" {
			if werr := multiStore.UpdateOAuthRefreshToken(xuid, rotatedRT); werr != nil {
				slog.WarnContext(ctx, "watcher_refresh: persistance multi-store échouée",
					"xuid", xuid, "err", werr,
				)
			}
		}
	}

	// 5. Persister access_token (et RT pour compat) dans le legacy store.
	if updErr := store.UpdateOAuth(newAccessToken, rtToStore, oauthAccessTokenTTL); updErr != nil {
		slog.WarnContext(ctx, "watcher_refresh: persistence access_token echouee", "err", updErr)
	} else {
		slog.InfoContext(ctx, "watcher_refresh: access_token rafraichi",
			"source", source, "gamertag", gamertag,
		)
	}

	return newAccessToken, nil
}

// lookupRefreshToken applique la priorité ADR 0023 pour trouver un refresh_token :
//  1. MultiUserTokenStore (canonique) — par gamertag
//  2. TokenStore legacy (watcher_tokens.json)
//  3. env var SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG> (DEPRECATED)
//
// Retourne (rt, xuid, source) — xuid est non vide uniquement si le RT vient du
// multi-store, et permet de persister la rotation au bon endroit.
func lookupRefreshToken(
	ctx context.Context,
	multiStore *MultiUserTokenStore,
	tokens *StoredTokens,
	gamertag string,
) (rt, xuid, source string) {
	if multiStore != nil && gamertag != "" {
		if user, err := multiStore.LoadByGamertag(gamertag); err == nil && user != nil && user.OAuthRefreshToken != "" {
			return user.OAuthRefreshToken, user.XUID, "multi_user_store"
		}
	}
	if tokens != nil && tokens.RefreshToken != "" {
		return tokens.RefreshToken, "", "watcher_tokens.json"
	}
	if envToken := RefreshTokenFromEnv(gamertag); envToken != "" {
		slog.WarnContext(ctx, "watcher_refresh: legacy env var utilisée — à migrer",
			"gamertag", gamertag, "deprecated_since", "ADR-0023")
		observability.RecordLegacySourceUsed(observability.LegacySourceEnvOAuth)
		return envToken, "", "env_var"
	}
	return "", "", ""
}
