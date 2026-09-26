//go:build research

// tailles.go — les tailles de paquet et le contexte d un litteral (lot J4.5 : scinde de main.go, deplacement pur).

package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/weapons/filmshell"
)

// sizesMode liste les paquets type-0 d'un chunk par taille décroissante et compte les
// littéraux d'armes complets de chacun. Hypothèse : les GROS paquets type-0 sont des
// mini-keyframes (re-sync full-state des bipeds) qui retransmettent ~8 armes (1/joueur).
func sizesMode(chunkIdx int) {
	d := chunk(chunkIdx)
	pkts := listPackets(d)
	type info struct {
		idx, size, lits int
		armes           []string
	}
	var infos []info
	i0 := 0
	for _, p := range pkts {
		if p.typ != 0 {
			continue
		}
		total := len(p.payload) * 8
		var armes []string
		for bp := 0; bp+64 <= total; bp++ {
			hi := uint32(bitsAt(p.payload, bp, 32))
			if _, ok := knownHigh32(hi); !ok {
				continue
			}
			lo := uint32(bitsAt(p.payload, bp+32, 32))
			if nm, ok := filmshell.WeaponIDToName[(uint64(hi)<<32)|uint64(lo)]; ok {
				armes = append(armes, nm)
			}
		}
		infos = append(infos, info{i0, p.size, len(armes), armes})
		i0++
	}
	// stats taille
	var sizes []int
	for _, in := range infos {
		sizes = append(sizes, in.size)
	}
	sort.Ints(sizes)
	med := sizes[len(sizes)/2]
	fmt.Printf("=== chunk_%02d : %d paquets type-0 ; taille mediane=%d, min=%d, max=%d ===\n",
		chunkIdx, len(infos), med, sizes[0], sizes[len(sizes)-1])
	// top 15 par taille
	sort.Slice(infos, func(a, b int) bool { return infos[a].size > infos[b].size })
	fmt.Printf("  -- top 15 plus gros paquets type-0 (taille, #litteraux, armes) --\n")
	for k := 0; k < len(infos) && k < 15; k++ {
		in := infos[k]
		fmt.Printf("    #%-5d size=%-6d lits=%-3d %v\n", in.idx, in.size, in.lits, dedup(in.armes))
	}
	// corrélation : combien de littéraux dans les paquets > 2x médiane vs <= médiane
	bigLits, bigPkts, smallLits, smallPkts := 0, 0, 0, 0
	for _, in := range infos {
		if in.size > 2*med {
			bigLits += in.lits
			bigPkts++
		} else {
			smallLits += in.lits
			smallPkts++
		}
	}
	fmt.Printf("  -- gros paquets (>2x mediane=%d) : %d paquets, %d litteraux (%.1f/pkt)\n",
		2*med, bigPkts, bigLits, ratio(bigLits, bigPkts))
	fmt.Printf("  -- paquets normaux (<=2x mediane) : %d paquets, %d litteraux (%.3f/pkt)\n",
		smallPkts, smallLits, ratio(smallLits, smallPkts))
}

func ratio(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

// litCtx dump, pour un paquet type-0 donné, l'en-tête (1er record header décodé sous
// idLowBits=11) et, pour chaque littéral d'arme, le contexte de bits autour (les 32
// bits avant le high32, le high32, le low32, et les bits après) pour comprendre la
// structure du record qui le porte (NEW d'entité-arme ? autre ?).
func litCtx(reg *grammar.Registry, chunkIdx, pktIdx int) {
	d := chunk(chunkIdx)
	pkts := listPackets(d)
	var t0 []packet
	for _, p := range pkts {
		if p.typ == 0 {
			t0 = append(t0, p)
		}
	}
	if pktIdx >= len(t0) {
		fmt.Println("pktIdx hors borne")
		return
	}
	p := t0[pktIdx]
	total := len(p.payload) * 8
	fmt.Printf("=== chunk_%02d type-0 #%d : %d octets (%d bits), ts=%d ===\n", chunkIdx, pktIdx, p.size, total, p.ts)

	// header sous différents idLowBits pour voir le 1er record
	for _, idLow := range []int{9, 11} {
		bp := 0
		typ := 0
		if bitsAt(p.payload, bp, 1) == 1 {
			typ = 3
			bp = 1
		} else {
			typ = int(bitsAt(p.payload, bp+1, 2))
			bp = 3
		}
		low := uint32(bitsAt(p.payload, bp, idLow))
		bp += idLow
		tag := uint32(bitsAt(p.payload, bp, 2))
		slot := (low) & 0x3fffffff
		fmt.Printf("  [idLow=%d] 1er record: type=%d low=%d tag=%d slot=%d\n", idLow, typ, low, tag, slot)
	}

	// littéraux + contexte
	for bp := 0; bp+64 <= total; bp++ {
		hi := uint32(bitsAt(p.payload, bp, 32))
		if _, ok := knownHigh32(hi); !ok {
			continue
		}
		lo := uint32(bitsAt(p.payload, bp+32, 32))
		id64 := (uint64(hi) << 32) | uint64(lo)
		nm, ok := filmshell.WeaponIDToName[id64]
		if !ok {
			continue
		}
		before := uint32(bitsAt(p.payload, bp-32, 32))
		after1 := uint32(bitsAt(p.payload, bp+64, 32))
		after2 := uint32(bitsAt(p.payload, bp+96, 32))
		fmt.Printf("\n  >>> %s @bit%d (id64=0x%016x)\n", nm, bp, id64)
		fmt.Printf("      [-32]=0x%08x | high=0x%08x low=0x%08x | [+64]=0x%08x [+96]=0x%08x\n",
			before, hi, lo, after1, after2)
		// Est-ce que [bp-1] ressemble à un gate de WST (1) suivi du high32 = pattern keyframe WST ?
		gateBit := bitsAt(p.payload, bp-1, 1)
		fmt.Printf("      bit[-1] (gate WST candidat)=%d ; si =1 -> structure WST keyframe (gate+variant R32)\n", gateBit)
	}
}

func freshWorld(reg *grammar.Registry, path string) *grammar.World {
	raw, _ := os.ReadFile(path)
	w := grammar.NewWorld(reg)
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		for _, tok := range strings.Fields(line) {
			parts := strings.SplitN(tok, ":", 2)
			if len(parts) != 2 {
				continue
			}
			slot, e1 := strconv.ParseUint(parts[0], 10, 32)
			ti, e2 := strconv.ParseUint(parts[1], 10, 32)
			if e1 != nil || e2 != nil {
				continue
			}
			w.BindFull(uint32(slot), uint32(ti))
		}
	}
	return w
}
