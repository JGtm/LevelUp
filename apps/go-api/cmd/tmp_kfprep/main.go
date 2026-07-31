// tmp_kfprep — prépare le terrain du durcissement keyframe + décodage positions.
//
//  1. Inventorie les frames type-2 (keyframes) dans chaque chunk d'un film
//     (offset, taille payload, timestamp).
//  2. Extrait les payloads type-2 bruts de chunk_02/04/06 (000d5950) en .bin
//     pour les décompileurs/implémenteurs (test gen>=2 + champ 26-bit non-nul).
//  3. Dump les offsets bit des records biped (slots 512-519 via l'oracle
//     world_dump) dans chunk_02 : header + début d'état (après header 64b).
//
// Réutilise filmdec.ParseRegistryChunk ; la logique de scan/ancre est reprise de
// cmd/tmp_kftable (tool jumeau, format keyframe type-2 confirmé). Outil OFFLINE.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_kfprep [filmDir] [worldDump] [crackDir]
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

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	defFilm  = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
	defDump  = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/.ai/re_dump/crack/world_dump_000d5950.txt`
	defCrack = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/.ai/re_dump/crack`
	sent     = 0xFFFFFFFF
	tableCap = 8192
)

func inflate(p string) []byte {
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); (e2 == nil || e2 == io.ErrUnexpectedEOF) && len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

type frameHdr struct {
	typ  uint16
	off  int
	size int
	ts   uint64
}

// scanFrames walks a chunk's frame list ([type:u16][?:u16][size:u32][ts:u64][payload]).
func scanFrames(d []byte) []frameHdr {
	var out []frameHdr
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		out = append(out, frameHdr{typ: typ, off: off, size: sz, ts: ts})
		off += 16 + sz
	}
	return out
}

// framePayloadFrom returns the payload of the first frame of type `want` in d.
func framePayloadFrom(d []byte, want uint16) (frameHdr, []byte, bool) {
	for _, f := range scanFrames(d) {
		if f.typ == want {
			return f, d[f.off+16 : f.off+16+f.size], true
		}
	}
	return frameHdr{}, nil, false
}

func readBits(buf []byte, pos, n int) uint64 {
	var r uint64
	for i := 0; i < n; i++ {
		p := pos + i
		var bit uint64
		if idx := p >> 3; idx < len(buf) {
			bit = uint64(buf[idx]>>(7-uint(p&7))) & 1
		}
		r = r<<1 | bit
	}
	return r
}

func bitAt(buf []byte, p int) uint64 {
	if idx := p >> 3; idx < len(buf) {
		return uint64(buf[idx]>>(7-uint(p&7))) & 1
	}
	return 0
}

// findAnchorExact : 1re position bit p >= from où readBits(p,32)==id ET readBits(p+32,32)==ti.
func findAnchorExact(buf []byte, from, id, ti, total int) int {
	if from+64 > total {
		return -1
	}
	target := uint64(uint32(id))<<32 | uint64(uint32(ti))
	acc := readBits(buf, from, 64)
	for p := from; p+64 <= total; p++ {
		if acc == target {
			return p
		}
		acc = acc<<1 | bitAt(buf, p+64)
	}
	return -1
}

func loadOracle(p string) map[int]int {
	f, err := os.Open(p)
	if err != nil {
		return nil
	}
	defer f.Close()
	m := map[int]int{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		for _, tok := range strings.Fields(line) {
			kv := strings.SplitN(tok, ":", 2)
			if len(kv) != 2 {
				continue
			}
			s, e1 := strconv.Atoi(kv[0])
			t, e2 := strconv.Atoi(kv[1])
			if e1 == nil && e2 == nil {
				m[s] = t
			}
		}
	}
	return m
}

