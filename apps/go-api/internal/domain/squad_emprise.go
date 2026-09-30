package domain

// squad_emprise.go — LE BLOC « EMPRISE » de /pages/teammates (lot L4 du plan
// PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26 ; maquette de l'onglet, copie
// `.ai/V7.5/MAQUETTE_ONGLET_TACTIQUE_ESCOUADE_2026-09-26.html`).
//
// Il répond, sur le périmètre D2 de la page (composition exacte ∩ filtres), à « qui a tenu la
// carte » : les prises de chaque RESSOURCE par notre camp et par l'adversaire, match par match
// et objet par objet, qui chez nous, ce que chaque camp en a produit, et d'habitude.
//
// # LES RESSOURCES SONT UNE LISTE
//
// `powerup` (bonus : camouflage + surbouclier, prises ATTRIBUÉES à un joueur, décision D4),
// `power_weapon` (armes spéciales : prises sur les socles de niveau `puissance`, D3), `rack`
// (armes de râtelier : socles de niveau `terrain`, grille match par match seulement). Une
// ressource sans donnée est ABSENTE, jamais à zéro ; `vehicle` (lot L7.3, squad_emprise_vehicles.go)
// est entrée dans ces listes comme une entrée de plus.
//
// # RÉSULTAT, SCORE ET DOMINANCE NE SONT PAS ICI
//
// Ils se joignent côté web depuis `match_history` par `match_id` (lot L3.2) : une seconde copie
// dans ce bloc divergerait de la première.

// Clés des ressources de l'Emprise.
const (
	EmpriseResourcePowerup     = "powerup"
	EmpriseResourcePowerWeapon = "power_weapon"
	EmpriseResourceRack        = "rack"
)

// États des niveaux de socle d'un match (SquadEmpriseMatch.Tiers).
const (
	// EmpriseTiersMeasured : la passe des niveaux a eu lieu et la carte est dans la référence.
	EmpriseTiersMeasured = "measured"
	// EmpriseTiersUnestablished : des socles, mais la carte n'est pas dans la référence — tout
	// est « non classé », ni arme spéciale ni râtelier ne se lit.
	EmpriseTiersUnestablished = "unestablished"
	// EmpriseTiersNotMeasured : aucune passe des niveaux pour ce match (projection absente).
	EmpriseTiersNotMeasured = "not_measured"
)

// Natures d'exposition d'une ressource (SquadEmpriseExposure.Kind).
const (
	// EmpriseExposureEffectMS : temps d'effet des bonus, en millisecondes.
	EmpriseExposureEffectMS = "effect_ms"
	// EmpriseExposurePickups : prises (armes spéciales).
	EmpriseExposurePickups = "pickups"
)

// EmpriseFilmUnsupported : le titre ne publie pas de résumé d'usage du film ; seules les
// grandeurs de la feuille de match sont servies (frags aux armes spéciales, D10).
const EmpriseFilmUnsupported = "film_unsupported"

// EmpriseFilmLoadFailed : la lecture du résumé d'usage a échoué ; même dégradation.
const EmpriseFilmLoadFailed = "film_load_failed"

// EmpriseSheetUnsupported / EmpriseSheetLoadFailed : la feuille de match n'est pas servie pour
// ce titre, ou sa lecture a échoué — les frags aux armes spéciales sont alors absents.
const (
	EmpriseSheetUnsupported = "sheet_unsupported"
	EmpriseSheetLoadFailed  = "sheet_load_failed"
)

