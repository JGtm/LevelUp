// Package api — server_presence.go : assemblage du service de présence en jeu.
//
// Wiring pur (aucune décision métier) : il branche les quatre sources dont
// service.PresenceService a besoin sur ce que le process possède déjà —
// l'état du watcher, les joueurs possédés, la liste d'amis des Réglages, et le
// client Xbox du tracker. Chaque source absente retire sa part de la réponse
// sans faire échouer l'endpoint (cf. presence_service.go).
//
// Extrait de server_apiv1.go, qui est un assembleur déjà exempté du seuil de
// 500 lignes : on n'y ajoute pas d'adaptateurs.
package api

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/api/wire"
	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	platform_duckdb "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/presence"
	"levelup/go-api/internal/service"
	"levelup/go-api/internal/watcher"
)

// presenceBatcher — la capacité « présence de plusieurs users » du daemon.
// Type-assert plutôt qu'ajout à watcher.DaemonController : contrôler le daemon
// (start/stop/auth/abonnements) et lui emprunter son client Xbox sont deux
// choses différentes, et l'interface de contrôle est déjà implémentée ailleurs.
type presenceBatcher interface {
	PresenceBatch(ctx context.Context, xuids []string) ([]presence.PresenceEvent, error)
}

// buildPresenceService assemble le service derrière GET /api/v1/presence.
func buildPresenceService(
	cfg *config.AppConfig,
	bootSvc *service.BootstrapService,
	daemon watcher.DaemonController,
	reg *wire.ServiceRegistry,
	titleReg *titlePkg.Registry,
) *service.PresenceService {
	var ownedPlayers service.OwnedPlayersFunc
	if bootSvc != nil {
		ownedPlayers = bootSvc.OwnedPlayers
	}
	svc := service.NewPresenceService(ownedPlayers, trackedPresenceFrom(daemon))

	var friendGamertags service.FriendGamertagsFunc
	if reg != nil {
		friendGamertags = reg.FriendGamertags
	}
	counter := service.NewFriendPresenceCounter(
		friendGamertags,
		friendXUIDResolverFrom(cfg),
		friendPresenceFrom(daemon),
		titleReg,
	)
	if counter != nil {
		svc = svc.WithFriends(counter)
	}
	return svc
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

// friendXUIDResolverFrom branche la résolution gamertag → xuid sur la vue
// partagée v_gamertag_lookup. Même source que la recherche de gamertags
// (cfg.SharedProvider) : nil en son absence (démo / shared indisponible).
func friendXUIDResolverFrom(cfg *config.AppConfig) service.FriendXUIDResolver {
	if cfg == nil || cfg.SharedProvider == nil {
		return nil
	}
	repo := platform_duckdb.NewGamertagRepo(cfg.SharedProvider)
	return repo.ResolveXUIDsByGamertags
}

// friendPresenceFrom emprunte au daemon son client Xbox (seul header XBL3.0
// maintenu à jour du process) et réduit chaque enregistrement à ce que le
// comptage exige : un xuid et le titre actif.
func friendPresenceFrom(daemon watcher.DaemonController) service.FriendPresenceFetcher {
	batcher, ok := daemon.(presenceBatcher)
	if !ok || batcher == nil {
		return nil
	}
	return func(ctx context.Context, xuids []string) ([]service.FriendPresence, error) {
		events, err := batcher.PresenceBatch(ctx, xuids)
		if err != nil {
			return nil, err
		}
		out := make([]service.FriendPresence, 0, len(events))
		for _, ev := range events {
			fp := service.FriendPresence{XUID: ev.XUID}
			if ev.PresenceDetail != nil {
				fp.TitleID = ev.PresenceDetail.TitleID
			}
			out = append(out, fp)
		}
		return out, nil
	}
}
