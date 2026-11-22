// Package timeline construit des domain.MatchTimeline depuis les rows DB et résout le coup
// d'envoi d'un match (kickoff.go).
//
// # UNE LECTURE DE LA CHRONOLOGIE DE JEU
//
// [ForMatch] est la SEULE règle qui lit le coup d'envoi rangé d'un match : coup d'envoi connu =
// la chronologie de jeu part de lui ; coup d'envoi inconnu (nil : Halo 5, estimation refusée,
// donnée absente) = elle part du début du match, et la durée de jeu vaut la durée du match. Les
// constructeurs ci-dessous et l'écriture ([GameplayTimeline]) la délèguent ; aucun lecteur ne
// remplace un coup d'envoi absent par zéro de son côté. Côté SQL, la même règle s'écrit une fois
// dans `sqlfrag.SQLT0Ms` (le coup d'envoi) et `sqlfrag.SQLGameplayDurationMs` (la durée de jeu).
//
// Pas de dépendance DB / HTTP : fonctions pures testables en isolation.
package timeline

import (
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// ForMatch rend la chronologie de jeu d'un match depuis sa durée (ms, countdown compris) et son
// coup d'envoi rangé (t0Ms, ms depuis le début canonique ; nil = inconnu). startUTC peut être
// la valeur zéro quand l'appelant n'a besoin que des durées et des recalages d'événements.
// Les valeurs négatives sont ramenées à 0 (domain.NewMatchTimelineAt).
func ForMatch(startUTC time.Time, durationMs int64, t0Ms *int64) domain.MatchTimeline {
	var t0 int64
	if t0Ms != nil {
		t0 = *t0Ms
	}
	return domain.NewMatchTimelineAt(startUTC, durationMs, t0)
}

// BuildTimelinesFromPlayerMatches indexe une MatchTimeline par match_id depuis les rows
// canoniques chargées par un service (Timeseries, Squad, MatchView). Le coup d'envoi vient de
// Summary.T0Ms (SQLT0Ms) ; nil = inconnu (voir [ForMatch]).
func BuildTimelinesFromPlayerMatches(rows []canonical.PlayerMatchRow) map[string]domain.MatchTimeline {
	out := make(map[string]domain.MatchTimeline, len(rows))
	for _, r := range rows {
		var durMs int64
		if r.Summary.DurationSeconds != nil {
			durMs = int64(*r.Summary.DurationSeconds) * 1000
		}
		out[r.Summary.MatchID] = ForMatch(time.Time{}, durMs, r.Summary.T0Ms)
	}
	return out
}

// BuildTimelinesFromSquadRows indexe une MatchTimeline par match_id depuis les SquadMatchRow
// (page Escouade / TeammatesService, Q30). Le coup d'envoi vient de SquadMatchRow.T0Ms ;
// nil = inconnu. Plusieurs rows peuvent partager le même match_id (une par coéquipier) : elles
// portent le même coup d'envoi, la dernière écrasant les précédentes sans effet observable.
func BuildTimelinesFromSquadRows(rows []domain.SquadMatchRow) map[string]domain.MatchTimeline {
	out := make(map[string]domain.MatchTimeline, len(rows))
	for _, r := range rows {
		out[r.MatchID] = ForMatch(time.Time{}, int64(r.DurationSeconds)*1000, r.T0Ms)
	}
	return out
}

// GameplayDurationsMS projette une map de MatchTimeline en map match_id → durée
// de gameplay en ms (countdown retranché). Sert de dénominateur canonique aux
// builders intensité / cadence (au lieu d'inférer la fin depuis les events).
// Les durées ≤ 0 sont omises (le builder retombe alors sur son proxy maxTime).
func GameplayDurationsMS(timelines map[string]domain.MatchTimeline) map[string]int64 {
	if len(timelines) == 0 {
		return nil
	}
	out := make(map[string]int64, len(timelines))
	for id, tl := range timelines {
		if d := tl.GameplayDurationMs(); d > 0 {
			out[id] = d
		}
	}
	return out
}
