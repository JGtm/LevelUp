//go:build research

// litteraux.go — les balayages de litteraux d arme (lot J4.5 : scinde de main.go, deplacement pur).

package main

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/weapons/filmshell"
)

// litScan balaie un chunk : pour chaque paquet, cherche un littéral d'arme complet
// (high32 catalogué suivi de son low32 catalogué) à TOUT offset de bit. Reporte par
// type de paquet + par arme. But : où les armes COMPLÈTES apparaissent-elles dans le
// flux gameplay (records NEW d'entité-arme / WST keyframe-style) ?
func litScan(chunkIdx int) {
	d := chunk(chunkIdx)
	pkts := listPackets(d)
	byType := map[uint16]int{}     // paquets par type
	hitsByType := map[uint16]int{} // littéraux complets par type de paquet
	hitsByWeapon := map[string]int{}
	pktsByType := map[uint16]int{}
	pktsWithHit := map[uint16]int{}
	totalLits := 0
	for _, p := range pkts {
		byType[p.typ]++
		pktsByType[p.typ]++
		total := len(p.payload) * 8
		hadHit := false
		for bp := 0; bp+64 <= total; bp++ {
			hi := uint32(bitsAt(p.payload, bp, 32))
			nm, ok := knownHigh32(hi)
			if !ok {
				continue
			}
			lo := uint32(bitsAt(p.payload, bp+32, 32))
			id64 := (uint64(hi) << 32) | uint64(lo)
			if real, ok := filmshell.WeaponIDToName[id64]; ok {
				hitsByType[p.typ]++
				hitsByWeapon[real]++
				totalLits++
				hadHit = true
				_ = nm
			}
		}
		if hadHit {
			pktsWithHit[p.typ]++
		}
	}
	fmt.Printf("=== chunk_%02d : %d octets, %d paquets ===\n", chunkIdx, len(d), len(pkts))
	var types []int
	for t := range byType {
		types = append(types, int(t))
	}
	sort.Ints(types)
	fmt.Printf("  paquets par type : ")
	for _, t := range types {
		fmt.Printf("t%d=%d ", t, byType[uint16(t)])
	}
	fmt.Println()
	fmt.Printf("  litteraux d'armes COMPLETS (high32|low32 catalogues) = %d\n", totalLits)
	fmt.Printf("  par type de paquet : ")
	for _, t := range types {
		if hitsByType[uint16(t)] > 0 {
			fmt.Printf("t%d=%d (dans %d/%d paquets) ", t, hitsByType[uint16(t)], pktsWithHit[uint16(t)], pktsByType[uint16(t)])
		}
	}
	fmt.Println()
	if len(hitsByWeapon) > 0 {
		fmt.Printf("  par arme : ")
		type kv struct {
			k string
			v int
		}
		var arr []kv
		for k, v := range hitsByWeapon {
			arr = append(arr, kv{k, v})
		}
		sort.Slice(arr, func(a, b int) bool { return arr[a].v > arr[b].v })
		for _, e := range arr {
			fmt.Printf("%s=%d ", e.k, e.v)
		}
		fmt.Println()
	}
}

