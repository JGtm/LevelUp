package main

import (
	"encoding/binary"
	"fmt"
	"levelup/go-api/internal/analysis"
	"os"
	"sort"
)

const dmgF = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/dmgcapture_run2.bin`
const killF = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/killcapture.bin`
const aBase = 0xEC500000
const kBase = 0xE1500000
const step = 0x10002

var h32 = map[uint32]string{}
var roster = map[int]string{0: "whiteknight2519", 1: "JAVIERLOLITO540", 2: "JGtm", 3: "LORD PEINX13", 4: "IKE ILYA", 5: "Akatsuki fire17", 6: "aldusbroncus", 7: "VitaminA1688"}

func idxFrom(h, base uint32) int {
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

type drec struct {
	atk int
	fam uint32
	tsc uint64
}
type krec struct {
	kil, vic int
	tsc      uint64
}

func main() {
	for id, n := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = n
	}
	dd, _ := os.ReadFile(dmgF)
	var dmg []drec
	for o := 0; o+32 <= len(dd); o += 32 {
		a := binary.LittleEndian.Uint32(dd[o:])
		if a == 0 {
			continue
		}
		dmg = append(dmg, drec{idxFrom(a, aBase), binary.LittleEndian.Uint32(dd[o+8:]), uint64(binary.LittleEndian.Uint32(dd[o+20:]))<<32 | uint64(binary.LittleEndian.Uint32(dd[o+16:]))})
	}
	sort.Slice(dmg, func(i, j int) bool { return dmg[i].tsc < dmg[j].tsc })
	kd, _ := os.ReadFile(killF)
	var kil []krec
	for o := 0; o+16 <= len(kd); o += 16 {
		v := binary.LittleEndian.Uint32(kd[o:])
		k := binary.LittleEndian.Uint32(kd[o+4:])
		t := uint64(binary.LittleEndian.Uint32(kd[o+12:]))<<32 | uint64(binary.LittleEndian.Uint32(kd[o+8:]))
		ki := idxFrom(k, kBase)
		vi := idxFrom(v, kBase)
		if ki < 0 {
			continue
		}
		kil = append(kil, krec{ki, vi, t})
	}
	sort.Slice(kil, func(i, j int) bool { return kil[i].tsc < kil[j].tsc })
	// dédup (le hook peut refire) : même tueur+victime à <1ms
	var ku []krec
	for _, k := range kil {
		dup := false
		for j := len(ku) - 1; j >= 0 && k.tsc-ku[j].tsc < 3000000; j-- {
			if ku[j].kil == k.kil && ku[j].vic == k.vic {
				dup = true
				break
			}
		}
		if !dup {
			ku = append(ku, k)
		}
	}
	fmt.Printf("=== %d dégâts ; %d kills valides (%d après dédup) ===\n\n", len(dmg), len(kil), len(ku))
	// ATTRIBUTION EXACTE : arme = dernier dégât du tueur avant le tsc du kill
	att := 0
	byK := map[int]map[string]int{}
	var lines []string
	for _, k := range ku {
		fam := "?"
		var bt uint64
		for _, d := range dmg {
			if d.atk != k.kil {
				continue
			}
			if d.tsc <= k.tsc && d.tsc > bt {
				bt = d.tsc
				fam = famName(d.fam)
			}
		}
		if fam != "?" {
			att++
		}
		if byK[k.kil] == nil {
			byK[k.kil] = map[string]int{}
		}
		byK[k.kil][fam]++
		lines = append(lines, fmt.Sprintf("  %-16s --[%-16s]--> %-16s", roster[k.kil], fam, roster[k.vic]))
	}
	fmt.Println("=== ARME PAR KILL (live exact, 30 premiers) ===")
	for i, l := range lines {
		if i < 30 {
			fmt.Println(l)
		}
	}
	fmt.Printf("\n=== COUVERTURE : %d/%d kills attribués ===\n", att, len(ku))
	fmt.Println("\n=== BREAKDOWN ARMES PAR JOUEUR (ground-truth exact) ===")
	for p := 0; p < 8; p++ {
		var fs []string
		tot := 0
		for f, c := range byK[p] {
			fs = append(fs, f)
			tot += c
		}
		sort.Slice(fs, func(i, j int) bool { return byK[p][fs[i]] > byK[p][fs[j]] })
		line := ""
		for _, f := range fs {
			line += fmt.Sprintf("%s:%d ", f, byK[p][f])
		}
		fmt.Printf("  %-16s (%2d kills) : %s\n", roster[p], tot, line)
	}
}
