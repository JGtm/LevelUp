package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"
)

// basecheck.go — exploite la capture CE filmdec_base_capture (base réelle du déser dégât + curseur
// kill-event, MÊME paquet, reader per-paquet donc [reader+0x2c] packet-local). Match dmg->kill par paquet,
// calcule L_real = cursor - base - 10 et le compare au port deserDamageV2 depuis la VRAIE base.

type ceRec struct {
	rdtsc   uint64
	val     uint32 // base (dmgdeser) ou cursor (killdeser)
	bytePtr uint32
	window  []byte // 112 octets
}

func parseCERecs(path string) []ceRec {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []ceRec
	for o := 0; o+0x80 <= len(b); o += 0x80 {
		out = append(out, ceRec{
			rdtsc:   uint64(binary.LittleEndian.Uint32(b[o:])) | uint64(binary.LittleEndian.Uint32(b[o+4:]))<<32,
			val:     binary.LittleEndian.Uint32(b[o+8:]),
			bytePtr: binary.LittleEndian.Uint32(b[o+0xc:]),
			window:  b[o+0x10 : o+0x80],
		})
	}
	return out
}

// locatePkt : trouve la fenêtre (48 premiers octets) dans les chunks inflatés et renvoie le payload du
// paquet frame contenant cette position + l'indice de chunk (pour identifier le paquet de façon unique).
func locatePkt(chunks [][]byte, window []byte) (pl []byte, chIdx, pktOff int, ok bool) {
	needle := window
	if len(needle) > 24 {
		needle = needle[:24]
	}
	for ci, d := range chunks {
		x := indexBytes(d, needle)
		if x < 0 {
			continue
		}
		off := 0
		for off+16 <= len(d) {
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			if x >= off+16 && x < off+16+sz {
				return d[off+16 : off+16+sz], ci, off, true
			}
			off += 16 + sz
		}
	}
	return nil, 0, 0, false
}

