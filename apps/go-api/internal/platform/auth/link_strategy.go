// Package auth — link_strategy.go : abstraction de la logique post-flow d'authentification.
//
// Quand le Device Code Flow réussit (Status=Authorized), AuthHandler appelle
// LinkStrategy.OnAuthSuccess() pour décider quoi faire de l'identité Xbox obtenue :
//   - PasswordLinkStrategy (mode password) : LinkIdentity du user déjà connecté
//   - XboxSSOLinkStrategy (mode xbox) : login direct via XUID, création si nouveau
//
// L'injection se fait au boot selon cfg.AuthMode (cf. server.go).
package auth

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/domain"
)

// LinkStrategy abstrait la logique "que faire quand le Device Code Flow réussit".
// L'implémentation modifie sess en place (Username, CurrentPlayerSlug, etc.)
// et retourne une erreur si l'opération échoue.
type LinkStrategy interface {
	OnAuthSuccess(ctx context.Context, attempt *Attempt, sess *domain.SessionData) error
}

// UserLinker est l'interface minimale dont a besoin PasswordLinkStrategy.
// Implémentée par *userstore.Store ; permet de mocker en test.
type UserLinker interface {
	LinkIdentity(username, gamertag, xuid string) error
}

// PasswordLinkStrategy implémente le flow "post-login" historique :
// l'utilisateur est déjà connecté via password, on lie son gamertag/XUID Halo
// à son user existant.
type PasswordLinkStrategy struct {
	users UserLinker
}

// NewPasswordLinkStrategy crée une PasswordLinkStrategy.
func NewPasswordLinkStrategy(users UserLinker) *PasswordLinkStrategy {
	return &PasswordLinkStrategy{users: users}
}

// Vérification compile-time.
var _ LinkStrategy = (*PasswordLinkStrategy)(nil)

// ErrSessionNotAuthenticated est retournée si la session n'a pas de username
// (cas "mode password sans login préalable" — ne devrait pas arriver mais safe).
var ErrSessionNotAuthenticated = errors.New("link_strategy: session sans username (auth requise)")

// OnAuthSuccess lie le gamertag/XUID au user déjà connecté.
// No-op si le gamertag est vide (failsafe) ou si la session n'a pas d'username.
func (s *PasswordLinkStrategy) OnAuthSuccess(ctx context.Context, attempt *Attempt, sess *domain.SessionData) error {
	if attempt.Gamertag == "" {
		slog.DebugContext(ctx, "password_link: gamertag vide, no-op")
		return nil
	}
	if sess.Username == nil {
		slog.WarnContext(ctx, "password_link: session sans username — ignoré")
		return ErrSessionNotAuthenticated
	}
	if err := s.users.LinkIdentity(*sess.Username, attempt.Gamertag, attempt.XUID); err != nil {
		return err
	}
	slug := attempt.Gamertag
	sess.CurrentPlayerSlug = &slug
	slog.InfoContext(ctx, "password_link: identité Halo liée",
		"username", *sess.Username, "gamertag", attempt.Gamertag)
	return nil
}
