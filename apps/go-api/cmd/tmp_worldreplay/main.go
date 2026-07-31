// tmp_worldreplay — REJOUER les deltas type-0 avec le World CAPTURÉ (slot->typeIndex).
//
// Différence majeure avec tmp_framedelta (World VIDE = OPTION B, tous les deltas
// désync à DesyncAt=0) : ici on charge le World capturé via debugger (world_dump.txt,
// 250 entités, bipeds typeIndex=35 = slots 512-519). Un delta sur un slot CONNU peut
// alors résoudre son archétype et décoder mask+composants.
//
// Étapes (cf. mission) :
//  1. parse world_dump.txt -> World rempli (BindFull(slot, typeIndex)).
//  2. extrait les type-0 d'un chunk gameplay (chunk_03 par défaut), framing
//     [u16 type][2o][u32 size][u64 ts][payload].
//  3. SWEEP FrameConfig (IDLowBits 8..16 × HasExtraFields {false,true}) ; pour chaque
//     combo : records décodés, slots World-connus, désyncs, records sur slots biped.
//  4. avec le meilleur combo : suit les bipeds 512-519, lit l'arme (WST i43-46).
//
// Usage : tmp_worldreplay [chunk_index] [npackets]
package main

import (
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

const bipedTypeIndex = 35

var bipedSlots = map[uint32]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}

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

// knownHigh32 : un id catalogue dont high-32 (uint32(id>>32)) == v.
func knownHigh32(v uint32) (string, bool) {
	for id, n := range analysis.WeaponIDToName {
		if uint32(id>>32) == v {
			return n, true
		}
	}
	return "", false
}

// knownLow32 : un id catalogue dont low-32 (uint32(id)) == v (le suffixe 42c9679f
// est partagé par presque toutes les armes → low-32 match = famille générique, peu
// discriminant ; on l'inclut pour diagnostic).
func knownLow32(v uint32) (string, bool) {
	for id, n := range analysis.WeaponIDToName {
		if uint32(id) == v {
			return n, true
		}
	}
	return "", false
}

// parseWorld lit world_dump.txt (paires slot:typeIndex séparées par espaces, # = commentaire)
// et binde chaque paire dans un World neuf.
func parseWorld(reg *filmdec.Registry, path string) (*filmdec.World, int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, err
	}
	w := filmdec.NewWorld(reg)
	n := 0
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
			w.BindFull(uint32(slot), uint32(ti)) // BindFull stocke à slot&0x3fffffff ; slots déjà bruts
			n++
		}
	}
	return w, n, nil
}

// freshWorld reconstruit un World propre (le décodage NEW/DEL le mute).
func freshWorld(reg *filmdec.Registry, path string) *filmdec.World {
	w, _, _ := parseWorld(reg, path)
	return w
}

type runResult struct {
	records     int
	knownSlots  int // records dont slot ∈ World (avant décodage)
	bipedSlots  int // records dont slot ∈ {512..519}
	desyncs     int
	cleanEnd    bool
	endBit      int
	totalBits   int
	desyncMsg   string
	typeHist    map[int]int
	desyncAtHst map[int]int // index composant du 1er désync
}

func tname(t int) string {
	switch t {
	case 1:
		return "new"
	case 2:
		return "del"
	case 3:
		return "delta"
	default:
		return fmt.Sprintf("t%d", t)
	}
}

