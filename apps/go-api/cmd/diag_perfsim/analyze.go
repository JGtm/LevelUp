package main

// analyze.go — exploitation des régimes simulés : purge des notes orphelines,
// concordance avec les notes stockées, sélection des matchs témoins.

import (
	"math"
	"sort"
	"time"

	"levelup/go-api/internal/sync/skill"
)

// ── Purge (D-D) ─────────────────────────────────────────────────────────────

type purgeReport struct {
	StoredScored   int
	Kept           int
	DNF            int
	Excluded       int
	BelowThreshold int
	OutOfUniverse  int
	ByChainStored  map[string]int
}

func (p purgeReport) Total() int {
	return p.DNF + p.Excluded + p.BelowThreshold + p.OutOfUniverse
}

// buildPurge classe chaque note DÉJÀ stockée : conservée, ou purgée avec sa
// cause. Ordre de priorité des causes : DNF > exclu > sous-seuil > hors univers.
func buildPurge(res *playerResult) purgeReport {
	out := purgeReport{ByChainStored: map[string]int{}}
	outcome := make(map[string]int, len(res.Universe))
	excluded := make(map[string]bool, len(res.Universe))
	for i := range res.Universe {
		m := &res.Universe[i]
		outcome[m.MatchID] = m.Outcome
		excluded[m.MatchID] = m.Excluded
	}
	ref := res.NewByW[ospmReference]

	for _, s := range res.Stored {
		if s.Score == nil {
			continue
		}
		out.StoredScored++
		oc, known := outcome[s.MatchID]
		sm := ref.Matches[s.MatchID]
		cause := ""
		switch {
		case !known:
			cause = "hors_univers"
			out.OutOfUniverse++
		case oc == 4:
			cause = "dnf"
			out.DNF++
		case excluded[s.MatchID] || s.Excluded:
			cause = "exclu"
			out.Excluded++
		case sm == nil || sm.Note == nil:
			cause = "sous_seuil"
			out.BelowThreshold++
		default:
			out.Kept++
		}
		if cause != "" {
			label := s.Chain
			if label == "" {
				label = "(NULL)"
			}
			out.ByChainStored[label]++
		}
	}
	return out
}

// ── Concordance réplique / stocké ───────────────────────────────────────────

type concordChain struct {
	N          int
	MedStored  float64
	MedReplica float64
	MeanAbs    float64
	Within1    int
	deltas     []float64
	stored     []float64
	replica    []float64
}

type concordance struct {
	Paired  int
	MeanAbs float64
	P90Abs  float64
	Within1 int
	ByChain map[string]*concordChain

	// Sonde : effet mesuré du retrait d'UNE métrique de poids 0.06
	// (dpm_damage), proxy empirique de l'absence de medal_exploit.
	DropN       int
	DropMeanAbs float64
	DropP90Abs  float64
	DropMaxAbs  float64
}

// buildConcordance compare la réplique du régime ACTUEL aux notes stockées, sur
// les seuls matchs dont la chaîne stockée est non vide ET identique à la chaîne
// recalculée (sinon la note stockée vient d'une autre population de référence).
func buildConcordance(res *playerResult) concordance {
	out := concordance{ByChain: map[string]*concordChain{}}
	var all []float64
	for _, s := range res.Stored {
		sm := res.Actual.Matches[s.MatchID]
		if s.Score == nil || s.Chain == "" || sm == nil || sm.Note == nil || sm.Chain != s.Chain {
			continue
		}
		d := math.Abs(*sm.Note - *s.Score)
		out.Paired++
		all = append(all, d)
		if d <= 1.0 {
			out.Within1++
		}
		cc := out.ByChain[s.Chain]
		if cc == nil {
			cc = &concordChain{}
			out.ByChain[s.Chain] = cc
		}
		cc.N++
		cc.deltas = append(cc.deltas, d)
		cc.stored = append(cc.stored, *s.Score)
		cc.replica = append(cc.replica, *sm.Note)
		if d <= 1.0 {
			cc.Within1++
		}
	}
	out.MeanAbs, out.P90Abs = meanAndP90(all)
	for _, cc := range out.ByChain {
		cc.MeanAbs, _ = meanAndP90(cc.deltas)
		cc.MedStored = quantile(cc.stored, 0.5)
		cc.MedReplica = quantile(cc.replica, 0.5)
	}
	fillDropProbe(res, &out)
	return out
}

