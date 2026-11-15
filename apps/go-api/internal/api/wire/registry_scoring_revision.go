// Package api — registry_scoring_revision.go : le câblage du passage de démarrage des révisions de
// mesure (service scoringrevision). Deux étapes par joueur, chacune avec sa révision rangée dans
// sync_meta : le rejeu des KPI après un changement de formule, sur le recalculateur de match (base
// partagée du titre par son provider, base metadata du titre, base de chaque joueur sous son lease) ;
// le recalage du départ des campagnes de l'axe Objectif après un changement de sa mesure.
package wire

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"time"

	"levelup/go-api/internal/campaign"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/playerresolve"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/scoringrevision"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/skill"
)

// NewScoringRevision construit le service de rejeu des KPI après un changement de formule. Le
// serveur lance son passage unique sur le contexte des tâches de fond (cmd/server).
func (r *ServiceRegistry) NewScoringRevision() *scoringrevision.Service {
	return scoringrevision.New(scoringRevisionApplier{r: r})
}

// scoringRevisionApplier implémente port.ScoringRevisionApplier sur le registre.
type scoringRevisionApplier struct {
	r *ServiceRegistry
}

var _ port.ScoringRevisionApplier = scoringRevisionApplier{}

func (a scoringRevisionApplier) FollowedPlayers(titleSlug string) ([]domain.PlayerSummary, error) {
	return a.r.cfg.LoadPlayers(titleSlug)
}

// ApplyForPlayer applique les deux révisions du joueur : KPI (notes et LUSR rejoués si sa révision de
// formule rangée diffère de celle du binaire) puis départ des campagnes de l'axe Objectif (recalé si sa
// révision de mesure rangée diffère). Les étapes sont indépendantes : l'échec de l'une ne retient pas
// l'autre, et chacune garde sa révision rangée tant qu'elle n'a pas réussi. Rend true quand l'une a
// recalculé quelque chose. Un profil sans base joueur (jamais synchronisé) n'a rien à recalculer : il
// n'est pas créé ici.
func (a scoringRevisionApplier) ApplyForPlayer(
	ctx context.Context, titleSlug string, player domain.PlayerSummary,
) (bool, error) {
	playerPath := config.PlayerDBPath(a.r.cfg, titleSlug, player.Gamertag)
	if _, err := os.Stat(playerPath); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return false, fmt.Errorf("base joueur illisible: %w", err)
		}
		slog.InfoContext(ctx, "scoring revision: profil sans base joueur, rien à recalculer",
			"player", player.Gamertag, "titleSlug", titleSlug)
		return false, nil
	}
	kpi, kpiErr := a.applyKPI(ctx, titleSlug, player, playerPath)
	campaigns, campaignErr := a.applyCampaignObjectiveRev(ctx, titleSlug, player)
	return kpi || campaigns, errors.Join(kpiErr, campaignErr)
}

// applyKPI rejoue les KPI du joueur si sa révision de formule rangée diffère de celle du binaire.
func (a scoringRevisionApplier) applyKPI(
	ctx context.Context, titleSlug string, player domain.PlayerSummary, playerPath string,
) (bool, error) {
	return syncpkg.NewMatchRecomputer(
		playerPath,
		config.SharedDBPath(a.r.cfg, titleSlug),
		config.MetadataDBPath(a.r.cfg, titleSlug),
		player.XUID,
		player.Gamertag,
		a.r.sharedProviderForTitle(titleSlug),
	).WithTitle(titleSlug).ApplyScoringRevision(ctx, skill.ScoringRevision())
}

// applyCampaignObjectiveRev recale le départ des campagnes en cours du joueur sur l'axe Objectif si sa
// révision de mesure rangée diffère de campaign.ObjectiveAxisRev ; rend true quand une campagne l'a été.
func (a scoringRevisionApplier) applyCampaignObjectiveRev(
	ctx context.Context, titleSlug string, player domain.PlayerSummary,
) (bool, error) {
	pdb, err := playerresolve.ResolvePlayer(ctx, a.r.cfg, player.PlayerSlug, titleSlug)
	if err != nil {
		return false, fmt.Errorf("base joueur indisponible pour le recalage des campagnes: %w", err)
	}
	svc := campaign.NewService(duckdb.NewCampaignRepo(pdb.Player), duckdb.NewCampaignSampleProvider(pdb))
	store := duckdb.NewSyncMetaRevisionStore(pdb, campaign.SyncMetaObjectiveAxisRev)
	n, err := svc.ApplyObjectiveAxisRev(ctx, store, player.XUID, titleSlug, time.Now().UTC())
	return n > 0, err
}