func histStr(h map[int]int) string {
	keys := make([]int, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	s := ""
	for _, k := range keys {
		s += fmt.Sprintf("%s=%d ", tname(k), h[k])
	}
	if s == "" {
		return "(0)"
	}
	return s
}

// decodeOne décode un payload avec un World FRAIS (rechargé) et un cfg donné.
func decodeOne(payload []byte, reg *filmdec.Registry, worldPath string, cfg filmdec.FrameConfig) runResult {
	w := freshWorld(reg, worldPath)
	br := filmdec.NewBitReader(payload)
	recs, err := filmdec.DecodeFrameRecords(br, w, cfg)
	res := runResult{
		records:     len(recs),
		cleanEnd:    err == nil,
		endBit:      br.BitPos(),
		totalBits:   len(payload) * 8,
		typeHist:    map[int]int{},
		desyncAtHst: map[int]int{},
	}
	if err != nil {
		res.desyncMsg = err.Error()
	}
	// reconstruit un World pour tester l'appartenance des slots AVANT mutation
	ref := freshWorld(reg, worldPath)
	for _, r := range recs {
		res.typeHist[r.Type]++
		if _, ok := ref.ArchetypeForSlot(r.Slot); ok {
			res.knownSlots++
		}
		if bipedSlots[r.Slot] {
			res.bipedSlots++
		}
		if r.DesyncAt != -1 {
			res.desyncs++
			res.desyncAtHst[r.DesyncAt]++
		}
	}
	return res
}

func main() {
	chunkIdx := 3
	npkts := 12
	if len(os.Args) >= 2 {
		fmt.Sscanf(os.Args[1], "%d", &chunkIdx)
	}
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[2], "%d", &npkts)
	}
	filmdec.SetRecordStateParam(2) // bipeds (comme au keyframe)

	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	worldPath := cache + "/world_dump.txt"
	w0, nBound, err := parseWorld(reg, worldPath)
	if err != nil {
		panic(err)
	}
	nBiped := 0
	for s := uint32(512); s <= 519; s++ {
		if ti, ok := w0.ArchetypeForSlot(s); ok && ti == bipedTypeIndex {
			nBiped++
		}
	}
	fmt.Printf("=== WORLD chargé : %d entités bindées (Bound=%d) ; bipeds 512-519 présents=%d/8 ===\n",
		nBound, w0.Bound(), nBiped)

	chunkPath := fmt.Sprintf("%s/chunk_%02d.bin", cache, chunkIdx)
	data := inflate(chunkPath)
	pkts := listPackets(data)
	var t0 []packet
	for _, p := range pkts {
		if p.typ == 0 {
			t0 = append(t0, p)
		}
	}
	fmt.Printf("chunk_%02d : %d octets inflatés, %d paquets total, %d type-0 (FRAME)\n",
		chunkIdx, len(data), len(pkts), len(t0))
	if len(t0) == 0 {
		fmt.Println("aucun type-0 — abandon")
		return
	}

	// ---- SWEEP FrameConfig sur le 1er type-0 ----
	fmt.Printf("\n=== SWEEP FrameConfig — type-0 #0 (size=%d, %d bits) — World REMPLI ===\n",
		t0[0].size, t0[0].size*8)
	fmt.Printf("%-6s %-9s %-8s %-8s %-8s %-8s %-9s %-11s %s\n",
		"extra", "idLowB", "records", "known", "biped", "desyncs", "cleanEnd", "endBit/tot", "types")
	type combo struct {
		cfg filmdec.FrameConfig
		res runResult
	}
	var all []combo
	for _, extra := range []bool{false, true} {
		for idLow := 8; idLow <= 16; idLow++ {
			cfg := filmdec.FrameConfig{HasExtraFields: extra, IDLowBits: idLow}
			res := decodeOne(t0[0].payload, reg, worldPath, cfg)
			fmt.Printf("%-6v %-9d %-8d %-8d %-8d %-8d %-9v %d/%d  %s\n",
				extra, idLow, res.records, res.knownSlots, res.bipedSlots, res.desyncs,
				res.cleanEnd, res.endBit, res.totalBits, histStr(res.typeHist))
			all = append(all, combo{cfg, res})
		}
	}
	// score : maximiser known-slots ; bonus cleanEnd ; pénaliser désyncs
	score := func(r runResult) int {
		s := r.knownSlots*10 + r.records
		if r.cleanEnd {
			s += 5000
		}
		s -= r.desyncs * 3
		return s
	}
	sort.SliceStable(all, func(i, j int) bool { return score(all[i].res) > score(all[j].res) })
	bc := all[0]
	fmt.Printf("\n>>> meilleur combo (#0) : extra=%v idLowBits=%d -> records=%d known=%d biped=%d desyncs=%d cleanEnd=%v end=%d/%d\n",
		bc.cfg.HasExtraFields, bc.cfg.IDLowBits, bc.res.records, bc.res.knownSlots, bc.res.bipedSlots,
		bc.res.desyncs, bc.res.cleanEnd, bc.res.endBit, bc.res.totalBits)
	if bc.res.desyncMsg != "" {
		fmt.Printf("    1er désync : %s\n", bc.res.desyncMsg)
	}

	// ---- même combo sur N paquets : stabilité ----
	fmt.Printf("\n=== meilleur combo sur les %d premiers type-0 ===\n", npkts)
	aggKnown, aggRec, aggDes, aggClean := 0, 0, 0, 0
	for i := 0; i < len(t0) && i < npkts; i++ {
		res := decodeOne(t0[i].payload, reg, worldPath, bc.cfg)
		aggKnown += res.knownSlots
		aggRec += res.records
		aggDes += res.desyncs
		if res.cleanEnd {
			aggClean++
		}
		fmt.Printf("  #%d size=%-5d records=%-4d known=%-4d biped=%-3d desyncs=%-3d cleanEnd=%-5v end=%d/%d types=%s\n",
			i, t0[i].size, res.records, res.knownSlots, res.bipedSlots, res.desyncs, res.cleanEnd,
			res.endBit, res.totalBits, histStr(res.typeHist))
	}
	fmt.Printf("  AGG : records=%d known=%d desyncs=%d cleanEnd=%d/%d\n", aggRec, aggKnown, aggDes, aggClean, min(npkts, len(t0)))

	// ---- SWEEP "1er record slot connu" sur TOUS les paquets (métrique robuste) ----
	// Critère : pour un idLowBits donné, sur combien de paquets le 1ER record (un delta,
	// bit0=1) tombe sur un slot du World ? Le bon idLowBits doit le faire sur ~tous.
	fmt.Printf("\n=== SWEEP 'slot du 1er record ∈ World' sur les %d type-0 (bit0, extra=false) ===\n", len(t0))
	fmt.Printf("%-9s %-12s %-12s %-12s\n", "idLowBits", "1er∈World", "1er∈biped", "ex.slots(5)")
	for idLow := 8; idLow <= 16; idLow++ {
		inWorld, inBiped := 0, 0
		var ex []uint32
		refW := freshWorld(reg, worldPath)
		for _, p := range t0 {
			br := filmdec.NewBitReader(p.payload)
			typ := readTypeLocal(br)
			id := readIDLocal(br, idLow)
			slot := id & 0x3fffffff
			_ = typ
			if _, ok := refW.ArchetypeForSlot(slot); ok {
				inWorld++
			}
			if bipedSlots[slot] {
				inBiped++
			}
			if len(ex) < 5 {
				ex = append(ex, slot)
			}
		}
		fmt.Printf("%-9d %-12d %-12d %v\n", idLow, inWorld, inBiped, ex)
	}

	// Verdict de calibration : le sweep ci-dessus tranche en faveur de extra=false /
	// idLowBits=11 (le 1er record de ~tous les paquets tombe sur un slot biped). On
	// fige ce combo pour le dump détaillé et le suivi des armes (le scoring "cleanEnd"
	// du #0 favorisait à tort un combo qui s'arrête vite).
	calCfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}
	fmt.Printf("\n=== COMBO CALIBRÉ retenu : extra=false idLowBits=11 ===\n")

	// ---- dump détaillé des records du paquet #0 ----
	fmt.Printf("\n=== DUMP records type-0 #0 (combo calibré) ===\n")
	dumpRecords(t0[0].payload, reg, worldPath, calCfg)

	// ---- suivi des bipeds + lecture d'arme avec le combo calibré ----
	fmt.Printf("\n=== SUIVI BIPEDS 512-519 + ARME (WST i43-46) — combo calibré ===\n")
	trackBipeds(t0, reg, worldPath, calCfg, npkts)

	// ---- anatomie du 1er delta biped (composants présents + largeur de chaque) ----
	fmt.Printf("\n=== ANATOMIE delta biped slot=519 du paquet #0 (combo calibré) ===\n")
	anatomyBipedDelta(t0[0].payload, reg, worldPath, calCfg)

	// ---- histogramme des désyncs biped sur N paquets ----
	fmt.Printf("\n=== HISTOGRAMME désyncs deltas biped (combo calibré, %d paquets) ===\n", npkts)
	desyncHistogram(t0, reg, worldPath, calCfg, npkts)

	// ---- scan d'offset WST : pour chaque WST présent dans un delta biped, chercher
	//      un R(32) == arme connue (high32) dans la fenêtre [start, start+w] ----
	fmt.Printf("\n=== SCAN OFFSET WST (arme connue high32 dans la fenêtre du composant) — %d paquets ===\n", npkts)
	scanWSTOffsets(t0, reg, worldPath, calCfg, npkts)

	// ---- brute-force largeur i63 (biped-action-component) par chaînage de records ----
	fmt.Printf("\n=== BRUTE-FORCE largeur i63 biped-action-component (chaînage records) ===\n")
	bruteForceI63(t0, reg, worldPath, calCfg, npkts)

	// ---- recherche d'un WST i43/i45 gate=1 (arme PRIMAIRE transmise) sur tout le chunk ----
	fmt.Printf("\n=== RECHERCHE WST i43/i45 gate=1 (arme primaire (re)transmise) sur TOUT le chunk ===\n")
	huntPrimaryWST(t0, reg, worldPath, calCfg)

	// ---- combien de bipeds DISTINCTS atteint-on dans un paquet (i63 PORTÉ, sans stub) ? ----
	fmt.Printf("\n=== BIPEDS DISTINCTS atteints par paquet (i63 PORTÉ, cas commun) ===\n")
	distinctBipeds(t0, reg, worldPath, calCfg, npkts)

	// ---- inventaire des typeIndex qui désync APRÈS i63 (= reste-à-porter pour le walk) ----
	fmt.Printf("\n=== RESTE-À-PORTER : typeIndex/composants qui désync (tout le chunk) ===\n")
	desyncByArchetype(t0, reg, worldPath, calCfg)

	// ---- couverture biped sur tout le chunk : quels slots, combien clean ----
	fmt.Printf("\n=== COUVERTURE BIPED sur tout le chunk (i63 porté) ===\n")
	bipedCoverage(t0, reg, worldPath, calCfg)

	// ---- diagnostic i63 : count1 R(4) lu, et sweep count2 (popcount RAM) ----
	fmt.Printf("\n=== DIAGNOSTIC i63 : distribution count1=R(4) sur deltas biped clean-jusqu'à-i63 ===\n")
	diagI63(t0, reg, worldPath, calCfg)
}

