package assets

import (
	"context"

	"levelup/go-api/internal/domain"
)

// TokenProvider est la fonction qui fournit les tokens Halo pour les requêtes authentifiées.
// Correspond à l'ancien BPTokenProvider dans handlers/assets.go.
type TokenProvider func(ctx context.Context) (*domain.HaloTokens, error)

// Fetcher est l'interface d'une source distante d'assets.
// Chaque implémentation gère un ou plusieurs Kinds.
type Fetcher interface {
	// Supports retourne true si ce fetcher peut gérer le Kind donné.
	Supports(k Kind) bool

	// Fetch récupère l'asset identifié par ref depuis la source distante.
	// Retourne ErrNotFound si l'asset n'existe pas sur la source.
	// Retourne ErrUpstreamUnavailable si la source est inaccessible.
	Fetch(ctx context.Context, ref Ref) (Payload, error)
}
