package port

import (
	"context"
	"errors"
	"fmt"
)

// WeaponKillFilters parametre la lecture aggregate des kills par arme.
//
// Garde-fou : la requete SQL aggrege sur shared.weapon_kills (ou equivalent
// title-specific). Sans MatchIDs ni Gamertag, scan complet → rejete par
// Validate().
type WeaponKillFilters struct {
	// MatchIDs ne garde que les kills de ces matchs. Recommande (Squad V2 :
	// matchs partages de l'intersection).
	MatchIDs []string

	// Gamertag filtre les kills attribues a ce joueur (cote killer).
	Gamertag string

	// XUIDs filtre par xuids (alternative a Gamertag pour multi-joueurs).
	// Utilise par squad weapons table (charge les armes des N membres en 1
	// requete groupee).
	XUIDs []string

	// MinKills exclut les armes avec total < N kills (slider min kills audit
	// section 21). 0 = pas de filtre.
	MinKills int

	// IncludeGrenadeMelee : si true, ajoute les grenade/melee kills (event
	// types specifiques highlight_events). Si false, ne renvoie que les kills
	// from weapon_kills (armes principales).
	IncludeGrenadeMelee bool
}

// ErrWeaponKillFiltersInvalid est retournee par Validate() pour combinaisons
// rejetees.
var ErrWeaponKillFiltersInvalid = errors.New("port: invalid WeaponKillFilters")

// ErrWeaponKillFiltersTooBroad est retournee si scan complet.
var ErrWeaponKillFiltersTooBroad = errors.New("port: WeaponKillFilters too broad (provide MatchIDs and (Gamertag or XUIDs))")

// Validate verifie que les filtres sont coherents.
//
// Cas rejetes :
//   - MinKills < 0
//   - MatchIDs vide
//   - Ni Gamertag ni XUIDs renseigne
func (f WeaponKillFilters) Validate() error {
	if f.MinKills < 0 {
		return fmt.Errorf("%w: MinKills must be >= 0, got %d",
			ErrWeaponKillFiltersInvalid, f.MinKills)
	}
	if len(f.MatchIDs) == 0 {
		return ErrWeaponKillFiltersTooBroad
	}
	if f.Gamertag == "" && len(f.XUIDs) == 0 {
		return ErrWeaponKillFiltersTooBroad
	}
	return nil
}

// WeaponKillRow est une ligne aggregat (xuid, weapon_id, total_kills).
//
// L'agregation est faite cote DuckDB (GROUP BY xuid, effective_weapon_id).
// Le Label EN/FR est resolu cote service via metadata.weapon_labels apres
// chargement (le repo reste agnostique).
type WeaponKillRow struct {
	XUID     string `json:"xuid"`
	WeaponID int64  `json:"weapon_id"`
	Kills    int    `json:"kills"`
	// Label EN ou FR resolu cote service. Vide cote repo.
	Label string `json:"label,omitempty"`
	// IsGrenadeMelee : true si la valeur vient d'highlight_events (grenade ou
	// melee), false si elle vient de weapon_kills (arme primaire).
	IsGrenadeMelee bool `json:"is_grenade_melee,omitempty"`
}

// WeaponKillsRepository expose le loader aggregate weapon_kills.
//
// Capability gating : retourne games.ErrCapabilityNotSupported si le titre
// indique par slug n'a pas la capability "match.detail.weapon_kills".
type WeaponKillsRepository interface {
	// LoadWeaponKillsAggregated charge les kills aggregees du titre filtres
	// par WeaponKillFilters. Le service appelant doit appeler filters.Validate()
	// avant.
	LoadWeaponKillsAggregated(
		ctx context.Context,
		slug string,
		filters WeaponKillFilters,
	) ([]WeaponKillRow, error)
}
