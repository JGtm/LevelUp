// Package session — limits.go : bornes de ce qu'une session peut coûter au disque.
//
// Une session anonyme se crée sans identifiant ni cookie préalable et peut porter
// des champs choisis par le client : sans plafond de taille ni durée de vie courte,
// des requêtes anonymes répétées remplissent data/sessions/.
package session

import (
	"errors"
	"time"

	"levelup/go-api/internal/domain"
)

const (
	// MaxSessionBytes est la taille maximale d'une session sérialisée (8 Kio). Une
	// session authentifiée avec ses jetons Halo en occupe une fraction ; au-delà, le
	// contenu n'est pas légitime.
	MaxSessionBytes = 8 << 10

	// AnonymousTTL est la durée de vie (glissante, sur last_seen_at) d'une session
	// sans identité authentifiée. Elle plafonne DefaultTTL, jamais l'inverse.
	AnonymousTTL = 24 * time.Hour
)

// ErrSessionTooLarge : la session sérialisée dépasse MaxSessionBytes ; rien n'est écrit.
var ErrSessionTooLarge = errors.New("session: serialized session exceeds size limit")

// carriesIdentity : la session porte une identité authentifiée (compte local, SSO
// Xbox, device-code Halo) et relève donc du TTL complet du Store.
func carriesIdentity(sess *domain.SessionData) bool {
	return sess.Username != nil || sess.AuthReady ||
		sess.LinkedHaloIdentity != nil || sess.HaloTokens != nil
}
