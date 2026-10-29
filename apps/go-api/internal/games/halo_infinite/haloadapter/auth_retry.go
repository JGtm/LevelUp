package haloadapter

// auth_retry.go — le filet 401/403 des lectures live du fournisseur et le verdict d'échec
// d'authentification des services Halo.

import (
	"context"

	"levelup/go-api/internal/haloapi/infinite"
	"levelup/go-api/internal/platform/halo"
)

// IsAuthError dit si `err` est un refus d'authentification d'un service Halo (HTTP 401/403),
// verdict TYPÉ de la brique. Un blob du CDN public n'en est jamais un. Les chemins live qui
// passent par le pool ou par le filet de jetons frais s'en servent comme prédicat.
func IsAuthError(err error) bool { return infinite.IsAuthError(err) }

// retryOnAuth : le filet du fournisseur. Sur un refus d'authentification d'une lecture live
// signée du jeton du lecteur, et si le contexte porte un xuid : cache de jetons du joueur
// invalidé, jeton re-dérivé, UN nouvel essai.
func retryOnAuth[T any](ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	return halo.RetryWithFreshTokens(ctx, IsAuthError, fn)
}