// SquadEmpriseBlock — le bloc publié. Nil quand le périmètre n'a aucun match.
type SquadEmpriseBlock struct {
	// MatchesTotal : les matchs du périmètre D2.
	MatchesTotal int `json:"matches_total"`
	// MatchesMeasured : parmi eux, ceux dont le film a été résumé ET notre camp connu : un match
	// filmé au camp inconnu n'apporte aucun compte, il n'est pas mesuré.
	MatchesMeasured int `json:"matches_measured"`
	// FilmUnavailable : raison machine quand AUCUNE grandeur du film n'est servie
	// (EmpriseFilmUnsupported, EmpriseFilmLoadFailed). Vide sinon.
	FilmUnavailable string `json:"film_unavailable,omitempty"`
	// SheetUnavailable : raison machine quand la feuille de match (frags aux armes spéciales) n'a
	// pas pu être lue (EmpriseSheetUnsupported, EmpriseSheetLoadFailed). Vide sinon.
	SheetUnavailable string `json:"sheet_unavailable,omitempty"`
	// Players : le joueur de la page puis les coéquipiers sélectionnés — l'ordre des fiches et
	// des entrées `squad` de chaque objet. Le reste du camp n'est pas une entrée.
	Players []SessionUsageSquadPlayer `json:"players"`
	// Resources : le bilan de la soirée, camp contre camp (powerup, power_weapon).
	Resources []SquadEmpriseResource `json:"resources"`
	// Objects : la soirée objet par objet (fiches « Répartition des prises dans l'escouade »),
	// du plus pris par notre camp au moins pris.
	Objects []SquadEmpriseObject `json:"objects"`
	// Matches : le périmètre match par match, dans l'ordre chronologique.
	Matches []SquadEmpriseMatch `json:"matches"`
	// Production : frags obtenus avec chaque ressource, exposition et rendement, par camp.
	Production []SquadEmpriseProduction `json:"production"`
	// Habit : les soirées précédentes comparables (D5). Nil sans coéquipier sélectionné ou
	// sans film.
	Habit *SquadEmpriseHabit `json:"habit,omitempty"`
	// Placement : « Groupés ou isolés », le placement et le rendement de chaque vie de la
	// composition (squad_emprise_placement.go). Nil sans `film.kill_positions`, lecture en échec
	// ou aucune vie écrite pour la composition sur le périmètre.
	Placement *SquadEmprisePlacement `json:"placement,omitempty"`
	// Vehicles : la couverture de la ressource véhicules (squad_emprise_vehicles.go). Nil quand
	// le titre ne la mesure pas (capability `film.vehicle_usage`), jamais un zéro.
	Vehicles *SquadEmpriseVehicles `json:"vehicles,omitempty"`
}

// SquadEmpriseCount — un compte camp contre camp.
type SquadEmpriseCount struct {
	Us   int `json:"us"`
	Them int `json:"them"`
}

// SquadEmpriseResource — une ressource du bilan.
type SquadEmpriseResource struct {
	Resource string `json:"resource"`
	// Taken : prises attribuées de chaque camp, sur les matchs qui mesurent la ressource.
	Taken SquadEmpriseCount `json:"taken"`
	// MatchesMeasured : les matchs dont ces prises sont lues (film ; niveaux de socle pour les
	// armes spéciales).
	MatchesMeasured int `json:"matches_measured"`
	// Outcomes : les quatre issues de chaque camp — bonus seulement (« Bonus perdus »).
	Outcomes *SquadEmpriseCampOutcomes `json:"outcomes,omitempty"`
}

// SquadEmpriseCampOutcomes — pris / utilisé / gardé / lâché, par camp (D11).
type SquadEmpriseCampOutcomes struct {
	Us   SquadEmpriseOutcomeCounts `json:"us"`
	Them SquadEmpriseOutcomeCounts `json:"them"`
}

// SquadEmpriseOutcomeCounts — les quatre comptes d'issue ; perdus = gardé + lâché.
type SquadEmpriseOutcomeCounts struct {
	Taken   int `json:"taken"`
	Used    int `json:"used"`
	Kept    int `json:"kept"`
	Dropped int `json:"dropped"`
}

