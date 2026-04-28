package temporal

import (
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// buildCurveParams regroupe les inputs de buildEngagementCurve. Mis dans une
// struct dediee pour respecter la limite de 5 parametres par fonction (cf
// CLAUDE.md règle 15).
type buildCurveParams struct {
	PlayerEvents      []canonical.HighlightEvent
	TeamEvents        []canonical.HighlightEvent
	LobbyEvents       []canonical.HighlightEvent
	NTeam             int
	NHumansLobby      int
	MatchStartMS      int64
	MatchEndMS        int64
	WindowMS          int64
	SamplingMS        int64
	CoefForExpected   float64
	DenominatorEvents []canonical.HighlightEvent
	DenominatorN      int
}

// buildEngagementCurve construit la serie temporelle des paces (joueur, team,
// attendu, lobby) sur la duree du match.
//
// Pour chaque instant t echantillonne tous les SamplingMS sur [MatchStartMS,
// MatchEndMS], on calcule en fenetre glissante centree de largeur WindowMS :
//   - pace_joueur(t)  = nb_events_joueur dans [t-W/2, t+W/2] / (W/60s)
//   - pace_team(t)    = nb_events_team dans [t-W/2, t+W/2] / NTeam / (W/60s)
//   - pace_lobby(t)   = nb_events_lobby dans [t-W/2, t+W/2] / NHumans / (W/60s)
//   - pace_attendu(t) = coef_for_expected * pace_denominator(t)
//
// Le pace_attendu utilise le denominateur (team ou lobby selon mode FFA, cf
// selectExpectedReference dans engagement_score.go).
//
// Les events doivent avoir des TimeMS dans le repere du match (relatifs a 0
// ou absolus mais coherents avec MatchStartMS/MatchEndMS).
//
// Les flags PostDeathFlag et IsPassiveDeath sont laisses a false ici ;
// annotateDeaths les positionne dans une passe ulterieure.
func buildEngagementCurve(p buildCurveParams) []domain.EngagementPoint {
	if p.MatchEndMS <= p.MatchStartMS || p.SamplingMS <= 0 || p.WindowMS <= 0 {
		return nil
	}

	durationMS := p.MatchEndMS - p.MatchStartMS
	nPoints := int(durationMS/p.SamplingMS) + 1
	if nPoints < 1 {
		return nil
	}

	curve := make([]domain.EngagementPoint, 0, nPoints)
	halfWindow := p.WindowMS / 2

	// Pre-extraction des TimeMS de chaque slice pour eviter d'iterer sur les
	// events complets a chaque echantillon. Tries en amont par defaut (les
	// events sont stockes par ordre chronologique dans highlight_events).
	playerTimes := extractTimes(p.PlayerEvents)
	teamTimes := extractTimes(p.TeamEvents)
	lobbyTimes := extractTimes(p.LobbyEvents)
	denomTimes := extractTimes(p.DenominatorEvents)

	windowMin := float64(p.WindowMS) / 60_000.0

	for i := 0; i < nPoints; i++ {
		t := p.MatchStartMS + int64(i)*p.SamplingMS
		windowStart := t - halfWindow
		windowEnd := t + halfWindow

		paceJoueur := countInWindow(playerTimes, windowStart, windowEnd) / windowMin
		paceTeamRaw := countInWindow(teamTimes, windowStart, windowEnd) / windowMin
		paceLobbyRaw := countInWindow(lobbyTimes, windowStart, windowEnd) / windowMin

		paceTeam := safePerPlayer(paceTeamRaw, p.NTeam)
		paceLobby := safePerPlayer(paceLobbyRaw, p.NHumansLobby)

		// Pace attendu base sur le denominateur (team ou lobby selon mode).
		var paceAttendu float64
		if p.DenominatorN > 0 {
			denomRaw := countInWindow(denomTimes, windowStart, windowEnd) / windowMin
			paceAttendu = p.CoefForExpected * (denomRaw / float64(p.DenominatorN))
		}

		curve = append(curve, domain.EngagementPoint{
			TimeMS:      t,
			PaceJoueur:  paceJoueur,
			PaceTeam:    paceTeam,
			PaceAttendu: paceAttendu,
			PaceLobby:   paceLobby,
		})
	}

	return curve
}

// extractTimes recupere les TimeMS d'une liste d'events.
//
// Note : les events arrivent normalement deja tries par TimeMS (convention de
// l'ingestion highlight_events). On ne re-trie pas pour eviter le coût O(n
// log n). Si l'invariant est viole en pratique, ajouter un sort.Slice ici.
func extractTimes(events []canonical.HighlightEvent) []int64 {
	times := make([]int64, len(events))
	for i, e := range events {
		times[i] = e.TimeMS
	}
	return times
}

// countInWindow compte les events dont TimeMS appartient a [windowStart, windowEnd].
//
// Implementation lineaire (suffisante pour ~50-100 events/match). Si l'on
// devait scaler a des matches massifs, basculer sur sort.Search pour O(log n).
func countInWindow(times []int64, windowStart, windowEnd int64) float64 {
	count := 0
	for _, t := range times {
		if t >= windowStart && t <= windowEnd {
			count++
		}
	}
	return float64(count)
}

// safePerPlayer divise un pace agrege par le nombre de joueurs, en gerant le
// cas degenere (lobby vide post-quitters).
func safePerPlayer(rawPace float64, n int) float64 {
	if n <= 0 {
		return 0
	}
	return rawPace / float64(n)
}
