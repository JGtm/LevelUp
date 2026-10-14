package domain

// squad_emprise_vehicles.go — LA RESSOURCE « VEHICULES » DU BLOC EMPRISE (plan
// `.ai/V7.5/PLAN_EMPRISE_VEHICULES_2026-09-28.md`, lot L7.3, décisions D2 à D10).
//
// Elle entre dans les listes existantes du bloc (une ressource = une entrée de liste, masquée si
// absente) : `Resources` (prises par camp), `Objects` et `Matches[].Resources[].Objects` (un objet
// par FAMILLE de véhicule, clé = la famille publiée par le calque du rejeu : `warthog`, `ghost`,
// `unknown`…), `Production` (frags depuis un véhicule, temps à bord, rendement par minute à bord),
// `Habit` (notre part des prises). Ce fichier ne porte que ce qui n'a pas de place dans ces
// listes : l'état d'un match.
//
// # NON LU ET ZÉRO SE DISTINGUENT (D8)
//
// Un match dont l'artefact n'a pas d'occupation lue (schéma < 67, calque non balayé, pas d'image
// absent), que la dérivation n'a pas encore traité ou dont notre camp est inconnu est
// `not_measured` : il n'entre dans AUCUN compte de la ressource et sa case de grille reste vide,
// sans texte. Un match `measured` sans prise est un zéro : la ressource n'a pas d'entrée dans sa
// liste, l'état le dit.
//
// # LE RENDEMENT SANS BIAIS (D9)
//
// « Frags obtenus avec les ressources » garde tous les frags de classe véhicule ou tourelle
// (`IsEngineFragClass`, la définition de la Répartition des frags) ; « Rendement » ne compte au
// numérateur que ceux tombés PENDANT un épisode publié de leur tueur (appariés à la dérivation) :
// diviser tous les frags par le seul temps à bord publié gonflerait le rendement. Les deux se
// lisent sur les MÊMES matchs (ceux dont la dérivation a apparié les frags).

// EmpriseResourceVehicle : la ressource véhicules (prises de véhicule, tourelles fixes comprises).
const EmpriseResourceVehicle = "vehicle"

// EmpriseExposureAboardMS : temps à bord des véhicules, en millisecondes (SquadEmpriseExposure.Kind).
const EmpriseExposureAboardMS = "aboard_ms"

// États des véhicules d'un match (SquadEmpriseMatch.Vehicles).
const (
	// EmpriseVehiclesMeasured : l'occupation est lue et notre camp connu ; zéro prise = zéro mesuré.
	EmpriseVehiclesMeasured = "measured"
	// EmpriseVehiclesNotMeasured : rien de la ressource ne se lit sur ce match.
	EmpriseVehiclesNotMeasured = "not_measured"
)
