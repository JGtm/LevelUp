// rdata_weapon_scan — THROWAWAY (mission: trouver le mécanisme de SWAP d'arme).
//
// Étape 1 (ce fichier) : cartographier le registre ECS. Pour chaque typeIndex
// présent dans le World (data/cache/film_chunks/000d5950/world_dump.txt), afficher
// le nom de l'archétype (= 1er composant), le nombre de composants, et signaler ceux
// qui portent des composants liés à l'arme/loadout/variant (weapon-state-type-info,
// player-engine-loadout-index, object-multiplayer-properties = 'obje' qui porte un
// variant-name, etc.). But : identifier le typeIndex + slots des entités-armes et de
// l'archétype loadout.
package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

func inflate(p string) []byte {
	raw, _ := os.ReadFile(p)
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

// loadWorld parses world_dump.txt -> slot:typeIndex map and typeIndex->slots.
func loadWorld() (map[int]int, map[int][]int) {
	f, err := os.Open(cache + "/world_dump.txt")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	slotType := map[int]int{}
	typeSlots := map[int][]int{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		for _, tok := range strings.Fields(line) {
			parts := strings.SplitN(tok, ":", 2)
			if len(parts) != 2 {
				continue
			}
			slot, e1 := strconv.Atoi(parts[0])
			ti, e2 := strconv.Atoi(parts[1])
			if e1 != nil || e2 != nil {
				continue
			}
			slotType[slot] = ti
			typeSlots[ti] = append(typeSlots[ti], slot)
		}
	}
	return slotType, typeSlots
}

// weaponish reports whether a component name is weapon/loadout/variant-bearing.
func weaponish(name string) bool {
	for _, k := range []string{"weapon", "loadout", "variant", "equip", "pickup", "item", "grenade", "ability", "multiplayer-properties"} {
		if strings.Contains(name, k) {
			return true
		}
	}
	return false
}

// ---- littéraux d'armes dans le flux ----

func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

func knownHigh32(v uint32) (string, bool) {
	for id, n := range analysis.WeaponIDToName {
		if uint32(id>>32) == v {
			return n, true
		}
	}
	return "", false
}

type packet struct {
	typ     uint16
	off     int
	size    int
	ts      uint64
	payload []byte
}

func listPackets(d []byte) []packet {
	var out []packet
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		out = append(out, packet{typ, off, sz, ts, d[off+16 : off+16+sz]})
		off += 16 + sz
	}
	return out
}

// litScan balaie un chunk : pour chaque paquet, cherche un littéral d'arme complet
// (high32 catalogué suivi de son low32 catalogué) à TOUT offset de bit. Reporte par
// type de paquet + par arme. But : où les armes COMPLÈTES apparaissent-elles dans le
// flux gameplay (records NEW d'entité-arme / WST keyframe-style) ?
func litScan(chunkIdx int) {
	chunkPath := fmt.Sprintf("%s/chunk_%02d.bin", cache, chunkIdx)
	d := inflate(chunkPath)
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
			if real, ok := analysis.WeaponIDToName[id64]; ok {
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
	fmt.Printf("  littéraux d'armes COMPLETS (high32|low32 catalogués) = %d\n", totalLits)
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
func litLoc(reg *filmdec.Registry, worldPath string, chunkIdx, maxPkts int) {
	d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, chunkIdx))
	pkts := listPackets(d)
	cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}
	filmdec.SetRecordStateParam(2)
	// stub i63 pour franchir le dernier composant biped et enchaîner les records.
	filmdec.SetUnportedStubWidth("biped-action-component", 48)
	defer filmdec.SetUnportedStubWidth("biped-action-component", -1)

	var t0 []packet
	for _, p := range pkts {
		if p.typ == 0 {
			t0 = append(t0, p)
		}
	}
	fmt.Printf("=== chunk_%02d : %d paquets type-0 ; localisation des littéraux d'armes ===\n", chunkIdx, len(t0))

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
			if nm, ok := analysis.WeaponIDToName[id64]; ok {
				litBits = append(litBits, bp)
				litNames = append(litNames, nm)
			}
		}
		if len(litBits) == 0 {
			continue
		}
		// décode les records de ce paquet
		w := freshWorld(reg, worldPath)
		br := filmdec.NewBitReader(p.payload)
		recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
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
				hn := "(hors record décodé)"
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
	fmt.Printf("  >>> littéraux DANS un record décodé=%d ; HORS record décodé=%d\n", inRecord, outRecord)
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

