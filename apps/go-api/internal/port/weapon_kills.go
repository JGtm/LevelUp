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

	// ResolveRoles : si true, le repo renseigne WeaponKillRow.Role ET
	// WeaponKillRow.Class via le registre d'armes (passage par weapons/weapon_ids
	// dans metadata, une SEULE passe). Best-effort (Role/Class restent vides si le
	// registre est absent). Off par defaut → zero cout pour les consommateurs qui
	// n'en ont pas besoin (timeseries top weapons, squad).
	ResolveRoles bool
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
	// LabelEN : meme libelle que Label mais EN-first (repli FR), resolu dans la MEME passe
	// (resolveWeaponMeta). Ajoute le 2026-08-29 (V2.1) pour porter le nom d'engin EN jusqu'a
	// FragRoleEntry.LabelEN (classes vehicule/tourelle, domain.IsPerWeaponFragClass) — vide
	// pour les autres classes (non lu ailleurs). Vide cote repo.
	LabelEN string `json:"label_en,omitempty"`
	// Role : fonction de combat de l'arme (automatic/precision/sniper/shotgun/
	// sidearm/power/special/melee/grenade), resolu depuis le referentiel
	// metadata.weapons (duckdb/weapon_resolver.go) quand
	// WeaponKillFilters.ResolveRoles=true. Vide sinon.
	Role string `json:"role,omitempty"`
	// Class : axe manipulation de l'arme (shoulder/sidearm/heavy/melee/grenade +
	// buckets non-combat H5 vehicle/turret/…), resolu dans la MÊME passe que Role
	// quand ResolveRoles=true (COALESCE(w.class) du registre). Vide sinon. Sert
	// l'agregation par CLASSE (buildFragDistribution, sunburst v2).
	Class string `json:"class,omitempty"`
	// Family : famille d'arme registre (battle_rifle/frag_grenade/…), resolue dans la
	// MÊME passe que Role/Class (COALESCE(w.family_key)). Vide sinon. Sert le niveau 2
	// de la classe Grenade (TYPE de grenade, V72-15.2 : frag/plasma/dynamo/splinter).
	Family string `json:"family,omitempty"`
	// WeaponKey : cle canonique du registre ("h5_vehicle_warthog", "hinf_br75"), resolue
	// dans la MÊME passe que Role/Class/Family (COALESCE(w.weapon_key)). Vide sinon. Sert
	// le niveau 2 des classes ventilees PAR ENGIN (domain.IsPerWeaponFragClass :
	// vehicule/tourelle), ou Role et Family valent la classe elle-meme et ne distinguent
	// donc aucun engin (V73-3.2).
	WeaponKey string `json:"weapon_key,omitempty"`
	// IsGrenadeMelee : true si la valeur vient d'highlight_events (grenade ou
	// melee), false si elle vient de weapon_kills (arme primaire).
	IsGrenadeMelee bool `json:"is_grenade_melee,omitempty"`
	// MechanicKills : sous-ensemble de Kills qui ne sont PAS des kills d'ARME mais des
	// mécaniques natives Halo 5 (mêlée/assassinat/coup au sol/charge d'épaule) attribuées
	// à l'arme TENUE dans weapon_kills (kill_kind <> 'weapon'). Sur Infinite kill_kind est
	// NULL → 0. buildFragDistribution les RETIRE des classes gun (anti-double-comptage
	// V72-15.3) : ces kills sont déjà servis par les compteurs natifs (Mêlée/Capacités
	// spartanes). Le breakdown par arme, lui, garde Kills complet (arme tenue).
	MechanicKills int `json:"mechanic_kills,omitempty"`
}

// KillMechanicsRow agrège les mécaniques de kill NATIVES Halo 5 par joueur (xuid)
// sur un ensemble de matchs — assassinats + compétences spartiate (ground pound,
// shoulder bash). Source : shared.match_participants (GROUP BY xuid). Alimente le
// breakdown « mécaniques par coéquipier » de la page Escouade. 0 pour les titres
// qui ne fournissent pas ces colonnes (Infinite).
type KillMechanicsRow struct {
	XUID           string `json:"xuid"`
	Assassinations int    `json:"assassinations"`
	GroundPound    int    `json:"ground_pound_kills"`
	ShoulderBash   int    `json:"shoulder_bash_kills"`
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
