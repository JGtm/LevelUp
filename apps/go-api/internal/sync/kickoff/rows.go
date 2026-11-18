package kickoff

// rows.go — le coup d'envoi et le temps de jeu posés sur les lignes d'un match AVANT leur insertion
// (synchronisation, import OpenSpartan) : aucune mise à jour après coup, et la même résolution que la
// publication d'un rejeu et que le rattrapage ([Resolve]).

import (
	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/domain"
)

// ApplyToRows résout le coup d'envoi d'un match depuis ses lignes extraites (avant l'insertion, le
// rejeu n'existe pas encore : l'ordre des sources commence donc par l'estimation), le pose sur reg
// (`RealStartTime`, `RealStartSource`) et pose sur chaque participant le temps de jeu calculé sur ce
// coup d'envoi ; un participant dont la présence est inconnue garde le temps de jeu de l'API. Rend le
// coup d'envoi résolu, que l'appelant observe ([Observe]).
func ApplyToRows(reg *domain.MatchRegistryRow, parts []domain.MatchParticipantRow) timeline.Kickoff {
	m := Match{
		MatchID:         reg.MatchID,
		StartUTC:        reg.StartTime.UTC(),
		DurationSeconds: reg.DurationSeconds,
		RealStart:       reg.RealStartTime,
		Source:          timeline.KickoffSource(reg.RealStartSource),
		Participants:    make([]Participant, 0, len(parts)),
	}
	for _, p := range parts {
		m.Participants = append(m.Participants, Participant{
			XUID:               p.XUID,
			FirstJoined:        p.FirstJoinedTime,
			LastLeave:          p.LastLeaveTime,
			PresentAtBeginning: p.PresentAtBeginning != nil && *p.PresentAtBeginning,
			TimePlayed:         p.TimePlayedSeconds,
		})
	}
	plan := Resolve(m, Film{State: FilmAbsent})
	reg.RealStartTime = plan.RealStart
	reg.RealStartSource = string(plan.Kickoff.Source)
	changed := make(map[string]int, len(plan.TimePlayed))
	for _, c := range plan.TimePlayed {
		changed[c.XUID] = c.After
	}
	for i := range parts {
		if v, ok := changed[parts[i].XUID]; ok {
			parts[i].TimePlayedSeconds = &v
		}
	}
	return plan.Kickoff
}