// diagI63 mesure, pour les deltas biped dont i63 est présent, la valeur de count1=R(4)
// (les 4 bits juste après les 96 bits du sous-bloc début), et sweep bipedActionLoop2Count
// pour voir si un count2 fixe augmente la fraction de records biped clean.
func diagI63(t0 []packet, reg *filmdec.Registry, worldPath string, cfg filmdec.FrameConfig) {
	// distribution de count1 : on relit les bits à partir du start d'i63 dans les records
	// dont i63 est présent (porté ou désync), via le StartBit du composant.
	count1Hist := map[uint64]int{}
	i63Present := 0
	for i := range t0 {
		w := freshWorld(reg, worldPath)
		br := filmdec.NewBitReader(t0[i].payload)
		recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
		for _, r := range recs {
			if !bipedSlots[r.Slot] {
				continue
			}
			for _, c := range r.Trace.Comps {
				if c.Name != "biped-action-component" {
					continue
				}
				i63Present++
				// i63 layout : [96 bits sous-bloc][R(4) count1]...
				count1 := bitsAt(t0[i].payload, c.StartBit+96, 4)
				count1Hist[count1]++
			}
		}
	}
	fmt.Printf("  i63 présents (StartBit capturé)=%d ; distribution count1=R(4)@+96 :\n", i63Present)
	var keys []uint64
	for k := range count1Hist {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool { return keys[a] < keys[b] })
	for _, k := range keys {
		fmt.Printf("    count1=%d : %d\n", k, count1Hist[k])
	}

	// sweep count2 (FUN_1409fe718 popcount RAM, invisible en delta)
	fmt.Printf("  -- sweep bipedActionLoop2Count (popcount RAM) : records biped clean total --\n")
	for c2 := 0; c2 <= 8; c2++ {
		filmdec.SetBipedActionLoop2Count(c2)
		clean := 0
		for i := range t0 {
			w := freshWorld(reg, worldPath)
			br := filmdec.NewBitReader(t0[i].payload)
			recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
			for _, r := range recs {
				if bipedSlots[r.Slot] && r.DesyncAt == -1 {
					clean++
				}
			}
		}
		fmt.Printf("    count2=%d : bipeds clean=%d\n", c2, clean)
	}
	filmdec.SetBipedActionLoop2Count(0)
}

