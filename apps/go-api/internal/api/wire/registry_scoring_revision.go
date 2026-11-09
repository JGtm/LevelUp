// Package api — registry_scoring_revision.go : le câblage du rejeu des KPI après un changement de
// formule (service scoringrevision) sur le recalculateur de match : base partagée du titre par son
// provider, base metadata du titre, base de chaque joueur sous son lease.
package wire

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
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

// ApplyForPlayer rejoue les KPI du joueur si sa révision rangée diffère de celle du binaire. Un
// profil sans base joueur (jamais synchronisé) n'a rien à recalculer : il n'est pas créé ici.
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
	return syncpkg.NewMatchRecomputer(
		playerPath,
		config.SharedDBPath(a.r.cfg, titleSlug),
		config.MetadataDBPath(a.r.cfg, titleSlug),
		player.XUID,
		player.Gamertag,
		a.r.sharedProviderForTitle(titleSlug),
	).WithTitle(titleSlug).ApplyScoringRevision(ctx, skill.ScoringRevision())
}
