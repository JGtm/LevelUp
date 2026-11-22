// Package wire — registry_scoring_revision.go : le câblage du passage de démarrage des révisions de
// mesure (service scoringrevision). Deux étapes par joueur, chacune avec sa révision rangée dans
// sync_meta : le rejeu des KPI après un changement de formule, sur le recalculateur de match (base
// partagée du titre par son provider, base metadata du titre, base de chaque joueur sous son lease) ;
// le recalage du départ des campagnes de l'axe Objectif après un changement de sa mesure. Le câblage
// n'injecte que les profils et ces deux étapes ; l'ordre et l'indépendance sont ceux du service.
package wire

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"levelup/go-api/internal/campaign"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/playerresolve"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/scoringrevision"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/skill"
)

// NewScoringRevision construit le service de rejeu des KPI après un changement de formule. Le
// serveur lance son passage unique sur le contexte des tâches de fond (cmd/server). Le câblage injecte
// les profils et les deux étapes ; l'ordre et l'indépendance des étapes sont décidés par le service.
func (r *ServiceRegistry) NewScoringRevision() *scoringrevision.Service {
	return scoringrevision.New(scoringRevisionPlayers{r: r}, kickoffRevisionStep{r: r},
		kpiRevisionStep{r: r}, campaignRevisionStep{r: r})
}

// scoringRevisionPlayers implémente port.ScoringRevisionPlayers sur le registre.
type scoringRevisionPlayers struct{ r *ServiceRegistry }

var _ port.ScoringRevisionPlayers = scoringRevisionPlayers{}

func (p scoringRevisionPlayers) FollowedPlayers(titleSlug string) ([]domain.PlayerSummary, error) {
	return p.r.cfg.LoadPlayers(titleSlug)
}

// HasPlayerDB dit si le fichier de la base joueur existe, sans le créer.
func (p scoringRevisionPlayers) HasPlayerDB(titleSlug string, player domain.PlayerSummary) (bool, error) {
	if _, err := os.Stat(config.PlayerDBPath(p.r.cfg, titleSlug, player.Gamertag)); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// kpiRevisionStep : l'étape des notes, sur le recalculateur de match (base partagée du titre par son
// provider, base metadata du titre, base du joueur sous son lease).
type kpiRevisionStep struct{ r *ServiceRegistry }

var _ port.ScoringRevisionStep = kpiRevisionStep{}

// Apply rejoue les KPI du joueur si sa révision de formule rangée diffère de celle du binaire.
func (s kpiRevisionStep) Apply(ctx context.Context, titleSlug string, player domain.PlayerSummary) (bool, error) {
	return syncpkg.NewMatchRecomputer(
		config.PlayerDBPath(s.r.cfg, titleSlug, player.Gamertag),
		config.SharedDBPath(s.r.cfg, titleSlug),
		config.MetadataDBPath(s.r.cfg, titleSlug),
		player.XUID,
		player.Gamertag,
		s.r.sharedProviderForTitle(titleSlug),
	).WithTitle(titleSlug).ApplyScoringRevision(ctx, skill.ScoringRevision())
}

// campaignRevisionStep : l'étape du départ des campagnes de l'axe Objectif.
type campaignRevisionStep struct{ r *ServiceRegistry }

var _ port.ScoringRevisionStep = campaignRevisionStep{}

// Apply recale le départ des campagnes en cours du joueur sur l'axe Objectif si sa révision de mesure
// rangée diffère de campaign.ObjectiveAxisRev ; rend true quand une campagne l'a été.
func (s campaignRevisionStep) Apply(ctx context.Context, titleSlug string, player domain.PlayerSummary) (bool, error) {
	pdb, err := playerresolve.ResolvePlayer(ctx, s.r.cfg, player.PlayerSlug, titleSlug)
	if err != nil {
		return false, fmt.Errorf("base joueur indisponible pour le recalage des campagnes: %w", err)
	}
	svc := campaign.NewService(duckdb.NewCampaignRepo(pdb.Player), duckdb.NewCampaignSampleProvider(pdb)).
		WithObjectiveMeasure(s.r.capabilitiesForPDB(pdb).Has(games.CapMatchObjectiveStats))
	store := duckdb.NewSyncMetaRevisionStore(pdb, campaign.SyncMetaObjectiveAxisRev)
	n, err := svc.ApplyObjectiveAxisRev(ctx, store, player.XUID, titleSlug, time.Now().UTC())
	return n > 0, err
}
