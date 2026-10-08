// Package teammates — teammates_squad_impact.go : LES RÔLES D'IMPACT DE L'ESCOUADE, lus et
// calculés une fois par requête pour leurs deux surfaces de l'onglet Contributions : la matrice
// d'impact (teammates.07, un match par colonne, sur la population escouade) et les points
// d'impact par soirée (la soirée affichée et les soirées précédentes de la composition).
//
// Une lecture des événements d'impact (Q32, partagée par la mémoire de la requête,
// teammates_service_loads.go) et une lecture du journal des morts (badge « Voleur ») couvrent
// les matchs des deux surfaces ; chaque match passe UNE fois par squadimpact.RolesOfMatch. Q32
// est lu par GROUPES (groupesDImpact) : chaque groupe reçoit ce que sa lecture dédiée lui
// aurait rendu, repli des frags reconstitués compris.
package teammates

import (
	"cmp"
	"context"
	"slices"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/squadimpact"
	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/timing"
)

// impactEscouade : ce que lisent la matrice d'impact et les points par soirée.
type impactEscouade struct {
	// rows : la population escouade (allSquadRows), celle de la matrice.
	rows []domain.SquadMatchRow
	// timeline : l'historique de la composition, dont viennent les soirées précédentes (et le
	// T0 de leurs matchs).
	timeline []domain.SquadMatchRow
	// evenings : les soirées publiées (soireesDImpact).
	evenings  []squadimpact.Evening
	mainXUID  string
	selected  []string
	teammates []domain.TeammateRow
	// allies : l'équipe alliée du joueur principal par match (Q32b), chargée une fois par
	// GetPage sur l'union de la population et de l'historique.
	allies []domain.AllyParticipant
}

// soireesDImpact rend les soirées des points d'impact : celles de la population escouade,
// précédées des soirées antérieures de la composition (squadimpact.SelectEvenings). Sans
// coéquipier sélectionné : aucune.
func soireesDImpact(rows, timelineRows []domain.SquadMatchRow, selection bool) []squadimpact.Evening {
	if !selection {
		return nil
	}
	return squadimpact.SelectEvenings(eveningMatches(rows), eveningMatches(timelineRows))
}

// eveningMatches projette les lignes escouade sur l'entrée de la sélection des soirées.
func eveningMatches(rows []domain.SquadMatchRow) []squadimpact.EveningMatch {
	out := make([]squadimpact.EveningMatch, 0, len(rows))
	for _, r := range rows {
		m := squadimpact.EveningMatch{MatchID: r.MatchID, StartTime: r.StartTime, Won: r.Outcome == domain.OutcomeWin}
		if r.SessionLabel != nil {
			m.SessionLabel = *r.SessionLabel
		}
		out = append(out, m)
	}
	return out
}

// groupesDImpact — les groupes de matchs dont les rôles sont calculés : la population (celle
// que lisent la matrice et les autres blocs), puis chaque soirée publiée hors de la population
// (les soirées précédentes). Chaque groupe est ce qu'une lecture dédiée aurait lu : le repli
// des frags reconstitués se décide groupe par groupe. Le préchargement de Q32 lit exactement
// ces groupes.
func groupesDImpact(rows []domain.SquadMatchRow, evenings []squadimpact.Evening) [][]string {
	population := collectSharedMatchIDsForDigest(rows)
	if len(population) == 0 {
		return nil
	}
	dans := make(map[string]bool, len(population))
	for _, id := range population {
		dans[id] = true
	}
	out := [][]string{population}
	for _, e := range evenings {
		var g []string
		for _, m := range e.Matches {
			if !dans[m.MatchID] {
				dans[m.MatchID] = true
				g = append(g, m.MatchID)
			}
		}
		if len(g) > 0 {
			out = append(out, g)
		}
	}
	return out
}

// buildSquadImpact rend la matrice d'impact et les points par soirée, tirés des mêmes rôles.
// Nil pour chacun quand la population est vide, sans coéquipier sélectionné, ou quand aucun
// rôle ne tombe sur l'escouade (titre sans événements ni équipe alliée).
func (s *TeammatesService) buildSquadImpact(
	ctx context.Context, in impactEscouade,
) (*domain.SquadImpactMatrix, *domain.SquadImpactHistory) {
	defer timing.FromContext(ctx).Section("squad_impact")()
	if len(in.rows) == 0 || len(in.selected) == 0 {
		return nil, nil
	}
	// L'APPARTENANCE SE DÉCIDE PAR XUID (teammates[].XUID), sous le nom de la ligne (le nom
	// choisi) : Q29 et Q32b nomment chacune par l'annuaire de SES matchs, un même xuid peut y
	// porter deux casses.
	squad := resolveSquadScope(in.rows, s.gamertag, in.mainXUID, in.teammates).gtByXUID
	roles := s.rolesDesMatchs(ctx, in, groupesDImpact(in.rows, in.evenings), squad)
	players := joueursDeLEscouade(s.gamertag, in.selected)
	history := squadimpact.BuildHistory(squadimpact.HistoryInput{
		Evenings: in.evenings, Players: players, Roles: roles,
	})
	return assemblerMatrice(in.rows, players, roles), history
}

