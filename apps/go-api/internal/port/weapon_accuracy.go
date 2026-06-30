package port

import (
	"context"
	"errors"
)

// WeaponAccuracyFilters paramètre la lecture agrégée de la précision par arme
// (table weapon_accuracy : tirs tirés / tirs au but par (xuid, weapon_id)).
//
// Garde-fou identique à WeaponKillFilters : la requête SQL agrège sur la table
// weapon_accuracy partagée. Sans MatchIDs ni (Gamertag/XUIDs), scan complet →
// rejeté par Validate().
type WeaponAccuracyFilters struct {
	// MatchIDs ne garde que les rows de ces matchs (scope filtré de la Synthèse).
	MatchIDs []string

	// Gamertag filtre les tirs attribués à ce joueur (résolu via xuid_aliases).
	Gamertag string

	// XUIDs filtre par xuids (alternative à Gamertag pour multi-joueurs).
	XUIDs []string
}

// ErrWeaponAccuracyFiltersInvalid est retournée par Validate() pour combinaisons
// rejetées.
var ErrWeaponAccuracyFiltersInvalid = errors.New("port: invalid WeaponAccuracyFilters")

// ErrWeaponAccuracyFiltersTooBroad est retournée si scan complet.
var ErrWeaponAccuracyFiltersTooBroad = errors.New("port: WeaponAccuracyFilters too broad (provide MatchIDs and (Gamertag or XUIDs))")

// Validate vérifie que les filtres sont cohérents (mêmes règles que
// WeaponKillFilters) :
//   - MatchIDs requis (sinon scan complet)
//   - Ni Gamertag ni XUIDs renseigné → rejeté
func (f WeaponAccuracyFilters) Validate() error {
	if len(f.MatchIDs) == 0 {
		return ErrWeaponAccuracyFiltersTooBroad
	}
	if f.Gamertag == "" && len(f.XUIDs) == 0 {
		return ErrWeaponAccuracyFiltersTooBroad
	}
	return nil
}

// WeaponAccuracyRow est une ligne agrégat (xuid, weapon_id, Σ shots_fired,
// Σ shots_landed). L'agrégation est faite côté DuckDB (GROUP BY xuid, weapon_id).
// La précision (shots_landed/shots_fired) et le Label EN/FR sont calculés/résolus
// côté service après chargement (le repo reste agnostique).
type WeaponAccuracyRow struct {
	XUID        string `json:"xuid"`
	WeaponID    int64  `json:"weapon_id"`
	ShotsFired  int    `json:"shots_fired"`
	ShotsLanded int    `json:"shots_landed"`
	// Label EN ou FR résolu côté service. Vide côté repo.
	Label string `json:"label,omitempty"`
}

// WeaponAccuracyRepository expose le loader agrégé weapon_accuracy.
//
// Capability gating : retourne games.ErrCapabilityNotSupported si la table
// weapon_accuracy est absente (titre qui ne peuple pas cette donnée, ex. Halo
// Infinite) → dégradation gracieuse côté service (best-effort nil).
type WeaponAccuracyRepository interface {
	// LoadWeaponAccuracyAggregated charge la précision agrégée par (xuid,
	// weapon_id) du titre indiqué, filtrée par WeaponAccuracyFilters. Le service
	// appelant doit appeler filters.Validate() avant.
	LoadWeaponAccuracyAggregated(
		ctx context.Context,
		slug string,
		filters WeaponAccuracyFilters,
	) ([]WeaponAccuracyRow, error)
}
