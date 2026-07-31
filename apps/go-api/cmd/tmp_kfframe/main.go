// tmp_kfframe — THROWAWAY. Question make-or-break : peut-on énumérer les records
// (id, typeIndex) du KEYFRAME type-2 SANS décoder intégralement les composants ?
//
// On inspecte le DÉBUT du payload type-2 (compte d'entités ? table (id,typeIndex) ?),
// et on mesure l'espacement des marqueurs byte-alignés candidats (a0 7b 42, etc.).
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

func inflate(p string) []byte {
	raw, err := os.ReadFile(p)
	if err != nil {
		panic(err)
	}
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

type pkt struct {
	typ     uint16
	size    int
	ts      uint64
	payload []byte
	off     int
}

// allPackets walks the [Type u16@0][?u16@2][Size u32@4][Ts u64@8][payload@16] framing.
func allPackets(d []byte) []pkt {
	var out []pkt
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		out = append(out, pkt{typ, sz, ts, d[off+16 : off+16+sz], off})
		off += 16 + sz
	}
	return out
}

func hexdump(b []byte, n int) string {
	if n > len(b) {
		n = len(b)
	}
	s := ""
	for i := 0; i < n; i++ {
		s += fmt.Sprintf("%02x ", b[i])
		if (i+1)%16 == 0 {
			s += "\n  "
		}
	}
	return s
}

// bit reader helpers (MSB-first, same convention as filmdec.BitReader).
func bitAt(d []byte, p int) uint64 {
	if p < 0 || p>>3 >= len(d) {
		return 0
	}
	return uint64((d[p>>3] >> uint(7-(p&7))) & 1)
}
func rb(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		v = (v << 1) | bitAt(d, bp+i)
	}
	return v
}

// countMarker counts byte-aligned occurrences of pat in d and returns their offsets.
func countMarker(d, pat []byte) []int {
	var pos []int
	for i := 0; i+len(pat) <= len(d); i++ {
		if bytes.Equal(d[i:i+len(pat)], pat) {
			pos = append(pos, i)
		}
	}
	return pos
}

func gaps(pos []int) (min, max, mean int) {
	if len(pos) < 2 {
		return 0, 0, 0
	}
	min = 1 << 30
	sum := 0
	for i := 1; i < len(pos); i++ {
		g := pos[i] - pos[i-1]
		if g < min {
			min = g
		}
		if g > max {
			max = g
		}
		sum += g
	}
	mean = sum / (len(pos) - 1)
	return
}

