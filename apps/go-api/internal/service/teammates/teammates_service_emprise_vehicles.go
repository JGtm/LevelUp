// Package teammates — teammates_service_emprise_vehicles.go : LA RESSOURCE « VEHICULES » du bloc
// Emprise (plan `.ai/V7.5/PLAN_EMPRISE_VEHICULES_2026-09-28.md`, lot L7.3 ; calcul :
// analysis/squademprise/vehicles.go, type publié : domain/squad_emprise_vehicles.go).
//
// La lecture est `squadagg.EmpriseLecteur.Vehicules` (partagée avec les Séries temporelles) : UNE
// lecture (port.SquadVehicleRepository, ADR 0036), bornée par les matchs du périmètre ET ceux de
// l'habitude, sous sa propre section de durée. Indépendante du film (`film.usage_summary`) : la
// ressource vient de l'artefact de rejeu, dérivée au sync sous la capability `film.vehicle_usage`.
package teammates

import "levelup/go-api/internal/port"

// WithVehicleUsage injecte le lecteur de la ressource véhicules. Câblé sous `film.vehicle_usage` ;
// sans lui, la ressource est absente du bloc Emprise.
func (s *TeammatesService) WithVehicleUsage(repo port.SquadVehicleRepository) *TeammatesService {
	s.vehicleRepo = repo
	return s
}
