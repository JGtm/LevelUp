package narrative

import (
	"sort"

	"levelup/go-api/internal/games/canonical"
)

// IntensityProfile decrit l'intensite d'un match : densite d'events
// (kills + deaths + medals + assists) par bucket normalise [0..N-1].
//
// Contrairement a CadenceProfile (buckets a duree fixe en secondes),
// IntensityProfile decoupe le match en N buckets de duree relative pour
// permettre la comparaison entre matchs de durees differentes (heatmap
// match x phase).
//
// Le score d'un bucket est le nombre brut d'events. Le wrapper de rendu
// peut normaliser par bucket le plus actif (cf. ComputeIntensityNormalized).
type IntensityProfile struct {
	MatchID  string `json:"match_id"`
	NBuckets int    `json:"n_buckets"`
	Buckets  []int  `json:"buckets"` // events par bucket
	Total    int    `json:"total"`   // nombre total d'events filmes
	MaxTime  int64  `json:"max_time_ms"`
}

// ComputeMatchIntensityProfiles aggrege les highlight_events par match en
// N buckets normalises (par defaut 10).
//
// Tous les events avec MatchID non vide sont comptes (kill, death, medal,
// assist, finisher, clutch, first_kill, first_death). Les events sans
// MatchID sont ignores. La duree du match est inferee depuis le timestamp
// max observe (proxy, audit signale que match_registry fournit la duree
// canonique amont).
//
// Tri : par MatchID asc.
func ComputeMatchIntensityProfiles(
	events []canonical.HighlightEvent,
	nBuckets int,
) []IntensityProfile {
	if nBuckets <= 0 {
		nBuckets = 10
	}
	if len(events) == 0 {
		return nil
	}

	type acc struct {
		times   []int64
		maxTime int64
	}
	groups := make(map[string]*acc)
	for _, ev := range events {
		if ev.MatchID == "" {
			continue
		}
		a, ok := groups[ev.MatchID]
		if !ok {
			a = &acc{}
			groups[ev.MatchID] = a
		}
		a.times = append(a.times, ev.TimeMS)
		if ev.TimeMS > a.maxTime {
			a.maxTime = ev.TimeMS
		}
	}

	out := make([]IntensityProfile, 0, len(groups))
	for matchID, a := range groups {
		buckets := make([]int, nBuckets)
		// Si le match a une duree non nulle, distribuer ; sinon tout dans bucket 0.
		if a.maxTime <= 0 {
			buckets[0] = len(a.times)
		} else {
			for _, t := range a.times {
				b := int(int64(nBuckets) * t / a.maxTime)
				if b >= nBuckets {
					b = nBuckets - 1
				}
				if b < 0 {
					b = 0
				}
				buckets[b]++
			}
		}
		out = append(out, IntensityProfile{
			MatchID:  matchID,
			NBuckets: nBuckets,
			Buckets:  buckets,
			Total:    len(a.times),
			MaxTime:  a.maxTime,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].MatchID < out[j].MatchID
	})
	return out
}

// NormalizeIntensityBuckets renvoie les buckets normalises [0.0..1.0] :
// chaque valeur divisee par le max global du profil. Un profil sans event
// renvoie un slice de zeros.
//
// Utile pour le rendu heatmap (gradient continu sans biais inter-bucket).
func NormalizeIntensityBuckets(buckets []int) []float64 {
	if len(buckets) == 0 {
		return nil
	}
	maxV := 0
	for _, v := range buckets {
		if v > maxV {
			maxV = v
		}
	}
	out := make([]float64, len(buckets))
	if maxV == 0 {
		return out
	}
	for i, v := range buckets {
		out[i] = float64(v) / float64(maxV)
	}
	return out
}
