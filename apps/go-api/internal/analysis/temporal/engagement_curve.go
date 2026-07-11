package temporal

import (
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// buildCurveParams regroupe les inputs de buildEngagementCurve. Mis dans une
// struct dediee pour respecter la limite de 5 parametres par fonction (cf
// CLAUDE.md règle 15).
type buildCurveParams struct {
	PlayerEvents []canonical.HighlightEvent
	TeamEvents   []canonical.HighlightEvent
	LobbyEvents  []canonical.HighlightEvent
	NTeam        int
	NHumansLobby int
	MatchStartMS int64
	MatchEndMS   int64
	WindowMS     int64
	SamplingMS   int64
}

// buildEngagementCurve construit la serie temporelle des paces (joueur, team,
// lobby) sur la duree du match. PaceAttendu est laisse a 0 : il est pose en 2e
// passe par applyExpectedToCurve (modele lobby-anchored — l'attendu depend de
// l'intensite moyenne du match, connue seulement une fois la courbe construite).
//
// Pour chaque instant t echantillonne tous les SamplingMS sur [MatchStartMS,
// MatchEndMS], on calcule en fenetre glissante centree de largeur WindowMS :
//   - pace_joueur(t)  = poids_events_joueur dans [t-W/2, t+W/2] / (W/60s)
//   - pace_team(t)    = poids_events_team dans [t-W/2, t+W/2] / NTeam / (W/60s)
//   - pace_lobby(t)   = poids_events_lobby dans [t-W/2, t+W/2] / NHumans / (W/60s)
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
	playerPts := extractWeightedPoints(p.PlayerEvents)
	teamPts := extractWeightedPoints(p.TeamEvents)
	lobbyPts := extractWeightedPoints(p.LobbyEvents)

	windowMin := float64(p.WindowMS) / 60_000.0

	for i := 0; i < nPoints; i++ {
		t := p.MatchStartMS + int64(i)*p.SamplingMS
		windowStart := t - halfWindow
		windowEnd := t + halfWindow

		paceJoueur := sumWeightInWindow(playerPts, windowStart, windowEnd) / windowMin
		paceTeamRaw := sumWeightInWindow(teamPts, windowStart, windowEnd) / windowMin
		paceLobbyRaw := sumWeightInWindow(lobbyPts, windowStart, windowEnd) / windowMin

		paceTeam := safePerPlayer(paceTeamRaw, p.NTeam)
		paceLobby := safePerPlayer(paceLobbyRaw, p.NHumansLobby)

		// PaceAttendu laisse a 0 : pose en 2e passe (applyExpectedToCurve).
		curve = append(curve, domain.EngagementPoint{
			TimeMS:     t,
			PaceJoueur: paceJoueur,
			PaceTeam:   paceTeam,
			PaceLobby:  paceLobby,
		})
	}

	return curve
}

// weightedPoint = un event réduit à (instant, poids d'engagement de son type).
type weightedPoint struct {
	t int64
	w float64
}

// extractWeightedPoints réduit chaque event à (TimeMS, engagementEventWeight(type)).
//
// Note : les events arrivent normalement deja tries par TimeMS (convention de
// l'ingestion highlight_events). On ne re-trie pas pour eviter le coût O(n
// log n). Si l'invariant est viole en pratique, ajouter un sort.Slice ici.
func extractWeightedPoints(events []canonical.HighlightEvent) []weightedPoint {
	pts := make([]weightedPoint, len(events))
	for i, e := range events {
		pts[i] = weightedPoint{t: e.TimeMS, w: engagementEventWeight(e.EventType)}
	}
	return pts
}

// sumWeightInWindow somme les POIDS des events dont TimeMS ∈ [windowStart, windowEnd]
// (rythme PONDÉRÉ ; cf. engagementEventWeight). Avant la pondération c'était un simple
// comptage (poids 1 par event).
//
// Implementation lineaire (suffisante pour ~50-100 events/match). Si l'on
// devait scaler a des matches massifs, basculer sur sort.Search pour O(log n).
func sumWeightInWindow(pts []weightedPoint, windowStart, windowEnd int64) float64 {
	var sum float64
	for _, p := range pts {
		if p.t >= windowStart && p.t <= windowEnd {
			sum += p.w
		}
	}
	return sum
}

// safePerPlayer divise un pace agrege par le nombre de joueurs, en gerant le
// cas degenere (lobby vide post-quitters).
func safePerPlayer(rawPace float64, n int) float64 {
	if n <= 0 {
		return 0
	}
	return rawPace / float64(n)
}