// bipedCoverage compte, sur tout le chunk, les records biped par slot et la fraction
// qui décode sans désync (DesyncAt=-1) — pour quantifier l'attribution d'arme atteignable.
func bipedCoverage(t0 []packet, reg *filmdec.Registry, worldPath string, cfg filmdec.FrameConfig) {
	type stat struct{ total, clean, withWST int }
	per := map[uint32]*stat{}
	firstRecBiped, firstRecBipedClean := 0, 0
	for i := range t0 {
		w := freshWorld(reg, worldPath)
		br := filmdec.NewBitReader(t0[i].payload)
		recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
		for j, r := range recs {
			if !bipedSlots[r.Slot] {
				continue
			}
			s := per[r.Slot]
			if s == nil {
				s = &stat{}
				per[r.Slot] = s
			}
			s.total++
			if r.DesyncAt == -1 {
				s.clean++
			}
			for _, c := range r.Trace.Comps {
				if c.Name == "weapon-state-type-info" {
					s.withWST++
					break
				}
			}
			if j == 0 {
				firstRecBiped++
				if r.DesyncAt == -1 {
					firstRecBipedClean++
				}
			}
		}
	}
	var slots []int
	for s := range per {
		slots = append(slots, int(s))
	}
	sort.Ints(slots)
	for _, s := range slots {
		st := per[uint32(s)]
		fmt.Printf("  slot=%d : %d records biped, clean=%d (%.0f%%), portant un WST=%d\n",
			s, st.total, st.clean, 100*float64(st.clean)/float64(st.total), st.withWST)
	}
	fmt.Printf("  1er-record-du-paquet est un biped : %d/%d paquets ; dont clean : %d\n",
		firstRecBiped, len(t0), firstRecBipedClean)
}

// distinctBipeds mesure, avec i63 RÉELLEMENT PORTÉ (plus de stub), combien de bipeds
// DISTINCTS (512-519) un paquet expose et où le record loop bute ensuite.
func distinctBipeds(t0 []packet, reg *filmdec.Registry, worldPath string, cfg filmdec.FrameConfig, npkts int) {
	for i := 0; i < len(t0) && i < npkts && i < 8; i++ {
		w := freshWorld(reg, worldPath)
		br := filmdec.NewBitReader(t0[i].payload)
		recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
		seen := map[uint32]bool{}
		firstDesync := ""
		bipedClean := 0
		for _, r := range recs {
			if bipedSlots[r.Slot] {
				seen[r.Slot] = true
				if r.DesyncAt == -1 {
					bipedClean++
				}
			}
			if r.DesyncAt != -1 && firstDesync == "" {
				if arch, ok := reg.Archetype(int(r.TypeIndex)); ok && r.DesyncAt < len(arch.Components) {
					firstDesync = fmt.Sprintf("slot=%d typeIdx=%d @i%d:%s", r.Slot, r.TypeIndex, r.DesyncAt, arch.Components[r.DesyncAt])
				}
			}
		}
		var slots []int
		for s := range seen {
			slots = append(slots, int(s))
		}
		sort.Ints(slots)
		fmt.Printf("  pkt#%d : %d records, bipeds distincts=%v (clean=%d) ; 1er désync: %s\n",
			i, len(recs), slots, bipedClean, firstDesync)
	}
}

