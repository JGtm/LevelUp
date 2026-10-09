// Package service — session_page_tools.go : « OUTILS DE DESTRUCTION » DE LA PAGE SESSIONS (plan
// `.ai/V7.5/PLAN_SESSIONS_EMPRISE_2026-10-06.md`, D6) — chaque frag du joueur de la page nommé, par le
// builder de l'Escouade (squadagg.BuildWeaponTools) sur un seul joueur.
//
// Lignes d'arme : celles que la répartition des frags vient de lire. Feuille de match : les compteurs
// canoniques déjà agrégés. Catégories de source du film (objets explosifs, chute) : l'interface
// OPTIONNELLE port.KillSourceCategoryRepository du même lecteur d'armes — absente (titre sans film)
// ou non supportée, les deux lignes manquent et leurs frags restent dans le reliquat.
package service

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/squadagg"
)

// sessionWeaponTools rend les outils de la session, ou nil sans ligne.
func (s *SessionPageService) sessionWeaponTools(
	ctx context.Context, rows []port.WeaponKillRow, counts domain.FragKillTypeCounts, matchIDs []string, hasMechanics bool,
) *domain.SquadWeaponTools {
	gtByXUID := map[string]string{}
	if s.sessionXUID != "" {
		gtByXUID[s.sessionXUID] = s.gamertag
	}
	for _, r := range rows {
		// Les lignes sont filtrées sur le joueur de la page : leur xuid est le sien.
		if r.XUID != "" {
			gtByXUID[r.XUID] = s.gamertag
		}
	}
	tools := squadagg.BuildWeaponTools(squadagg.WeaponToolInputs{
		Rows:           rows,
		Categories:     s.loadSessionToolCategories(ctx, matchIDs),
		PlayersOrdered: []string{s.gamertag},
		GtByXUID:       gtByXUID,
		Sheet:          map[string]domain.FragKillTypeCounts{s.gamertag: counts},
		HasMechanics:   hasMechanics,
	})
	if tools != nil {
		slog.DebugContext(ctx, "session page: outils de destruction", "title", s.titleSlug,
			"lines", len(tools.Lines), "players_above_sheet", squadagg.PlayersAboveSheet(tools, map[string]domain.FragKillTypeCounts{s.gamertag: counts}))
	}
	return tools
}

// loadSessionToolCategories lit les frags par catégorie de source du film. nil sans l'interface,
// capability non supportée (Debug) ou lecture en échec (Warn) — jamais une erreur de page.
func (s *SessionPageService) loadSessionToolCategories(ctx context.Context, matchIDs []string) []port.KillSourceCategoryRow {
	loader, ok := s.weaponKillsRepo.(port.KillSourceCategoryRepository)
	if !ok || s.gamertag == "" {
		return nil
	}
	rows, err := loader.LoadKillSourceCategoryKills(ctx, s.titleSlug, port.WeaponKillFilters{MatchIDs: matchIDs, Gamertag: s.gamertag})
	switch {
	case errors.Is(err, games.ErrCapabilityNotSupported):
		slog.DebugContext(ctx, "session page: catégories de source non supportées", "title", s.titleSlug)
		return nil
	case err != nil:
		slog.WarnContext(ctx, "session page: catégories de source en échec",
			"title", s.titleSlug, "match_count", len(matchIDs), "err", err)
		return nil
	}
	return rows
}
