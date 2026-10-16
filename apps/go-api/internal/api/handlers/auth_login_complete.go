// Package handlers — auth_login_complete.go : ce qu'une connexion Microsoft réussie
// écrit dans la session, commun au device-flow et au flux OAuth.
package handlers

import (
	"context"
	"errors"

	"levelup/go-api/internal/domain"
	auth_platform "levelup/go-api/internal/platform/auth"
)

// completeLogin applique la stratégie post-flux à une COPIE de la session, puis,
// seulement si elle réussit, recopie le résultat dans sess avec AuthReady, l'identité
// Halo et les jetons Halo de la tentative. En cas d'échec (instance verrouillée,
// compte impossible à créer…), sess reste intacte : aucune trace d'authentification.
//
// ErrSessionNotAuthenticated n'est pas un échec : elle signale qu'il n'existe aucun
// compte local à lier (stratégie mot de passe sans connexion préalable, mode sans
// propriété) ; l'identité Halo prouvée par Microsoft est alors posée seule.
// Sans stratégie, ou sans gamertag, la stratégie n'est pas appelée.
func completeLogin(
	ctx context.Context, strategy auth_platform.LinkStrategy, attempt *auth_platform.Attempt, sess *domain.SessionData,
) error {
	candidate := *sess
	if strategy != nil && attempt.Gamertag != "" {
		err := strategy.OnAuthSuccess(ctx, attempt, &candidate)
		if err != nil && !errors.Is(err, auth_platform.ErrSessionNotAuthenticated) {
			return err
		}
	}
	candidate.AuthReady = true
	if attempt.Gamertag != "" {
		candidate.LinkedHaloIdentity = &domain.HaloIdentity{Gamertag: attempt.Gamertag, XUID: attempt.XUID}
	}
	if attempt.SpartanToken != "" {
		candidate.HaloTokens = &domain.HaloTokens{
			SpartanToken:     attempt.SpartanToken,
			ClearanceToken:   attempt.ClearanceToken,
			SpartanExpiresAt: attempt.SpartanExpiresAt,
		}
	}
	*sess = candidate
	return nil
}

// loginFailureCode rend le code exposé au client pour une stratégie post-flux en échec.
func loginFailureCode(err error) string {
	if errors.Is(err, auth_platform.ErrInstanceLocked) {
		return "instance_locked"
	}
	return "link_failed"
}
