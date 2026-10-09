// Package teammates — teammates_service_objective_history.go : L'HISTORIQUE D'OBJECTIF de la
// composition, carte « Rapport de force, soirée après soirée » de l'onglet Contributions (lot L3
// du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26, décisions D6 et D7).
//
// Orchestration seule : le calcul est pur (analysis/squadformes.BuildObjectiveHistory), la
// lecture passe par le lecteur des colonnes d'objectif déjà câblé pour le bloc « formes
// retenues » (port.SquadFormesObjectiveRepository.LoadObjectiveColumnRows, vue `_latest`), et
// le camp de chaque match est l'équipe alliée du joueur principal que GetPage charge déjà
// (loadMainTeamAllies, sur l'union de l'historique et du périmètre). Aucune lecture de plus
// que celle des colonnes d'objectif.
package teammates

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/squadformes"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/timing"
)

// WithObjectiveHistory injecte le savoir du titre sur les modes que les parts de rôle de
// l'escouade écartent — l'historique d'objectif ET le fil de la session (bloc formes,
// `excluded_from_balance`) — (D6 : le drapeau neutre, où personne ne peut renvoyer le drapeau
// et où la part d'un camp est mécanique). Nil ⇒ aucun mode écarté. Le prédicat vient du paquet du titre, posé au
// câblage (jamais une comparaison de slug ici).
func (s *TeammatesService) WithObjectiveHistory(modeEcarte func(pairName string) bool) *TeammatesService {
	s.objectiveModeEcarte = modeEcarte
	return s
}

// loadObjectiveHistory publie l'historique : la soirée affichée (les lignes escouade du
// périmètre D2) et les soirées précédentes de la composition (timelineRows). Nil quand la
// source des colonnes d'objectif n'est pas câblée (titre sans stats d'objectif), quand le
// périmètre est vide, ou quand la lecture échoue — la carte se retire, jamais la page.
func (s *TeammatesService) loadObjectiveHistory(
	ctx context.Context, current, timeline []domain.SquadMatchRow, camp map[string]map[string]struct{},
	pairNames map[string]string,
) *domain.SquadObjectiveHistory {
	if s.formesObjectiveRepo == nil || len(current) == 0 {
		return nil
	}
	if camp == nil {
		// L'équipe alliée n'a pas été lue (échec déjà journalisé par loadMainTeamAllies) : sans
		// camp, aucune part ne se calcule.
		slog.WarnContext(ctx, "teammates_objective_history_sans_camp",
			"player", s.gamertag, "matchs", len(current))
		return nil
	}
	defer timing.FromContext(ctx).Section("objective_history")()
	ids := matchIDsDistincts(current, timeline)
	rows, err := s.formesObjectiveRepo.LoadObjectiveColumnRows(ctx, ids)
	if err != nil {
		slog.WarnContext(ctx, "teammates_objective_history_lecture_en_echec",
			"player", s.gamertag, "matchs", len(ids), "err", err)
		return nil
	}
	h := squadformes.BuildObjectiveHistory(squadformes.HistoryInput{
		Current:  historyMatches(current, s.modesEcartes(pairNames)),
		Timeline: historyMatches(timeline, s.modesEcartes(pairNames)),
		Rows:     rows,
		Camp:     camp,
	})
	slog.DebugContext(ctx, "teammates_objective_history",
		"player", s.gamertag, "matchs_lus", len(ids), "lignes", len(rows),
		"soiree_matchs_objectif", h.Current.ObjectiveMatches, "soirees_precedentes", len(h.Previous),
		"soirees_avec_objectif", h.EveningsWithObjective, "soirees_sous_minimum", h.EveningsBelowMinimum)
	return &h
}

// modesEcartes — le prédicat « mode écarté » d'un match : le pair_name brut des lignes
// canoniques du joueur (pairNames) d'abord, celui passé par l'appelant en repli, jugé par le
// prédicat du titre (WithObjectiveHistory). Le fil de la session (bloc formes) et l'historique
// passent par LUI SEUL : même prédicat, même source, donc la fin du fil égale le point du soir.
func (s *TeammatesService) modesEcartes(pairNames map[string]string) func(matchID, pairName string) bool {
	return func(matchID, pairName string) bool {
		if s.objectiveModeEcarte == nil {
			return false
		}
		if p, ok := pairNames[matchID]; ok {
			pairName = p
		}
		return s.objectiveModeEcarte(pairName)
	}
}

// historyMatches projette les lignes escouade sur l'entrée du calcul : session, heure,
// victoire du camp du joueur principal, et mode écarté (ecarte).
func historyMatches(rows []domain.SquadMatchRow, ecarte func(matchID, pairName string) bool) []squadformes.HistoryMatch {
	out := make([]squadformes.HistoryMatch, 0, len(rows))
	for _, r := range rows {
		m := squadformes.HistoryMatch{
			MatchID: r.MatchID, StartTime: r.StartTime, Won: r.Outcome == domain.OutcomeWin,
		}
		if r.SessionLabel != nil {
			m.SessionLabel = *r.SessionLabel
		}
		m.Excluded = ecarte(r.MatchID, r.PairName)
		out = append(out, m)
	}
	return out
}

// matchIDsDistincts — les identifiants de match des deux listes, sans doublon.
func matchIDsDistincts(a, b []domain.SquadMatchRow) []string {
	vus := make(map[string]bool, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, list := range [][]domain.SquadMatchRow{a, b} {
		for _, r := range list {
			if !vus[r.MatchID] {
				vus[r.MatchID] = true
				out = append(out, r.MatchID)
			}
		}
	}
	return out
}
