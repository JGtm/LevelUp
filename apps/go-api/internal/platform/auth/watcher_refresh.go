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
//        (meme convention que internal/api/registry.go et auto_sync.go).
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
// La convention de transformation du gamertag est partagee avec :
//   - internal/api/registry.go::oauthRefreshTokenForPlayer
//   - internal/scheduler/auto_sync.go::defaultTokenReader (env var fallback)
//   - internal/platform/auth/pool/discovery.go::readOAuthRefreshTokenFromEnv
//
// Cette fonction exportee centralise la logique pour le watcher (et plus tard
// pourra remplacer les copies dispersees).
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
//   - store    : TokenStore pointant sur watcher_tokens.json (lecture + Save).
//   - provider : TokenProvider pour TryOAuthRefresh (MSAL ou SISU).
//   - gamertag : gamertag a utiliser comme cle SPNKR_OAUTH_REFRESH_TOKEN_*.
//     Typiquement tokens.XSTSGamertag, ou un override explicite (config).
//
// Retours :
//   - access_token : la chaine a utiliser pour AcquireXSTSForRTA. "" signifie
//     "aucun moyen d'obtenir un token" (aucun refresh_token trouve ou refresh
//     echoue).
//   - error : seulement pour les erreurs structurelles (impossible de lire le
//     store). Une absence de refresh_token ou un refresh echoue retourne
//     ("", nil) — pas une erreur — pour permettre au caller de retomber sur
//     un mode degrade (XSTS deja stocke encore valide, par exemple).
func EnsureWatcherAccessToken(
	ctx context.Context,
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

	tokens, err := store.Load()
	if err != nil {
		return "", fmt.Errorf("EnsureWatcherAccessToken: load tokens: %w", err)
	}

	// 1. Si access_token courant est encore valide (marge confortable), le
	//    reutiliser tel quel — pas besoin de bruler un refresh_token.
	if tokens.IsOAuthValid(accessTokenSafetyMargin) {
		slog.DebugContext(ctx, "watcher_refresh: access_token courant encore valide",
			"expires_at", tokens.OAuthExpiresAt,
		)
		return tokens.AccessToken, nil
	}

	// 2. Chercher un refresh_token utilisable.
	refreshToken := tokens.RefreshToken
	source := "watcher_tokens.json"
	if refreshToken == "" {
		// Fallback : env var SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG>.
		envToken := RefreshTokenFromEnv(gamertag)
		if envToken != "" {
			refreshToken = envToken
			source = "env_var"
		}
	}

	if refreshToken == "" {
		slog.InfoContext(ctx, "watcher_refresh: aucun refresh_token disponible",
			"gamertag", gamertag,
			"hint", "definir SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG> dans .env.local",
		)
		return "", nil
	}

	// 3. Echange refresh_token → access_token.
	newAccessToken, err := provider.TryOAuthRefresh(ctx, refreshToken)
	if err != nil {
		slog.WarnContext(ctx, "watcher_refresh: TryOAuthRefresh erreur",
			"source", source,
			"gamertag", gamertag,
			"err", err,
		)
		return "", nil //nolint:nilerr // erreur de refresh = non fatale, caller decide
	}
	if newAccessToken == "" {
		slog.InfoContext(ctx, "watcher_refresh: TryOAuthRefresh retourne vide (refresh_token revoke ?)",
			"source", source,
			"gamertag", gamertag,
		)
		return "", nil
	}

	// 4. Persister le nouveau access_token (et le refresh_token, surtout s'il
	//    vient de l'env var — on le met dans le fichier pour les prochains
	//    redemarrages).
	if updErr := store.UpdateOAuth(newAccessToken, refreshToken, oauthAccessTokenTTL); updErr != nil {
		// Echec persistance non fatal : on a quand meme un token en memoire.
		slog.WarnContext(ctx, "watcher_refresh: persistence access_token echouee",
			"err", updErr,
		)
	} else {
		slog.InfoContext(ctx, "watcher_refresh: access_token rafraichi",
			"source", source,
			"gamertag", gamertag,
		)
	}

	return newAccessToken, nil
}