// desyncByArchetype balaie TOUT le chunk et histogramme, par (typeIndex, composant),
// le 1er composant qui désync — pour cartographier le reste-à-porter du walk complet.
func desyncByArchetype(t0 []packet, reg *filmdec.Registry, worldPath string, cfg filmdec.FrameConfig) {
	hist := map[string]int{}
	totalRecs, cleanRecs := 0, 0
	for i := range t0 {
		w := freshWorld(reg, worldPath)
		br := filmdec.NewBitReader(t0[i].payload)
		recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
		for _, r := range recs {
			totalRecs++
			if r.DesyncAt == -1 {
				cleanRecs++
				continue
			}
			key := fmt.Sprintf("typeIdx=%-3d i%-2d", r.TypeIndex, r.DesyncAt)
			if arch, ok := reg.Archetype(int(r.TypeIndex)); ok && r.DesyncAt < len(arch.Components) {
				key = fmt.Sprintf("typeIdx=%-3d i%-2d %s", r.TypeIndex, r.DesyncAt, arch.Components[r.DesyncAt])
			}
			hist[key]++
		}
	}
	fmt.Printf("  records totaux=%d ; clean(DesyncAt=-1)=%d ; désync=%d\n", totalRecs, cleanRecs, totalRecs-cleanRecs)
	type kv struct {
		k string
		v int
	}
	var arr []kv
	for k, v := range hist {
		arr = append(arr, kv{k, v})
	}
	sort.Slice(arr, func(a, b int) bool { return arr[a].v > arr[b].v })
	for k := 0; k < len(arr) && k < 25; k++ {
		fmt.Printf("    désync @%-45s : %d\n", arr[k].k, arr[k].v)
	}
}

// huntPrimaryWST balaie TOUT le chunk et, pour les deltas biped portant un WST i43 ou
// i45 (slots d'arme primaire/secondaire) avec gate interne=1, dump le handle (high32)
// et teste l'arme. Au keyframe l'arme est dans handle@+1 (=high32 famille). Si la
// traversée delta est bit-exacte jusqu'à i43, un gate=1 doit y exposer une arme connue.
func huntPrimaryWST(t0 []packet, reg *filmdec.Registry, worldPath string, cfg filmdec.FrameConfig) {
	found, dumped := 0, 0
	for i := range t0 {
		w := freshWorld(reg, worldPath)
		br := filmdec.NewBitReader(t0[i].payload)
		recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
		for _, r := range recs {
			if !bipedSlots[r.Slot] {
				continue
			}
			for _, c := range r.Trace.Comps {
				if c.Name != "weapon-state-type-info" || (c.Index != 43 && c.Index != 45) {
					continue
				}
				if bitsAt(t0[i].payload, c.StartBit, 1) != 1 {
					continue // gate=0 : arme inchangée
				}
				found++
				h := uint32(bitsAt(t0[i].payload, c.StartBit+1, 32))
				v := uint32(bitsAt(t0[i].payload, c.StartBit+33, 32))
				hn, hOk := knownHigh32(h)
				vn, vOk := knownHigh32(v)
				if (hOk || vOk) || dumped < 20 {
					tag := ""
					if hOk {
						tag = " HANDLE=" + hn
					}
					if vOk {
						tag += " VARIANT=" + vn
					}
					fmt.Printf("    pkt#%d slot=%d i%d gate=1 handle=0x%08x variant=0x%08x%s\n",
						i, r.Slot, c.Index, h, v, tag)
					dumped++
				}
			}
		}
	}
	fmt.Printf("  >>> %d WST i43/i45 gate=1 trouvés sur %d paquets\n", found, len(t0))

	// Scan: pour chaque WST i43/i45 gate=1, chercher une arme connue (high32) à TOUT
	// offset interne [start, start+260] — révèle si le handle est présent mais décalé.
	fmt.Printf("  -- scan arme connue (high32) à tout offset interne des WST i43/i45 gate=1 --\n")
	offHits := map[int]int{}
	withArme := 0
	for i := range t0 {
		w := freshWorld(reg, worldPath)
		br := filmdec.NewBitReader(t0[i].payload)
		recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
		for _, r := range recs {
			if !bipedSlots[r.Slot] {
				continue
			}
			for _, c := range r.Trace.Comps {
				if c.Name != "weapon-state-type-info" || (c.Index != 43 && c.Index != 45) {
					continue
				}
				if bitsAt(t0[i].payload, c.StartBit, 1) != 1 {
					continue
				}
				hit := false
				for off := 0; off <= 260; off++ {
					vv := uint32(bitsAt(t0[i].payload, c.StartBit+off, 32))
					if _, ok := knownHigh32(vv); ok {
						offHits[off]++
						hit = true
					}
				}
				if hit {
					withArme++
				}
			}
		}
	}
	fmt.Printf("  WST i43/i45 gate=1 dont la fenêtre contient une arme connue (à un offset quelconque) : %d\n", withArme)
	type kv struct{ off, n int }
	var arr []kv
	for o, n := range offHits {
		arr = append(arr, kv{o, n})
	}
	sort.Slice(arr, func(a, b int) bool { return arr[a].n > arr[b].n })
	for k := 0; k < len(arr) && k < 12; k++ {
		fmt.Printf("    offset +%-4d : %d hits\n", arr[k].off, arr[k].n)
	}
}

