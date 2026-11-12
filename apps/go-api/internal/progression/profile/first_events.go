package profile

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/analysis"
)

// computeFKFD compte les matchs de la fenêtre dont le premier frag (First Kill) puis la première
// mort (First Death) sont ceux du joueur. Les événements viennent du chargeur unifié
// HighlightEventsRepo (kill/death de highlight_events pour Infinite ; repli sur les frags
// horodatés du titre quand highlight_events n'en porte pas, ce qui couvre Halo 5), la règle de
// décompte est analysis.CountFirstKillsAndDeaths (celle de la vue match). Un match sans événement
// ne compte pour personne. Toute erreur de lecture remonte à l'appelant, qui la journalise.
func (s *Service) computeFKFD(ctx context.Context, userID string, since, until time.Time) (int, int, error) {
	matchIDs, err := s.playerWindowMatchIDs(ctx, userID, since, until)
	if err != nil {
		return 0, 0, fmt.Errorf("computeFKFD: matchs de la fenêtre: %w", err)
	}
	if len(matchIDs) == 0 {
		return 0, 0, nil
	}
	events, err := s.store.KillDeathEvents(ctx, matchIDs)
	if err != nil {
		return 0, 0, fmt.Errorf("computeFKFD: événements: %w", err)
	}
	fk, fd := analysis.CountFirstKillsAndDeaths(events, userID)
	return fk, fd, nil
}

// playerWindowMatchIDs rend les match_id où le joueur a participé dans la fenêtre (Campagne
// exclue), lus sur la base partagée. Borne la lecture des événements aux matchs du joueur :
// windowMatchIDs, elle, rend tout le registre de la fenêtre.
func (s *Service) playerWindowMatchIDs(ctx context.Context, userID string, since, until time.Time) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	return s.store.PlayerWindowMatchIDs(ctx, userID, since, until)
}
