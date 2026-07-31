package main

import (
	"fmt"
	"sort"
)

// bracketcheck.go — exploite la capture CE filmdec_bracket (4 hooks bitstream : D=dmgdeser base,
// C=counter start, S=section start, K=killdeser cursor ; positions packet-local [reader+0x2c]). Match
// D/C/S par rdtsc au K (kill), localise le paquet via la fenêtre K, exécute deserDamageV2 depuis base=D.pos
// et compare dbgPosCounter/dbgPosSection/L aux positions CE réelles -> PINPOINTE le segment fautif.

func precedingByRdtsc(recs []ceRec, t uint64) (ceRec, bool) {
	lo, hi := 0, len(recs)
	for lo < hi {
		mid := (lo + hi) / 2
		if recs[mid].rdtsc < t {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo == 0 {
		return ceRec{}, false
	}
	return recs[lo-1], true
}

// followingByRdtsc : premier record avec rdtsc >= t (le C/S du MÊME déser que D fire juste après D).
func followingByRdtsc(recs []ceRec, t uint64) (ceRec, bool) {
	lo, hi := 0, len(recs)
	for lo < hi {
		mid := (lo + hi) / 2
		if recs[mid].rdtsc < t {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo >= len(recs) {
		return ceRec{}, false
	}
	return recs[lo], true
}

func runBracketCheck(m string) {
	ceDir := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/`
	D := parseCERecs(ceDir + m + "_brk_D.bin")
	C := parseCERecs(ceDir + m + "_brk_C.bin")
	S := parseCERecs(ceDir + m + "_brk_S.bin")
	K := parseCERecs(ceDir + m + "_brk_K.bin")
	if len(K) == 0 || len(D) == 0 {
		fmt.Printf("=== BRACKETCHECK %s : K=%d D=%d (capture absente ?) ===\n", m, len(K), len(D))
		return
	}
	for _, s := range [][]ceRec{D, C, S} {
		sort.Slice(s, func(i, j int) bool { return s[i].rdtsc < s[j].rdtsc })
	}
	cache := root + "/" + m
	var chunks [][]byte
	for ch := 0; ch <= 41; ch++ {
		chunks = append(chunks, inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch)))
	}
	fmt.Printf("=== BRACKETCHECK %s : D=%d C=%d S=%d K=%d ===\n", m, len(D), len(C), len(S), len(K))

	matched := 0
	okCounter, okSection, okCursor := 0, 0, 0
	// premier segment fautif par kill (diagnostic)
	firstBad := map[string]int{}
	var samples []string
	for _, k := range K {
		d, okd := precedingByRdtsc(D, k.rdtsc)
		if !okd || int(d.val) >= int(k.val) {
			continue
		}
		pl, _, _, okp := locatePkt(chunks, k.window)
		if !okp {
			continue
		}
		base := int(d.val)
		L := deserDamageV2(pl, dwV2{base: base, e494: 67, tail84: 4, guard: 0})
		predCursor := base + L + 10
		matched++
		// positions CE réelles : le C/S du MÊME déser fatal = premier APRÈS D (helpers génériques -> ne PAS
		// prendre "avant K" qui attrape des appels étrangers).
		c, okc := followingByRdtsc(C, d.rdtsc)
		s, oks := followingByRdtsc(S, d.rdtsc)
		realCounter, realSection := -1, -1
		if okc && c.rdtsc < k.rdtsc && int(c.val) > base && int(c.val) < int(k.val) {
			realCounter = int(c.val)
		}
		if oks && s.rdtsc < k.rdtsc && int(s.val) > base && int(s.val) < int(k.val) {
			realSection = int(s.val)
		}
		cOK := realCounter >= 0 && dbgPosCounter == realCounter
		sOK := realSection >= 0 && dbgPosSection == realSection
		curOK := predCursor == int(k.val)
		if cOK {
			okCounter++
		}
		if sOK {
			okSection++
		}
		if curOK {
			okCursor++
		}
		// 1er segment fautif
		bad := "OK"
		if realCounter >= 0 && !cOK {
			bad = "PRE-COUNTER"
		} else if realSection >= 0 && !sOK {
			bad = "COUNTERS+HITS+LOOP2"
		} else if !curOK {
			bad = "SECTION+TAIL"
		}
		firstBad[bad]++
		if len(samples) < 16 {
			samples = append(samples, fmt.Sprintf("  mk=0x%02X base=%d | counter port=%d CE=%d | section port=%d CE=%d | cursor pred=%d CE=%d | cnt1=%d cnt2=%d -> %s",
				pl[0], base, dbgPosCounter, realCounter, dbgPosSection, realSection, predCursor, k.val, dbgCnt1, dbgCnt2, bad))
		}
	}
	fmt.Printf(">>> matchés %d | counter OK %d | section OK %d | cursor OK %d\n", matched, okCounter, okSection, okCursor)
	fmt.Printf(">>> 1er segment fautif : %v\n", firstBad)
	for _, s := range samples {
		fmt.Println(s)
	}
}
