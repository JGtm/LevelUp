// Package tokencache — auth_retry.go : filet defense-in-depth sur échec auth live.
//
// Le mécanisme PRINCIPAL anti-401 est le cache expiry-aware (player_token_cache.go) :
// on ne sert jamais un Spartan qu'on ne peut pas garantir valide. Ce filet ne couvre
// QUE le résiduel qu'aucun suivi d'expiry ne peut prévoir — révocation côté serveur,
// changement de flight, dérive d'horloge : un token *valide-par-expiry* qui se fait
// quand même 401/403. Dans ce cas : invalider le cache → re-minter (singleflight) →
// réessayer EXACTEMENT une fois. Ce n'est pas le mécanisme, c'est le filet.
package tokencache

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/ctxkeys"
)

// RetryWithFreshTokens exécute fn (un fetch live token-gated lisant ses tokens depuis le
// ctx). Sur échec auth — détecté par le predicate isAuthErr — ET si un xuid est présent
// dans le ctx, invalide le cache token du joueur, re-minte (singleflight) et réessaie fn
// UNE fois avec le token frais ré-injecté dans le ctx. Retry strictement 1× : un 2ᵉ échec
// auth (révocation réelle) rend l'erreur d'origine, pas de boucle.
//
// Chaque appelant passe son prédicat d'échec d'authentification (le fournisseur et le client
// de synchronisation : le verdict typé 401/403 de la brique, `haloadapter.IsAuthError`).
func RetryWithFreshTokens[T any](ctx context.Context, isAuthErr func(error) bool, fn func(context.Context) (T, error)) (T, error) {
	res, err := fn(ctx)
	if err == nil || isAuthErr == nil || !isAuthErr(err) {
		return res, err
	}
	xuid := ctxkeys.HaloXUID(ctx)
	if xuid == "" {
		return res, err
	}
	// Un token réputé valide-par-expiry s'est fait 401/403 → signal anormal
	// (révocation serveur / changement de flight / dérive d'horloge). Info, pas Debug.
	slog.InfoContext(ctx, "halo: filet auth déclenché (401/403 sur token valide-par-expiry) — re-mint + retry unique",
		"xuid", xuid, "err", err)
	InvalidateCachedPlayerTokens(xuid)
	fresh, rerr := ResolveFreshPlayerTokens(ctx, xuid)
	if rerr != nil || fresh == nil {
		slog.WarnContext(ctx, "halo: filet auth — re-mint échoué, dégradation",
			"xuid", xuid, "err", rerr)
		return res, err // re-mint impossible → erreur d'origine
	}
	return fn(ctxkeys.WithHaloAuth(ctx, fresh, xuid))
}
