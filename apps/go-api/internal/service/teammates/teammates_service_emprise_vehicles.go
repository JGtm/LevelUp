// Package teammates — teammates_service_emprise_vehicles.go : LA RESSOURCE « VEHICULES » du bloc
// Emprise (plan `.ai/PLAN_EMPRISE_VEHICULES_2026-09-28.md`, lot L7.3 ; calcul :
// analysis/squademprise/vehicles.go, type publié : domain/squad_emprise_vehicles.go).
//
// Orchestration seule : UNE lecture (port.SquadVehicleRepository, ADR 0036), bornée par les matchs
// du périmètre ET ceux de l'habitude, sous sa propre section de durée. Indépendante du film
// (`film.usage_summary`) : la ressource vient de l'artefact de rejeu, dérivée au sync sous la
// capability `film.vehicle_usage`.
//
// Capability absente (le lecteur n'est pas câblé, ou la table manque) : ressource absente,
// games.ErrCapabilityNotSupported journalisé en Debug. Lecture en échec : ressource absente ET le
// bloc le dit (EmpriseVehiclesLoadFailed), jamais un zéro.
package teammates

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/analysis/squademprise"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/squadagg"
)

// WithVehicleUsage injecte le lecteur de la ressource véhicules. Câblé sous `film.vehicle_usage` ;
// sans lui, la ressource est absente du bloc Emprise.
func (s *TeammatesService) WithVehicleUsage(repo port.SquadVehicleRepository) *TeammatesService {
	s.vehicleRepo = repo
	return s
}

// lireVehiculesEmprise charge la ressource véhicules du périmètre et des matchs de l'habitude, et
// les noms des familles qualifiées. Rend (nil, "") quand le titre ne la mesure pas.
func (s *TeammatesService) lireVehiculesEmprise(
	ctx context.Context, playerXUID string, current, timeline []squademprise.Match, locale string,
) (*squademprise.VehicleRead, string, map[string]string) {
	if s.vehicleRepo == nil {
		slog.DebugContext(ctx, "teammates_emprise_vehicules_capability_absente",
			"player", s.gamertag, "capability", string(games.CapFilmVehicleUsage),
			"err", games.ErrCapabilityNotSupported)
		return nil, "", nil
	}
	defer timing.FromContext(ctx).Section("emprise_vehicules")()
	ids := append(matchIDsOf(current), squademprise.HabitCandidates(current, timeline)...)
	read, err := s.vehicleRepo.LoadVehicleUsage(ctx, ids, playerXUID)
	switch {
	case errors.Is(err, games.ErrCapabilityNotSupported):
		slog.DebugContext(ctx, "teammates_emprise_vehicules_capability_absente",
			"player", s.gamertag, "capability", string(games.CapFilmVehicleUsage), "err", err)
		return nil, "", nil
	case err != nil:
		slog.ErrorContext(ctx, "teammates_emprise_vehicules_en_echec",
			"player", s.gamertag, "matchs", len(ids), "err", err)
		return nil, domain.EmpriseVehiclesLoadFailed, nil
	}
	slog.DebugContext(ctx, "teammates_emprise_vehicules",
		"player", s.gamertag, "matchs", len(ids), "passes", len(read.Passes), "prises", len(read.Rows),
		"matchs_avec_frags", len(read.EventsRead))
	return &read, "", squadagg.VehicleFamilyLabels(ctx, s.repoRoot, s.titleSlug, locale)
}
