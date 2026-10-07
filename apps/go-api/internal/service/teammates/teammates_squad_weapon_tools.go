// Package teammates — teammates_squad_weapon_tools.go : « Outils de destruction » de
// l'Escouade, chaque frag nommé (décision D8 du plan
// .ai/PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26.md).
//
// Les LECTURES et leurs traces vivent ici (feuille de match par joueur, catégories de source du
// film) ; le builder pur est `squadagg.BuildWeaponTools`, partagé avec la page Sessions.
package teammates

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/squadagg"
)

// playerFragCounts : compteurs de la feuille de match d'un joueur sur les matchs partagés,
// mécaniques natives comprises quand le chargeur les a rendues.
func playerFragCounts(
	pts []domain.SquadPerformanceSeriesPoint, mech port.KillMechanicsRow, hasMech bool,
) domain.FragKillTypeCounts {
	counts := aggregateFragCounts(pts)
	if hasMech {
		counts.Assassination = mech.Assassinations
		counts.GroundPound = mech.GroundPound
		counts.ShoulderBash = mech.ShoulderBash
	}
	return counts
}

// loadSquadToolCategories charge les frags par catégorie de source du film. nil (sans
// erreur) si le chargeur ne sait pas les lire ou si le titre n'a pas de film : les deux
// lignes manquent alors, leurs frags restent dans le reliquat. Toute autre erreur est
// journalisée avant la dégradation.
func (s *TeammatesService) loadSquadToolCategories(
	ctx context.Context, sharedMatches, xuids []string,
) []port.KillSourceCategoryRow {
	loader, ok := s.squadLoader.(squadagg.SquadKillSourceCategoryLoader)
	if !ok {
		return nil
	}
	rows, err := loader.LoadKillSourceCategories(ctx, s.titleSlug, port.WeaponKillFilters{
		MatchIDs: sharedMatches,
		XUIDs:    xuids,
	})
	if err != nil {
		if errors.Is(err, games.ErrCapabilityNotSupported) {
			slog.DebugContext(ctx, "teammates_weapon_tools_categories_unsupported", "title", s.titleSlug)
			return nil
		}
		slog.WarnContext(ctx, "teammates_weapon_tools_categories_load_failed",
			"title", s.titleSlug, "matches", len(sharedMatches), "err", err)
		return nil
	}
	return rows
}

// squadSheet : la feuille de match de chaque joueur sur les matchs partagés.
func squadSheet(
	players []string,
	perf map[string][]domain.SquadPerformanceSeriesPoint,
	mechByGT map[string]port.KillMechanicsRow,
) map[string]domain.FragKillTypeCounts {
	sheet := make(map[string]domain.FragKillTypeCounts, len(players))
	for _, gt := range players {
		m, ok := mechByGT[gt]
		sheet[gt] = playerFragCounts(perf[gt], m, ok)
	}
	return sheet
}

// buildSquadToolsSection charge les catégories de source du film, assemble les lignes
// D8 et trace le résultat. Un joueur dont les lignes dépassent sa feuille de match (le
// film compte un frag que la feuille ignore) n'a pas de « Non attribué » : l'écart est
// tracé, pas avalé.
func (s *TeammatesService) buildSquadToolsSection(
	ctx context.Context,
	sc squadScope,
	rows []port.WeaponKillRow,
	sheet map[string]domain.FragKillTypeCounts,
	hasMechanics bool,
) *domain.SquadWeaponTools {
	tools := squadagg.BuildWeaponTools(squadagg.WeaponToolInputs{
		Rows:           rows,
		Categories:     s.loadSquadToolCategories(ctx, sc.sharedMatches, sc.xuids),
		PlayersOrdered: sc.playersOrdered,
		GtByXUID:       sc.gtByXUID,
		Sheet:          sheet,
		HasMechanics:   hasMechanics,
	})
	lineCount, overSheet := 0, 0
	if tools != nil {
		lineCount = len(tools.Lines)
		overSheet = squadagg.PlayersAboveSheet(tools, sheet)
	}
	slog.DebugContext(ctx, "teammates_weapon_tools_built", "title", s.titleSlug,
		"lines", lineCount, "players_above_sheet", overSheet)
	return tools
}