// fillDropProbe mesure |note(ACTUEL) − note(ACTUEL sans dpm_damage)| : ordre de
// grandeur empirique de l'absence d'une métrique de poids 0.06 (medal_exploit).
func fillDropProbe(res *playerResult, out *concordance) {
	var deltas []float64
	for id, sm := range res.Actual.Matches {
		other := res.ActualNoDmg.Matches[id]
		if sm.Note == nil || other == nil || other.Note == nil {
			continue
		}
		d := math.Abs(*sm.Note - *other.Note)
		deltas = append(deltas, d)
		out.DropMaxAbs = math.Max(out.DropMaxAbs, d)
	}
	out.DropN = len(deltas)
	out.DropMeanAbs, out.DropP90Abs = meanAndP90(deltas)
}

// ── Témoins ─────────────────────────────────────────────────────────────────

// combatKeys — percentiles purement « combat » servant à qualifier un match
// d'écrasé (ils excluent délibérément pspm, apm et ospm).
var combatKeys = []string{
	skill.MetricKeyKPM, skill.MetricKeyKDA, skill.MetricKeyAccuracy,
	skill.MetricKeyDPMDamage, skill.MetricKeyKillsVsExpected,
	skill.MetricKeyOffensiveConv,
}

type witness struct {
	Gamertag   string
	MatchID    string
	Start      time.Time
	Pair       string
	Chain      string
	Kills      float64
	Deaths     float64
	PCombat    float64
	POspm      float64
	NoteActual float64
	NoteNew    map[float64]float64
}

func (w witness) gap() float64 { return w.POspm - w.PCombat }

// collectWitnesses rassemble, sur les 4 joueurs, les matchs de chaîne objectif
// notés sous les deux régimes et porteurs d'un percentile ospm.
func collectWitnesses(results []*playerResult) []witness {
	var out []witness
	for _, pr := range results {
		ref := pr.NewByW[ospmReference]
		for id, sm := range ref.Matches {
			old := pr.Actual.Matches[id]
			if sm.Note == nil || old == nil || old.Note == nil || !isObjectiveChain(sm.Chain) {
				continue
			}
			pOspm, ok := sm.Pct[metricKeyOSPM]
			if !ok {
				continue
			}
			w := witness{
				Gamertag: pr.Player.Gamertag, MatchID: id, Start: sm.Start, Pair: sm.Pair,
				Chain: sm.Chain, Kills: sm.Kills, Deaths: sm.Deaths,
				PCombat: meanPct(sm.Pct, combatKeys), POspm: pOspm,
				NoteActual: *old.Note, NoteNew: map[float64]float64{},
			}
			for _, wt := range ospmVariants {
				if v := pr.NewByW[wt].Matches[id]; v != nil && v.Note != nil {
					w.NoteNew[wt] = *v.Note
				}
			}
			out = append(out, w)
		}
	}
	return out
}

// topWitnesses rend les 5 meilleurs témoins. active=true : écrasé au combat mais
// actif à l'objectif. active=false : contre-témoin (fort au combat, absent de
// l'objectif) — il vérifie que la métrique ne récompense pas l'absence de combat.
func topWitnesses(all []witness, active bool) []witness {
	pool := make([]witness, len(all))
	copy(pool, all)
	sort.Slice(pool, func(i, j int) bool {
		if active {
			return pool[i].gap() > pool[j].gap()
		}
		return pool[i].gap() < pool[j].gap()
	})
	var strict []witness
	for _, w := range pool {
		if active && w.PCombat <= 40 && w.POspm >= 60 {
			strict = append(strict, w)
		}
		if !active && w.PCombat >= 60 && w.POspm <= 40 {
			strict = append(strict, w)
		}
	}
	if len(strict) < 5 {
		strict = pool
	}
	if len(strict) > 5 {
		strict = strict[:5]
	}
	return strict
}

func meanPct(pct map[string]float64, keys []string) float64 {
	sum, n := 0.0, 0
	for _, k := range keys {
		if v, ok := pct[k]; ok {
			sum += v
			n++
		}
	}
	if n == 0 {
		return 50.0
	}
	return sum / float64(n)
}

// ── Statistiques ────────────────────────────────────────────────────────────

// quantile interpole linéairement le quantile q d'un échantillon (copie triée).
func quantile(values []float64, q float64) float64 {
	if len(values) == 0 {
		return math.NaN()
	}
	s := make([]float64, len(values))
	copy(s, values)
	sort.Float64s(s)
	if len(s) == 1 {
		return s[0]
	}
	pos := q * float64(len(s)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return s[lo]
	}
	return s[lo] + (s[hi]-s[lo])*(pos-float64(lo))
}

func meanAndP90(values []float64) (mean, p90 float64) {
	if len(values) == 0 {
		return math.NaN(), math.NaN()
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values)), quantile(values, 0.90)
}
