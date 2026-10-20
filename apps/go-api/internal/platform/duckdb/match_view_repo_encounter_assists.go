// Package duckdb — match_view_repo_encounter_assists.go : assistances échangées avec les
// joueurs du match, sur tout l'historique commun (colonne « Assistances » de
// l'historique des rencontres). Même requête que la page Relations (Q28c).
package duckdb

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
)

// GetMatchEncounterAssists retourne, pour chaque joueur du match qui a déjà été
// coéquipier du joueur sur un match mesuré, les assistances échangées (Q28c). Map vide
// quand rien n'est mesuré.
func (r *MatchViewRepo) GetMatchEncounterAssists(
	ctx context.Context, matchID, myXUID string,
) (map[string]domain.RelationAssists, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	db, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchEncounterAssists: shared reader: %w", err)
	}
	defer release()
	out, err := queryRelationAssists(ctx, db, assistExchangeQuery{me: myXUID, partnersOfMatch: matchID})
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchEncounterAssists: %w", err)
	}
	return out, nil
}
