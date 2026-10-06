// Package service — session_page_objectif.go : LA FEUILLE D'OBJECTIF ET L'EMBLÈME DE LA PAGE
// SESSIONS (plan `.ai/PLAN_SESSIONS_EMPRISE_2026-10-06.md`, D7).
//
// Le bloc `formes_retenues` (réduit à l'objectif) de l'Escouade et des Séries temporelles, sur les
// matchs de chaque session, le joueur de la page SEUL dans l'escouade du bloc (V1) ; les prises
// nettes de drapeau y sont une colonne optionnelle du rôle « prendre ».
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/service/squadagg"
)

// attachSessionFormes pose `formes_retenues` (et la version comparée, tiroir ouvert).
func (s *SessionPageService) attachSessionFormes(
	ctx context.Context, resp *domain.SessionPageResponse, sc sessionBlocksScope,
	canon []canonical.PlayerMatchRow, lus lecturesDesSessions,
) {
	resp.FormesRetenues = s.sessionFormes(ctx, lignesCanoniquesDe(canon, sc.Matches), sc.Locale, lus.courant)
	if len(sc.CompareMatches) > 0 {
		resp.CompareFormesRetenues = s.sessionFormes(ctx, lignesCanoniquesDe(canon, sc.CompareMatches), sc.Locale, lus.compare)
	}
}

// sessionFormes rend le bloc d'une session, ou nil (aucun match).
func (s *SessionPageService) sessionFormes(
	ctx context.Context, rows []canonical.PlayerMatchRow, locale string, lu *squadagg.LecturesUsage,
) *domain.SquadFormesBlock {
	if len(rows) == 0 {
		return nil
	}
	defer timing.FromContext(ctx).Section("squad_formes")()
	return squadagg.BuildSquadFormesBlock(ctx, squadagg.SquadFormesQuery{
		Repo:         s.sessionUsageRepo,
		Objectives:   s.formesObjectives,
		PlayerXUID:   s.sessionXUID,
		MainGamertag: s.gamertag,
		Metas:        timeseriesFormesMetas(rows, locale),
		Lectures:     lu,
	})
}

// attachSessionEmblem pose l'emblème du joueur de la page (best-effort : rien sans chargeur).
func (s *SessionPageService) attachSessionEmblem(ctx context.Context, resp *domain.SessionPageResponse) {
	if s.emblemLoader == nil || s.gamertag == "" {
		slog.DebugContext(ctx, "sessions_emblem_sans_chargeur", "player", s.gamertag)
		return
	}
	resp.PlayerEmblemURL = s.emblemLoader.LoadEmblemURLs(ctx, s.titleSlug, []string{s.gamertag})[s.gamertag]
}
