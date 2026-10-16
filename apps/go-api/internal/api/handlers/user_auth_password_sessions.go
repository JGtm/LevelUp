// Package handlers — user_auth_password_sessions.go : ce qu'un changement de mot de passe
// fait aux sessions du compte.
package handlers

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
)

// closeOtherSessions termine un changement de mot de passe réussi : la session courante
// reçoit un nouvel identifiant (Rotate : l'ancien cookie ne vaut plus rien), puis toutes
// les AUTRES sessions du compte sont révoquées (Store.RevokeAccount) : un tiers qui
// tenait une session ouverte avec l'ancien mot de passe la perd.
//
// Le mot de passe est déjà changé quand on arrive ici : un échec est journalisé en ERROR
// (sessions restées ouvertes, nombre et erreur), il ne transforme pas le changement en
// échec côté client.
func (h *UserAuthHandler) closeOtherSessions(ctx context.Context, sess *domain.SessionData) {
	username := *sess.Username
	if _, err := h.sessionStore.Rotate(sess); err != nil {
		slog.ErrorContext(ctx, "auth: rotation de la session au changement de mot de passe échouée",
			"username", username, "err", err)
	}
	revoked, err := h.sessionStore.RevokeAccount(username, sess.SessionID)
	if err != nil {
		slog.ErrorContext(ctx, "auth: révocation des autres sessions du compte incomplète",
			"username", username, "revoked", revoked, "err", err)
		return
	}
	slog.InfoContext(ctx, "auth: autres sessions du compte révoquées au changement de mot de passe",
		"username", username, "revoked", revoked)
}