// upstreamScan : pour chaque littéral d'arme (WST gate à bit B-1), cherche en amont,
// à TOUS les offsets [B-600, B], un header de record `[1 delta][R(11) low][R(2) tag]`
// dont le slot (low&0x3fffffff) ∈ 512-519. Mesure : combien de WST ont un slot biped
// plausible en amont, et la distribution des distances (révèle la structure du record
// biped : header -> ... -> WST). Test du POC "attribution par remontée".
func upstreamScan(chunkIdx int) {
	d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, chunkIdx))
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
			if _, ok := analysis.WeaponIDToName[(uint64(hi)<<32)|uint64(lo)]; !ok {
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
	fmt.Printf("=== chunk_%02d : %d WST gate=1 (littéraux d'armes) ===\n", chunkIdx, totalWST)
	fmt.Printf("  avec un header biped (slot 512-519) en amont [<=700 bits] : %d (%.0f%%)\n", withBiped, pct(withBiped, totalWST))
	fmt.Printf("  -- slots biped trouvés en amont (1er match) : ")
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
	d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, chunkIdx))
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
			if nm, ok := analysis.WeaponIDToName[(uint64(hi)<<32)|uint64(lo)]; ok {
				hits = append(hits, hit{pktIdx, p.ts, slot, bipedSlot[slot], nm, b})
			}
		}
		pktIdx++
	}
	fmt.Printf("=== chunk_%02d : %d littéraux d'armes ; 1er record du paquet = joueur ===\n", chunkIdx, len(hits))
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
	fmt.Printf("  >>> littéraux dont le paquet commence par un biped (512-519) : %d/%d\n", inBip, len(hits))
	_ = perSlot
	// Groupement par 1er slot du paquet (proxy d'entité), séquence temporelle d'armes.
	bySlot := map[uint32][]string{}
	for _, h := range hits {
		bySlot[h.firstSlot] = append(bySlot[h.firstSlot], h.arme)
	}
	fmt.Printf("  -- séquence temporelle d'armes par 1er-slot du paquet (idLow=11) --\n")
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
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, chunkIdx))
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
				if _, ok := analysis.WeaponIDToName[(uint64(hi)<<32)|uint64(lo)]; ok {
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
	fmt.Printf("=== litPattern sur chunks %v : %d littéraux d'armes complets ===\n", chunks, total)
	fmt.Printf("  gate bit[-1]=1 (pattern WST gate+variant) : %d (%.0f%%)\n", gate1, pct(gate1, total))
	fmt.Printf("  gate bit[-1]=0                            : %d (%.0f%%)\n", gate0, pct(gate0, total))
	fmt.Printf("  littéraux avec un 2e littéral dans [+64,+576] (paire d'armes) : %d (%.0f%%)\n", pairWithin, pct(pairWithin, total))
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}

