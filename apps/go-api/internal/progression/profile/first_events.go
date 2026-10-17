package profile

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
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
	events, err := duckdb.NewHighlightEventsRepo(s.pdb).Load(ctx, port.HighlightEventFilters{
		MatchIDs:   matchIDs,
		EventTypes: []canonical.HighlightEventType{canonical.EventKill, canonical.EventDeath},
	})
	if err != nil {
		return 0, 0, fmt.Errorf("computeFKFD: événements: %w", err)
	}
	fk, fd := analysis.CountFirstKillsAndDeaths(events, userID)
	return fk, fd, nil
}

// playerWindowMatchIDs rend les match_id où le joueur a participé dans la fenêtre (Campagne
// exclue par campaignExcl), lus sur la base partagée. Borne la lecture des événements aux matchs
// du joueur : windowMatchIDs, elle, rend tout le registre de la fenêtre.
func (s *Service) playerWindowMatchIDs(ctx context.Context, userID string, since, until time.Time) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	sharedDB, release, err := s.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT DISTINCT mp.match_id
		FROM match_participants mp
		JOIN match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?`+s.campaignExcl("mr")+`
		  AND `+startTimeCanonicalMR+` >= ? AND `+startTimeCanonicalMR+` <= ?
	`, userID, since, until)
	if err != nil {
		return nil, fmt.Errorf("requête: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("lecture: %w", err)
	}
	return ids, nil
}
