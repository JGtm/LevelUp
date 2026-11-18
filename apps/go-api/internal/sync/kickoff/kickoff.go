// Package kickoff résout, range et rattrape le coup d'envoi des matchs d'une base partagée et le
// temps de jeu de leurs joueurs (ADR 0049).
//
// # UNE RÉSOLUTION
//
// [Resolve] est le seul point où se décide ce qu'un match doit porter : le coup d'envoi par
// `timeline.ResolveKickoff` (rejeu, sinon estimation, sinon inconnu), puis le temps de jeu de
// chaque joueur par `timeline.PlayedSeconds` sur CE coup d'envoi. Les trois écrivains l'appellent
// : la synchronisation (lignes en mémoire avant leur insertion, [ApplyToRows]), la
// publication d'un rejeu (`replayartifacts.Deriver`) et le rattrapage ([Backfill]).
//
// # UNE ÉCRITURE
//
// Hors des insertions de la synchronisation, tout passe par `persist.KickoffPersister`, un match par
// transaction, sous le writer partagé tenu par l'appelant : [Persist] ne lui passe que ce qui diffère
// de la base, donc un second passage n'écrit rien.
package kickoff

import (
	"time"

	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/persist"
)

// Participant est ce que la résolution lit d'une ligne de `match_participants`.
type Participant struct {
	XUID string
	// FirstJoined / LastLeave : arrivée et départ du joueur (UTC) ; nil = inconnu.
	FirstJoined *time.Time
	LastLeave   *time.Time
	// PresentAtBeginning : le joueur était là au début (seuls ceux-là entrent dans l'estimation).
	PresentAtBeginning bool
	// TimePlayed : le temps de jeu rangé ; nil = absent.
	TimePlayed *int
}

// Match est l'état rangé d'un match, tel que la résolution le lit.
type Match struct {
	MatchID string
	// StartUTC : début canonique (`start_time_utc`, sinon `start_time` lu en UTC).
	StartUTC        time.Time
	DurationSeconds *int
	// RealStart / Source : le coup d'envoi rangé et sa source.
	RealStart    *time.Time
	Source       timeline.KickoffSource
	Participants []Participant
}

// TimePlayedChange est un temps de jeu qui change.
type TimePlayedChange struct {
	XUID string
	// Before : la valeur rangée ; nil = absente.
	Before *int
	After  int
}

// Plan est ce que la résolution décide pour un match, et ce qui diffère de la base.
type Plan struct {
	MatchID string
	Kickoff timeline.Kickoff
	// RealStart : le coup d'envoi résolu (nil = inconnu).
	RealStart *time.Time
	// Before / BeforeSource : le coup d'envoi rangé avant la résolution.
	Before       *time.Time
	BeforeSource timeline.KickoffSource
	// TimePlayed : les joueurs dont le temps de jeu change.
	TimePlayed []TimePlayedChange
}

// ValueChanged dit si l'instant du coup d'envoi change (inconnu compris).
func (p Plan) ValueChanged() bool { return !sameInstant(p.RealStart, p.Before) }

// SourceChanged dit si la source du coup d'envoi change.
func (p Plan) SourceChanged() bool { return p.Kickoff.Source != p.BeforeSource }

// KickoffChanged dit si le registre du match est à réécrire.
func (p Plan) KickoffChanged() bool { return p.ValueChanged() || p.SourceChanged() }

// Unchanged dit si le match porte déjà tout ce que la résolution décide.
func (p Plan) Unchanged() bool { return !p.KickoffChanged() && len(p.TimePlayed) == 0 }

// Resolve décide ce que le match doit porter. filmT0Ms est le coup d'envoi tel que le document de
// rejeu du match le publie (nil : pas de rejeu, ou rejeu sans coup d'envoi). Fonction pure.
func Resolve(m Match, filmT0Ms *int64) Plan {
	k := timeline.ResolveKickoff(timeline.KickoffInput{
		StartUTC: m.StartUTC, FilmT0Ms: filmT0Ms, Participations: estimateInputs(m.Participants),
	})
	p := Plan{
		MatchID: m.MatchID, Kickoff: k, RealStart: k.RealStart(m.StartUTC),
		Before: m.RealStart, BeforeSource: m.Source,
	}
	tl, hasWindow := timeline.GameplayTimeline(m.StartUTC, m.DurationSeconds, k)
	for _, part := range m.Participants {
		if part.FirstJoined == nil {
			continue // présence inconnue : la valeur rangée (celle de l'API) reste
		}
		secs, ok := timeline.PlayedSeconds(timeline.TimePlayedInput{
			FirstJoinedTime: part.FirstJoined.UTC(), LastLeaveTime: part.LastLeave,
		}, tl, hasWindow)
		if !ok || (part.TimePlayed != nil && *part.TimePlayed == secs) {
			continue
		}
		p.TimePlayed = append(p.TimePlayed, TimePlayedChange{XUID: part.XUID, Before: part.TimePlayed, After: secs})
	}
	return p
}

// Write rend l'écriture du persisteur pour ce plan (vide quand rien ne change).
func (p Plan) Write() persist.MatchKickoffWrite {
	w := persist.MatchKickoffWrite{MatchID: p.MatchID}
	if p.KickoffChanged() {
		w.Kickoff = &persist.KickoffValue{RealStart: p.RealStart, Source: string(p.Kickoff.Source)}
	}
	for _, c := range p.TimePlayed {
		w.TimePlayed = append(w.TimePlayed, persist.PlayerTimePlayed{XUID: c.XUID, Seconds: c.After})
	}
	return w
}

// estimateInputs rend les arrivées que l'estimation lit : joueurs (pas les bots) dont l'heure
// d'arrivée est connue.
func estimateInputs(parts []Participant) []timeline.ParticipationT0Input {
	out := make([]timeline.ParticipationT0Input, 0, len(parts))
	for _, p := range parts {
		if p.FirstJoined == nil {
			continue
		}
		out = append(out, timeline.ParticipationT0Input{
			FirstJoinedTime:    p.FirstJoined.UTC(),
			PresentAtBeginning: p.PresentAtBeginning,
			IsBot:              !timeline.HumanXUID(p.XUID),
		})
	}
	return out
}

// sameInstant compare deux instants optionnels (deux absences sont égales).
func sameInstant(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}
