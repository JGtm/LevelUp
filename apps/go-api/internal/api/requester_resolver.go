// Package api — requester_resolver.go : le demandeur des écritures sociales (médias).
package api

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/authz"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
)

// requesterPlayerSlugResolver rend, pour une requête, le slug du profil POSSÉDÉ par
// l'utilisateur de la session dans le titre courant : le profil de db_profiles.json
// dont le xuid est le xuid lié de l'utilisateur (ADR 0029). "" quand l'utilisateur
// n'en possède aucun (non connecté, non lié, admin sans profil). C'est l'identité du
// demandeur d'une suppression, d'un like ou d'une association de média, et le viewer
// des galeries (lire et écrire désignent le même viewer).
//
// Le joueur courant de la session n'entre jamais dans cette résolution quand la
// propriété est appliquée : POST /session/context le laisse choisir parmi les profils
// lisibles (ceux des amis compris), il ne prouve aucune propriété. Hors propriété
// appliquée (mono-utilisateur, démo), aucune identité n'est opposable : le joueur
// choisi dans l'interface est rendu.
func requesterPlayerSlugResolver(cfg *config.AppConfig, users authz.UserLookup) func(context.Context) string {
	if !authz.Enforced(cfg.DemoMode, cfg.AuthMode) || users == nil {
		return middleware.SessionPlayerSlug
	}
	return func(ctx context.Context) string {
		user := authz.CurrentUser(middleware.GetSession(ctx), users)
		if user == nil || user.XUID == "" {
			return ""
		}
		titleSlug := ctxkeys.TitleSlug(ctx)
		players, err := cfg.LoadPlayers(titleSlug)
		if err != nil {
			slog.ErrorContext(ctx, "demandeur: profils illisibles — aucun profil possédé résolu",
				"err", err, "titleSlug", titleSlug)
			return ""
		}
		for i := range players {
			if players[i].XUID == user.XUID && !players[i].AuthOnly {
				return players[i].PlayerSlug
			}
		}
		return ""
	}
}
