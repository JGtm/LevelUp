package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

func (s *SquadServiceV2) loadAllPlayers(
	ctx context.Context,
	slug, mainGT string,
	teammateGTs []string,
	filters port.PlayerMatchFilters,
) (map[string][]canonical.PlayerMatchRow, []canonical.CapabilityGap, error) {
	allGTs := append([]string{mainGT}, teammateGTs...)

	g, gctx := errgroup.WithContext(ctx)
	var mu sync.Mutex
	perPlayer := make(map[string][]canonical.PlayerMatchRow, len(allGTs))
	var capGaps []canonical.CapabilityGap

	for _, gt := range allGTs {
		gt := gt
		g.Go(func() error {
			rows, err := s.loader.LoadFor(gctx, slug, gt, filters)
			if err != nil {
				if errors.Is(err, games.ErrCapabilityNotSupported) {
					mu.Lock()
					capGaps = append(capGaps, canonical.CapabilityGap{
						CapabilityKey: string(games.CapMatchHistory),
						ReasonCode:    "match_history_unsupported",
						Severity:      "warning",
						Message:       fmt.Sprintf("match.history non supporté pour %s", gt),
					})
					mu.Unlock()
					return nil
				}
				return fmt.Errorf("LoadFor(%s): %w", gt, err)
			}
			mu.Lock()
			perPlayer[gt] = rows
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, nil, err
	}
	return perPlayer, capGaps, nil
}