// bruteForceI63 teste des largeurs pour i63 (seul deser manquant sur le chemin biped)
// et mesure, pour chaque largeur, combien de records décodent au total et combien
// tombent sur des slots du World. La bonne largeur maximise les records-connus enchaînés
// (le record suivant doit démarrer proprement). Évalué sur les premiers paquets.
func bruteForceI63(t0 []packet, reg *filmdec.Registry, worldPath string, cfg filmdec.FrameConfig, npkts int) {
	const comp = "biped-action-component"
	type score struct {
		w           int
		recs        int
		known       int
		bipeds      int
		desyncOther int // désyncs sur un AUTRE composant que i63 (signe de mauvaise largeur)
	}
	var results []score
	refW := freshWorld(reg, worldPath)
	for w := 0; w <= 80; w++ {
		filmdec.SetUnportedStubWidth(comp, w)
		sc := score{w: w}
		for i := 0; i < len(t0) && i < npkts; i++ {
			wd := freshWorld(reg, worldPath)
			br := filmdec.NewBitReader(t0[i].payload)
			recs, _ := filmdec.DecodeFrameRecords(br, wd, cfg)
			for _, r := range recs {
				sc.recs++
				if _, ok := refW.ArchetypeForSlot(r.Slot); ok {
					sc.known++
				}
				if bipedSlots[r.Slot] {
					sc.bipeds++
				}
				if r.DesyncAt != -1 {
					if arch, ok := reg.Archetype(int(r.TypeIndex)); ok && r.DesyncAt < len(arch.Components) &&
						arch.Components[r.DesyncAt] != comp {
						sc.desyncOther++
					}
				}
			}
		}
		results = append(results, sc)
	}
	filmdec.SetUnportedStubWidth(comp, -1) // clear

	// meilleure largeur = max known (puis max recs), avec peu de désyncs-autres
	sort.SliceStable(results, func(a, b int) bool {
		if results[a].known != results[b].known {
			return results[a].known > results[b].known
		}
		return results[a].recs > results[b].recs
	})
	fmt.Println("  top 12 largeurs i63 (par records-connus enchaînés) :")
	for k := 0; k < len(results) && k < 12; k++ {
		r := results[k]
		fmt.Printf("    w=%-3d : recs=%-5d known=%-5d bipeds=%-4d desyncAutres=%d\n",
			r.w, r.recs, r.known, r.bipeds, r.desyncOther)
	}
}

