// tmp_keyframe_sweep — THROWAWAY : valide le décodeur QUANTIFIÉ (param_5=1) sur le
// keyframe type-2 (dense). Trois étapes :
//  1. extrait le payload type-2, mesure densité (ratio de zéros) vs type-1 sparse.
//  2. cherche l'ancre string-id 0x67abd42a à TOUTES les alignements bit.
//  3. sweep (startBit, statWidth) : compte les records DecodeEntityRecordQ
//     consécutifs « plausibles » (variant_name haute-entropie, pas un masque rond).
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"math/bits"
	"os"

	"levelup/go-api/internal/analysis/filmdec"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tmp_keyframe_sweep <chunk.bin> [anchorHex]")
		os.Exit(1)
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	data := raw
	if len(raw) >= 2 && raw[0] == 0x78 {
		zr, _ := zlib.NewReader(bytes.NewReader(raw))
		data, _ = readAll(zr)
	}
	payload := extractPacket(data, 2)
	if payload == nil {
		fmt.Println("pas de paquet type-2")
		return
	}
	fmt.Printf("keyframe type-2 : %d octets\n", len(payload))

	// --- étape 1 : densité ---
	zeros := 0
	for _, b := range payload {
		if b == 0 {
			zeros++
		}
	}
	fmt.Printf("ratio zéros = %.1f%%  (type-1 sparse ~>80%% ; dense attendu <40%%)\n",
		100*float64(zeros)/float64(len(payload)))
	head := payload
	if len(head) > 48 {
		head = head[:48]
	}
	fmt.Printf("head: % x\n\n", head)

	// --- étape 2 : ancre ---
	anchor := uint32(0x67abd42a)
	if len(os.Args) >= 3 {
		var a uint64
		fmt.Sscanf(os.Args[2], "%x", &a)
		anchor = uint32(a)
	}
	hits := findAnchorBits(payload, anchor)
	fmt.Printf("ancre 0x%08x : %d occurrence(s) bit-alignées\n", anchor, len(hits))
	for i, h := range hits {
		if i >= 12 {
			fmt.Printf("  ... (+%d autres)\n", len(hits)-12)
			break
		}
		fmt.Printf("  bit %d (octet %d.%d)\n", h, h/8, h%8)
	}
	fmt.Println()

	// --- étape 3 : sweep ---
	widths := []uint{13, 12, 11, 14, 8}
	maxStart := 16384
	if maxStart > len(payload)*8-64 {
		maxStart = len(payload)*8 - 64
	}
	for _, w := range widths {
		bestRun, bestOff := 0, -1
		for start := 0; start < maxStart; start++ {
			run := decodeRun(payload, start, w, 8)
			if run > bestRun {
				bestRun, bestOff = run, start
			}
		}
		fmt.Printf("statWidth=%-2d : meilleur run=%d @bit %d", w, bestRun, bestOff)
		if bestOff >= 0 && bestRun > 0 {
			fmt.Printf("  variants=%s", sampleVariants(payload, bestOff, w, bestRun))
		}
		fmt.Println()
	}

	// Si l'ancre a été trouvée : tente de verrouiller un record dessus.
	if len(hits) > 0 {
		fmt.Println("\n=== lock sur l'ancre ===")
		for _, w := range widths {
			lockAnchor(payload, hits, anchor, w)
		}
		// Détail du record-ancre + framing avant/après (frontières connues).
		fmt.Println("\n=== détail record-ancre (start=148, w=13) ===")
		printRecordDetail(payload, 148, 13)
		// Tuilage en avant depuis l'ancre : observe la succession des records.
		fmt.Println("\n=== tuilage depuis l'ancre (start=148, w=13) ===")
		tileFrom(payload, 148, 13, 16)
		// Tuilage depuis bit 0 : la keyframe est-elle un flux plat dès le début ?
		fmt.Println("\n=== tuilage depuis bit 0 (w=13) ===")
		tileFrom(payload, 0, 13, 16)
	}
}

// tileFrom décode des records consécutifs et affiche leurs frontières/champs.
func tileFrom(payload []byte, start int, w uint, maxRec int) {
	br := filmdec.NewBitReader(payload)
	br.Skip(start)
	for r := 0; r < maxRec; r++ {
		s := br.BitPos()
		if s+40 > len(payload)*8 {
			break
		}
		ok := func() (survived bool) {
			defer func() {
				if recover() != nil {
					survived = false
				}
			}()
			rec := filmdec.DecodeEntityRecordQ(br, w)
			e := br.BitPos()
			fmt.Printf("  [%2d] %d..%d (len=%3d) var=0x%08x stats=%d binds=%d valid=%v pos=%v plaus=%v\n",
				r, s, e, e-s, rec.VariantName, len(rec.StatChans), len(rec.Bindings),
				rec.Valid, rec.PosValid, plausibleVariant(rec.VariantName))
			return e > s
		}()
		if !ok {
			fmt.Printf("  [%2d] @%d : STOP (panic/no-advance)\n", r, s)
			break
		}
	}
}

