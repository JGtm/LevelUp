package main

import (
	"fmt"
	"math/rand"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// lives.go — les vies, le fil des morts, et leur appariement.

// lifeGapUS : au-dela de ce trou dans un meme slot, on ouvre une nouvelle vie. Un slot ne
// se reutilise pas a chaud ; 5 s est tres au-dela du pas de replication (~16 ms) et bien
// en deca du temps de reapparition mesure (mediane 8,0 s).
const lifeGapUS = 5_000_000

// matchWindowMS : ecart maximal accepte entre la fin d'une vie et une mort du fil. La
// mediane mesuree etant nulle, cette fenetre borne le bruit d'horloge, pas le signal.
const matchWindowMS = 150

// Life est une vie de biped : les positions d'un meme slot sans trou majeur.
type Life struct {
	Slot       uint32
	StartUS    int64
	EndUS      int64
	XUID       uint64 // identite lue dans le fil des morts (0 = non nommee)
	Gamertag   string
	DeathNamed bool
}

// Death est une mort du fil, lue dans le chunk highlight du film.
type Death struct {
	XUID      uint64
	Gamertag  string
	TimeMS    int64
	LifeIndex int
}

// buildLives regroupe les positions par slot puis par continuite temporelle.
func buildLives(pos []filmdec.BipedPosition) []Life {
	bySlot := map[uint32][]filmdec.BipedPosition{}
	for _, p := range pos {
		bySlot[p.Slot] = append(bySlot[p.Slot], p)
	}
	var slots []uint32
	for s := range bySlot {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	var out []Life
	for _, s := range slots {
		ps := bySlot[s]
		sort.Slice(ps, func(i, j int) bool { return ps[i].TimestampUS < ps[j].TimestampUS })
		start, last := int64(ps[0].TimestampUS), int64(ps[0].TimestampUS)
		for _, p := range ps[1:] {
			t := int64(p.TimestampUS)
			if t-last > lifeGapUS {
				out = append(out, Life{Slot: s, StartUS: start, EndUS: last})
				start = t
			}
			last = t
		}
		out = append(out, Life{Slot: s, StartUS: start, EndUS: last})
	}
	return out
}

// endsMS rend la fin de chaque vie, en millisecondes de l'horloge du film.
func endsMS(lives []Life) []int64 {
	out := make([]int64, len(lives))
	for i, l := range lives {
		out[i] = l.EndUS / 1000
	}
	return out
}

// bestOffset balaie le decalage entre l'horloge du fil des morts (relative au debut du
// match) et celle du film (absolue), et retient celui qui apparie le plus de morts.
//
// LE MAXIMUM EST UN PLATEAU : toute la largeur de la fenetre d'acceptation donne le meme
// compte. On retient son CENTRE — sinon le decalage retenu est au bord et tous les ecarts
// d'appariement valent la demi-fenetre, ce qui ferait passer un mauvais calage pour bon.
func bestOffset(lives []Life, deaths []Death) (int64, int) {
	ends := endsMS(lives)
	if len(ends) == 0 || len(deaths) == 0 {
		return 0, 0
	}
	lo, hi := ends[0], ends[0]
	for _, e := range ends {
		if e < lo {
			lo = e
		}
		if e > hi {
			hi = e
		}
	}
	bestN := -1
	var plateau []int64
	// La borne du balayage est la plage des fins de vie elle-meme : l'origine du fil des
	// morts est le debut du match, qui tombe forcement dedans.
	for off := lo - 60_000; off <= hi; off += 10 {
		if n := countMatches(ends, deaths, off); n > bestN {
			bestN, plateau = n, []int64{off}
		} else if n == bestN {
			plateau = append(plateau, off)
		}
	}
	return plateau[len(plateau)/2], bestN
}

// countMatches compte les morts appariables a une fin de vie, chaque vie servant une fois.
func countMatches(ends []int64, deaths []Death, off int64) int {
	used := make([]bool, len(ends))
	n := 0
	for _, d := range deaths {
		target := d.TimeMS + off
		bi, bd := -1, int64(matchWindowMS+1)
		for i, e := range ends {
			if used[i] {
				continue
			}
			delta := e - target
			if delta < 0 {
				delta = -delta
			}
			if delta < bd {
				bd, bi = delta, i
			}
		}
		if bi >= 0 {
			used[bi] = true
			n++
		}
	}
	return n
}

// matchDeaths apparie definitivement — glouton par ecart croissant, ce qui rend le
// resultat independant de l'ordre d'iteration — et pose l'identite de la victime sur la
// vie. Rend le nombre de vies nommees.
func matchDeaths(lives []Life, deaths []Death, off int64) int {
	ends := endsMS(lives)
	type pair struct {
		di, li int
		d      int64
	}
	var ps []pair
	for di, d := range deaths {
		target := d.TimeMS + off
		for li, e := range ends {
			delta := e - target
			if delta < 0 {
				delta = -delta
			}
			if delta <= matchWindowMS {
				ps = append(ps, pair{di, li, delta})
			}
		}
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].d < ps[j].d })
	usedD := make([]bool, len(deaths))
	usedL := make([]bool, len(lives))
	n := 0
	var deltas []float64
	for _, p := range ps {
		if usedD[p.di] || usedL[p.li] {
			continue
		}
		usedD[p.di], usedL[p.li] = true, true
		deaths[p.di].LifeIndex = p.li
		lives[p.li].XUID = deaths[p.di].XUID
		lives[p.li].Gamertag = deaths[p.di].Gamertag
		lives[p.li].DeathNamed = true
		deltas = append(deltas, float64(p.d))
		n++
	}
	sort.Float64s(deltas)
	fmt.Printf("ecart median d'appariement : %.0f ms (p90 %.0f, max %.0f)\n",
		median(deltas), quantile(deltas, 0.9), lastOf(deltas))
	return n
}

// controlMatch — TEMOIN : les memes morts, replacees uniformement au hasard sur la plage
// des fins de vie. Il donne le nombre d'appariements attendu SANS information temporelle.
func controlMatch(lives []Life, deaths []Death, off int64) int {
	ends := endsMS(lives)
	if len(ends) == 0 {
		return 0
	}
	lo, hi := ends[0], ends[0]
	for _, e := range ends {
		if e < lo {
			lo = e
		}
		if e > hi {
			hi = e
		}
	}
	r := rand.New(rand.NewSource(1))
	fake := make([]Death, len(deaths))
	for i := range fake {
		fake[i].TimeMS = lo + r.Int63n(hi-lo+1) - off
	}
	sort.Slice(fake, func(i, j int) bool { return fake[i].TimeMS < fake[j].TimeMS })
	return countMatches(ends, fake, off)
}

func lastOf(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	return v[len(v)-1]
}