// scanWSTOffsets : pour chaque WST porté dans un delta biped, balaie chaque offset de
// bit dans [start, end] et teste si le R(32) à cet offset matche une arme du catalogue
// (high32 OU low32). Révèle si l'arme est présente mais à un offset différent de +33.
func scanWSTOffsets(t0 []packet, reg *filmdec.Registry, worldPath string, cfg filmdec.FrameConfig, npkts int) {
	hitsByOff := map[int]int{}
	wstTotal := 0
	gate1 := 0 // WST avec gate interne = 1 (slot répliqué = arme (re)transmise dans ce delta)
	armed := 0 // WST dont la fenêtre contient au moins une arme connue
	for i := 0; i < len(t0) && i < npkts; i++ {
		w := freshWorld(reg, worldPath)
		br := filmdec.NewBitReader(t0[i].payload)
		recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
		for _, r := range recs {
			if !bipedSlots[r.Slot] {
				continue
			}
			comps := r.Trace.Comps
			for k, c := range comps {
				if c.Name != "weapon-state-type-info" {
					continue
				}
				wstTotal++
				if bitsAt(t0[i].payload, c.StartBit, 1) == 1 {
					gate1++
				}
				end := c.StartBit
				if k+1 < len(comps) {
					end = comps[k+1].StartBit
				} else {
					end = c.StartBit + 256
				}
				found := false
				for off := c.StartBit; off+32 <= end; off++ {
					v := uint32(bitsAt(t0[i].payload, off, 32))
					if _, ok := knownHigh32(v); ok {
						hitsByOff[off-c.StartBit]++
						found = true
					}
				}
				if found {
					armed++
				}
			}
		}
	}
	// dump des 1ers WST : gate + handle + variant lus par le code, + 12 R(32) à partir du start
	fmt.Printf("  -- dump bits des 4 premiers WST portés (gate=bit0, code lit handle@+1 variant@+33) --\n")
	shown := 0
	for i := 0; i < len(t0) && i < npkts && shown < 4; i++ {
		w := freshWorld(reg, worldPath)
		br := filmdec.NewBitReader(t0[i].payload)
		recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
		for _, r := range recs {
			if !bipedSlots[r.Slot] {
				continue
			}
			for _, c := range r.Trace.Comps {
				if c.Name != "weapon-state-type-info" || shown >= 4 {
					continue
				}
				gate := bitsAt(t0[i].payload, c.StartBit, 1)
				h := uint32(bitsAt(t0[i].payload, c.StartBit+1, 32))
				v := uint32(bitsAt(t0[i].payload, c.StartBit+33, 32))
				fmt.Printf("    pkt#%d i%d start=%d gate=%d handle@+1=0x%08x variant@+33=0x%08x | R(32)@+0..+5: ",
					i, c.Index, c.StartBit, gate, h, v)
				for o := 0; o <= 5; o++ {
					fmt.Printf("0x%08x ", uint32(bitsAt(t0[i].payload, c.StartBit+o, 32)))
				}
				fmt.Println()
				shown++
			}
		}
	}

	fmt.Printf("  WST portés=%d ; gate interne=1 (arme transmise)=%d ; gate=0 (arme inchangée)=%d ; arme connue dans la fenêtre=%d\n",
		wstTotal, gate1, wstTotal-gate1, armed)
	type kv struct{ off, n int }
	var arr []kv
	for o, n := range hitsByOff {
		arr = append(arr, kv{o, n})
	}
	sort.Slice(arr, func(a, b int) bool { return arr[a].n > arr[b].n })
	fmt.Println("  hits par offset relatif au start du WST (top 15) :")
	for k := 0; k < len(arr) && k < 15; k++ {
		fmt.Printf("    +%-4d : %d hits\n", arr[k].off, arr[k].n)
	}
}

// anatomyBipedDelta décode le paquet jusqu'au 1er delta biped et liste chaque composant
// présent (mask) avec son start/end bit et sa largeur — pour repérer où la largeur dérive.
func anatomyBipedDelta(payload []byte, reg *filmdec.Registry, worldPath string, cfg filmdec.FrameConfig) {
	w := freshWorld(reg, worldPath)
	br := filmdec.NewBitReader(payload)
	recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
	for _, r := range recs {
		if !bipedSlots[r.Slot] {
			continue
		}
		fmt.Printf("  delta slot=%d typeIdx=%d mask=0x%016x (popcount=%d) desyncAt=i%d\n",
			r.Slot, r.TypeIndex, r.Trace.Mask, popcount(r.Trace.Mask), r.DesyncAt)
		comps := r.Trace.Comps
		for k, c := range comps {
			end := c.StartBit
			if k+1 < len(comps) {
				end = comps[k+1].StartBit
			}
			w := end - c.StartBit
			extra := ""
			if c.Name == "weapon-state-type-info" && c.Variant != noVariantU {
				if hn, ok := knownHigh32(c.Variant); ok {
					extra = " ARME(variant.high32)=" + hn
				}
			}
			fmt.Printf("    i%-2d %-45s start=%-6d w=%-4d ported=%-5v%s\n",
				c.Index, c.Name, c.StartBit, w, c.Ported, extra)
		}
		return
	}
	fmt.Println("  (aucun delta biped dans ce paquet)")
}

// desyncHistogram décode N paquets et compte, pour les deltas biped, quel composant
// est le 1er à désync (par nom) — révèle les desers manquants par fréquence.
func desyncHistogram(t0 []packet, reg *filmdec.Registry, worldPath string, cfg filmdec.FrameConfig, npkts int) {
	hist := map[string]int{}
	total, clean := 0, 0
	for i := 0; i < len(t0) && i < npkts; i++ {
		w := freshWorld(reg, worldPath)
		br := filmdec.NewBitReader(t0[i].payload)
		recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
		for _, r := range recs {
			if !bipedSlots[r.Slot] {
				continue
			}
			total++
			if r.DesyncAt == -1 {
				clean++
				continue
			}
			name := fmt.Sprintf("i%d", r.DesyncAt)
			if arch, ok := reg.Archetype(int(r.TypeIndex)); ok && r.DesyncAt < len(arch.Components) {
				name = fmt.Sprintf("i%d:%s", r.DesyncAt, arch.Components[r.DesyncAt])
			}
			hist[name]++
		}
	}
	fmt.Printf("  deltas biped: %d ; sans désync: %d\n", total, clean)
	type kv struct {
		k string
		v int
	}
	var arr []kv
	for k, v := range hist {
		arr = append(arr, kv{k, v})
	}
	sort.Slice(arr, func(a, b int) bool { return arr[a].v > arr[b].v })
	for _, e := range arr {
		fmt.Printf("    désync @%-45s : %d\n", e.k, e.v)
	}
}