func main() {
	// ---- registry (chunk_00) : combien d'archétypes, largeur typeIndex ----
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	fmt.Printf("registry: %d archetypes (typeIndex sur R(6)=6 bits, max 63)\n", len(reg.Archetypes))

	// ---- type-2 packet ----
	d2 := inflate(cache + "/chunk_02.bin")
	pkts := allPackets(d2)
	fmt.Printf("chunk_02 inflated = %d bytes, %d packets\n", len(d2), len(pkts))
	fmt.Println("packet types présents :")
	byType := map[uint16][]pkt{}
	for _, p := range pkts {
		byType[p.typ] = append(byType[p.typ], p)
	}
	var types []int
	for t := range byType {
		types = append(types, int(t))
	}
	sort.Ints(types)
	for _, t := range types {
		ps := byType[uint16(t)]
		fmt.Printf("  type=%-3d : %d paquet(s), tailles=", t, len(ps))
		for i, p := range ps {
			if i < 6 {
				fmt.Printf("%d ", p.size)
			}
		}
		fmt.Println()
	}

	t2 := byType[2]
	if len(t2) == 0 {
		fmt.Println("PAS de paquet type-2 dans chunk_02 — réexamen du framing requis")
		return
	}
	pay := t2[0].payload
	fmt.Printf("\n=== PAYLOAD type-2 #0 : %d bytes (ts=%d) ===\n", len(pay), t2[0].ts)
	fmt.Printf("--- premiers 128 octets ---\n  %s\n", hexdump(pay, 128))

	// ---- hypothèse "compte d'entités en tête" : essaie u16/u32 LE/BE @0 ----
	fmt.Println("--- lecture spéculative d'un en-tête de compte ---")
	if len(pay) >= 4 {
		fmt.Printf("  u16 LE@0 = %d | u16 BE@0 = %d\n",
			binary.LittleEndian.Uint16(pay), binary.BigEndian.Uint16(pay))
		fmt.Printf("  u32 LE@0 = %d | u32 BE@0 = %d\n",
			binary.LittleEndian.Uint32(pay), binary.BigEndian.Uint32(pay))
	}
	// premiers champs en lecture bit MSB-first (id low 13 bits, tag 2 bits, R6…)
	fmt.Println("--- lecture bit MSB-first du début (hypothèses largeurs) ---")
	fmt.Printf("  R(1)=%d R(2)=%d R(6)=%d R(13)=%d\n",
		rb(pay, 0, 1), rb(pay, 0, 2), rb(pay, 0, 6), rb(pay, 0, 13))

	// ---- marqueurs byte-alignés candidats ----
	fmt.Println("\n=== marqueurs byte-alignés ===")
	cands := [][]byte{
		{0xa0, 0x7b, 0x42},
		{0xa0, 0x7b},
		{0x7b, 0x42},
		{0x42, 0xc9, 0x67, 0x9f}, // variant low-32 connu
	}
	for _, c := range cands {
		pos := countMarker(pay, c)
		mn, mx, mean := gaps(pos)
		fmt.Printf("  % x : %d occurrences ; gaps min/mean/max = %d/%d/%d\n",
			c, len(pos), mn, mean, mx)
		if len(pos) > 0 && len(pos) <= 40 {
			fmt.Printf("      offsets = %v\n", pos)
		}
	}

	// ---- histogramme des octets récurrents (cherche un séparateur dense) ----
	fmt.Println("\n=== top octets récurrents (séparateur dense ?) ===")
	var freq [256]int
	for _, b := range pay {
		freq[b]++
	}
	type bf struct {
		b byte
		n int
	}
	var bfs []bf
	for i := 0; i < 256; i++ {
		bfs = append(bfs, bf{byte(i), freq[i]})
	}
	sort.Slice(bfs, func(i, j int) bool { return bfs[i].n > bfs[j].n })
	for i := 0; i < 8; i++ {
		fmt.Printf("  0x%02x : %d (%.1f%%)\n", bfs[i].b, bfs[i].n, 100*float64(bfs[i].n)/float64(len(pay)))
	}

	// ---- test "table (id,typeIndex)" : si une spawn-list existe en tête, les premiers
	//      octets devraient ressembler à une suite régulière de petits typeIndex (<64).
	//      On scanne la densité de valeurs R(6) plausibles en lecture séquentielle naïve.
	fmt.Println("\n=== densité de typeIndex plausibles (<", len(reg.Archetypes), ") en tête (R6 packés) ===")
	br := filmdec.NewBitReader(pay)
	plausible := 0
	for i := 0; i < 64 && br.Remaining() >= 6; i++ {
		v := br.ReadBits(6)
		ok := int(v) < len(reg.Archetypes)
		mark := ""
		if ok {
			plausible++
			mark = " <typeIdx?"
		}
		if i < 32 {
			fmt.Printf("  R6[%d]=%d%s\n", i, v, mark)
		}
	}
	fmt.Printf("  => %d/64 valeurs R6 < nArchetypes (bruit attendu ~%d/64)\n",
		plausible, 64*len(reg.Archetypes)/64)

	// ---- TEST DÉCISIF : le type-2 est-il une cascade de records partageant la
	//      grammaire de la boucle type-0 ? On rejoue le record-loop (prefix-code type
	//      + id table-driven + body) et on observe combien de records on énumère
	//      AVANT desync composant, et quels typeIndex/armes apparaissent. Si l'on
	//      n'avance qu'en décodant les composants -> énumération PAS cheap. ----
	fmt.Println("\n=== rejeu record-loop sur payload type-2 (cfg par défaut) ===")
	w := filmdec.NewWorld(reg)
	br2 := filmdec.NewBitReader(pay)
	recs, err := filmdec.DecodeFrameRecords(br2, w, filmdec.DefaultFrameConfig())
	fmt.Printf("  records énumérés avant arrêt : %d\n", len(recs))
	if err != nil {
		fmt.Printf("  arrêt : %v\n", err)
	}
	nNew, nDel, nDelta, desync := 0, 0, 0, 0
	tiSeen := map[uint32]int{}
	for _, r := range recs {
		switch r.Type {
		case 1:
			nNew++
		case 2:
			nDel++
		case 3:
			nDelta++
		}
		if r.DesyncAt != -1 {
			desync++
		}
		tiSeen[r.TypeIndex]++
	}
	fmt.Printf("  NEW=%d DEL=%d DELTA=%d ; desync=%d ; typeIndex distincts=%d\n",
		nNew, nDel, nDelta, desync, len(tiSeen))
	for i, r := range recs {
		if i >= 12 {
			fmt.Printf("  ... (%d records au total)\n", len(recs))
			break
		}
		fmt.Printf("  R%d type=%d slot=%d typeIdx=%d desyncAt=%d held=0x%x endBit=%d\n",
			i, r.Type, r.Slot, r.TypeIndex, r.DesyncAt, r.Trace.HeldWeapon, r.Trace.EndBit)
	}

	// ---- combien d'armes (variant 0x42c9679f) le payload contient-il au total ? ----
	tot := len(pay) * 8
	hits := 0
	for bp := 0; bp+32 <= tot; bp++ {
		if uint32(rb(pay, bp, 32)) == 0x42c9679f {
			hits++
		}
	}
	fmt.Printf("\n=== variant 0x42c9679f : %d occurrences (n'importe quel alignement bit) ===\n", hits)

	// ---- Hypothèse "cascade de NEW" : un keyframe = état complet, donc chaque entité
	//      devrait être un NEW full-state (typeIndex R6 dans le stream). On force
	//      l'interprétation NEW au 1er record, en balayant un en-tête de longueur
	//      variable (header bits avant le 1er record) pour voir si un offset donne
	//      une cascade de NEW dont le typeIndex est plausible et qui contient une arme.
	fmt.Println("\n=== balayage : 1er record forcé NEW à offset de départ variable ===")
	bestOff, bestRun := -1, 0
	for startBit := 0; startBit <= 64; startBit++ {
		brN := filmdec.NewBitReader(pay)
		brN.Skip(startBit)
		// header NEW = R6 typeIndex + default-state(0) + gate + mask + comps
		tr := filmdec.TraverseEntity(brN, reg, 0)
		run := 0
		if tr.DesyncAt == -1 || tr.DesyncAt > 5 {
			run = tr.DesyncAt
			if tr.DesyncAt == -1 {
				run = 999
			}
		}
		if run > bestRun {
			bestRun, bestOff = run, startBit
		}
		if startBit <= 20 {
			fmt.Printf("  start=%2d : typeIdx=%2d gate=%v maskbits=%d desyncAt=%d held=0x%x\n",
				startBit, tr.TypeIndex, tr.Gate, popcount(tr.Mask), tr.DesyncAt, tr.HeldWeapon)
		}
	}
	fmt.Printf("  meilleur offset NEW = %d (run=%d composants avant desync)\n", bestOff, bestRun)

	// ---- espacement des records d'arme : les 63 variants 0x42c9679f sont-ils
	//      régulièrement espacés (=> table) ou irréguliers (=> état dense) ? ----
	fmt.Println("\n=== espacement (en bits) des 0x42c9679f consécutifs ===")
	var wpos []int
	for bp := 0; bp+32 <= tot; bp++ {
		if uint32(rb(pay, bp, 32)) == 0x42c9679f {
			wpos = append(wpos, bp)
		}
	}
	for i := 1; i < len(wpos) && i <= 20; i++ {
		fmt.Printf("  arme %2d @bit %7d  (gap=%d bits, %d octets)\n",
			i, wpos[i], wpos[i]-wpos[i-1], (wpos[i]-wpos[i-1])/8)
	}
}

func popcount(x uint64) int {
	n := 0
	for x != 0 {
		n += int(x & 1)
		x >>= 1
	}
	return n
}
