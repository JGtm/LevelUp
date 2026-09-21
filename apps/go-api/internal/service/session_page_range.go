// Package service — session_page_range.go : le bloc « portée des engagements » de la page
// détail de session (plan .ai/V7.5/PLAN_AJUSTEMENTS_SUPP_PRE_V75_2026-09-21.md, lot N2,
// décision D22-4).
//
// PATRON D'ATTACHEMENT identique au bloc usage : repo OPTIONNEL injecté à la DI, attaché à
// la réponse existante de POST .../pages/sessions/detail — pas d'endpoint dédié, pas de clé
// de cache de plus.
//
// # CE QUI EST PUBLIÉ, ET CE QUI NE L'EST PAS
//
// Un profil par match de la session : la médiane du LOBBY (calculée sur tous les frags
// mesurés du match, camp adverse compris) et UNE SEULE ligne de joueur, celle du joueur
// consulté. Le lobby sert de RÉFÉRENTIEL — c'est ce qui rend l'écart lisible d'une carte à
// l'autre (D22-4) — il n'est jamais publié nominativement : la page Session n'est pas un
// tableau du lobby.
//
// # BEST-EFFORT, TOUJOURS
//
// Repo non câblé (titre sans décodeur de film), xuid inconnu, lecture en échec, aucun match
// décodé : le bloc reste nil et la page se rend. Une erreur est JOURNALISÉE avant toute
// dégradation.
package service

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
)

// WithMatchRange injecte le lecteur de portée « tout le lobby » et le xuid du joueur
// consulté. Câblé INCONDITIONNELLEMENT : c'est le repo qui sait si ce titre a des positions
// par kill (il rend games.ErrCapabilityNotSupported), et un `if capability` ici prendrait la
// même décision à deux endroits qui divergeraient — même doctrine que le câblage Timeseries
// de WeaponRangeRepo.
func (s *SessionPageService) WithMatchRange(repo port.MatchRangeRepository, xuid string) *SessionPageService {
	s.matchRangeRepo = repo
	s.matchRangeXUID = xuid
	return s
}

// attachSessionRange attache le bloc portée de la session COURANTE et, en comparaison, celui
// de la session COMPARÉE — miroir d'attachSessionUsage : les deux colonnes du drawer parlent
// des mêmes matchs, et l'écart au lobby est comparable sans retraitement (c'est précisément
// ce que la normalisation garantit).
func (s *SessionPageService) attachSessionRange(
	ctx context.Context, resp *domain.SessionPageResponse,
	matches, compareMatches []legacymatch.StatsMatchRow,
) {
	resp.RangeProfiles = s.buildSessionRange(ctx, matches)
	if len(compareMatches) > 0 {
		resp.CompareRangeProfiles = s.buildSessionRange(ctx, compareMatches)
	}
}

// buildSessionRange calcule le bloc portée d'UNE session. nil dans tous les cas où il n'y a
// rien à dire — jamais un bloc vide, qui se lirait comme une mesure à zéro.
func (s *SessionPageService) buildSessionRange(
	ctx context.Context, matches []legacymatch.StatsMatchRow,
) *domain.MatchRangeBlock {
	if s.matchRangeRepo == nil || s.matchRangeXUID == "" || len(matches) == 0 {
		return nil
	}
	scope := make([]analysis.MatchRangeMatch, 0, len(matches))
	ids := make([]string, 0, len(matches))
	for i := range matches {
		scope = append(scope, analysis.MatchRangeMatch{
			MatchID:  matches[i].MatchID,
			PlayedAt: matches[i].StartTime,
			MapName:  sessionRangeMapName(matches[i]),
		})
		ids = append(ids, matches[i].MatchID)
	}

	read, err := s.matchRangeRepo.LoadMatchRangeKills(ctx, s.titleSlug, port.WeaponRangeFilters{
		MatchIDs:   ids,
		AllPlayers: true,
	})
	if err != nil {
		if errors.Is(err, games.ErrCapabilityNotSupported) {
			// Titre sans positions par kill : une absence de capacité, pas une panne.
			slog.DebugContext(ctx, "session page: range block unsupported for title",
				"slug", s.titleSlug, "match_count", len(ids))
			return nil
		}
		slog.ErrorContext(ctx, "session page: range block load failed",
			"err", err, "match_count", len(ids))
		return nil
	}

	profiles := analysis.MatchRangeProfiles(analysis.MatchRangeInput{
		Kills:        read.Kills,
		Matches:      scope,
		Publish:      map[string]string{s.matchRangeXUID: s.gamertag},
		PublishOrder: []string{s.matchRangeXUID},
	})
	if len(profiles) == 0 {
		slog.InfoContext(ctx, "session page: range block empty, section omitted",
			"match_count", len(ids),
			"cause", "aucun match de la session ne porte de frag mesure")
		return nil
	}
	return &domain.MatchRangeBlock{
		Profiles:      profiles,
		KillsMeasured: len(read.Kills),
		KillsTotal:    read.KillsTotal,
	}
}

// sessionRangeMapName rend le nom de carte de l'étiquette, en préférant la traduction FR
// quand la base la porte — la MÊME préférence que les lignes du tableau de session.
func sessionRangeMapName(row legacymatch.StatsMatchRow) string {
	if row.MapNameFR != "" {
		return row.MapNameFR
	}
	return row.MapName
}