const noVariantU uint32 = 0xFFFFFFFF

func popcount(x uint64) int {
	n := 0
	for x != 0 {
		n += int(x & 1)
		x >>= 1
	}
	return n
}

// readTypeLocal : prefix-code R(1); 1->delta(3); sinon R(2).
func readTypeLocal(br *filmdec.BitReader) int {
	if br.ReadBit() {
		return 3
	}
	return int(br.ReadBits(2))
}

// readIDLocal : low=R(idLowBits) ; tag=R(2)<<30.
func readIDLocal(br *filmdec.BitReader, idLowBits int) uint32 {
	var low uint32
	if idLowBits > 0 {
		low = uint32(br.ReadBits(uint(idLowBits)))
	}
	tag := uint32(br.ReadBits(2))
	return (tag << 30) | (low & 0x3fffffff)
}

// dumpRecords décode un paquet et liste chaque record (slot, type, typeIndex, known?, desyncAt).
func dumpRecords(payload []byte, reg *filmdec.Registry, worldPath string, cfg filmdec.FrameConfig) {
	ref := freshWorld(reg, worldPath)
	w := freshWorld(reg, worldPath)
	br := filmdec.NewBitReader(payload)
	recs, err := filmdec.DecodeFrameRecords(br, w, cfg)
	for i, r := range recs {
		_, known := ref.ArchetypeForSlot(r.Slot)
		desyncName := ""
		if r.DesyncAt != -1 {
			if arch, ok := reg.Archetype(int(r.TypeIndex)); ok && r.DesyncAt < len(arch.Components) {
				desyncName = " @" + arch.Components[r.DesyncAt]
			}
		}
		fmt.Printf("  rec#%d type=%-5s slot=%-5d id=0x%08x typeIdx=%d known=%-5v desyncAt=i%d%s comps=%d\n",
			i, tname(r.Type), r.Slot, r.ID, r.TypeIndex, known, r.DesyncAt, desyncName, len(r.Trace.Comps))
	}
	if err != nil {
		fmt.Printf("  err: %v\n", err)
	}
}

// trackBipeds décode N paquets avec le meilleur combo et, pour chaque record DELTA/NEW
// sur un slot biped 512-519, regarde si la trace porte un weapon-state-type-info (i43-46),
// lit handle/variant et tente un match catalogue.
func trackBipeds(t0 []packet, reg *filmdec.Registry, worldPath string, cfg filmdec.FrameConfig, npkts int) {
	bipedRecords, bipedDecoded, wstSeen, wstHit := 0, 0, 0, 0
	for i := 0; i < len(t0) && i < npkts; i++ {
		w := freshWorld(reg, worldPath)
		br := filmdec.NewBitReader(t0[i].payload)
		recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
		for _, r := range recs {
			if !bipedSlots[r.Slot] {
				continue
			}
			bipedRecords++
			ok := r.DesyncAt == -1
			if ok {
				bipedDecoded++
			}
			compsReached := len(r.Trace.Comps)
			deepest := -1
			hasWST := false
			for _, c := range r.Trace.Comps {
				if c.Index > deepest {
					deepest = c.Index
				}
				if c.Name == "weapon-state-type-info" {
					hasWST = true
					wstSeen++
					handle := uint32(bitsAt(t0[i].payload, c.StartBit+1, 32))
					variant := uint32(bitsAt(t0[i].payload, c.StartBit+filmdec.VariantBitOffsetInWST, 32))
					hn, hOk := knownHigh32(handle)
					vn, vOk := knownHigh32(variant)
					_, vLowOk := knownLow32(variant)
					_, hLowOk := knownLow32(handle)
					tag := ""
					if hOk {
						tag += " handleHIGH=" + hn
						wstHit++
					}
					if vOk {
						tag += " variantHIGH=" + vn
						wstHit++
					}
					if vLowOk {
						tag += " variantLOW(suffix-only)"
					}
					if hLowOk {
						tag += " handleLOW(suffix-only)"
					}
					fmt.Printf("    pkt#%d slot=%d WST@i%d var=0x%08x handle=0x%08x desyncAt=%d%s\n",
						i, r.Slot, c.Index, variant, handle, r.DesyncAt, tag)
				}
			}
			if !hasWST {
				fmt.Printf("    pkt#%d slot=%d type=%s comps=%d deepest=i%d desyncAt=%d (pas de WST dans ce delta)\n",
					i, r.Slot, tname(r.Type), compsReached, deepest, r.DesyncAt)
			}
		}
	}
	fmt.Printf("  >>> bipedRecords=%d décodés-sans-désync=%d ; WST rencontrés=%d ; hits arme(high32)=%d\n",
		bipedRecords, bipedDecoded, wstSeen, wstHit)
}