func main() {
	dir, dump, crack := defFilm, defDump, defCrack
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if len(os.Args) > 2 {
		dump = os.Args[2]
	}
	if len(os.Args) > 3 {
		crack = os.Args[3]
	}

	// --- 1. Inventaire des keyframes type-2 par chunk ---
	fmt.Printf("=== INVENTAIRE frames (dir=%s) ===\n", dir)
	fmt.Printf("%-10s %-8s %-9s %-12s %-9s %s\n", "chunk", "nFrames", "types", "kf@off", "kfsize", "kfts")
	for i := 0; i < 40; i++ {
		p := fmt.Sprintf("%s/chunk_%02d.bin", dir, i)
		if _, err := os.Stat(p); err != nil {
			continue
		}
		d := inflate(p)
		frames := scanFrames(d)
		if len(frames) == 0 {
			fmt.Printf("chunk_%02d  (pas de frames : registre/annexe, %d o)\n", i, len(d))
			continue
		}
		typeSet := map[uint16]int{}
		for _, f := range frames {
			typeSet[f.typ]++
		}
		var typeStr []string
		var types []int
		for t := range typeSet {
			types = append(types, int(t))
		}
		sort.Ints(types)
		for _, t := range types {
			typeStr = append(typeStr, fmt.Sprintf("%d×%d", typeSet[uint16(t)], t))
		}
		kf, _, has := framePayloadFrom(d, 2)
		kfOff, kfSize, kfTs := "-", "-", "-"
		if has {
			kfOff = strconv.Itoa(kf.off)
			kfSize = strconv.Itoa(kf.size)
			kfTs = strconv.FormatUint(kf.ts, 10)
		}
		fmt.Printf("chunk_%02d  %-8d %-20s %-12s %-9s %s\n", i, len(frames), strings.Join(typeStr, ","), kfOff, kfSize, kfTs)
	}

	// registre (info)
	if reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin")); err == nil {
		fmt.Printf("registre : %d archétypes\n", len(reg.Archetypes))
	}

	// --- 2. Extraction payloads bruts chunk_02/04/06 ---
	oracle := loadOracle(dump)
	writeCrack := oracle != nil // seulement pour le film qui a un oracle (000d5950)
	if writeCrack {
		fmt.Printf("\n=== EXTRACTION payloads type-2 -> %s ===\n", crack)
		for _, ci := range []int{2, 4, 6} {
			p := fmt.Sprintf("%s/chunk_%02d.bin", dir, ci)
			d := inflate(p)
			f, pay, has := framePayloadFrom(d, 2)
			if !has {
				fmt.Printf("chunk_%02d : PAS de type-2\n", ci)
				continue
			}
			outP := fmt.Sprintf("%s/kf_type2_chunk%02d.bin", crack, ci)
			if err := os.WriteFile(outP, pay, 0o644); err != nil {
				fmt.Printf("chunk_%02d : écriture échouée : %v\n", ci, err)
				continue
			}
			fmt.Printf("chunk_%02d : type-2 @off=%d ts=%d, %d o -> %s\n", ci, f.off, f.ts, len(pay), outP)
		}
	}

	// --- 3. Offsets bit des records (biped 512-519) dans chunk_02 ---
	if writeCrack {
		d := inflate(dir + "/chunk_02.bin")
		_, pay, has := framePayloadFrom(d, 2)
		if !has {
			fmt.Println("chunk_02 : type-2 introuvable, pas d'offsets biped")
			return
		}
		total := len(pay) * 8

		slots := make([]int, 0, len(oracle))
		for s := range oracle {
			slots = append(slots, s)
		}
		sort.Ints(slots)

		type rec struct{ slot, ti, hdr int }
		var recs []rec
		var missing []int
		pos := 0
		for _, slot := range slots {
			ti := oracle[slot]
			id := 0x40000000 | slot
			p := findAnchorExact(pay, pos, id, ti, total)
			if p < 0 {
				missing = append(missing, slot)
				continue
			}
			recs = append(recs, rec{slot, ti, p})
			pos = p + 64
		}

		bipeds := map[int]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}
		var b strings.Builder
		fmt.Fprintf(&b, "# Offsets bit des records biped (ti=35, slots 512-519) dans 000d5950/chunk_02 (frame type-2).\n")
		fmt.Fprintf(&b, "# Header keyframe (RE FUN_141f86704) = [id:32][champ:26 (=0 au spawn)][ti:6] = 64 bits.\n")
		fmt.Fprintf(&b, "# id = gen<<30|slot ; au spawn gen==1 -> id=0x40000000|slot. Etat i0 (position) DEBUTE a hdrBit+64.\n")
		fmt.Fprintf(&b, "# Localisation via ancre exacte (id||ti 64-bit), pos monotone en ordre de slot croissant (%d entités oracle).\n", len(slots))
		fmt.Fprintf(&b, "# Colonnes : slot ti hdrBit stateBit(=hdrBit+64) idHex\n")
		fmt.Fprintf(&b, "# payload_bits_total=%d\n", total)
		fmt.Fprintf(&b, "#\n")
		count := 0
		for _, r := range recs {
			if !bipeds[r.slot] {
				continue
			}
			fmt.Fprintf(&b, "%d %d %d %d 0x%08x\n", r.slot, r.ti, r.hdr, r.hdr+64, uint32(0x40000000|r.slot))
			count++
		}
		var bmiss []int
		for s := range bipeds {
			found := false
			for _, r := range recs {
				if r.slot == s {
					found = true
					break
				}
			}
			if !found {
				bmiss = append(bmiss, s)
			}
		}
		sort.Ints(bmiss)
		fmt.Fprintf(&b, "# bipeds_localisés=%d/8 manquants=%v\n", count, bmiss)
		fmt.Fprintf(&b, "# entités_totales_localisées=%d/%d manquantes_totales=%v\n", len(recs), len(slots), missing)

		outP := crack + "/biped_record_offsets.txt"
		if err := os.WriteFile(outP, []byte(b.String()), 0o644); err != nil {
			fmt.Printf("écriture biped offsets échouée : %v\n", err)
			return
		}
		fmt.Printf("\n=== OFFSETS biped -> %s ===\n", outP)
		fmt.Print(b.String())
	}
}