// litLoc décode les records type-0 d'un chunk (combo calibré extra=false idLowBits=11)
// et, pour chaque littéral d'arme complet trouvé, indique dans QUEL record (slot,
// typeIndex, type) et à quelle position relative il tombe. But : les armes complètes
// vivent-elles dans des records NEW d'entité-arme (ti=42) ou loadout (ti=5), ou dans
// les bipeds (ti=35), ou hors de tout record décodé ?
func litLoc(reg *grammar.Registry, worldPath string, chunkIdx, maxPkts int) {
	d := chunk(chunkIdx)
	pkts := listPackets(d)
	cfg := grammar.FrameConfig{HasExtraFields: false, IDLowBits: 11,
		Profil: grammar.ProfilDeBalayageParDefaut()}
	// stub i63 pour franchir le dernier composant biped et enchaîner les records.
	cfg.Profil.Grammaire.LargeursBouchon = map[string]int{"biped-action-component": 48}

	var t0 []packet
	for _, p := range pkts {
		if p.typ == 0 {
			t0 = append(t0, p)
		}
	}
	fmt.Printf("=== chunk_%02d : %d paquets type-0 ; localisation des litteraux d'armes ===\n", chunkIdx, len(t0))

	typeIndexHits := map[uint32]int{} // littéraux par typeIndex de record englobant
	inRecord, outRecord := 0, 0
	shown := 0
	for pi := 0; pi < len(t0) && pi < maxPkts; pi++ {
		p := t0[pi]
		// trouve les littéraux de ce paquet
		var litBits []int
		var litNames []string
		total := len(p.payload) * 8
		for bp := 0; bp+64 <= total; bp++ {
			hi := uint32(bitsAt(p.payload, bp, 32))
			if _, ok := knownHigh32(hi); !ok {
				continue
			}
			lo := uint32(bitsAt(p.payload, bp+32, 32))
			id64 := (uint64(hi) << 32) | uint64(lo)
			if nm, ok := filmshell.WeaponIDToName[id64]; ok {
				litBits = append(litBits, bp)
				litNames = append(litNames, nm)
			}
		}
		if len(litBits) == 0 {
			continue
		}
		// décode les records de ce paquet
		w := freshWorld(reg, worldPath)
		br := grammar.LecteurSur(p.payload)
		recs, _ := grammar.DecodeFrameRecords(br, w, cfg)
		// borne chaque record [startBit?, endBit]. On reconstruit les bornes via re-décodage :
		// approxime par la séquence cumulée des EndBit (Trace.EndBit) — chaque record finit là.
		type span struct {
			start, end int
			slot       uint32
			ti         uint32
			typ        int
		}
		var spans []span
		prevEnd := 0
		for _, r := range recs {
			st := prevEnd
			en := r.Trace.EndBit
			if en <= st {
				en = st + 1
			}
			spans = append(spans, span{st, en, r.Slot, r.TypeIndex, r.Type})
			prevEnd = r.Trace.EndBit
		}
		for li, bp := range litBits {
			var hostTI uint32 = 0xFFFFFFFF
			var hostSlot uint32
			var hostTyp int = -1
			for _, s := range spans {
				if bp >= s.start && bp < s.end {
					hostTI = s.ti
					hostSlot = s.slot
					hostTyp = s.typ
					break
				}
			}
			if hostTI != 0xFFFFFFFF {
				inRecord++
				typeIndexHits[hostTI]++
			} else {
				outRecord++
			}
			if shown < 30 {
				hn := "(hors record decode)"
				if hostTI != 0xFFFFFFFF {
					arch, _ := reg.Archetype(int(hostTI))
					an := ""
					if len(arch.Components) > 0 {
						an = arch.Components[0]
					}
					hn = fmt.Sprintf("record slot=%d ti=%d(%s) type=%d", hostSlot, hostTI, an, hostTyp)
				}
				fmt.Printf("  pkt#%d bit=%-6d %-22s -> %s\n", pi, bp, litNames[li], hn)
				shown++
			}
		}
	}
	fmt.Printf("  >>> litteraux DANS un record decode=%d ; HORS record decode=%d\n", inRecord, outRecord)
	if len(typeIndexHits) > 0 {
		fmt.Printf("  par typeIndex de record englobant : ")
		var tis []int
		for ti := range typeIndexHits {
			tis = append(tis, int(ti))
		}
		sort.Ints(tis)
		for _, ti := range tis {
			arch, _ := reg.Archetype(ti)
			an := ""
			if len(arch.Components) > 0 {
				an = arch.Components[0]
			}
			fmt.Printf("ti=%d(%s):%d ", ti, an, typeIndexHits[uint32(ti)])
		}
		fmt.Println()
	}
}
