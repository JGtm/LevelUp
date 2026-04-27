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
// Les profiles retournes sont tries par MatchID asc puis XUID asc (stabilite).
func ComputeCadenceProfiles(
	events []canonical.HighlightEvent,
	squad []string,
	phaseSeconds int,
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
		matchEnd := matchMaxTime[k.matchID]
		bucketCount := int(matchEnd/phaseMS) + 1
		if bucketCount < 1 {
			bucketCount = 1
		}
		buckets := make([]int, bucketCount)
		for _, t := range a.killTimes {
			b := int(t / phaseMS)
			if b >= bucketCount {
				b = bucketCount - 1
			}
			if b < 0 {
				b = 0
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
