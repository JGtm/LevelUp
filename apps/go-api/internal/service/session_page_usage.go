// Package service — session_page_usage.go : le bloc « usages d'équipement,
// socles et objectifs » de la page détail de session (chantier session-usage S2,
// .ai/HANDOFF_SESSION_USAGE_BDD_2026-09-04.md §5/S2).
//
// PATRON D'ATTACHEMENT (miroir IntensityRows/FirstBlood) : repo optionnel injecté
// à la DI, attaché à la réponse existante de POST .../pages/sessions/detail —
// PAS d'endpoint dédié. Capability film.usage_summary absente ⇒ repo nil ⇒ bloc
// Available=false avec raison machine (réponse partielle propre, jamais un 500).
// Le contexte Solo/Escouade est résolu EN AMONT par Filters.MatchContext : le
// bloc agrège les matchs de la session AFFICHÉE ; en contexte escouade il porte
// en plus une ligne par coéquipier suivi.
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/teammates"
)

// objectiveRoleRowsLoader est la capability OPTIONNELLE du repo objectifs :
// lignes (match, joueur, famille) projetées par rôle, les deux camps. Seul
// duckdb.ObjectiveStatsRepo l'implémente ; sans elle (titre sans
// match.objective.stats), le sous-bloc Objectives est simplement omis.
type objectiveRoleRowsLoader interface {
	LoadObjectiveRoleRows(ctx context.Context, matchIDs []string) ([]sessionusage.ObjectiveRow, error)
}

// WithSessionUsage injecte le repo du résumé d'usage S1 (vues _latest), le xuid
// du joueur suivi et le résolveur d'amis configurés (restriction des coéquipiers
// suivis, même source que l'accueil). Câblé UNIQUEMENT pour les titres portant
// film.usage_summary (registry_pages, jamais slug==) — nil ⇒ bloc indisponible.
func (s *SessionPageService) WithSessionUsage(
	repo port.SessionUsageRepository, xuid string, friends teammates.FriendGamertagsResolver,
) *SessionPageService {
	s.sessionUsageRepo = repo
	s.usageXUID = xuid
	s.usageFriends = friends
	return s
}

// attachSessionUsage attache le bloc usage de la session COURANTE. Best-effort :
// une erreur de lecture est loggée PUIS dégradée en Available=false (raison
// machine) — jamais d'échec de la page. Session sans match ⇒ bloc nil.
func (s *SessionPageService) attachSessionUsage(
	ctx context.Context, resp *domain.SessionPageResponse,
	matches []legacymatch.StatsMatchRow, matchContext string,
) {
	if len(matches) == 0 {
		return
	}
	if s.sessionUsageRepo == nil || s.usageXUID == "" {
		resp.Usage = &domain.SessionUsageBlock{
			UnavailableReason: domain.SessionUsageUnsupported, MatchesTotal: len(matches),
		}
		return
	}
	ids := matchIDsFromStatsRows(matches)
	films, filmsErr := s.sessionUsageRepo.LoadUsageFilms(ctx, ids)
	players, playersErr := s.sessionUsageRepo.LoadUsagePlayers(ctx, ids)
	participants, partErr := s.sessionUsageRepo.LoadParticipants(ctx, ids)
	for _, err := range []error{filmsErr, playersErr, partErr} {
		if err != nil {
			slog.ErrorContext(ctx, "session page: usage block load failed", "err", err,
				"match_count", len(ids))
			resp.Usage = &domain.SessionUsageBlock{
				UnavailableReason: domain.SessionUsageLoadFailed, MatchesTotal: len(matches),
			}
			return
		}
	}

	tc := sessionusage.BuildTeamContext(s.usageXUID, participants)
	in := buildSessionUsageInput(s.usageXUID, matches, films, players, tc)
	// Un match mesuré sans échelle de temps (duration_ms <= 0 et aucun repli
	// stats) est exclu des cadences par ComputeUsage — jamais en silence.
	if n := countMeasuredWithoutDuration(in.Matches); n > 0 {
		slog.WarnContext(ctx, "session page: measured matches without time scale excluded from rates",
			"matches_without_duration", n, "match_count", len(ids))
	}
	var squad []domain.SessionUsageSquadPlayer
	if matchContext == domain.MatchContextSquad {
		squad = sessionusage.ResolveTrackedSquad(s.usageXUID, ids, participants, s.friendGamertags(ctx))
		for _, member := range squad {
			in.SquadXUIDs = append(in.SquadXUIDs, member.XUID)
		}
	}
	block := sessionusage.ComputeUsage(in)
	block.SquadPlayers = squad
	s.attachSessionObjectives(ctx, &block, ids, tc, in.SquadXUIDs)
	resp.Usage = &block
}

