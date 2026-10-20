//go:build research

// amont.go — les balayages en amont des litteraux et leur motif (lot J4.5 : scinde de main.go, deplacement pur).

package main

import (
	"fmt"
	"sort"
	"strings"

	"levelup/go-api/internal/games/weapons/filmshell"
)

// upstreamScan : pour chaque littéral d'arme (WST gate à bit B-1), cherche en amont,
// à TOUS les offsets [B-600, B], un header de record `[1 delta][R(11) low][R(2) tag]`
// dont le slot (low&0x3fffffff) ∈ 512-519. Mesure : combien de WST ont un slot biped
// plausible en amont, et la distribution des distances (révèle la structure du record
// biped : header -> ... -> WST). Test du POC "attribution par remontée".
func upstreamScan(chunkIdx int) {
	d := chunk(chunkIdx)
	pkts := listPackets(d)
	bip := map[uint32]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}
	totalWST := 0
	withBiped := 0
	distHist := map[int]int{}
	slotHist := map[uint32]int{}
	for _, p := range pkts {
		if p.typ != 0 {
			continue
		}
		tot := len(p.payload) * 8
		for bp := 0; bp+64 <= tot; bp++ {
			hi := uint32(bitsAt(p.payload, bp, 32))
			if _, ok := knownHigh32(hi); !ok {
				continue
			}
			lo := uint32(bitsAt(p.payload, bp+32, 32))
			if _, ok := filmshell.WeaponIDToName[(uint64(hi)<<32)|uint64(lo)]; !ok {
				continue
			}
			totalWST++
			gateBit := bp - 1 // le gate WST est juste avant le high32
			// remonte : un header delta `[1][R11 low][R2 tag]` se termine 14 bits avant le
			// 1er composant du record. On cherche un header dont le slot ∈ biped à toute
			// distance d entre le header-start et le gate.
			found := false
			for hs := gateBit - 1; hs >= gateBit-700 && hs >= 0; hs-- {
				if bitsAt(p.payload, hs, 1) != 1 { // type delta
					continue
				}
				low := uint32(bitsAt(p.payload, hs+1, 11))
				slot := low & 0x3fffffff
				if bip[slot] {
					if !found {
						withBiped++
						distHist[gateBit-hs]++
						slotHist[slot]++
						found = true
					}
				}
			}
		}
	}
	fmt.Printf("=== chunk_%02d : %d WST gate=1 (litteraux d'armes) ===\n", chunkIdx, totalWST)
	fmt.Printf("  avec un header biped (slot 512-519) en amont [<=700 bits] : %d (%.0f%%)\n", withBiped, pct(withBiped, totalWST))
	fmt.Printf("  -- slots biped trouves en amont (1er match) : ")
	var ss []int
	for s := range slotHist {
		ss = append(ss, int(s))
	}
	sort.Ints(ss)
	for _, s := range ss {
		fmt.Printf("%d:%d ", s, slotHist[uint32(s)])
	}
	fmt.Println()
	// Note : à <=700 bits, un slot biped finit presque toujours par apparaître par
	// hasard (R(11) a 1/8 chance de tomber sur 512-519 ~ en fait 8/2048). On reporte la
	// distribution des distances pour voir s'il y a un PIC structurel (vraie position du
	// header) vs un bruit uniforme.
	type kv struct{ d, n int }
	var arr []kv
	for dd, n := range distHist {
		arr = append(arr, kv{dd, n})
	}
	sort.Slice(arr, func(a, b int) bool { return arr[a].n > arr[b].n })
	fmt.Printf("  -- top distances header-biped -> gate WST (pic structurel ?) --\n")
	for k := 0; k < len(arr) && k < 10; k++ {
		fmt.Printf("    dist=%-4d : %d\n", arr[k].d, arr[k].n)
	}
}