// rolesDesMatchs calcule, par match des groupes, les rôles tombés sur l'escouade. Les
// événements sont ramenés au référentiel gameplay (T0 retranché) ; l'équipe alliée complète du
// joueur principal sert au calcul team-wide (sans xuid du principal, aucune équipe n'est
// attribuable) ; le « Voleur » vient du journal des morts, lu sur l'escouade seule.
func (s *TeammatesService) rolesDesMatchs(
	ctx context.Context, in impactEscouade, groupes [][]string, squad map[string]string,
) map[string][]squadimpact.Attribution {
	var matchIDs []string
	for _, g := range groupes {
		matchIDs = append(matchIDs, g...)
	}
	t0Rows := append(append([]domain.SquadMatchRow(nil), in.timeline...), in.rows...)
	eventsByMatch := s.loadImpactEventsByMatch(ctx, groupes, timeline.BuildTimelinesFromSquadRows(t0Rows))
	thiefByMatch := s.loadThiefBadgesByMatch(ctx, matchIDs, squad)
	allyByMatch := map[string][]analysis.ParticipantSnap{}
	if in.mainXUID != "" {
		for _, a := range in.allies {
			allyByMatch[a.MatchID] = append(allyByMatch[a.MatchID], analysis.ParticipantSnap{
				XUID: a.XUID, Outcome: a.Outcome, Kills: a.Kills, Deaths: a.Deaths, Assists: a.Assists,
			})
		}
	}
	out := make(map[string][]squadimpact.Attribution, len(matchIDs))
	for _, mid := range matchIDs {
		r := squadimpact.RolesOfMatch(squadimpact.MatchInput{
			Events: eventsByMatch[mid], Allies: allyByMatch[mid], Thief: thiefByMatch[mid],
		}, squad)
		if len(r) > 0 {
			out[mid] = r
		}
	}
	return out
}

// joueursDeLEscouade — le joueur principal puis les coéquipiers sélectionnés, sans doublon.
func joueursDeLEscouade(main string, selected []string) []string {
	out := make([]string, 0, 1+len(selected))
	seen := map[string]bool{}
	for _, gt := range append([]string{main}, selected...) {
		if !seen[gt] {
			seen[gt] = true
			out = append(out, gt)
		}
	}
	return out
}

// assemblerMatrice construit la matrice d'impact sur la population : une colonne par match qui
// attribue au moins un rôle à l'escouade (du plus ancien au plus récent), une ligne par joueur
// (score décroissant, nom croissant à égalité), ses comptes par rôle dans l'ordre canonique.
// Nil quand aucun match n'attribue de rôle.
func assemblerMatrice(
	rows []domain.SquadMatchRow, players []string, roles map[string][]squadimpact.Attribution,
) *domain.SquadImpactMatrix {
	outcome := make(map[string]int, len(rows))
	start := make(map[string]time.Time, len(rows))
	order := make([]string, 0, len(rows))
	for _, m := range rows {
		if _, ok := outcome[m.MatchID]; ok {
			continue
		}
		outcome[m.MatchID], start[m.MatchID] = m.Outcome, m.StartTime
		order = append(order, m.MatchID)
	}
	slices.SortStableFunc(order, func(a, b string) int { return start[a].Compare(start[b]) })

	counts := make(map[string]map[string]int, len(players))
	score := make(map[string]float64, len(players))
	for _, p := range players {
		counts[p] = map[string]int{}
	}
	out := &domain.SquadImpactMatrix{Cells: []domain.SquadImpactCell{}, BadgeOrd: squadimpact.RoleOrder()}
	for _, mid := range order {
		byPlayer := map[string][]string{}
		for _, a := range roles[mid] {
			if c, ok := counts[a.Player]; ok {
				w, _ := squadimpact.Weight(a.Role)
				c[a.Role]++
				score[a.Player] += w
				byPlayer[a.Player] = append(byPlayer[a.Player], a.Role)
			}
		}
		if len(byPlayer) == 0 {
			continue
		}
		out.Matches = append(out.Matches, domain.SquadImpactMatchHeader{MatchID: mid, Outcome: outcome[mid]})
		for _, p := range players {
			if keys := byPlayer[p]; len(keys) > 0 {
				out.Cells = append(out.Cells, domain.SquadImpactCell{Player: p, MatchID: mid, BadgeKeys: keys})
			}
		}
	}
	if len(out.Matches) == 0 {
		return nil
	}
	out.Players = matrixPlayers(players, counts, score)
	return out
}

// matrixPlayers — les lignes de la matrice, score décroissant, nom croissant à égalité.
func matrixPlayers(
	players []string, counts map[string]map[string]int, score map[string]float64,
) []domain.SquadImpactPlayerSummary {
	ordered := slices.Clone(players)
	slices.SortFunc(ordered, func(a, b string) int {
		if c := cmp.Compare(score[b], score[a]); c != 0 {
			return c
		}
		return cmp.Compare(a, b)
	})
	out := make([]domain.SquadImpactPlayerSummary, 0, len(ordered))
	for _, p := range ordered {
		c := make([]domain.SquadImpactBadgeCount, 0, len(counts[p]))
		for _, role := range squadimpact.RoleOrder() {
			c = append(c, domain.SquadImpactBadgeCount{BadgeKey: role, Count: counts[p][role]})
		}
		out = append(out, domain.SquadImpactPlayerSummary{Player: p, Counts: c, Score: round2(score[p])})
	}
	return out
}