// SquadEmpriseObject — un objet (un bonus, une arme de socle), sur la soirée ou sur un match.
type SquadEmpriseObject struct {
	Resource string `json:"resource"`
	// Key : famille du bonus (`powerup_camo`, `powerup_overshield`) ou clé de famille d'arme
	// (huit hexadécimaux). Le web nomme les bonus ; les armes portent leur nom du titre.
	Key string `json:"key"`
	// WeaponKey / Label : clé canonique du registre d'armes et nom du titre dans la langue de
	// la requête. Vides pour un bonus ou une arme hors catalogue.
	WeaponKey string            `json:"weapon_key,omitempty"`
	Label     string            `json:"label,omitempty"`
	Taken     SquadEmpriseCount `json:"taken"`
	// PadsEmptied : socles de ce bonus vidés, avec ou sans ramasseur nommé (les prises non
	// attribuées ne comptent dans aucun camp, D4). Bonus seulement.
	PadsEmptied *int `json:"pads_emptied,omitempty"`
	// Aboard : temps à bord de chaque camp sur cette famille, en ms (véhicules seulement, D4).
	Aboard *SquadEmpriseCount `json:"aboard_ms,omitempty"`
	// Squad : les prises de notre camp, une entrée par joueur de Players dans l'ordre, puis le
	// reste du camp (XUID vide).
	Squad []SquadEmpriseObjectShare `json:"squad"`
}

// SquadEmpriseObjectShare — ce qu'un joueur de notre camp (ou le reste du camp) a pris d'un
// objet ; pour un bonus, combien il en a perdu.
type SquadEmpriseObjectShare struct {
	// XUID : vide pour le reste du camp.
	XUID  string `json:"xuid,omitempty"`
	Taken int    `json:"taken"`
	// Kept / Dropped : bonus gardés sans être activés / lâchés en mourant. Bonus seulement.
	Kept    *int `json:"kept,omitempty"`
	Dropped *int `json:"dropped,omitempty"`
	// AboardMS : son temps à bord, en ms (véhicules seulement, D4).
	AboardMS *int64 `json:"aboard_ms,omitempty"`
}

// SquadEmpriseMatch — un match du périmètre.
type SquadEmpriseMatch struct {
	MatchID string `json:"match_id"`
	// HasFilm : le résumé d'usage du film existe (sinon la case dit « sans film »).
	HasFilm bool `json:"has_film"`
	// TeamKnown : le camp du joueur de la page est connu — sans lui, aucun compte camp contre
	// camp ne se publie pour ce match.
	TeamKnown bool `json:"team_known"`
	// Tiers : état des niveaux de socle (EmpriseTiers*), qui séparent armes spéciales et
	// râteliers. Vide sans film.
	Tiers string `json:"tiers,omitempty"`
	// Resources : ressource puis chaque objet du match. Une ressource absente de la carte (ou
	// non mesurée) n'a pas d'entrée.
	Resources []SquadEmpriseMatchResource `json:"resources"`
	// PowerWeaponKills : frags aux armes spéciales de chaque camp (feuille de match). Nil quand
	// la feuille ne le dit pas ou que le camp est inconnu.
	PowerWeaponKills *SquadEmpriseCount `json:"power_weapon_kills,omitempty"`
	// Vehicles : l'état des véhicules du match (EmpriseVehicles*), qui sépare « non mesuré » (D8)
	// d'un zéro mesuré ; vide quand le titre ne mesure pas la ressource. VehiclesReason : la raison
	// machine d'un match non mesuré.
	Vehicles       string `json:"vehicles,omitempty"`
	VehiclesReason string `json:"vehicles_reason,omitempty"`
}

// SquadEmpriseMatchResource — une ressource d'un match et ses objets.
type SquadEmpriseMatchResource struct {
	Resource string               `json:"resource"`
	Taken    SquadEmpriseCount    `json:"taken"`
	Objects  []SquadEmpriseObject `json:"objects"`
}