// litPlayer : pour chaque littéral d'arme, décode le header du 1er record du paquet
// (idLowBits=11) pour récupérer le slot du joueur, et liste (joueur, ts, arme). Permet
// de voir l'ÉVOLUTION TEMPORELLE de l'arme par joueur sur le chunk. Le 1er record d'un
// paquet biped tombe sur 512-519 = le joueur "propriétaire" du paquet.
func litPlayer(chunkIdx int) {
	d := chunk(chunkIdx)
	pkts := listPackets(d)
	bipedSlot := map[uint32]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}
	pktIdx := 0
	type hit struct {
		pkt        int
		ts         uint64
		firstSlot  uint32
		firstIsBip bool
		arme       string
		bit        int
	}
	var hits []hit
	for _, p := range pkts {
		if p.typ != 0 {
			continue
		}
		// 1er record header sous idLowBits=11
		bp := 0
		if bitsAt(p.payload, 0, 1) == 1 {
			bp = 1 // delta
		} else {
			bp = 3
		}
		low := uint32(bitsAt(p.payload, bp, 11))
		slot := low & 0x3fffffff
		tot := len(p.payload) * 8
		for b := 0; b+64 <= tot; b++ {
			hi := uint32(bitsAt(p.payload, b, 32))
			if _, ok := knownHigh32(hi); !ok {
				continue
			}
			lo := uint32(bitsAt(p.payload, b+32, 32))
			if nm, ok := filmshell.WeaponIDToName[(uint64(hi)<<32)|uint64(lo)]; ok {
				hits = append(hits, hit{pktIdx, p.ts, slot, bipedSlot[slot], nm, b})
			}
		}
		pktIdx++
	}
	fmt.Printf("=== chunk_%02d : %d litteraux d'armes ; 1er record du paquet = joueur ===\n", chunkIdx, len(hits))
	inBip := 0
	perSlot := map[uint32][]string{}
	for _, h := range hits {
		mark := ""
		if h.firstIsBip {
			inBip++
			mark = " (BIPED)"
			perSlot[h.firstSlot] = append(perSlot[h.firstSlot], h.arme)
		}
		fmt.Printf("  pkt#%-5d ts=%-12d 1erSlot=%-5d%s  bit=%-6d %s\n", h.pkt, h.ts, h.firstSlot, mark, h.bit, h.arme)
	}
	fmt.Printf("  >>> litteraux dont le paquet commence par un biped (512-519) : %d/%d\n", inBip, len(hits))
	_ = perSlot
	// Groupement par 1er slot du paquet (proxy d'entité), séquence temporelle d'armes.
	bySlot := map[uint32][]string{}
	for _, h := range hits {
		bySlot[h.firstSlot] = append(bySlot[h.firstSlot], h.arme)
	}
	fmt.Printf("  -- sequence temporelle d'armes par 1er-slot du paquet (idLow=11) --\n")
	var slots []int
	for s := range bySlot {
		slots = append(slots, int(s))
	}
	sort.Ints(slots)
	for _, s := range slots {
		seq := bySlot[uint32(s)]
		// compresse les répétitions consécutives
		var comp []string
		for i, a := range seq {
			if i == 0 || seq[i-1] != a {
				comp = append(comp, a)
			}
		}
		fmt.Printf("    slot %-5d (%d hits) : %s\n", s, len(seq), strings.Join(comp, " -> "))
	}
}

// litPattern agrège la structure binaire autour de chaque littéral d'arme complet sur
// plusieurs chunks. Vérifie l'hypothèse "littéral = composant WST keyframe-style"
// (gate=bit[-1]=1, puis variant=R(32), puis la suite du deser WST : R(12)+R(7)...).
// Mesure : distribution de bit[-1] (gate), et combien de littéraux ont un 2e littéral
// d'arme proche (paire primaire/secondaire comme au keyframe biped).
func litPattern(chunks []int) {
	gate1, gate0 := 0, 0
	total := 0
	pairWithin := 0 // littéraux ayant un autre littéral dans [+64, +64+512] (paire arme)
	for _, chunkIdx := range chunks {
		d := chunk(chunkIdx)
		pkts := listPackets(d)
		for _, p := range pkts {
			if p.typ != 0 {
				continue
			}
			tot := len(p.payload) * 8
			var lits []int
			for bp := 0; bp+64 <= tot; bp++ {
				hi := uint32(bitsAt(p.payload, bp, 32))
				if _, ok := knownHigh32(hi); !ok {
					continue
				}
				lo := uint32(bitsAt(p.payload, bp+32, 32))
				if _, ok := filmshell.WeaponIDToName[(uint64(hi)<<32)|uint64(lo)]; ok {
					lits = append(lits, bp)
				}
			}
			for _, bp := range lits {
				total++
				if bp >= 1 && bitsAt(p.payload, bp-1, 1) == 1 {
					gate1++
				} else {
					gate0++
				}
				for _, bp2 := range lits {
					if bp2 > bp+64 && bp2 <= bp+64+512 {
						pairWithin++
						break
					}
				}
			}
		}
	}
	fmt.Printf("=== litPattern sur chunks %v : %d litteraux d'armes complets ===\n", chunks, total)
	fmt.Printf("  gate bit[-1]=1 (pattern WST gate+variant) : %d (%.0f%%)\n", gate1, pct(gate1, total))
	fmt.Printf("  gate bit[-1]=0                            : %d (%.0f%%)\n", gate0, pct(gate0, total))
	fmt.Printf("  litteraux avec un 2e litteral dans [+64,+576] (paire d'armes) : %d (%.0f%%)\n", pairWithin, pct(pairWithin, total))
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}
