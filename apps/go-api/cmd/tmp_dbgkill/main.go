package main

import (
	"encoding/binary"
	"fmt"
	"levelup/go-api/internal/analysis"
	"os"
	"sort"
)

const dF = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/tools/ce/9b191a7f_dmg.bin`
const kF = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/tools/ce/9b191a7f_kill.bin`

var h32 = map[uint32]string{}

func idxD(h uint32) int {
	if h < 0xEC500000 || h > 0xEC600000 {
		return -1
	}
	return int((h - 0xEC500000) / 0x10002)
}
func idxK(h uint32) int {
	if h < 0xE1500000 || h > 0xE1600000 {
		return -1
	}
	return int((h - 0xE1500000) / 0x10002)
}
func main() {
	for id := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = analysis.WeaponIDToName[id]
	}
	dd, _ := os.ReadFile(dF)
	kd, _ := os.ReadFile(kF)
	type d struct {
		atk int
		w   string
		tsc uint64
	}
	var ds []d
	for o := 0; o+32 <= len(dd); o += 32 {
		a := binary.LittleEndian.Uint32(dd[o:])
		if a == 0 {
			continue
		}
		tsc := uint64(binary.LittleEndian.Uint32(dd[o+20:]))<<32 | uint64(binary.LittleEndian.Uint32(dd[o+16:]))
		ds = append(ds, d{idxD(a), h32[binary.LittleEndian.Uint32(dd[o+8:])], tsc})
	}
	sort.Slice(ds, func(i, j int) bool { return ds[i].tsc < ds[j].tsc })
	// ranges
	fmt.Printf("dmg: %d records ; tsc[0]=%d tsc[last]=%d ; atk idx range check:\n", len(ds), ds[0].tsc, ds[len(ds)-1].tsc)
	ai := map[int]int{}
	neg := 0
	for _, x := range ds {
		if x.atk < 0 {
			neg++
		} else {
			ai[x.atk]++
		}
	}
	fmt.Printf("  atk idx<0 (hors 0xEC5x)=%d ; par idx: %v\n", neg, ai)
	// kills
	fmt.Printf("kill: %d records (16o)\n", len(kd)/16)
	kbad := 0
	ki := map[int]int{}
	var ktsc []uint64
	for o := 0; o+16 <= len(kd); o += 16 {
		k := idxK(binary.LittleEndian.Uint32(kd[o+4:]))
		if k < 0 {
			kbad++
		} else {
			ki[k]++
		}
		ktsc = append(ktsc, uint64(binary.LittleEndian.Uint32(kd[o+12:]))<<32|uint64(binary.LittleEndian.Uint32(kd[o+8:])))
	}
	sort.Slice(ktsc, func(i, j int) bool { return ktsc[i] < ktsc[j] })
	fmt.Printf("  killer idx<0=%d ; par idx: %v\n  kill tsc[0]=%d tsc[last]=%d\n", kbad, ki, ktsc[0], ktsc[len(ktsc)-1])
	// trace 3 premiers kills d'idx0 : leurs dégâts autour
	fmt.Println("=== trace : dégâts d'atk=0 (10 derniers avant les 3 premiers kills d'idx0) ===")
	var kil0 []uint64
	for o := 0; o+16 <= len(kd); o += 16 {
		if idxK(binary.LittleEndian.Uint32(kd[o+4:])) == 0 {
			kil0 = append(kil0, uint64(binary.LittleEndian.Uint32(kd[o+12:]))<<32|uint64(binary.LittleEndian.Uint32(kd[o+8:])))
		}
	}
	sort.Slice(kil0, func(i, j int) bool { return kil0[i] < kil0[j] })
	for n := 0; n < 3 && n < len(kil0); n++ {
		kt := kil0[n]
		fmt.Printf("kill idx0 #%d tsc=%d -> derniers dégâts atk=0 avant:\n", n, kt)
		cnt := 0
		for i := len(ds) - 1; i >= 0 && cnt < 5; i-- {
			if ds[i].atk == 0 && ds[i].tsc <= kt {
				fmt.Printf("   tsc=%d %s (delta=%d)\n", ds[i].tsc, ds[i].w, kt-ds[i].tsc)
				cnt++
			}
		}
	}
}
