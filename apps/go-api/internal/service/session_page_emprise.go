// Package service — session_page_emprise.go : L'EMPRISE ET « MES VIES » DE LA PAGE SESSIONS (plan
// `.ai/V7.5/PLAN_SESSIONS_EMPRISE_2026-10-06.md`, D3, D4, D5).
//
// Le joueur de la page SEUL, en contexte escouade comme en solo (décision utilisateur V1) : aucun
// coéquipier suivi n'a de fiche. Orchestration seule : l'assemblage est `buildSoloEmpriseBlock`
// (sans grille par carte) et la lecture des vies `lireViesPresOuSeul`, partagés avec les Séries
// temporelles ; la carte et le résultat de chaque match viennent des lignes canoniques déjà chargées.
package service

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/service/squadagg"
)

// attachSessionEmprise pose `emprise` (et `compare_emprise`, tiroir ouvert).
func (s *SessionPageService) attachSessionEmprise(
	ctx context.Context, resp *domain.SessionPageResponse, sc sessionBlocksScope,
	canon []canonical.PlayerMatchRow, lus lecturesDesSessions,
) {
	resp.Emprise = s.sessionEmprise(ctx, lignesCanoniquesDe(canon, sc.Matches), sc.Locale, lus.courant)
	if len(sc.CompareMatches) > 0 {
		resp.CompareEmprise = s.sessionEmprise(ctx, lignesCanoniquesDe(canon, sc.CompareMatches), sc.Locale, lus.compare)
	}
}

// sessionEmprise rend le bloc d'une session, ou nil (aucun match, joueur inconnu).
func (s *SessionPageService) sessionEmprise(
	ctx context.Context, rows []canonical.PlayerMatchRow, locale string, lu *squadagg.LecturesUsage,
) *domain.SoloEmpriseBlock {
	defer timing.FromContext(ctx).Section("emprise")()
	if len(rows) == 0 || s.sessionXUID == "" {
		return nil
	}
	return buildSoloEmpriseBlock(ctx, soloEmpriseQuery{
		Page: pageSessions, Player: s.gamertag, PlayerXUID: s.sessionXUID,
		RepoRoot: s.repoRoot, TitleSlug: s.titleSlug, Locale: locale,
		Current: timeseriesEmpriseMatches(rows, locale), Lectures: lu,
		UsageRepo: s.sessionUsageRepo, EmpriseRepo: s.empriseRepo, VehicleRepo: s.vehicleRepo,
	})
}

// attachSessionLives pose `lives_near_teammate` (et la version comparée) : une lecture bornée par
// session, sur ses seuls matchs et le joueur de la page.
func (s *SessionPageService) attachSessionLives(ctx context.Context, resp *domain.SessionPageResponse, sc sessionBlocksScope) {
	resp.LivesNearTeammate = s.sessionLives(ctx, sc.Matches)
	if len(sc.CompareMatches) > 0 {
		resp.CompareLivesNearTeammate = s.sessionLives(ctx, sc.CompareMatches)
	}
}

func (s *SessionPageService) sessionLives(ctx context.Context, matches []legacymatch.StatsMatchRow) *domain.TimeseriesLivesNearTeammate {
	if len(matches) == 0 || s.sessionXUID == "" {
		return nil
	}
	return lireViesPresOuSeul(ctx, viesQuery{
		Page: pageSessions, Player: s.gamertag, PlayerXUID: s.sessionXUID,
		Repo: s.livesRepo, Radar: s.radarRange, MatchIDs: matchIDsFromStatsRows(matches),
	})
}