// SquadEmpriseProduction — ce que chaque camp a produit d'une ressource.
type SquadEmpriseProduction struct {
	Resource string `json:"resource"`
	// Kills : bonus = frags pendant l'effet (film, matchs dont le film a une échelle de temps) ;
	// armes spéciales = frags obtenus avec (feuille de match, tous les matchs à camp connu).
	Kills SquadEmpriseCount `json:"kills"`
	// Exposure : ce qui a permis ces frags. Nil sans mesure du film.
	Exposure *SquadEmpriseExposure `json:"exposure,omitempty"`
	// YieldUs / YieldThem : frags par minute d'effet (bonus), frags par prise (armes spéciales),
	// calculés sur Exposure.Kills (véhicules : Exposure.PairedKills). Nil quand l'exposition du camp est nulle.
	YieldUs   *float64 `json:"yield_us,omitempty"`
	YieldThem *float64 `json:"yield_them,omitempty"`
	// RelativeGap : notre rendement / le sien − 1 (0 = autant que l'adversaire).
	RelativeGap *float64 `json:"relative_gap,omitempty"`
}

// SquadEmpriseExposure — l'exposition d'une ressource, et les frags sur le MÊME périmètre.
type SquadEmpriseExposure struct {
	Kind  string            `json:"kind"`
	Value SquadEmpriseCount `json:"value"`
	// Kills : les frags des matchs où l'exposition est mesurée. Pour les armes spéciales, c'est
	// Production.Kills privé des matchs sans niveaux de socle : un rendement se lit sur un seul
	// périmètre. Pour les véhicules, ce sont tous les frags de classe véhicule des matchs du
	// rendement (D5) ; Value est lu sur les mêmes matchs.
	Kills SquadEmpriseCount `json:"kills"`
	// PairedKills : les frags APPARIÉS aux épisodes de leur tueur, le numérateur du rendement quand
	// il diffère de Kills (véhicules, D9). Nil ailleurs : le rendement lit alors Kills.
	PairedKills *SquadEmpriseCount `json:"paired_kills,omitempty"`
}

// SquadEmpriseHabit — « par rapport à d'habitude » (D5).
type SquadEmpriseHabit struct {
	// Families : les familles de mode jouées dans le périmètre (libellé de mode normalisé, le
	// même que `match_history[].mode_ui`). Une soirée précédente n'est lue que sur celles-ci.
	Families []string `json:"families"`
	// Current : la soirée affichée (le périmètre).
	Current SquadEmpriseEvening `json:"current"`
	// Previous : jusqu'à SquadEmpriseHabitMaxPrevious soirées précédentes de la composition,
	// de la plus ancienne à la plus récente, chacune avec au moins un match comparable filmé.
	Previous []SquadEmpriseEvening `json:"previous"`
}

// SquadEmpriseHabitMaxPrevious — les dix soirées précédentes (D5).
const SquadEmpriseHabitMaxPrevious = 10

// SquadEmpriseEvening — une soirée.
type SquadEmpriseEvening struct {
	SessionLabel string `json:"session_label"`
	// StartTime : premier match de la soirée (UTC, RFC 3339).
	StartTime string `json:"start_time"`
	// MatchCount : matchs de la soirée pour la composition, lus dans composition_sessions (ADR
	// 0033). 0 = soirée sans entrée (périmètre à cheval sur plusieurs sessions).
	MatchCount int `json:"match_count"`
	// MeasuredMatches : matchs comparables (familles de ce soir) dont le film est résumé.
	MeasuredMatches int `json:"measured_matches"`
	// Shares : notre part des prises, par ressource ; une ressource sans prise est absente.
	Shares []SquadEmpriseShare `json:"shares"`
}

// SquadEmpriseShare — notre part des prises d'une ressource sur une soirée.
type SquadEmpriseShare struct {
	Resource string            `json:"resource"`
	Taken    SquadEmpriseCount `json:"taken"`
	// Share : Taken.Us / (Taken.Us + Taken.Them), unité 0..1 (ADR 0006).
	Share float64 `json:"share"`
}
