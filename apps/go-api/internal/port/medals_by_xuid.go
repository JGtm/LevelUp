package port

import (
	"context"
	"errors"
	"fmt"
)

// MedalsByXUIDFilters parametre la lecture des medailles par (xuid, match_id).
//
// Garde-fou : scan complet rejete via Validate(). MatchIDs requis.
type MedalsByXUIDFilters struct {
	// MatchIDs : liste fermee des matchs a interroger. Requis.
	MatchIDs []string

	// XUIDs : liste fermee des xuids a interroger. Requis (multi-joueurs squad).
	XUIDs []string

	// Limit borne le nombre de lignes retournees (0 = pas de limite). La
	// galerie squad audit § 22 plafonne a 20 matchs partages — Limit defait
	// la pagination cote service.
	Limit int
}

// ErrMedalsByXUIDFiltersInvalid est retournee par Validate().
var ErrMedalsByXUIDFiltersInvalid = errors.New("port: invalid MedalsByXUIDFilters")

// ErrMedalsByXUIDFiltersTooBroad est retournee si scan complet.
var ErrMedalsByXUIDFiltersTooBroad = errors.New("port: MedalsByXUIDFilters too broad (MatchIDs and XUIDs required)")

// Validate verifie que les filtres sont coherents.
func (f MedalsByXUIDFilters) Validate() error {
	if f.Limit < 0 {
		return fmt.Errorf("%w: Limit must be >= 0, got %d",
			ErrMedalsByXUIDFiltersInvalid, f.Limit)
	}
	if len(f.MatchIDs) == 0 || len(f.XUIDs) == 0 {
		return ErrMedalsByXUIDFiltersTooBroad
	}
	return nil
}

// MedalRow est une ligne brute medaille du titre indexee par (xuid, match_id, medal_id).
//
// Label EN ou FR est resolu cote service via metadata.medal_citation_mappings
// (le repo reste agnostique des libelles).
type MedalRow struct {
	XUID    string `json:"xuid"`
	MatchID string `json:"match_id"`
	MedalID int64  `json:"medal_id"`
	Count   int    `json:"count"`
	// Label resolu cote service.
	Label string `json:"label,omitempty"`
}

// MedalsByXUIDRepository expose le loader medailles aggregat par (xuid, match).
//
// Capability gating : games.ErrCapabilityNotSupported si le titre n'a pas la
// capability "match.detail.medals".
type MedalsByXUIDRepository interface {
	LoadMedalsForMatchesByXUID(
		ctx context.Context,
		slug string,
		filters MedalsByXUIDFilters,
	) ([]MedalRow, error)
}
