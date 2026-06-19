package narrative

import (
	"sort"

	"levelup/go-api/internal/games/canonical"
)

// CadenceProfile decrit la cadence de kills d'un joueur sur un match :
// repartition des kills par phase de N secondes (default 60).
//
// Le slice Buckets a la longueur du match en buckets : si le match dure 600s
// avec phaseSeconds=60, on a 10 buckets. Chaque bucket = nombre de kills sur
// la phase. La duree exacte du match n'est pas determinee ici (l'audit
// signale que le service amont fournit la duree max via match_registry) ;
// CadenceProfile.Buckets se base sur le timestamp max observe dans les events
// du match comme proxy de duree.
type CadenceProfile struct {
	XUID         string `json:"xuid"`
	MatchID      string `json:"match_id"`
	PhaseSeconds int    `json:"phase_seconds"`
	Buckets      []int  `json:"buckets"` // kills par phase
	TotalKills   int    `json:"total_kills"`
}

// ComputeCadenceProfiles aggrege les kill events par xuid et par phase de
// PhaseSeconds. Renvoie 1 profil par (xuid in squad × match_id observe).
//
// phaseSeconds : default 60 si <= 0.
//
// Les events sans EventKill ou KillerXUID nil sont ignores. Les events hors
// squad (KillerXUID pas dans squad) sont aussi ignores.
//
// Les events au TimeMS negatif (frag de countdown pre-T0, apres
// timeline.CorrectEvents) sont ignores — cohérent avec first_events.go (badges
// "premier frag / premiere mort"). Sans ce garde, un frag de countdown serait
// replie dans le bucket 0 (phase_00), divergeant des badges/evtList.
//
// Les profiles retournes sont tries par MatchID asc puis XUID asc (stabilite).
//
// gameplayDurationsMS (nil-safe) fournit la VRAIE durée de gameplay par match
// (countdown retranché, source match_registry). Quand présente, elle fixe le
// nombre de phases ; sinon fallback sur le timestamp de kill max observé (proxy).
func ComputeCadenceProfiles(
	events []canonical.HighlightEvent,
	squad []string,
	phaseSeconds int,
	gameplayDurationsMS map[string]int64,
) []CadenceProfile {
	if phaseSeconds <= 0 {
		phaseSeconds = 60
	}
	if len(events) == 0 || len(squad) == 0 {
		return nil
	}
	squadSet := make(map[string]bool, len(squad))
	for _, x := range squad {
		squadSet[x] = true
	}

	type key struct{ matchID, xuid string }
	type acc struct {
		killTimes []int64 // ms
		maxTime   int64
	}
	groups := make(map[key]*acc)
	matchMaxTime := make(map[string]int64)

	for _, ev := range events {
		if ev.MatchID == "" || ev.EventType != string(canonical.EventKill) {
			continue
		}
		if ev.KillerXUID == nil || !squadSet[*ev.KillerXUID] {
			continue
		}
		if ev.TimeMS < 0 {
			// Frag de countdown (pre-T0 apres CorrectEvents) — ignore, cohérent
			// avec first_events.go. Ne pas le replier dans phase_00.
			continue
		}
		k := key{ev.MatchID, *ev.KillerXUID}
		a, ok := groups[k]
		if !ok {
			a = &acc{}
			groups[k] = a
		}
		a.killTimes = append(a.killTimes, ev.TimeMS)
		if ev.TimeMS > a.maxTime {
			a.maxTime = ev.TimeMS
		}
		if ev.TimeMS > matchMaxTime[ev.MatchID] {
			matchMaxTime[ev.MatchID] = ev.TimeMS
		}
	}

	phaseMS := int64(phaseSeconds) * 1000
	out := make([]CadenceProfile, 0, len(groups))
	for k, a := range groups {
		// Fin de match : durée gameplay canonique si fournie, sinon kill max.
		matchEnd := matchMaxTime[k.matchID]
		if d := gameplayDurationsMS[k.matchID]; d > 0 {
			matchEnd = d
		}
		bucketCount := int(matchEnd/phaseMS) + 1
		if bucketCount < 1 {
			bucketCount = 1
		}
		buckets := make([]int, bucketCount)
		for _, t := range a.killTimes {
			// t >= 0 garanti (frags pre-T0 ignores ci-dessus) → b >= 0.
			b := int(t / phaseMS)
			if b >= bucketCount {
				b = bucketCount - 1
			}
			buckets[b]++
		}
		out = append(out, CadenceProfile{
			XUID:         k.xuid,
			MatchID:      k.matchID,
			PhaseSeconds: phaseSeconds,
			Buckets:      buckets,
			TotalKills:   len(a.killTimes),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].MatchID != out[j].MatchID {
			return out[i].MatchID < out[j].MatchID
		}
		return out[i].XUID < out[j].XUID
	})
	return out
}
