package main

import (
	"fmt"
	"math"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// bridge.go — LE PONT index de tir -> XUID, et la couverture qui en decoule.
//
// POURQUOI CE PONT EXISTE ENCORE APRES LE REPLI. Le fil des morts nomme les vies par XUID,
// qui est l'identite. L'event de tir, lui, porte un INDEX de joueur (0..7) — un ordre
// interne au film. Les deux numerotations n'ont aucune raison de coincider, et c'est
// exactement l'erreur que la decision n°2 du plan interdit de refaire.
//
// CE QUI CHANGE PAR RAPPORT AU VOTE QU'ON REMPLACE. Le vote de owners.go devait nommer
// 99 vies avec 70 lancers de grenade, soit un probleme sous-determine par construction —
// d'ou les 26 slots couverts. Ici le probleme porte sur HUIT valeurs, avec plusieurs
// centaines d'events, et il est resolu GLOBALEMENT : on cherche la permutation de cout
// minimal sur les 8! = 40 320 affectations, pas un maximum local par slot. La marge entre
// la meilleure et la deuxieme meilleure permutation est publiee : c'est elle qui dit si la
// solution est tranchee ou si deux lectures se valent.

// bridgeResult porte le pont resolu et de quoi le juger.
type bridgeResult struct {
	IndexToXUID map[int]uint64
	XUIDs       []uint64
	Indices     []int
	Cost        [][]int // cout[i][j] = events de l'index i tombant HORS des vies du xuid j
	Best        int
	Second      int
	Placeable   int // events tombant dans exactement une vie de leur xuid
}

// solveBridge resout l'affectation index -> xuid par cout minimal global.
//
// LE COUT est un compte de CONTRADICTIONS, pas une ressemblance : cout[i][j] = nombre
// d'events de l'index i dont l'instant ne tombe dans AUCUNE vie du xuid j. Un joueur ne
// tire pas quand il est mort ; la bonne affectation est donc celle qui contredit le moins.
func solveBridge(lives []Life, fire []filmdec.FireEvent) bridgeResult {
	xs := namedXUIDs(lives)
	idxs := firingIndices(fire)
	res := bridgeResult{IndexToXUID: map[int]uint64{}, XUIDs: xs, Indices: idxs}
	if len(xs) == 0 || len(idxs) == 0 {
		return res
	}
	spans := lifeSpansByXUID(lives)
	res.Cost = make([][]int, len(idxs))
	for a, i := range idxs {
		res.Cost[a] = make([]int, len(xs))
		for b, x := range xs {
			miss := 0
			for _, e := range fire {
				if e.FilmIndex != i {
					continue
				}
				if !insideAny(spans[x], int64(e.TimestampUS)) {
					miss++
				}
			}
			res.Cost[a][b] = miss
		}
	}
	perm, best, second := bestAssignment(res.Cost)
	res.Best, res.Second = best, second
	for a, b := range perm {
		if a < len(idxs) && b < len(xs) {
			res.IndexToXUID[idxs[a]] = xs[b]
		}
	}
	return res
}

// namedXUIDs rend les xuids nommes par le fil des morts, en ordre stable.
func namedXUIDs(lives []Life) []uint64 {
	seen := map[uint64]bool{}
	var out []uint64
	for _, l := range lives {
		if l.XUID != 0 && !seen[l.XUID] {
			seen[l.XUID] = true
			out = append(out, l.XUID)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// firingIndices rend les index de joueur presents dans les events de tir, en ordre stable.
func firingIndices(fire []filmdec.FireEvent) []int {
	seen := map[int]bool{}
	var out []int
	for _, e := range fire {
		if !seen[e.FilmIndex] {
			seen[e.FilmIndex] = true
			out = append(out, e.FilmIndex)
		}
	}
	sort.Ints(out)
	return out
}

// span est l'intervalle temporel d'une vie, en microsecondes.
type span struct{ from, to int64 }

// lifeSpansByXUID regroupe les intervalles de vie par identite.
func lifeSpansByXUID(lives []Life) map[uint64][]span {
	out := map[uint64][]span{}
	for _, l := range lives {
		if l.XUID == 0 {
			continue
		}
		out[l.XUID] = append(out[l.XUID], span{l.StartUS, l.EndUS})
	}
	return out
}

func insideAny(ss []span, t int64) bool {
	for _, s := range ss {
		if t >= s.from && t <= s.to {
			return true
		}
	}
	return false
}

// bestAssignment enumere les permutations et rend celle de cout total minimal, plus le
// cout du deuxieme meilleur. L'enumeration exhaustive est EXACTE et sans dependance
// d'ordre, la ou un glouton departage les ex aequo par l'ordre d'iteration.
func bestAssignment(cost [][]int) (perm []int, best, second int) {
	n := len(cost)
	if n == 0 {
		return nil, 0, 0
	}
	m := len(cost[0])
	cur := make([]int, m)
	for i := range cur {
		cur[i] = i
	}
	best, second = math.MaxInt32, math.MaxInt32
	var bestPerm []int
	var rec func(k int)
	rec = func(k int) {
		if k == m {
			c := 0
			for i := 0; i < n && i < m; i++ {
				c += cost[i][cur[i]]
			}
			switch {
			case c < best:
				second, best = best, c
				bestPerm = append([]int(nil), cur...)
			case c < second:
				second = c
			}
			return
		}
		for i := k; i < m; i++ {
			cur[k], cur[i] = cur[i], cur[k]
			rec(k + 1)
			cur[k], cur[i] = cur[i], cur[k]
		}
	}
	rec(0)
	return bestPerm, best, second
}

// reportBridge publie le pont et sa marge.
func reportBridge(b bridgeResult) {
	if len(b.IndexToXUID) == 0 {
		fmt.Println("pont non resolu : aucune vie nommee ou aucun event de tir.")
		return
	}
	fmt.Printf("index de tir presents : %v\n", b.Indices)
	fmt.Printf("xuids nommes par le fil des morts : %d\n", len(b.XUIDs))
	fmt.Printf("cout de la MEILLEURE permutation : %d contradictions ; DEUXIEME : %d (marge %d)\n",
		b.Best, b.Second, b.Second-b.Best)
	var ks []int
	for k := range b.IndexToXUID {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	for _, k := range ks {
		fmt.Printf("  index %d -> xuid %d\n", k, b.IndexToXUID[k])
	}
}

// reportCoverage publie ce que le repli rattache, et sur quel denominateur.
//
// LE DENOMINATEUR EST LE POINT. Publier « N tirs rattaches » sans dire combien existent
// est precisement le defaut que l'etape 4 doit corriger ; on l'anticipe ici parce que le
// gate de l'etape 2 porte sur un RAPPORT, pas sur un compte.
func reportCoverage(lives []Life, fire []filmdec.FireEvent, b bridgeResult, filmDir string) {
	spans := lifeSpansByXUID(lives)
	placed, ambiguous, noXUID, outside := 0, 0, 0, 0
	for _, e := range fire {
		x, ok := b.IndexToXUID[e.FilmIndex]
		if !ok {
			noXUID++
			continue
		}
		n := 0
		for _, l := range lives {
			if l.XUID != x {
				continue
			}
			if int64(e.TimestampUS) >= l.StartUS && int64(e.TimestampUS) <= l.EndUS {
				n++
			}
		}
		switch {
		case n == 1:
			placed++
		case n > 1:
			ambiguous++
		default:
			outside++
		}
	}
	total := len(fire)
	fmt.Printf("events de tir LONGS decodes : %d\n", total)
	fmt.Printf("  rattaches (une seule vie de leur identite couvre l'instant) : %d (%.1f %%)\n",
		placed, pct(placed, total))
	fmt.Printf("  rejetes — index sans identite   : %d\n", noXUID)
	fmt.Printf("  rejetes — aucune vie ne couvre  : %d\n", outside)
	fmt.Printf("  rejetes — plusieurs vies        : %d\n", ambiguous)
	if placed+noXUID+outside+ambiguous != total {
		fmt.Printf("  ATTENTION : la somme ne fait pas le total — il y a une fuite.\n")
	}
	_ = spans
	_ = filmDir
}
