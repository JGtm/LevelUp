// Package api — server_presence.go : assemblage du service de présence en jeu.
//
// Wiring pur (aucune décision métier) : il branche les deux sources dont
// service.PresenceService a besoin sur ce que le process possède déjà — l'état
// du watcher, et le service de bootstrap qui sait quels joueurs la session voit
// et lesquels lui appartiennent. Chaque source absente retire sa part de la
// réponse sans faire échouer l'endpoint (cf. presence_service.go).
//
// Aucune sortie réseau ici : le compteur « amis en jeu » se déduit de l'état du
// watcher et de la propriété des profils (décision produit du 2026-08-25).
//
// Extrait de server_apiv1.go, qui est un assembleur déjà exempté du seuil de
// 500 lignes : on n'y ajoute pas d'adaptateurs.
package api

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/service"
	"levelup/go-api/internal/watcher"
)

// buildPresenceService assemble le service derrière GET /api/v1/presence.
func buildPresenceService(
	bootSvc *service.BootstrapService,
	daemon watcher.DaemonController,
) *service.PresenceService {
	if bootSvc == nil {
		// Sans bootstrap : ni liste de joueurs, ni propriété — réponse vide.
		return service.NewPresenceService(nil, trackedPresenceFrom(daemon))
	}
	return service.NewPresenceService(bootSvc.OwnedPlayers, trackedPresenceFrom(daemon)).
		WithFriends(bootSvc.DirectOwnerFor)
}

// trackedPresenceFrom adapte l'état du watcher en source de présence. nil si le
// watcher est désactivé ; tranche vide si le daemon est arrêté — dans les deux
// cas la réponse ne liste aucun joueur, ce qui est exact : on ne sait rien.
//
// `last_event_at` traverse avec le titre : c'est le témoin de vivacité du poll,
// dont le service se sert pour ne pas servir un titre figé (cf.
// service.presenceFreshnessWindow). Le watcher le publie en RFC3339 ; un champ
// vide ou illisible donne le temps zéro, que le service lit comme « aucune
// information », donc « pas en jeu ».
func trackedPresenceFrom(daemon watcher.DaemonController) service.TrackedPresenceSource {
	if daemon == nil {
		return nil
	}
	return func() []service.TrackedPresence {
		status := daemon.GetStatus()
		if !status.Running {
			return nil
		}
		out := make([]service.TrackedPresence, 0, len(status.Players))
		for _, p := range status.Players {
			var lastEventAt time.Time
			if p.LastEventAt != "" {
				parsed, err := time.Parse(time.RFC3339, p.LastEventAt)
				if err != nil {
					slog.WarnContext(context.Background(),
						"presence: last_event_at illisible — joueur traité comme hors jeu",
						"gamertag", p.Gamertag, "value", p.LastEventAt, "err", err)
				} else {
					lastEventAt = parsed
				}
			}
			out = append(out, service.TrackedPresence{
				Gamertag:    p.Gamertag,
				TitleSlug:   p.TitleSlug,
				TitleName:   p.TitleName,
				LastEventAt: lastEventAt,
			})
		}
		return out
	}
}
