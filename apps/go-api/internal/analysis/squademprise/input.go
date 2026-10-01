// Package squademprise — LE BLOC « EMPRISE » de la page Escouade (lot L4 du plan
// PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26) : les prises de chaque ressource par notre camp et
// par l'adversaire, match par match et objet par objet, qui chez nous, ce que chaque camp en a
// produit, et d'habitude. Le type publié et la définition des ressources vivent dans
// domain.SquadEmpriseBlock.
//
// # LES SOURCES, ET CE QUE CHACUNE MESURE
//
//	bonus (D4)            `match_usage_players_latest` : prises (`taken_json`, canal des
//	                      ramassages, ATTRIBUÉES), issues (gardé / lâché, même bascule que la
//	                      barre de Sessions), temps et frags d'effet ; socles vidés au grain du
//	                      match (`powerup_pickups_json`, anonymes).
//	armes spéciales (D3)  `match_pad_pickups_by_tier_latest`, niveau `puissance` ; les râteliers
//	                      au niveau `terrain`. Jamais la classe de l'arme.
//	frags aux armes       `match_participants.power_weapon_kills` (feuille de match, les deux
//	spéciales             camps) : la seule grandeur servie sans film (D10).
//
// # CAMP
//
// Notre camp est celui du joueur de la page dans CE match. Un participant sans camp connu dans un
// match à camp connu compte pour l'adversaire (même règle que computeOutcomes) ; un match dont le
// camp du joueur est inconnu ne publie aucun compte camp contre camp.
//
// Pur : aucune ouverture de base, aucune horloge, aucune chaîne de langue. Les familles de mode,
// les noms d'armes et les comptes de session sont résolus par le service.
package squademprise

import (
	"time"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/analysis/squadformes"
	"levelup/go-api/internal/domain"
)

// Match — un match tel que la page le connaît.
type Match struct {
	MatchID      string
	StartTime    time.Time
	SessionLabel string
	// Family : la famille de mode (libellé de mode normalisé de l'historique de l'escouade).
	// Vide = inconnue : le match n'est comparable à aucune soirée.
	Family string
}

// FilmData — les lectures du résumé d'usage et des niveaux de socle. Un match absent de Films
// n'est pas mesuré ; un match absent de PadTiers n'a pas de niveaux.
type FilmData struct {
	Films        map[string]sessionusage.FilmRow
	Players      []sessionusage.PlayerRow
	Participants []sessionusage.ParticipantRow
	PadTiers     []sessionusage.PadTierRow
}

// PowerKillRow — une ligne (match, joueur) de la feuille de match. Kills nil = la feuille ne le
// dit pas (jamais un zéro inventé).
type PowerKillRow struct {
	MatchID string
	XUID    string
	TeamID  *int
	Kills   *int
}

// PlacementRow — une vie de `match_life_placement_latest` (plan Emprise vies, lot V3), telle que
// le sync l'a écrite. MedianM nil = vie non mesurée ; RadarM et BeyondMS nil ENSEMBLE = variante
// sans portée connue à l'écriture.
type PlacementRow struct {
	MatchID            string
	XUID               string
	StartMS            int64
	EndMS              int64
	DurationMS         int64
	MeasuredMS         int64
	MedianM            *float64
	BeyondMS           *int64
	RadarM             *float64
	CarrierMS          int64
	TeamDownMS         int64
	UnplacedMS         int64
	TeammateUnplacedMS int64
	Kills              int
}

// PlacementRead — la lecture bornée du placement : les vies des joueurs demandés sur les matchs
// demandés, et la variante de chaque match qui en porte (`match_registry.game_variant_name`, la
// clé de la portée courante du radar).
type PlacementRead struct {
	Rows     []PlacementRow
	Variants map[string]string
}

// Input — tout ce que le calcul demande.
type Input struct {
	PlayerXUID string
	// Players : le joueur de la page puis les coéquipiers sélectionnés (fiches).
	Players []domain.SessionUsageSquadPlayer
	// Current : les matchs du périmètre D2.
	Current []Match
	// Timeline : l'historique de la composition (habitude). Vide = pas d'habitude.
	Timeline []Match
	// Film : nil quand le titre n'a pas de résumé d'usage ou que sa lecture a échoué ;
	// FilmUnavailable dit alors pourquoi.
	Film            *FilmData
	FilmUnavailable string
	// PowerKills : la feuille de match du périmètre. Nil = non lue ; SheetUnavailable dit pourquoi.
	PowerKills       []PowerKillRow
	SheetUnavailable string
	// Weapons : clé de famille d'arme -> nom du titre et clé canonique.
	Weapons map[string]squadformes.WeaponInfo
	// SessionMatchCounts : libellé de session -> matchs de la composition (ADR 0033).
	SessionMatchCounts map[string]int
	// Vehicles : la ressource véhicules (lecture de `match_vehicle_takes_latest`, périmètre ET
	// matchs de l'habitude). Nil = non lue ; VehiclesUnavailable dit alors l'échec (vide quand le
	// titre ne mesure simplement pas la ressource). Indépendante du film.
	Vehicles            *VehicleRead
	VehiclesUnavailable string
	// VehicleLabels : famille de véhicule -> nom du titre dans la langue de la requête, pour les
	// seules familles que le manifeste du titre qualifie (une tourelle fixe). Les autres sont des
	// noms propres du jeu, que le client affiche depuis leur clé.
	VehicleLabels map[string]string
}

// resourceOrder — l'ordre de publication des ressources.
var resourceOrder = []string{
	domain.EmpriseResourcePowerup, domain.EmpriseResourcePowerWeapon,
	domain.EmpriseResourceVehicle, domain.EmpriseResourceRack,
}

// resourceRank — le rang d'une ressource dans resourceOrder.
func resourceRank(resource string) int {
	for i, r := range resourceOrder {
		if r == resource {
			return i
		}
	}
	return len(resourceOrder)
}

// tierResource — la ressource d'un niveau de socle ; "" pour les niveaux hors Emprise (base, non
// classé, bonus, aucune prise).
func tierResource(tier string) string {
	switch tier {
	case domain.PadTierPower:
		return domain.EmpriseResourcePowerWeapon
	case domain.PadTierGround:
		return domain.EmpriseResourceRack
	}
	return ""
}
