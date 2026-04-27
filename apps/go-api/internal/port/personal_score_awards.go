package port

import (
	"context"
	"errors"
)

// PersonalScoreAwardsFilters parametre la lecture des awards.
//
// Garde-fou : MatchIDs ET XUIDs requis (rejet de scan complet).
type PersonalScoreAwardsFilters struct {
	// MatchIDs : liste fermee des matchs. Requis.
	MatchIDs []string

	// XUIDs : liste fermee des joueurs. Requis (multi-joueurs ou solo).
	XUIDs []string
}

// ErrPersonalScoreAwardsFiltersInvalid est retournee par Validate().
var ErrPersonalScoreAwardsFiltersInvalid = errors.New("port: invalid PersonalScoreAwardsFilters")

// ErrPersonalScoreAwardsFiltersTooBroad est retournee si scan complet.
var ErrPersonalScoreAwardsFiltersTooBroad = errors.New("port: PersonalScoreAwardsFilters too broad (MatchIDs and XUIDs required)")

// Validate verifie la coherence des filtres.
func (f PersonalScoreAwardsFilters) Validate() error {
	if len(f.MatchIDs) == 0 || len(f.XUIDs) == 0 {
		return ErrPersonalScoreAwardsFiltersTooBroad
	}
	return nil
}

// PersonalScoreAwardRow est une ligne aggregat (xuid, match_id, award_name, total).
//
// L'agregation est faite cote DuckDB (GROUP BY xuid, match_id, award_name).
// Le mapping award_name -> ParticipationAxis est laisse au consommateur
// (cf. narrative.ParticipationAxis : combat / survival / support / score /
// objective / impact). Les definitions sont title-specific et viendront d'un
// TOML mappings (Phase 2 ou 3).
//
// Phase 1 pilote : le consommateur (radar MatchView/Squad) utilise une table
// de mapping inline avec les award_name connus de Halo Infinite. Les
// award_name inconnus sont ignores (pas de panic).
type PersonalScoreAwardRow struct {
	XUID      string `json:"xuid"`
	MatchID   string `json:"match_id"`
	AwardName string `json:"award_name"`
	Total     int    `json:"total"` // somme de award_count sur le couple (xuid, match_id, award_name)
}

// PersonalScoreAwardsRepository expose le loader aggregat des awards.
//
// Implemente par internal/platform/duckdb.PersonalScoreAwardsRepo (chunk MV4.B).
//
// Capability gating : retourne games.ErrCapabilityNotSupported si la table
// personal_score_awards est absente de la DB cible (titre sans cette
// capability ou DB de test minimaliste).
type PersonalScoreAwardsRepository interface {
	LoadPersonalScoreAwards(
		ctx context.Context,
		slug string,
		filters PersonalScoreAwardsFilters,
	) ([]PersonalScoreAwardRow, error)
}
