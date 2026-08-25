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
//  3. Sinon, lit le refresh_token OAuth v2 du MultiUserTokenStore (source unique
//     ADR 0023, data/auth/watcher_tokens/{xuid}.json) pour le gamertag du watcher.
//  4. Echange ce refresh_token via provider.TryOAuthRefreshWithRotation pour
//     obtenir un access_token frais.
//  5. Persiste la rotation dans le MultiUserTokenStore, puis l'access_token dans
//     watcher_tokens.json via store.UpdateOAuth (etat propre du watcher RTA).
//
// ADR 0023 Phase 5 (2026-08-25) : les sources de repli (refresh_token du store
// mono-user, variable d environnement) sont supprimees.
package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/observability/logging"
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

// EnsureWatcherAccessToken garantit qu'un access_token Microsoft frais est
// disponible pour le watcher RTA. Voir doc du fichier pour la sequence.
//
// Parametres :
//   - multiStore : MultiUserTokenStore (ADR 0023). SEULE source du refresh_token,
//     résolue via LoadByGamertag.
//   - store      : TokenStore mono-user (watcher_tokens.json) — état propre du
//     watcher : access_token courant + XSTS. Jamais une source de credentials.
//   - provider   : TokenProvider pour TryOAuthRefreshWithRotation (SISU).
//   - gamertag   : gamertag du watcher, pour LoadByGamertag dans le multi-store.
//
// Retours :
//   - access_token : "" si le store ne donne pas de RT utilisable.
//   - error : seulement pour erreurs structurelles (store watcher illisible).
//     Absence de RT ou refresh raté → ("", nil), pour permettre au caller de
//     retomber sur un mode dégradé (XSTS déjà stocké encore valide).
//
// Persistance rotation : Microsoft rotate le RT à chaque usage. Le nouveau RT est
// persisté dans le multi-store (canonique). Sans persistance, le prochain refresh
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

	// 2. Chercher le refresh_token dans le MultiUserTokenStore (source unique).
	refreshToken, xuid, source := lookupRefreshToken(ctx, multiStore, gamertag)
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

	// 4. Persister le RT rotaté dans le multi-store (source unique ADR 0023).
	if rotatedRT != "" && rotatedRT != refreshToken && multiStore != nil && xuid != "" {
		if werr := multiStore.UpdateOAuthRefreshToken(xuid, rotatedRT); werr != nil {
			slog.WarnContext(ctx, "watcher_refresh: persistance multi-store échouée",
				"xuid", xuid, "err", werr,
			)
		}
	}

	// 5. Persister l'access_token dans l'état du watcher (jamais le refresh_token :
	//    le store mono-user n'est plus une source de credentials — ADR 0023 Phase 5).
	if updErr := store.UpdateOAuth(newAccessToken, oauthAccessTokenTTL); updErr != nil {
		slog.WarnContext(ctx, "watcher_refresh: persistence access_token echouee", "err", updErr)
	} else {
		slog.InfoContext(ctx, "watcher_refresh: access_token rafraichi",
			"source", source, "gamertag", gamertag,
		)
	}

	return newAccessToken, nil
}

// lookupRefreshToken lit le refresh_token du MultiUserTokenStore (source unique
// ADR 0023) pour le gamertag du watcher.
//
// Retourne (rt, xuid, source) — le xuid permet de persister la rotation.
//
// Revue adversariale r1 : l'erreur de LoadByGamertag n'est plus jetée en
// silence. Un store ILLISIBLE (I/O, JSON corrompu) doit se voir : sans ce log,
// il s'affichait comme « aucun refresh_token, lancer token-capture » — et
// token-capture ne répare pas un fichier corrompu. Symétrique du traitement de
// access_token_store_first.go.
func lookupRefreshToken(ctx context.Context, multiStore *MultiUserTokenStore, gamertag string) (rt, xuid, source string) {
	if multiStore == nil || gamertag == "" {
		return "", "", ""
	}
	user, err := multiStore.LoadByGamertag(gamertag)
	if err != nil {
		if errors.Is(err, ErrUserTokensNotFound) {
			// Cas bénin et fréquent : ce gamertag n'a jamais été authentifié.
			slog.DebugContext(ctx, "watcher_refresh: aucune entrée store pour ce gamertag",
				"gamertag", gamertag)
			return "", "", ""
		}
		slog.ErrorContext(ctx, "watcher_refresh: lecture du store échouée — état anormal (I/O ou JSON corrompu), pas une absence de token",
			"gamertag", gamertag, "err", err)
		return "", "", ""
	}
	if user == nil || user.OAuthRefreshToken == "" {
		return "", "", ""
	}
	return user.OAuthRefreshToken, user.XUID, "multi_user_store"
}
