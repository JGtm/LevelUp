// tmp_entityprobe — THROWAWAY : décode les records d'entité du full-state (type-1)
// d'un chunk film via filmdec.DecodeEntityRecord, et applique la stratégie
// "find first record" (scan d'offsets bit + validation variant_name/Valid).
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"levelup/go-api/internal/analysis/filmdec"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tmp_entityprobe <chunk.bin>")
		os.Exit(1)
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	data := raw
	if len(raw) >= 2 && raw[0] == 0x78 {
		zr, zerr := zlib.NewReader(bytes.NewReader(raw))
		if zerr != nil {
			panic(zerr)
		}
		data, _ = io.ReadAll(zr)
	}

	// Type de paquet cible (defaut 1 = full state). Arg 2 optionnel.
	targetType := uint16(1)
	if len(os.Args) >= 3 {
		var t int
		fmt.Sscanf(os.Args[2], "%d", &t)
		targetType = uint16(t)
	}
	var payload []byte
	off := 0
	for off+16 <= len(data) {
		typ := binary.LittleEndian.Uint16(data[off:])
		size := int(binary.LittleEndian.Uint32(data[off+4:]))
		if off+16+size > len(data) {
			break
		}
		if typ == targetType {
			payload = data[off+16 : off+16+size]
			fmt.Printf("paquet type-%d trouve : payload %d octets\n", targetType, size)
			break
		}
		off += 16 + size
	}
	if payload == nil {
		fmt.Printf("pas de paquet type-%d\n", targetType)
		return
	}

	// Scan des offsets bit de depart : compter les records consecutifs valides.
	plausible := func(v uint32) bool { return v != 0 && v != 0xFFFFFFFF }
	bestOff, bestRun := -1, -1
	for start := 0; start < 256; start++ {
		br := filmdec.NewBitReader(payload)
		br.Skip(start)
		run := 0
		for r := 0; r < 8; r++ {
			rec := filmdec.DecodeEntityRecord(br, true)
			if rec.Valid && plausible(rec.VariantName) && br.BitPos() <= len(payload)*8 {
				run++
			} else {
				break
			}
		}
		if run > bestRun {
			bestRun, bestOff = run, start
		}
	}
	fmt.Printf("meilleur start bit = %d  (run de %d records valides consecutifs)\n\n", bestOff, bestRun)

	// Caracterisation du flux : run-length des variant_name + 1ere apparition de
	// chaque variant distinct, pour reperer la frontiere table/entites.
	br := filmdec.NewBitReader(payload)
	br.Skip(bestOff)
	var prevVar uint32 = 0xdeadbeef
	runStart, runCount := 0, 0
	distinct := map[uint32]int{}
	maxRecords := 4000
	printRun := func(v uint32, startBit, count int) {
		fmt.Printf("  variant 0x%08x : %d records (depuis bit %d)\n", v, count, startBit)
	}
	r := 0
	for ; r < maxRecords; r++ {
		startBit := br.BitPos()
		if startBit+32 > len(payload)*8 {
			break
		}
		rec := filmdec.DecodeEntityRecord(br, true)
		if !rec.Valid || br.BitPos() <= startBit || br.BitPos() > len(payload)*8 {
			fmt.Printf("  [STOP @rec %d bit %d : valid=%v]\n", r, startBit, rec.Valid)
			break
		}
		distinct[rec.VariantName]++
		if rec.VariantName != prevVar {
			if runCount > 0 {
				printRun(prevVar, runStart, runCount)
			}
			prevVar, runStart, runCount = rec.VariantName, startBit, 1
		} else {
			runCount++
		}
	}
	if runCount > 0 {
		printRun(prevVar, runStart, runCount)
	}
	fmt.Printf("\n=== %d records decodes, %d variant_name distincts ===\n", r, len(distinct))
}