// sizesMode liste les paquets type-0 d'un chunk par taille décroissante et compte les
// littéraux d'armes complets de chacun. Hypothèse : les GROS paquets type-0 sont des
// mini-keyframes (re-sync full-state des bipeds) qui retransmettent ~8 armes (1/joueur).
func sizesMode(chunkIdx int) {
	d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, chunkIdx))
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
			if nm, ok := analysis.WeaponIDToName[(uint64(hi)<<32)|uint64(lo)]; ok {
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
	fmt.Printf("=== chunk_%02d : %d paquets type-0 ; taille médiane=%d, min=%d, max=%d ===\n",
		chunkIdx, len(infos), med, sizes[0], sizes[len(sizes)-1])
	// top 15 par taille
	sort.Slice(infos, func(a, b int) bool { return infos[a].size > infos[b].size })
	fmt.Printf("  -- top 15 plus gros paquets type-0 (taille, #littéraux, armes) --\n")
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
	fmt.Printf("  -- gros paquets (>2x médiane=%d) : %d paquets, %d littéraux (%.1f/pkt)\n",
		2*med, bigPkts, bigLits, ratio(bigLits, bigPkts))
	fmt.Printf("  -- paquets normaux (<=2x médiane) : %d paquets, %d littéraux (%.3f/pkt)\n",
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
func litCtx(reg *filmdec.Registry, chunkIdx, pktIdx int) {
	d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, chunkIdx))
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
		nm, ok := analysis.WeaponIDToName[id64]
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

func freshWorld(reg *filmdec.Registry, path string) *filmdec.World {
	raw, _ := os.ReadFile(path)
	w := filmdec.NewWorld(reg)
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

func main() {
	// Mode upstream : pour chaque WST gate=1 (littéral d'arme), cherche en AMONT un header
	// de record dont le slot ∈ 512-519 (biped joueur). Teste l'attribution par remontée.
	if len(os.Args) >= 2 && os.Args[1] == "upstream" {
		chunkIdx, _ := strconv.Atoi(os.Args[2])
		upstreamScan(chunkIdx)
		return
	}

	// Mode litplayer : pour chaque littéral, donne le slot du 1er record du paquet (=joueur).
	if len(os.Args) >= 2 && os.Args[1] == "litplayer" {
		chunkIdx, _ := strconv.Atoi(os.Args[2])
		litPlayer(chunkIdx)
		return
	}

	// Mode litpattern : agrège la structure bit autour de TOUS les littéraux d'un chunk.
	if len(os.Args) >= 2 && os.Args[1] == "litpattern" {
		chunks := []int{3, 4, 10, 15, 20, 25}
		if len(os.Args) >= 3 {
			chunks = nil
			for _, a := range os.Args[2:] {
				if v, e := strconv.Atoi(a); e == nil {
					chunks = append(chunks, v)
				}
			}
		}
		litPattern(chunks)
		return
	}

	// Mode sizes : distribution des tailles de paquets type-0 + littéraux par gros paquet.
	if len(os.Args) >= 2 && os.Args[1] == "sizes" {
		chunkIdx, _ := strconv.Atoi(os.Args[2])
		sizesMode(chunkIdx)
		return
	}

	// Mode litctx : dump du contexte de bits autour des littéraux d'un paquet précis.
	// usage: litctx <chunk> <pktIndex>
	if len(os.Args) >= 2 && os.Args[1] == "litctx" {
		reg, _ := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
		chunkIdx, _ := strconv.Atoi(os.Args[2])
		pktIdx, _ := strconv.Atoi(os.Args[3])
		litCtx(reg, chunkIdx, pktIdx)
		return
	}

	// Mode litloc : localise les littéraux dans les records décodés.
	if len(os.Args) >= 2 && os.Args[1] == "litloc" {
		reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
		if err != nil {
			panic(err)
		}
		chunkIdx, maxPkts := 3, 1199
		if len(os.Args) >= 3 {
			chunkIdx, _ = strconv.Atoi(os.Args[2])
		}
		if len(os.Args) >= 4 {
			maxPkts, _ = strconv.Atoi(os.Args[3])
		}
		litLoc(reg, cache+"/world_dump.txt", chunkIdx, maxPkts)
		return
	}

	// Mode litscan : scan des littéraux d'armes dans des chunks gameplay.
	if len(os.Args) >= 2 && os.Args[1] == "litscan" {
		chunks := []int{2, 3, 4, 5, 10, 15, 20, 25}
		if len(os.Args) >= 3 {
			chunks = nil
			for _, a := range os.Args[2:] {
				if v, e := strconv.Atoi(a); e == nil {
					chunks = append(chunks, v)
				}
			}
		}
		for _, c := range chunks {
			litScan(c)
			fmt.Println()
		}
		return
	}

	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	_, typeSlots := loadWorld()
	fmt.Printf("registre : %d archétypes ; world : %d typeIndex distincts\n\n", len(reg.Archetypes), len(typeSlots))

	// Liste triée des typeIndex présents dans le World.
	var tis []int
	for ti := range typeSlots {
		tis = append(tis, ti)
	}
	sort.Ints(tis)

	fmt.Printf("================ ARCHÉTYPES PRÉSENTS DANS LE WORLD ================\n")
	for _, ti := range tis {
		arch, ok := reg.Archetype(ti)
		name := "(hors registre)"
		ncomp := 0
		if ok {
			ncomp = len(arch.Components)
			if ncomp > 0 {
				name = arch.Components[0]
			}
		}
		slots := typeSlots[ti]
		sort.Ints(slots)
		// résumé compact des slots
		slotStr := fmt.Sprintf("%d slots", len(slots))
		if len(slots) <= 12 {
			slotStr = fmt.Sprintf("slots=%v", slots)
		} else {
			slotStr = fmt.Sprintf("%d slots [%d..%d]", len(slots), slots[0], slots[len(slots)-1])
		}
		// composants weapon-ish
		var wcomp []string
		for _, c := range arch.Components {
			if weaponish(c) {
				wcomp = append(wcomp, c)
			}
		}
		flag := ""
		if len(wcomp) > 0 {
			flag = "  <<< WEAPON/LOADOUT: " + strings.Join(dedup(wcomp), ",")
		}
		fmt.Printf("  ti=%-3d  %-44s  ncomp=%-3d  %s%s\n", ti, truncate(name, 44), ncomp, slotStr, flag)
	}

	// Dump détaillé des composants des archétypes candidats (ceux weapon-ish ou
	// passés en argument).
	fmt.Printf("\n================ COMPOSANTS DES ARCHÉTYPES CANDIDATS ================\n")
	candidates := map[int]bool{}
	for _, ti := range tis {
		arch, ok := reg.Archetype(ti)
		if !ok {
			continue
		}
		for _, c := range arch.Components {
			if weaponish(c) {
				candidates[ti] = true
				break
			}
		}
	}
	// args = typeIndex supplémentaires à dumper
	for _, a := range os.Args[1:] {
		if v, e := strconv.Atoi(a); e == nil {
			candidates[v] = true
		}
	}
	var cands []int
	for ti := range candidates {
		cands = append(cands, ti)
	}
	sort.Ints(cands)
	for _, ti := range cands {
		arch, _ := reg.Archetype(ti)
		fmt.Printf("\n--- ti=%d (%d composants, %d slots dans le World) ---\n", ti, len(arch.Components), len(typeSlots[ti]))
		for i, c := range arch.Components {
			mark := ""
			if weaponish(c) {
				mark = "  <<<"
			}
			fmt.Printf("  i%-2d %s%s\n", i, c, mark)
		}
	}
}

func dedup(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