// printRecordDetail décode un record et expose tous ses champs + le bit de fin,
// puis dumpe les bits bruts juste avant le start et juste après la fin (pour
// reverser le framing inter-records).
func printRecordDetail(payload []byte, start int, w uint) {
	br := filmdec.NewBitReader(payload)
	br.Skip(start)
	rec := filmdec.DecodeEntityRecordQ(br, w)
	end := br.BitPos()
	fmt.Printf("start=%d end=%d (len=%d bits)\n", start, end, end-start)
	fmt.Printf("  RawFlag=%v ModeFlag=%v HeaderA=0x%02x Field0C=0x%x ID5=0x%x LocalID=0x%x\n",
		rec.RawFlag, rec.ModeFlag, rec.HeaderA, rec.Field0C, rec.ID5, rec.LocalID)
	fmt.Printf("  VariantName=0x%08x B1D=%d Field02=%d Valid=%v\n",
		rec.VariantName, rec.B1D, rec.Field02, rec.Valid)
	fmt.Printf("  statChans=%d bindings=%d posValid=%v\n",
		len(rec.StatChans), len(rec.Bindings), rec.PosValid)
	for i, b := range rec.Bindings {
		fmt.Printf("    bind[%d] hdr4=0x%x present=%v sub=%d idx=%d word16=0x%04x vec=%v\n",
			i, b.Hdr4, b.Present, b.SubVal, b.Index, b.Word16, b.Vec)
	}
	fmt.Printf("  bits AVANT start [%d..%d]: %s\n", start-64, start, dumpBits(payload, start-64, 64))
	fmt.Printf("  bits APRÈS  end  [%d..%d]: %s\n", end, end+96, dumpBits(payload, end, 96))
}

func dumpBits(payload []byte, start, n int) string {
	if start < 0 {
		start = 0
	}
	br := filmdec.NewBitReader(payload)
	br.Skip(start)
	out := ""
	for i := 0; i < n; i++ {
		if i > 0 && i%8 == 0 {
			out += " "
		}
		if br.ReadBit() {
			out += "1"
		} else {
			out += "0"
		}
	}
	return out
}

// decodeRun décode jusqu'à maxRec records consécutifs et renvoie le nombre de
// records « plausibles » d'affilée à partir de startBit.
func decodeRun(payload []byte, startBit int, w uint, maxRec int) int {
	defer func() { _ = recover() }()
	br := filmdec.NewBitReader(payload)
	br.Skip(startBit)
	run := 0
	for r := 0; r < maxRec; r++ {
		before := br.BitPos()
		rec := filmdec.DecodeEntityRecordQ(br, w)
		after := br.BitPos()
		if after <= before || after > len(payload)*8 {
			break
		}
		if rec.Valid && plausibleVariant(rec.VariantName) {
			run++
		} else {
			break
		}
	}
	return run
}

func sampleVariants(payload []byte, startBit int, w uint, n int) string {
	br := filmdec.NewBitReader(payload)
	br.Skip(startBit)
	out := "["
	for r := 0; r < n && r < 5; r++ {
		rec := filmdec.DecodeEntityRecordQ(br, w)
		if r > 0 {
			out += " "
		}
		out += fmt.Sprintf("0x%08x", rec.VariantName)
	}
	return out + "]"
}

// lockAnchor : pour chaque hit, balaie le start du record dans [hit-52, hit-13]
// et garde ceux où variant_name == anchor avec préambule sain.
func lockAnchor(payload []byte, hits []int, anchor uint32, w uint) {
	found := 0
	for _, varBit := range hits {
		lo := varBit - 52
		if lo < 0 {
			lo = 0
		}
		for start := lo; start <= varBit-13; start++ {
			func() {
				defer func() { _ = recover() }()
				br := filmdec.NewBitReader(payload)
				br.Skip(start)
				rec := filmdec.DecodeEntityRecordQ(br, w)
				if rec.VariantName == anchor && rec.Valid {
					next := filmdec.DecodeEntityRecordQ(br, w)
					fmt.Printf("  w=%d start=%d : ANCRE OK  stats=%d binds=%d  next=0x%08x(plausible=%v)\n",
						w, start, len(rec.StatChans), len(rec.Bindings),
						next.VariantName, plausibleVariant(next.VariantName))
					found++
				}
			}()
		}
	}
	if found == 0 {
		fmt.Printf("  w=%d : aucun lock\n", w)
	}
}

// plausibleVariant : un string-id Halo réel a une haute entropie ; on rejette les
// masques ronds 2^N-1 (et leurs compléments) symptomatiques d'un read désaligné
// sur un champ saturé.
func plausibleVariant(v uint32) bool {
	if v == 0 || v == 0xFFFFFFFF {
		return false
	}
	if v&(v+1) == 0 { // 2^N-1 : 0x1FFF, 0x7FFFFFFF, 0xFFFF, ...
		return false
	}
	if nv := ^v; nv&(nv+1) == 0 { // complément 2^N-1 : 0xFFFF0000, 0xFFFFFFFE, ...
		return false
	}
	pc := bits.OnesCount32(v)
	return pc >= 6 && pc <= 26
}

func findAnchorBits(payload []byte, anchor uint32) []int {
	var hits []int
	total := len(payload) * 8
	for start := 0; start+32 <= total; start++ {
		br := filmdec.NewBitReader(payload)
		br.Skip(start)
		if uint32(br.ReadBits(32)) == anchor {
			hits = append(hits, start)
		}
	}
	return hits
}

func extractPacket(data []byte, target uint16) []byte {
	off := 0
	for off+16 <= len(data) {
		typ := binary.LittleEndian.Uint16(data[off:])
		size := int(binary.LittleEndian.Uint32(data[off+4:]))
		if off+16+size > len(data) {
			break
		}
		if typ == target {
			return data[off+16 : off+16+size]
		}
		off += 16 + size
	}
	return nil
}

func readAll(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var out []byte
	buf := make([]byte, 1<<16)
	for {
		n, err := r.Read(buf)
		out = append(out, buf[:n]...)
		if err != nil {
			return out, nil
		}
	}
}
