// Package service — ExplorerService : matchs communs, recherche croisée.
//
// Port Go de apps/api/app/routers/explorer.py.
package service

import (
	"context"
	"fmt"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// ExplorerService orchestre les requêtes de l'Explorer.
type ExplorerService struct {
	repo port.ExplorerRepository
	xuid string
}

// NewExplorerService crée un ExplorerService.
func NewExplorerService(repo port.ExplorerRepository, xuid string) *ExplorerService {
	return &ExplorerService{repo: repo, xuid: xuid}
}

// GetCommonMatches retourne les matchs en commun avec un autre joueur.
// Résout le gamertag de l'autre joueur, puis filtre via Q19.
func (s *ExplorerService) GetCommonMatches(
	ctx context.Context,
	otherGamertag string,
	limit int,
) (domain.ExplorerPlayerQueryResponse, error) {
	otherXUID, err := s.repo.ResolveXUIDByGamertag(ctx, otherGamertag)
	if err != nil {
		return domain.ExplorerPlayerQueryResponse{},
			fmt.Errorf("ExplorerService: résolution gamertag %q: %w", otherGamertag, err)
	}

	rawMatches, err := s.repo.GetCommonMatches(ctx, s.xuid, otherXUID)
	if err != nil {
		return domain.ExplorerPlayerQueryResponse{},
			fmt.Errorf("ExplorerService: matchs communs: %w", err)
	}

	matches := convertCommonMatches(rawMatches, limit)
	return domain.ExplorerPlayerQueryResponse{
		TargetGamertag: otherGamertag,
		TargetXUID:     otherXUID,
		CommonMatches:  matches,
		Total:          len(matches),
	}, nil
}

// convertCommonMatches convertit les lignes brutes en CommonMatchRow avec were_teammates.
func convertCommonMatches(raw []domain.CommonMatchRaw, limit int) []domain.CommonMatchRow {
	if len(raw) == 0 {
		return []domain.CommonMatchRow{}
	}

	max := len(raw)
	if limit > 0 && limit < max {
		max = limit
	}

	result := make([]domain.CommonMatchRow, 0, max)
	for i := range raw[:max] {
		r := &raw[i]
		wereTeammates := r.Player1TeamID != nil &&
			r.Player2TeamID != nil &&
			*r.Player1TeamID == *r.Player2TeamID

		result = append(result, domain.CommonMatchRow{
			MatchID:       r.MatchID,
			StartTime:     r.StartTime,
			MapUI:         r.MapUI,
			ModeUI:        r.ModeUI,
			WereTeammates: wereTeammates,
			PlayerOutcome: r.Player1Outcome,
			Kills:         r.Player1Kills,
			Deaths:        r.Player1Deaths,
			KDA:           r.Player1KDA,
		})
	}
	return result
}
