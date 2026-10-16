package handlers

import (
	"context"
	"log/slog"
	"strings"

	"levelup/go-api/internal/api/middleware"
)

// revokeTargetSessions termine la réinitialisation d'un mot de passe par l'admin : toutes
// les sessions du compte visé sont révoquées (Store.RevokeAccount), un tiers qui tenait
// une session ouverte avec l'ancien mot de passe la perd. Quand l'admin réinitialise son
// PROPRE compte par cette route, sa session courante est conservée.
//
// Le mot de passe est déjà changé quand on arrive ici : un échec est journalisé en ERROR,
// il ne transforme pas la réinitialisation en échec côté admin.
func (h *AdminHandler) revokeTargetSessions(ctx context.Context, target string) {
	if h.sessions == nil {
		return
	}
	keepID := ""
	if sess := middleware.GetSession(ctx); sess != nil && sess.Username != nil && strings.EqualFold(*sess.Username, target) {
		keepID = sess.SessionID
	}
	revoked, err := h.sessions.RevokeAccount(target, keepID)
	if err != nil {
		slog.ErrorContext(ctx, "admin: révocation des sessions du compte réinitialisé incomplète",
			"target", target, "revoked", revoked, "err", err)
		return
	}
	slog.InfoContext(ctx, "admin: sessions du compte réinitialisé révoquées", "target", target, "revoked", revoked)
}
