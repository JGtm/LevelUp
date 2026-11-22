// Package wire — registry_kickoff_revision.go : l'étape de démarrage de la base PARTAGÉE d'un titre,
// le rattrapage du coup d'envoi et du temps de jeu quand la révision rangée dans son `sync_meta` diffère
// de `timeline.KickoffRev` (ADR 0050). Lectures par le provider B-swap du titre, écritures en bursts
// courts du writer partagé ; le service scoringrevision la joue avant les notes.
package wire

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/kickoff"
)

// kickoffRevisionStep implémente port.SharedRevisionStep sur le registre.
type kickoffRevisionStep struct{ r *ServiceRegistry }

var _ port.SharedRevisionStep = kickoffRevisionStep{}

// Apply rattrape la base partagée du titre si sa révision rangée diffère de celle du binaire. Un titre sans
// base partagée n'a rien à rattraper (la base n'est pas créée).
func (s kickoffRevisionStep) Apply(ctx context.Context, titleSlug string) (bool, error) {
	path := config.SharedDBPath(s.r.cfg, titleSlug)
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("base partagée illisible: %w", err)
	}
	provider := s.r.sharedProviderForTitle(titleSlug)
	if provider == nil {
		return false, fmt.Errorf("aucun provider de la base partagée pour %s", titleSlug)
	}
	opts := kickoff.Options{TitleSlug: titleSlug, LotSize: kickoff.DefaultLotSize}
	caps, err := games.LoadCapabilityMap(s.r.cfg.Roots().Resources, titleSlug)
	if err != nil {
		return false, fmt.Errorf("capacités du titre illisibles: %w", err)
	}
	if caps.Has(games.CapFilmReplayArtifact) {
		paths := s.r.cfg.PathResolver()
		opts.ArtifactPath = func(matchID string) string { return paths.ReplayArtifactPath(titleSlug, matchID) }
	}
	acc := kickoff.Access{
		Read:  provider.Get,
		Write: syncpkg.NewSharedWriterAcquirer(provider, path, "kickoff_revision", acquireWriterTimeout),
	}
	applied, _, err := kickoff.ApplyRevision(ctx, acc, opts)
	if errors.Is(err, kickoff.ErrPartial) {
		return applied, fmt.Errorf("%w: %w", port.ErrSharedRevisionPartial, err)
	}
	return applied, err
}
