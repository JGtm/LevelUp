package port

import (
	"context"

	"levelup/go-api/internal/analysis/squademprise"
)

// SquadEmpriseRepository — la feuille de match du bloc « Emprise » de l'Escouade : les frags aux
// armes spéciales de chaque participant (les deux camps), seule grandeur de l'onglet servie sans
// film (décision D10 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26). Les grandeurs du film
// passent par SessionUsageRepository, câblé sous `film.usage_summary`.
//
// Implémenté par internal/platform/duckdb.SquadEmpriseRepo, câblé pour tout titre : la colonne
// `match_participants.power_weapon_kills` est écrite par les deux titres.
type SquadEmpriseRepository interface {
	LoadPowerWeaponKills(ctx context.Context, matchIDs []string) ([]squademprise.PowerKillRow, error)
}

// SquadLifePlacementRepository — le placement des vies du bloc « Groupés ou isolés » de l'Emprise
// (plan `.ai/V7.5/PLAN_EMPRISE_VIES_2026-09-28.md`, lot V3) : UN chargement par requête, borné par les
// matchs du périmètre ET les xuids de la composition (ADR 0036), sur la vue
// `match_life_placement_latest` seulement.
//
// Implémenté par internal/platform/duckdb.SquadLifePlacementRepo, câblé sous
// `film.kill_positions` (la porte des vies au sync). Table absente :
// games.ErrCapabilityNotSupported.
type SquadLifePlacementRepository interface {
	LoadLifePlacement(ctx context.Context, matchIDs, xuids []string) (squademprise.PlacementRead, error)
}

// SquadVehicleRepository — la ressource « véhicules » du bloc Emprise (plan
// `.ai/V7.5/PLAN_EMPRISE_VEHICULES_2026-09-28.md`, lot L7.3) : UN chargement par requête, borné par les
// matchs demandés (périmètre ET matchs de l'habitude ; ADR 0036 I2), sur `match_vehicle_takes_latest`
// (prises, temps à bord, frags appariés, couverture) puis, pour les seuls matchs dont la passe compte
// des frags d'engin, sur `match_kill_events_latest` (frags de classe véhicule par camp du tueur, D5).
// `playerXUID` est le joueur de la page : son camp dans chaque match est lu de `match_participants`.
//
// Implémenté par internal/platform/duckdb.SquadVehicleRepo, câblé sous `film.vehicle_usage`. Table
// absente : games.ErrCapabilityNotSupported.
type SquadVehicleRepository interface {
	LoadVehicleUsage(ctx context.Context, matchIDs []string, playerXUID string) (squademprise.VehicleRead, error)
}
