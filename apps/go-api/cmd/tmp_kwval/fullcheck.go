package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"
)

// fullcheck.go — exploite la capture CE filmdec_full_capture (dmgdeser base+fenêtre, killdeser curseur,
// consumer = struct de sortie param_3 REMPLI : victim/atk/variant/cnt2/cnt1/hits). Match dmgdeser->consumer
// par rdtsc pour obtenir (paquet, base, cnt1/cnt2 RÉELS) -> valide le décode des compteurs du port
// (source du désync post-variant). Le tableau de hits = les dégâts (Phase 3).

type consRec struct {
	rdtsc       uint64
	victim, atk uint32
	variant     uint32
	cnt2, cnt1  uint32
	out1d       uint32
	hits        []byte // 224 octets depuis param_3+0x40
}

func parseConsumer(path string) []consRec {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []consRec
	for o := 0; o+0x100 <= len(b); o += 0x100 {
		out = append(out, consRec{
			rdtsc:   uint64(binary.LittleEndian.Uint32(b[o:])) | uint64(binary.LittleEndian.Uint32(b[o+4:]))<<32,
			victim:  binary.LittleEndian.Uint32(b[o+8:]),
			atk:     binary.LittleEndian.Uint32(b[o+0xc:]),
			variant: binary.LittleEndian.Uint32(b[o+0x10:]),
			cnt2:    binary.LittleEndian.Uint32(b[o+0x14:]),
			cnt1:    binary.LittleEndian.Uint32(b[o+0x18:]),
			out1d:   binary.LittleEndian.Uint32(b[o+0x1c:]),
			hits:    b[o+0x20 : o+0x100],
		})
	}
	return out
}

func runFullCheck(m string) {
	ceDir := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/`
	dmg := parseCERecs(ceDir + m + "_full_dmgdeser.bin")
	cons := parseConsumer(ceDir + m + "_full_consumer.bin")
	if len(cons) == 0 {
		fmt.Printf("=== FULLCHECK %s : consumer vide (%s) ===\n", m, ceDir+m+"_full_consumer.bin")
		return
	}
	cache := root + "/" + m
	var chunks [][]byte
	for ch := 0; ch <= 41; ch++ {
		chunks = append(chunks, inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch)))
	}
	fmt.Printf("=== FULLCHECK %s : %d dmgdeser + %d consumer records ===\n", m, len(dmg), len(cons))

	// variant distribution (la CAUSE : 0x42C967xx firearm / 0x592CF3xx mêlée / 0x164B3Cxx grenade)
	clsHist := map[string]int{}
	for _, c := range cons {
		switch damageClass(c.variant) {
		case dmgFirearm:
			clsHist["firearm"]++
		case dmgMelee:
			clsHist["melee"]++
		case dmgGrenade:
			clsHist["grenade"]++
		default:
			clsHist[fmt.Sprintf("var-%06X", c.variant>>8)]++
		}
	}
	fmt.Printf("consumer variants (classe) : %v\n", clsHist)

	// match dmgdeser -> consumer par rdtsc (le consumer fire juste APRÈS le déser, même struct).
	sort.Slice(cons, func(i, j int) bool { return cons[i].rdtsc < cons[j].rdtsc })
	nextCons := func(t uint64) (consRec, bool) {
		lo, hi := 0, len(cons)
		for lo < hi {
			mid := (lo + hi) / 2
			if cons[mid].rdtsc < t {
				lo = mid + 1
			} else {
				hi = mid
			}
		}
		if lo >= len(cons) {
			return consRec{}, false
		}
		return cons[lo], true
	}

	cnt1OK, cnt2OK, matched := 0, 0, 0
	var samples []string
	for _, d := range dmg {
		pl, _, _, ok := locatePkt(chunks, d.window)
		if !ok {
			continue
		}
		c, okc := nextCons(d.rdtsc)
		if !okc || c.rdtsc-d.rdtsc > 200000 { // fenêtre rdtsc serrée (~même instant)
			continue
		}
		// le variant du consumer doit être présent dans le paquet (même record de dégât)
		matched++
		deserDamageV2(pl, dwV2{base: int(d.val), e494: 67, tail84: 4, guard: 0})
		if dbgCnt1 == int(c.cnt1) {
			cnt1OK++
		}
		if dbgCnt2 == int(c.cnt2) {
			cnt2OK++
		}
		if len(samples) < 16 {
			samples = append(samples, fmt.Sprintf("  base=%d var=0x%08X | RÉEL cnt1=%d cnt2=%d | port cnt1=%d cnt2=%d",
				d.val, c.variant, c.cnt1, c.cnt2, dbgCnt1, dbgCnt2))
		}
	}
	fmt.Printf(">>> dmgdeser<->consumer matchés : %d | cnt1 port==réel : %d/%d | cnt2 port==réel : %d/%d\n",
		matched, cnt1OK, matched, cnt2OK, matched)
	for _, s := range samples {
		fmt.Println(s)
	}
}
