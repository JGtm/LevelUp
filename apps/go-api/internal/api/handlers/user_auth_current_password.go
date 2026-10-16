// Package handlers — user_auth_current_password.go : le mot de passe actuel exigé
// pour en changer (POST /auth/password).
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/platform/userstore"
)

// checkCurrentPassword exige le mot de passe actuel quand le compte en a déjà un : une
// session volée ou laissée ouverte ne suffit pas à le remplacer. Un compte sans mot
// de passe (créé par la connexion Xbox) définit le premier sans lui. Un mot de passe
// actuel faux est un échec de connexion pour le limiteur (identifiant et IP), un
// juste remet les compteurs à zéro.
func (h *UserAuthHandler) checkCurrentPassword(ctx context.Context, username, current, ip string) error {
	user, err := h.users.Get(username)
	if errors.Is(err, userstore.ErrUserNotFound) {
		return humacore.NewError(http.StatusNotFound, "user_not_found", "user not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "auth: lecture du compte avant changement de mot de passe échouée", "username", username, "err", err)
		return errSetPassword()
	}
	if user.PasswordHash == "" {
		return nil
	}
	if current == "" {
		return humacore.NewError(http.StatusBadRequest, "current_password_required", "current password is required to change it")
	}
	if _, err := h.users.Authenticate(username, current); err != nil {
		if !errors.Is(err, userstore.ErrInvalidCredentials) {
			slog.ErrorContext(ctx, "auth: vérification du mot de passe actuel échouée", "username", username, "err", err)
			return errSetPassword()
		}
		h.loginAttempts.RecordFailure(username, ip)
		slog.WarnContext(ctx, "auth: changement de mot de passe refusé, mot de passe actuel faux", "username", username, "ip", ip)
		return humacore.NewError(http.StatusForbidden, "invalid_current_password", "current password is incorrect")
	}
	h.loginAttempts.RecordSuccess(username, ip)
	return nil
}

// errSetPassword : 500 set_password_error, détail au journal par l'appelant.
func errSetPassword() error {
	return humacore.NewError(http.StatusInternalServerError, "set_password_error", "could not set the password")
}
