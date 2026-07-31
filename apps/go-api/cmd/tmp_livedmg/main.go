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
var roster = map[int]string{0: "whiteknight2519", 1: "JAVIERLOLITO540", 2: "JGtm", 3: "LORD PEINX13", 4: "IKE ILYA", 5: "Akatsuki fire17", 6: "aldusbroncus", 7: "VitaminA1688"}
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
func famName(f uint32) string {
	if n := h32[f]; n != "" {
		return n
	}
	return fmt.Sprintf("0x%08x", f)
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
		v := binary.LittleEndian.Uint32(d[o+4:])
		f := binary.LittleEndian.Uint32(d[o+8:])
		tl := binary.LittleEndian.Uint32(d[o+16:])
		th := binary.LittleEndian.Uint32(d[o+20:])
		recs = append(recs, rec{atkPi(a), v, f, uint64(th)<<32 | uint64(tl)})
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].tsc < recs[j].tsc })
	tsc0, tscN := recs[0].tsc, recs[len(recs)-1].tsc
	// kill feed
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
	tMin, tMax := feed[0].t, feed[len(feed)-1].t
	// film_time approx de chaque record (tsc normalisé sur la fenêtre kills)
	ft := func(r rec) float64 { return float64(tMin) + float64(r.tsc-tsc0)/float64(tscN-tsc0)*float64(tMax-tMin) }
	// ATTRIBUTION : pour chaque kill, record attaquant==tueur le plus proche en temps approx
	att := 0
	byKiller := map[int]map[string]int{}
	var lines []string
	for _, k := range feed {
		bd, fam := 1e18, "?"
		for _, r := range recs {
			if r.atkPi != k.killer {
				continue
			}
			dd := ft(r) - float64(k.t)
			if dd < 0 {
				dd = -dd
			}
			if dd < bd {
				bd, fam = dd, famName(r.fam)
			}
		}
		if fam != "?" {
			att++
		}
		if byKiller[k.killer] == nil {
			byKiller[k.killer] = map[string]int{}
		}
		byKiller[k.killer][fam]++
		lines = append(lines, fmt.Sprintf("  t=%6.1fs  %-16s --[%-16s]--> %-16s", float64(k.t)/1000, roster[k.killer], fam, roster[k.victim]))
	}
	fmt.Printf("=== ATTRIBUTION ARME PAR KILL (live, %d records, %d kills) ===\n", len(recs), len(feed))
	for i, l := range lines {
		if i < 30 {
			fmt.Println(l)
		}
	}
	fmt.Printf("\n=== COUVERTURE : %d/%d kills attribués ===\n", att, len(feed))
	var ps []int
	for p := range byKiller {
		ps = append(ps, p)
	}
	sort.Ints(ps)
	fmt.Println("\n=== armes par tueur (live ground-truth) ===")
	for _, p := range ps {
		var fs []string
		for f := range byKiller[p] {
			fs = append(fs, f)
		}
		sort.Slice(fs, func(i, j int) bool { return byKiller[p][fs[i]] > byKiller[p][fs[j]] })
		line := ""
		for _, f := range fs {
			line += fmt.Sprintf("%s:%d ", f, byKiller[p][f])
		}
		fmt.Printf("  %-16s : %s\n", roster[p], line)
	}
}
