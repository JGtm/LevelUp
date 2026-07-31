package main

import (
	"encoding/binary"
	"fmt"
	"levelup/go-api/internal/analysis"
	"os"
	"sort"
)

const bin = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/dmgcapture_605.bin`
const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const base = 0xEC500000
const step = 0x10002

var h32 = map[uint32]string{}
var piXuid = map[int]uint64{0: 2535467794760703, 1: 2535437947245250, 2: 2533274823110022, 3: 2533274980284321, 4: 2533274815845110, 5: 2535444178793711, 6: 2533274882097883, 7: 2533274826120416}

func xpi(x uint64) int {
	for p, xu := range piXuid {
		if xu == x {
			return p
		}
	}
	return -1
}
func atkPi(h uint32) int {
	if h < base {
		return -1
	}
	d := h - base
	if d%step != 0 {
		return -1
	}
	p := int(d / step)
	if p < 0 || p > 7 {
		return -1
	}
	return p
}
func mustRead(p string) []byte { b, _ := os.ReadFile(p); return b }

type rec struct {
	atkPi    int
	vic, fam uint32
	tsc      uint64
}

func main() {
	for id, n := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = n
	}
	d, _ := os.ReadFile(bin)
	var recs []rec
	for o := 0; o+32 <= len(d); o += 32 {
		a := binary.LittleEndian.Uint32(d[o:])
		if a == 0 {
			continue
		}
		recs = append(recs, rec{atkPi(a), binary.LittleEndian.Uint32(d[o+4:]), binary.LittleEndian.Uint32(d[o+8:]), uint64(binary.LittleEndian.Uint32(d[o+20:]))<<32 | uint64(binary.LittleEndian.Uint32(d[o+16:]))})
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].tsc < recs[j].tsc })
	tsc0, tscN := recs[0].tsc, recs[len(recs)-1].tsc
	events, _ := analysis.ParseHighlightEvents(mustRead(cache+"/chunk_27.bin"), 0)
	type ev struct {
		x uint64
		t int
	}
	var kills, deaths []ev
	for _, e := range events {
		switch e.EventType {
		case analysis.EventTypeKill:
			kills = append(kills, ev{e.XUID, e.TimeMS})
		case analysis.EventTypeDeath:
			deaths = append(deaths, ev{e.XUID, e.TimeMS})
		}
	}
	sort.Slice(kills, func(i, j int) bool { return kills[i].t < kills[j].t })
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].t < deaths[j].t })
	used := make([]bool, len(deaths))
	type kf struct{ killer, victim, t int }
	var feed []kf
	for _, k := range kills {
		best, bd := -1, 400
		for i, dd := range deaths {
			if used[i] || dd.x == k.x {
				continue
			}
			dt := k.t - dd.t
			if dt < 0 {
				dt = -dt
			}
			if dt < bd {
				bd, best = dt, i
			}
		}
		if best >= 0 {
			used[best] = true
			feed = append(feed, kf{xpi(k.x), xpi(deaths[best].x), k.t})
		}
	}
	tMin, tMax := 13839, 481382
	ft := func(r rec) float64 { return float64(tMin) + float64(r.tsc-tsc0)/float64(tscN-tsc0)*float64(tMax-tMin) }
	matchedVic := make([]uint32, len(feed))
	for i, k := range feed {
		bd, vic := 1e18, uint32(0)
		for _, r := range recs {
			if r.atkPi != k.killer {
				continue
			}
			dd := ft(r) - float64(k.t)
			if dd < 0 {
				dd = -dd
			}
			if dd < bd {
				bd, vic = dd, r.vic
			}
		}
		matchedVic[i] = vic
	}
	bp2pi := map[uint32]map[int]int{}
	for i, k := range feed {
		if bp2pi[matchedVic[i]] == nil {
			bp2pi[matchedVic[i]] = map[int]int{}
		}
		bp2pi[matchedVic[i]][k.victim]++
	}
	consistent := 0
	for i, k := range feed {
		dom, domc := -1, 0
		for pi, c := range bp2pi[matchedVic[i]] {
			if c > domc {
				dom, domc = pi, c
			}
		}
		if dom == k.victim {
			consistent++
		}
	}
	pure := 0
	for _, m := range bp2pi {
		if len(m) == 1 {
			pure++
		}
	}
	fmt.Printf("=== VALIDATION matching (biped-victime du record retenu vs victime du kill) ===\n")
	fmt.Printf("  bipeds distincts retenus : %d ; mono-victime (purs) : %d\n", len(bp2pi), pure)
	fmt.Printf("  kills où le biped retenu correspond à la bonne victime : %d/%d (%.0f%%)\n", consistent, len(feed), 100*float64(consistent)/float64(len(feed)))
}
