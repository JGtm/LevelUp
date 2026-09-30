package domain

// squad_emprise_vehicles.go — LA RESSOURCE « VEHICULES » DU BLOC EMPRISE (plan
// `.ai/PLAN_EMPRISE_VEHICULES_2026-09-28.md`, lot L7.3, décisions D2 à D10).
//
// Elle entre dans les listes existantes du bloc (une ressource = une entrée de liste, masquée si
// absente) : `Resources` (prises par camp), `Objects` et `Matches[].Resources[].Objects` (un objet
// par FAMILLE de véhicule, clé = la famille publiée par le calque du rejeu : `warthog`, `ghost`,
// `unknown`…), `Production` (frags depuis un véhicule, temps à bord, rendement par minute à bord),
// `Habit` (notre part des prises). Ce fichier ne porte que ce qui n'a pas de place dans ces
// listes : la couverture, l'état d'un match, les raisons.
//
// # NON MESURÉ ET ZÉRO MESURÉ SE DISTINGUENT (D8)
//
// Un match dont l'artefact n'a pas d'occupation lue (schéma < 67, calque non balayé, pas d'image
// absent) ou que la dérivation n'a pas encore traité est `not_measured` : il n'entre dans AUCUN
// compte de la ressource et la case de la grille le dit. Un match `measured` sans prise est un zéro
// MESURÉ : la ressource n'a pas d'entrée dans sa liste, l'état le dit.
//
// # LE RENDEMENT SANS BIAIS (D9)
//
// « Frags obtenus avec les ressources » garde tous les frags de classe véhicule ou tourelle
// (`IsEngineFragClass`, la définition de la Répartition des frags) ; « Rendement » ne compte au
// numérateur que ceux tombés PENDANT un épisode publié de leur tueur (appariés à la dérivation) :
// diviser tous les frags par le seul temps à bord publié gonflerait le rendement. Les deux se
// lisent sur les MÊMES matchs (ceux dont la dérivation a apparié les frags) ; la part appariée
// est publiée en couverture.

// EmpriseResourceVehicle : la ressource véhicules (prises de véhicule, tourelles fixes comprises).
const EmpriseResourceVehicle = "vehicle"

// EmpriseExposureAboardMS : temps à bord des véhicules, en millisecondes (SquadEmpriseExposure.Kind).
const EmpriseExposureAboardMS = "aboard_ms"

// États des véhicules d'un match (SquadEmpriseMatch.Vehicles).
const (
	// EmpriseVehiclesMeasured : l'occupation est lue et notre camp connu ; zéro prise = zéro mesuré.
	EmpriseVehiclesMeasured = "measured"
	// EmpriseVehiclesNotMeasured : rien de la ressource ne se lit sur ce match
	// (SquadEmpriseMatch.VehiclesReason dit pourquoi).
	EmpriseVehiclesNotMeasured = "not_measured"
)

// Raisons d'un match `not_measured` propres au bloc (celles de l'artefact — schéma < 67, calque non
// balayé, pas d'image absent — traversent telles que la dérivation les a écrites).
const (
	// EmpriseVehiclesNoPass : la dérivation n'a écrit aucune passe pour ce match.
	EmpriseVehiclesNoPass = "no_pass"
	// EmpriseVehiclesTeamUnknown : le camp du joueur de la page est inconnu sur ce match.
	EmpriseVehiclesTeamUnknown = "team_unknown"
)

// EmpriseVehiclesLoadFailed : la lecture des véhicules a échoué ; la ressource est absente et le
// bloc le dit (SquadEmpriseVehicles.Unavailable).
const EmpriseVehiclesLoadFailed = "load_failed"

// SquadEmpriseVehicles — la couverture de la ressource sur le périmètre affiché (nil quand le titre
// ne la mesure pas : capability `film.vehicle_usage` absente ou table absente).
type SquadEmpriseVehicles struct {
	// Unavailable : EmpriseVehiclesLoadFailed quand la lecture a échoué. Vide sinon.
	Unavailable string `json:"unavailable,omitempty"`
	// MatchesMeasured / MatchesNotMeasured : les matchs du périmètre dont les véhicules se lisent
	// (D8), et les autres.
	MatchesMeasured    int `json:"matches_measured"`
	MatchesNotMeasured int `json:"matches_not_measured"`
	// EpisodesRead : épisodes d'occupation lus sur les matchs mesurés ; EpisodesUnnamed : sans
	// joueur nommé (un bot), ni prise ni temps (D10) ; EpisodesNoCamp : sans camp ;
	// ProximityEpisodes : datés par proximité plutôt que par le film (parmi ceux qui comptent).
	EpisodesRead      int `json:"episodes_read"`
	EpisodesUnnamed   int `json:"episodes_unnamed"`
	EpisodesNoCamp    int `json:"episodes_no_camp"`
	ProximityEpisodes int `json:"proximity_episodes"`
	// FragsMatches : les matchs du périmètre dont les frags de classe véhicule sont appariés aux
	// épisodes — le périmètre commun des frags, du temps à bord et du rendement (D9).
	FragsMatches int `json:"frags_matches"`
	// FragsTotal : les frags de classe véhicule de ces matchs, ceux qu'un tueur nommé a obtenus ;
	// FragsPaired : parmi eux, ceux tombés pendant un épisode publié de leur tueur (le numérateur
	// du rendement) ; PairedShare : FragsPaired / FragsTotal, unité 0..1 (ADR 0006), nil sans frag.
	FragsTotal  int      `json:"frags_total"`
	FragsPaired int      `json:"frags_paired"`
	PairedShare *float64 `json:"paired_share,omitempty"`
}
