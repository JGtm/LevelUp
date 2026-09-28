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
