// Package kickoff résout, range et rattrape le coup d'envoi des matchs d'une base partagée et le
// temps de jeu de leurs joueurs (ADR 0050).
//
// # UNE RÉSOLUTION
//
// [Resolve] est le seul point où se décide ce qu'un match doit porter : le coup d'envoi par
// `timeline.ResolveKickoff` (rejeu, sinon estimation, sinon inconnu), puis le temps de jeu de
// chaque joueur par `timeline.PlayedSeconds` sur CE coup d'envoi. Les trois écrivains l'appellent
// : la synchronisation (lignes en mémoire avant leur insertion, [ApplyToRows]), la
// publication d'un rejeu (`replayartifacts.Deriver`) et le rattrapage ([Backfill]).
//
// # CE QUE LE REJEU DIT, EN TROIS ÉTATS ([Film])
//
// Un rejeu PRÉSENT qui publie un coup d'envoi donne la source `film` ; un rejeu PRÉSENT qui n'en
// publie pas dit qu'il n'y en a pas : la source suivante décide. Un rejeu ABSENT (jamais généré,
// purgé par la rétention) ou ILLISIBLE ne dit rien de neuf : un match dont le coup d'envoi rangé est
// de source `film` le garde tel quel (coup d'envoi, source et temps de jeu), les autres se résolvent
// normalement.
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

// FilmState dit ce que la lecture du rejeu d'un match a donné.
type FilmState int

const (
	// FilmAbsent : aucun rejeu rangé (jamais généré, purgé, ou titre sans rejeu).
	FilmAbsent FilmState = iota
	// FilmUnreadable : un rejeu est rangé mais ne se lit pas (fichier verrouillé, remplacé, abîmé).
	FilmUnreadable
	// FilmRead : le rejeu est lu ; son coup d'envoi publié peut manquer.
	FilmRead
)

// Film est ce que le rejeu d'un match dit de son coup d'envoi.
type Film struct {
	State FilmState
	// T0Ms : le coup d'envoi publié par le document (`t0FilmMs`), en ms depuis le début canonique ;
	// nil quand le document n'en publie pas. Sans sens hors de FilmRead.
	T0Ms *int64
}

// FilmPublished : un rejeu lu qui publie (ou non, t0Ms nil) un coup d'envoi.
func FilmPublished(t0Ms *int64) Film { return Film{State: FilmRead, T0Ms: t0Ms} }

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
	// FilmKept : le coup d'envoi rangé, de source `film`, est conservé parce que le rejeu n'a rien
	// dit de neuf (absent ou illisible) ; le plan ne change rien.
	FilmKept bool
}

// ValueChanged dit si l'instant du coup d'envoi change (inconnu compris).
func (p Plan) ValueChanged() bool { return !sameInstant(p.RealStart, p.Before) }

// SourceChanged dit si la source du coup d'envoi change.
func (p Plan) SourceChanged() bool { return p.Kickoff.Source != p.BeforeSource }

// KickoffChanged dit si le registre du match est à réécrire.
func (p Plan) KickoffChanged() bool { return p.ValueChanged() || p.SourceChanged() }

// Unchanged dit si le match porte déjà tout ce que la résolution décide.
func (p Plan) Unchanged() bool { return !p.KickoffChanged() && len(p.TimePlayed) == 0 }

// Resolve décide ce que le match doit porter, d'après ce que son rejeu dit (f). Fonction pure.
func Resolve(m Match, f Film) Plan {
	if f.State != FilmRead && m.Source == timeline.KickoffSourceFilm {
		return keptFilm(m)
	}
	var filmT0 *int64
	if f.State == FilmRead {
		filmT0 = f.T0Ms
	}
	k := timeline.ResolveKickoff(timeline.KickoffInput{
		StartUTC: m.StartUTC, FilmT0Ms: filmT0, DurationSeconds: m.DurationSeconds,
		Participations: estimateInputs(m.Participants),
	})
	p := Plan{
		MatchID: m.MatchID, Kickoff: k, RealStart: k.RealStart(m.StartUTC),
		Before: m.RealStart, BeforeSource: m.Source,
	}
	tl, hasWindow := timeline.GameplayTimeline(m.StartUTC, m.DurationSeconds, k)
	for _, part := range m.Participants {
		if part.FirstJoined == nil {
			continue // présence inconnue : la valeur rangée reste
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

// keptFilm : le plan d'un match dont le coup d'envoi rangé, de source `film`, reste tel quel.
func keptFilm(m Match) Plan {
	k := timeline.Kickoff{Source: timeline.KickoffSourceFilm}
	if m.RealStart != nil {
		k.OffsetMs = m.RealStart.Sub(m.StartUTC).Milliseconds()
	}
	return Plan{
		MatchID: m.MatchID, Kickoff: k, RealStart: m.RealStart,
		Before: m.RealStart, BeforeSource: m.Source, FilmKept: true,
	}
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
