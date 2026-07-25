// Package service — match_view_teams.go : libellés et couleurs d'équipe du scoreboard.
//
// applyTeamNames renseigne row.TeamName / row.TeamColor depuis les capabilities
// OPTIONNELLES teamNameResolver / teamColorResolver exposées par l'adapter d'assets du
// titre (Halo 5 : team_colors). Title-agnostic : aucune comparaison de slug, la seule
// présence de la capability décide ; HINF ne les implémente pas → no-op gracieux et le
// front garde ses libellés/accents d'équipe existants. Extrait de l'ancienne voie
// canonique (purgée avec le fallback LIVE) car l'appelant vivant reste la voie repo
// (match_view_data_loaders.go).
package service

import (
	"context"
	"strconv"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// teamNameResolver est une capability OPTIONNELLE d'un TitleAssetURLAdapter : résout un
// team_id en libellé d'équipe localisé (Halo 5 : « Rouge »/« Red » depuis team_colors).
// Seul l'adapter H5 l'implémente ; HINF ne l'implémente pas → team_name reste vide et
// le front retombe sur sa résolution existante (Eagle/Cobra). Pattern optionnel
// (type-assertion) : la capability n'élargit PAS l'interface games.TitleAssetURLAdapter.
type teamNameResolver interface {
	TeamName(teamID int, locale string) string
}

// teamColorResolver est la capability OPTIONNELLE jumelle de teamNameResolver : résout un
// team_id en couleur d'identité hex (#RRGGBB, Halo 5 : team_colors.color). Même pattern
// optionnel (type-assertion) : HINF ne l'implémente pas → team_color reste vide et le
// front retombe sur sa map de couleurs par team_id.
type teamColorResolver interface {
	TeamColor(teamID int) string
}

// applyTeamNames renseigne row.TeamName ET row.TeamColor sur chaque ligne de scoreboard
// quand l'adapter d'assets du titre expose teamNameResolver / teamColorResolver (Halo 5).
// No-op si l'adapter ne les implémente pas (HINF), est nil, ou renvoie "" (team_colors
// vide) → dégradation gracieuse (le front garde libellé et accent d'équipe existants).
// Title-agnostic : aucune comparaison de slug, la capability seule décide.
func (s *MatchViewService) applyTeamNames(ctx context.Context, rows []domain.MatchScoreboardRow) {
	nameResolver, hasName := s.assetURL.(teamNameResolver)
	colorResolver, hasColor := s.assetURL.(teamColorResolver)
	if !hasName && !hasColor {
		return
	}
	locale := ctxkeys.Locale(ctx)
	for i := range rows {
		id, ok := teamSideToID(rows[i].TeamSide)
		if !ok {
			continue
		}
		if hasName {
			if name := nameResolver.TeamName(id, locale); name != "" {
				rows[i].TeamName = name
			}
		}
		if hasColor {
			if color := colorResolver.TeamColor(id); color != "" {
				rows[i].TeamColor = color
			}
		}
	}
}

// teamSideToID parse le team_side DTO "t{N}" en son entier N. (0, false) si nil ou
// format inattendu (le backend émet toujours fmt.Sprintf("t%d", teamID)).
func teamSideToID(teamSide *string) (int, bool) {
	if teamSide == nil {
		return 0, false
	}
	s := *teamSide
	if len(s) < 2 || s[0] != 't' {
		return 0, false
	}
	n, err := strconv.Atoi(s[1:])
	if err != nil {
		return 0, false
	}
	return n, true
}
