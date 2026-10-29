// Package halo — auth_retry.go : le filet 401/403 du fournisseur, sur le cache de jetons de
// `platform/auth/tokencache` (plan D3, lot 4.3 ; ce paquet est supprimé au lot 4.4).
package halo

import (
	"context"
	"errors"

	"levelup/go-api/internal/platform/auth/tokencache"
)

// errHaloAuthFailure marque un échec d'authentification (HTTP 401/403) renvoyé par doGet.
var errHaloAuthFailure = errors.New("halo: échec auth (401/403)")

// IsAuthFailure indique si une erreur provient d'un 401/403 Waypoint.
func IsAuthFailure(err error) bool { return errors.Is(err, errHaloAuthFailure) }

// retryOnAuth : filet auth pour le provider halo (predicate = sentinel errHaloAuthFailure).
func retryOnAuth[T any](ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	return tokencache.RetryWithFreshTokens(ctx, IsAuthFailure, fn)
}