func runBaseCheck(m string) {
	ceDir := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/`
	dmg := parseCERecs(ceDir + m + "_base_dmgdeser.bin")
	kill := parseCERecs(ceDir + m + "_base_killdeser.bin")
	if len(kill) == 0 {
		fmt.Printf("=== BASECHECK %s : killdeser vide (%s) ===\n", m, ceDir+m+"_base_killdeser.bin")
		return
	}
	cache := root + "/" + m
	var chunks [][]byte
	for ch := 0; ch <= 41; ch++ {
		chunks = append(chunks, inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch)))
	}
	fmt.Printf("=== BASECHECK %s : %d dmgdeser + %d killdeser records ===\n", m, len(dmg), len(kill))

	// DIAG localisation : la fenêtre est-elle dans les chunks brute ? byte-swappée par mots de 8 ?
	if len(os.Args) >= 4 && os.Args[3] == "diag" {
		swap8 := func(w []byte) []byte {
			out := make([]byte, len(w))
			for i := 0; i+8 <= len(w); i += 8 {
				for j := 0; j < 8; j++ {
					out[i+j] = w[i+7-j]
				}
			}
			return out
		}
		for i := 0; i < 6 && i < len(dmg); i++ {
			w := dmg[i].window
			rawHit := -1
			swHit := -1
			for _, d := range chunks {
				if x := indexBytes(d, w[:16]); x >= 0 {
					rawHit = x
					break
				}
			}
			sw := swap8(w)
			for _, d := range chunks {
				if x := indexBytes(d, sw[:16]); x >= 0 {
					swHit = x
					break
				}
			}
			fmt.Printf("  dmg[%d] bytePtr=%08X raw16=% X | rawHit=%d swHit=%d\n", i, dmg[i].bytePtr, w[:16], rawHit, swHit)
		}
		return
	}

	// APPARIEMENT PAR RDTSC : le déser dégât FATAL fire juste AVANT le kill deser (même reader, même
	// paquet). Le dmg deser fire aussi en sim-live (autre buffer) -> on n'exige PAS qu'il localise ; on
	// prend le dmg au rdtsc immédiatement < celui du kill (le plus proche dans le temps = le fatal).
	sort.Slice(dmg, func(i, j int) bool { return dmg[i].rdtsc < dmg[j].rdtsc })
	precedingDmg := func(tk uint64) (uint32, bool) {
		lo, hi := 0, len(dmg)
		for lo < hi {
			mid := (lo + hi) / 2
			if dmg[mid].rdtsc < tk {
				lo = mid + 1
			} else {
				hi = mid
			}
		}
		if lo == 0 {
			return 0, false
		}
		return dmg[lo-1].val, true
	}

	type kv3 struct{ k, v int }
	exact, matched, killLoc := 0, 0, 0
	ancExact, ancTot := 0, 0
	glocExact := 0
	resid := map[int]int{}
	ancResid := map[int]int{}
	glocResid := map[int]int{}
	baseHist := map[int]int{}
	var samples []string
	for _, k := range kill {
		pl, _, _, ok := locatePkt(chunks, k.window)
		if !ok {
			continue
		}
		killLoc++
		b, okb := precedingDmg(k.rdtsc)
		if !okb {
			continue
		}
		best := int(b)
		// sanity : base < cursor et longueur plausible (< bits du paquet)
		if best < 0 || best >= int(k.val) || int(k.val)-best > len(pl)*8 {
			continue
		}
		matched++
		baseHist[best]++
		// VALIDATION DU LOCATOR GRAMMATICAL (R7=85) vs le vrai curseur CE.
		gcur := locateKillEventCursor(pl)
		glocResid[gcur-int(k.val)]++
		if gcur == int(k.val) {
			glocExact++
		}
		Lreal := int(k.val) - best - 10
		w := dwV2{base: best, e494: 67, tail84: 4, guard: 0}
		Lport := deserDamageV2(pl, w)
		d := Lport - Lreal
		// ANCRÉ : trouve le variant fatal (avant le vrai curseur) et modélise le post-variant depuis là.
		famPos, _ := variantAnchorLast(pl, int(k.val))
		if famPos >= 0 {
			out0 := int(bitsAt(pl, best, 1))
			out1c := int(bitsAt(pl, best+1, 1))
			predCur := deserAnchoredCursor(pl, famPos+32, out0, out1c, w)
			ancResid[predCur-int(k.val)]++
			if predCur == int(k.val) {
				ancExact++
			}
			ancTot++
		}
		resid[d]++
		if d == 0 {
			exact++
		}
		if pl[0] == 0xD2 && len(samples) < 20 {
			// pré-variant : mon port lit le variant où, vs le vrai suffixe 0x42c9679f ?
			vpos, vval, _ := traceV2(pl, w)
			sufPos := -1
			for b := best; b+32 <= len(pl)*8; b++ {
				if uint32(bitsAt(pl, b, 32)) == sfx {
					sufPos = b
					break
				}
			}
			samples = append(samples, fmt.Sprintf("  0xD2 base=%d cur=%d Δ=%+d | monVariant@%d val=0x%08X (attendu 0x42C9679F) | suffixe réel@%d | cnt1=%d cnt2=%d",
				best, k.val, d, vpos, vval, sufPos, dbgCnt1, dbgCnt2))
		}
	}
	fmt.Printf(">>> kills localisés : %d/%d | matchés : %d | port(base) EXACT %d/%d | port ANCRÉ EXACT %d/%d\n",
		killLoc, len(kill), matched, exact, matched, ancExact, ancTot)
	fmt.Printf(">>> LOCATOR GRAMMATICAL (R7=85) vs curseur CE réel : EXACT %d/%d = %.1f%%\n",
		glocExact, matched, 100*float64(glocExact)/float64(max1(matched)))
	var gr []kv3
	for k, v := range glocResid {
		gr = append(gr, kv3{k, v})
	}
	sort.Slice(gr, func(i, j int) bool { return gr[i].v > gr[j].v })
	fmt.Printf("résidus LOCATOR (gcur-cur) : ")
	for i := 0; i < 14 && i < len(gr); i++ {
		fmt.Printf("%d:%d ", gr[i].k, gr[i].v)
	}
	fmt.Println()
	var ar []kv3
	for k, v := range ancResid {
		ar = append(ar, kv3{k, v})
	}
	sort.Slice(ar, func(i, j int) bool { return ar[i].v > ar[j].v })
	fmt.Printf("résidus ANCRÉ (predCur-cur) : ")
	for i := 0; i < 14 && i < len(ar); i++ {
		fmt.Printf("%d:%d ", ar[i].k, ar[i].v)
	}
	fmt.Println()
	// histogramme des bases réelles (le préambule dispatcher réel)
	var bh []kv3
	for k, v := range baseHist {
		bh = append(bh, kv3{k, v})
	}
	sort.Slice(bh, func(i, j int) bool { return bh[i].v > bh[j].v })
	fmt.Printf("bases réelles (top) : ")
	for i := 0; i < 12 && i < len(bh); i++ {
		fmt.Printf("%d:%d ", bh[i].k, bh[i].v)
	}
	fmt.Println()
	var rr []kv3
	for k, v := range resid {
		rr = append(rr, kv3{k, v})
	}
	sort.Slice(rr, func(i, j int) bool { return rr[i].v > rr[j].v })
	fmt.Printf("résidus (Lport-Lreal) : ")
	for i := 0; i < 14 && i < len(rr); i++ {
		fmt.Printf("%d:%d ", rr[i].k, rr[i].v)
	}
	fmt.Println()
	for _, s := range samples {
		fmt.Println(s)
	}
}
