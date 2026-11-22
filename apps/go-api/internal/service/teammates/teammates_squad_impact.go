// Package teammates — teammates_squad_impact.go : LES RÔLES D'IMPACT DE L'ESCOUADE, lus et
// calculés une fois par requête pour le tableau d'impact par session de l'onglet Contributions
// (la session affichée et les sessions précédentes de la composition).
//
// Une lecture des événements d'impact (Q32, partagée par la mémoire de la requête,
// teammates_service_loads.go), une lecture du journal des morts (badge « Voleur ») et une lecture
// des bris de bouclier du film (rôle « Os de verre ») couvrent les matchs des sessions ; chaque match passe UNE fois par squadimpact.RolesOfMatch. Q32
// est lu par GROUPES (groupesDImpact) : chaque groupe reçoit ce que sa lecture dédiée lui
// aurait rendu, repli des frags reconstitués compris.
package teammates

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/squadimpact"
	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/timing"
)

// impactEscouade : ce que lit le tableau d'impact par session.
type impactEscouade struct {
	// rows : la population escouade (allSquadRows), celle dont les sessions sont affichées.
	rows []domain.SquadMatchRow
	// timeline : l'historique de la composition, dont viennent les soirées précédentes (et le
	// T0 de leurs matchs).
	timeline []domain.SquadMatchRow
	// evenings : les sessions publiées (soireesDImpact).
	evenings  []squadimpact.Evening
	mainXUID  string
	selected  []string
	teammates []domain.TeammateRow
	// allies : l'équipe alliée du joueur principal par match (Q32b), chargée une fois par
	// GetPage sur l'union de la population et de l'historique.
	allies []domain.AllyParticipant
}

// soireesDImpact rend les sessions du tableau d'impact : celles de la population escouade,
// précédées des sessions antérieures de la composition (squadimpact.SelectEvenings). Sans
// coéquipier sélectionné : aucune.
func soireesDImpact(rows, timelineRows []domain.SquadMatchRow, selection bool) []squadimpact.Evening {
	if !selection {
		return nil
	}
	return squadimpact.SelectEvenings(eveningMatches(rows), eveningMatches(timelineRows))
}

// eveningMatches projette les lignes escouade sur l'entrée de la sélection des sessions.
func eveningMatches(rows []domain.SquadMatchRow) []squadimpact.EveningMatch {
	out := make([]squadimpact.EveningMatch, 0, len(rows))
	for _, r := range rows {
		m := squadimpact.EveningMatch{
			MatchID: r.MatchID, StartTime: r.StartTime, Outcome: r.Outcome,
		}
		if r.SessionLabel != nil {
			m.SessionLabel = *r.SessionLabel
		}
		out = append(out, m)
	}
	return out
}

// groupesDImpact — les groupes de matchs dont les rôles sont calculés : la population (celle
// que lisent les autres blocs), puis chaque session publiée hors de la population (les sessions
// précédentes). Chaque groupe est ce qu'une lecture dédiée aurait lu : le repli
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

// buildSquadImpact rend le tableau d'impact par session. Nil sans coéquipier sélectionné, quand la
// population est vide, ou quand aucun rôle ne tombe sur l'escouade (titre sans événements ni équipe
// alliée, ou lecture des événements en échec, journalisée par loadImpactEventsByMatch). Le nil
// d'une escouade pourtant sélectionnée se journalise avec sa raison (raisonSessionsAbsentes) : le
// front distingue le titre sans la fonction (capability match.events.timeline) du titre qui l'a et
// n'a rien à montrer. Les matchs de la population sans libellé de session n'ont pas de ligne ; leur
// nombre se journalise une fois par requête.
func (s *TeammatesService) buildSquadImpact(ctx context.Context, in impactEscouade) *domain.SquadImpactSessions {
	defer timing.FromContext(ctx).Section("squad_impact")()
	if len(in.rows) == 0 || len(in.selected) == 0 {
		return nil
	}
	journaliserMatchsSansSession(ctx, in.rows)
	// L'APPARTENANCE SE DÉCIDE PAR XUID (teammates[].XUID), sous le nom de la ligne (le nom
	// choisi) : Q29 et Q32b nomment chacune par l'annuaire de SES matchs, un même xuid peut y
	// porter deux casses.
	squad := resolveSquadScope(in.rows, s.gamertag, in.mainXUID, in.teammates).gtByXUID
	roles := s.rolesDesMatchs(ctx, in, groupesDImpact(in.rows, in.evenings), squad)
	players := joueursDeLEscouade(s.gamertag, in.selected)
	sessions := sessionsDImpact(in, players, squad, roles)
	if sessions == nil {
		slog.InfoContext(ctx, "teammates_squad_impact_sessions_absent",
			"reason", raisonSessionsAbsentes(in.evenings), "sessions", len(in.evenings),
			"squad_size", len(players), "matches", len(roles))
	}
	return sessions
}

// Les raisons d'un tableau d'impact absent alors qu'une escouade est sélectionnée.
const (
	sessionsSansSession = "no_session"
	sessionsSansRole    = "no_role_attributed"
)

// raisonSessionsAbsentes dit pourquoi squadimpact.BuildSessions n'a rien publié : aucune session
// (aucun match de la population ne porte de libellé de session) ou aucun rôle attribué à l'escouade
// sur les sessions (titre sans événements horodatés exploitables, ou lecture en échec).
func raisonSessionsAbsentes(evenings []squadimpact.Evening) string {
	if len(evenings) == 0 {
		return sessionsSansSession
	}
	return sessionsSansRole
}

// rolesDesMatchs calcule, par match des groupes, les rôles tombés sur l'escouade. Les
// événements sont ramenés au référentiel gameplay (T0 retranché) ; l'équipe alliée complète du
// joueur principal sert au calcul team-wide (sans xuid du principal, aucune équipe n'est
// attribuable) ; le « Voleur » vient du journal des morts, lu sur l'escouade seule ; « Os de
// verre » vient des bris de bouclier du film.
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
	shieldsByMatch := s.loadShieldsByMatch(ctx, matchIDs)
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
			Shields: shieldsByMatch[mid],
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
