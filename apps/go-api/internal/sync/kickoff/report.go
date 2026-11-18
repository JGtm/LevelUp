package kickoff

import (
	"sort"

	"levelup/go-api/internal/analysis/timeline"
)

// Report est le bilan d'un rattrapage : ce que la résolution décide, ce qui change, ce qui a été écrit.
type Report struct {
	// Matches : matchs résolus ; BySource : leur source après résolution ; Estimates : issues de
	// l'estimation quand elle a été jouée.
	Matches   int
	BySource  map[timeline.KickoffSource]int
	Estimates map[timeline.T0Quality]int
	// Artifacts : rejeux présents ; UnreadableArtifacts : rejeux présents mais illisibles.
	Artifacts           int
	UnreadableArtifacts int
	// ValueChanged / SourceChanged : matchs dont l'instant ou la source du coup d'envoi change ;
	// BecameUnknown : matchs dont le coup d'envoi rangé disparaît (estimation refusée, sans rejeu) ;
	// KickoffShiftsMs : écart nouveau − ancien des matchs dont les deux valeurs existent.
	ValueChanged    int
	SourceChanged   int
	BecameUnknown   int
	KickoffShiftsMs []int64
	// MatchesWithPlayerChanges / PlayersChanged : matchs et joueurs dont le temps de jeu change ;
	// PlayerGapsSeconds : écart nouveau − ancien (les valeurs absentes avant sont comptées dans
	// PlayersFromNothing).
	MatchesWithPlayerChanges int
	PlayersChanged           int
	PlayersFromNothing       int
	PlayerGapsSeconds        []int64
	// KeptFilm : matchs de source `film` gardés tels quels, leur rejeu absent ou illisible.
	// FilmOutOfBounds : rejeux dont le coup d'envoi publié tombait hors de la fenêtre du match.
	KeptFilm        int
	FilmOutOfBounds int
	// Missing : matchs disparus du registre entre la liste et leur lot (rien à rattraper).
	Missing int
	// Written : matchs écrits ; Failures : matchs en échec (écriture refusée, ou rejeu illisible
	// sur un coup d'envoi rangé `film` ou inconnu) ; FailedExample : le premier d'entre eux.
	Written       int
	Failures      int
	FailedExample string
}

// fail inscrit un match en échec.
func (r *Report) fail(matchID string) {
	if r.Failures == 0 {
		r.FailedExample = matchID
	}
	r.Failures++
}

func newReport() Report {
	return Report{BySource: map[timeline.KickoffSource]int{}, Estimates: map[timeline.T0Quality]int{}}
}

// add verse un plan au bilan.
func (r *Report) add(p Plan) {
	r.Matches++
	r.BySource[p.Kickoff.Source]++
	if p.FilmKept {
		r.KeptFilm++
	}
	if p.Kickoff.FilmOutOfBounds {
		r.FilmOutOfBounds++
	}
	if p.Kickoff.Estimate != "" {
		r.Estimates[p.Kickoff.Estimate]++
	}
	if p.SourceChanged() {
		r.SourceChanged++
	}
	if p.ValueChanged() {
		r.ValueChanged++
		switch {
		case p.RealStart == nil:
			r.BecameUnknown++
		case p.Before != nil:
			r.KickoffShiftsMs = append(r.KickoffShiftsMs, p.RealStart.Sub(*p.Before).Milliseconds())
		}
	}
	if len(p.TimePlayed) > 0 {
		r.MatchesWithPlayerChanges++
	}
	for _, c := range p.TimePlayed {
		r.PlayersChanged++
		if c.Before == nil {
			r.PlayersFromNothing++
			continue
		}
		r.PlayerGapsSeconds = append(r.PlayerGapsSeconds, int64(c.After-*c.Before))
	}
}

// Distribution résume une série : effectif, minimum, centiles 5 / 50 / 95, maximum, et le nombre
// de valeurs dont la valeur absolue ne dépasse pas la tolérance demandée.
type Distribution struct {
	N                      int
	Min, P5, P50, P95, Max int64
	Within                 int
}

// Summarize calcule la distribution d'une série, sans la modifier ; tol borne Within.
func Summarize(v []int64, tol int64) Distribution {
	d := Distribution{N: len(v)}
	if len(v) == 0 {
		return d
	}
	s := append([]int64(nil), v...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	at := func(q float64) int64 { return s[int(q*float64(len(s)-1))] }
	d.Min, d.P5, d.P50, d.P95, d.Max = s[0], at(0.05), at(0.5), at(0.95), s[len(s)-1]
	for _, x := range s {
		if x >= -tol && x <= tol {
			d.Within++
		}
	}
	return d
}