// attachSessionObjectives renseigne le sous-bloc objectifs (lecture seule de
// match_objective_stats_latest, les deux camps). Omis sans repo objectifs (titre
// sans match.objective.stats) ou sur erreur (best-effort, loggé).
func (s *SessionPageService) attachSessionObjectives(
	ctx context.Context, block *domain.SessionUsageBlock,
	matchIDs []string, tc sessionusage.TeamContext, squadXUIDs []string,
) {
	loader, ok := s.objectiveIndex.(objectiveRoleRowsLoader)
	if !ok {
		return
	}
	rows, err := loader.LoadObjectiveRoleRows(ctx, matchIDs)
	if err != nil {
		slog.WarnContext(ctx, "session page: objective role rows unavailable", "err", err)
		return
	}
	block.Objectives = sessionusage.ComputeObjectives(sessionusage.ObjectivesInput{
		PlayerXUID: s.usageXUID,
		SquadXUIDs: squadXUIDs,
		Rows:       rows,
		PlayerTeam: tc.PlayerTeam,
		TeamOf:     tc.TeamOf,
		TeamSize:   tc.TeamSize,
		LobbySize:  tc.LobbySize,
	})
}

// friendGamertags résout la liste des amis configurés (vide = aucune
// restriction sur les coéquipiers suivis, même convention que l'accueil).
func (s *SessionPageService) friendGamertags(ctx context.Context) []string {
	if s.usageFriends == nil {
		return nil
	}
	return s.usageFriends(ctx)
}

// countMeasuredWithoutDuration compte les matchs mesurés restés SANS durée
// après le repli stats : ils sortent des cadences (règle C6, voir ComputeUsage)
// et l'appelant le logge une fois par session.
func countMeasuredWithoutDuration(matches []sessionusage.MatchInput) int {
	n := 0
	for i := range matches {
		if matches[i].Measured && matches[i].DurationSeconds <= 0 {
			n++
		}
	}
	return n
}

// buildSessionUsageInput assemble l'entrée de sessionusage.ComputeUsage : un
// MatchInput par match de la session (ordre d'affichage), mesuré s'il a une
// ligne film. Durée du match mesuré : l'échelle de temps du film, repli sur la
// durée côté stats si le film n'en a pas — un match resté sans durée est exclu
// des cadences par ComputeUsage (totaux et parts conservés).
func buildSessionUsageInput(
	playerXUID string, matches []legacymatch.StatsMatchRow,
	films map[string]sessionusage.FilmRow, players []sessionusage.PlayerRow,
	tc sessionusage.TeamContext,
) sessionusage.Input {
	playersByMatch := make(map[string][]sessionusage.PlayerRow, len(films))
	for _, p := range players {
		playersByMatch[p.MatchID] = append(playersByMatch[p.MatchID], p)
	}
	in := sessionusage.Input{PlayerXUID: playerXUID}
	for i := range matches {
		row := &matches[i]
		film, measured := films[row.MatchID]
		m := sessionusage.MatchInput{
			MatchID:   row.MatchID,
			Measured:  measured,
			TeamOf:    tc.TeamOf[row.MatchID],
			TeamSize:  tc.TeamSize[row.MatchID],
			LobbySize: tc.LobbySize[row.MatchID],
			Players:   playersByMatch[row.MatchID],
		}
		if team, ok := tc.PlayerTeam[row.MatchID]; ok {
			t := team
			m.PlayerTeam = &t
		}
		if measured {
			m.DurationSeconds = float64(film.DurationMS) / 1000
			if m.DurationSeconds <= 0 && row.TimePlayedSeconds != nil {
				m.DurationSeconds = float64(*row.TimePlayedSeconds)
			}
			m.PadUnnamed = film.PadUnnamed
			m.PowerupPickups = film.PowerupPickups
		}
		in.Matches = append(in.Matches, m)
	}
	return in
}
