// Package halo — auth_retry.go : filet defense-in-depth sur échec auth live.
//
// Le mécanisme PRINCIPAL anti-401 est le cache expiry-aware (player_token_cache.go) :
// on ne sert jamais un Spartan qu'on ne peut pas garantir valide. Ce filet ne couvre
// QUE le résiduel qu'aucun suivi d'expiry ne peut prévoir — révocation côté serveur,
// changement de flight, dérive d'horloge : un token *valide-par-expiry* qui se fait
// quand même 401/403. Dans ce cas : invalider le cache → re-minter (singleflight) →
// réessayer EXACTEMENT une fois. Ce n'est pas le mécanisme, c'est le filet.
package halo

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/ctxkeys"
)

// errHaloAuthFailure marque un échec d'authentification (HTTP 401/403) renvoyé par
// doGet. Wrappé via %w pour être détectable avec errors.Is — sans ça, le filet ne peut
// pas distinguer un échec auth d'une autre erreur HTTP.
var errHaloAuthFailure = errors.New("halo: échec auth (401/403)")

// IsAuthFailure indique si une erreur provient d'un 401/403 Waypoint.
func IsAuthFailure(err error) bool { return errors.Is(err, errHaloAuthFailure) }

// retryOnAuth exécute fn (un fetch live token-gated lisant ses tokens depuis le ctx).
// Sur échec auth (401/403) ET si un xuid est présent dans le ctx, invalide le cache
// token du joueur, re-minte (singleflight) et réessaie fn UNE fois avec le token frais
// ré-injecté dans le ctx. Retry strictement 1× : un 2ᵉ 401 (révocation réelle) rend
// l'erreur d'origine, pas de boucle.
func retryOnAuth[T any](ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	res, err := fn(ctx)
	if err == nil || !errors.Is(err, errHaloAuthFailure) {
		return res, err
	}
	xuid := ctxkeys.HaloXUID(ctx)
	if xuid == "" {
		return res, err
	}
	InvalidateCachedPlayerTokens(xuid)
	fresh, rerr := ResolveFreshPlayerTokens(ctx, xuid)
	if rerr != nil || fresh == nil {
		return res, err // re-mint impossible → erreur d'origine
	}
	slog.DebugContext(ctx, "halo: filet 401 — token re-minté, retry unique", "xuid", xuid)
	return fn(ctxkeys.WithHaloAuth(ctx, fresh, xuid))
}
